package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	aiService "apipig/app/ai/service"
	reviewModel "apipig/app/apps/code-review/model"
	reviewResp "apipig/app/apps/code-review/model/response"
	"apipig/core/api"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit"

	"github.com/gofiber/fiber/v2"
)

type WebhookService struct {
	vault  aiService.CredentialVault
	runner *taskRunner
}

type normalizedEvent struct {
	EventKey, EventType, RepositoryName, RepositoryURL, Ref, BaseSHA, HeadSHA, Author, Title string
}

func (s *WebhookService) Handle(projectKey string, c *fiber.Ctx) (reviewResp.WebhookAccepted, error) {
	var result reviewResp.WebhookAccepted
	var project reviewModel.Project
	if err := global.DB.Where("webhook_key = ? AND status = ?", strings.TrimSpace(projectKey), reviewModel.ProjectStatusEnabled).First(&project).Error; err != nil {
		return result, errors.New("评审项目不存在或已禁用")
	}
	secret, err := s.vault.Decrypt(project.WebhookSecret)
	if err != nil {
		return result, err
	}
	body := append([]byte(nil), c.Body()...)
	if err = verifyWebhook(project.Provider, secret, c, body); err != nil {
		return result, err
	}
	event, ignored, err := parseWebhook(project, c, body)
	if err != nil {
		return result, err
	}
	if ignored {
		return reviewResp.WebhookAccepted{Status: reviewModel.TaskStatusIgnored}, nil
	}
	task := reviewModel.Task{
		MODEL:     api.MODEL{ID: db.GetId(), CreatedAt: toolkit.GetNowUnixMilli(), CreatedBy: "git-webhook"},
		ProjectID: project.ID, EventKey: event.EventKey, EventType: event.EventType, Provider: project.Provider,
		RepositoryName: event.RepositoryName, RepositoryURL: event.RepositoryURL, Ref: event.Ref,
		BaseSHA: event.BaseSHA, HeadSHA: event.HeadSHA, Author: event.Author, Title: event.Title,
		Status: reviewModel.TaskStatusQueued,
	}
	if err = global.DB.Create(&task).Error; err != nil {
		var existing reviewModel.Task
		if queryErr := global.DB.Where("project_id = ? AND event_key = ?", project.ID, event.EventKey).First(&existing).Error; queryErr == nil {
			return reviewResp.WebhookAccepted{TaskID: existing.ID.String(), Status: existing.Status, Duplicate: true}, nil
		}
		return result, err
	}
	s.runner.Enqueue(task.ID)
	return reviewResp.WebhookAccepted{TaskID: task.ID.String(), Status: task.Status}, nil
}

func verifyWebhook(providerName, secret string, c *fiber.Ctx, body []byte) error {
	if strings.TrimSpace(secret) == "" {
		return errors.New("项目未配置 WebHook Secret")
	}
	switch providerName {
	case "github", "gitee":
		signature := strings.TrimSpace(c.Get("X-Hub-Signature-256"))
		if signature == "" {
			signature = strings.TrimSpace(c.Get("X-Gitee-Token"))
			if signature != "" && hmac.Equal([]byte(signature), []byte(secret)) {
				return nil
			}
		}
		const prefix = "sha256="
		if !strings.HasPrefix(signature, prefix) {
			return errors.New("WebHook 签名缺失")
		}
		received, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
		if err != nil {
			return errors.New("WebHook 签名格式无效")
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		if !hmac.Equal(received, mac.Sum(nil)) {
			return errors.New("WebHook 签名校验失败")
		}
	case "gitlab":
		if !hmac.Equal([]byte(c.Get("X-Gitlab-Token")), []byte(secret)) {
			return errors.New("WebHook Token 校验失败")
		}
	default:
		return errors.New("不支持的 Git provider")
	}
	return nil
}

func parseWebhook(project reviewModel.Project, c *fiber.Ctx, body []byte) (normalizedEvent, bool, error) {
	if project.Provider == "gitlab" {
		return parseGitLabWebhook(project, c, body)
	}
	return parseGitHubWebhook(project, c, body)
}

func parseGitHubWebhook(project reviewModel.Project, c *fiber.Ctx, body []byte) (normalizedEvent, bool, error) {
	eventName := strings.ToLower(strings.TrimSpace(c.Get("X-GitHub-Event")))
	if eventName == "" {
		eventName = strings.ToLower(strings.TrimSpace(c.Get("X-Gitee-Event")))
	}
	delivery := strings.TrimSpace(c.Get("X-GitHub-Delivery"))
	if delivery == "" {
		delivery = strings.TrimSpace(c.Get("X-Gitee-Delivery"))
	}
	var payload struct {
		Before, After, Ref string
		Deleted            bool
		Sender             struct{ Login, Name string }
		Pusher             struct{ Name string }
		Repository         struct {
			FullName string `json:"full_name"`
			Name     string `json:"name"`
			CloneURL string `json:"clone_url"`
			HTMLURL  string `json:"html_url"`
		}
		PullRequest struct {
			Number int
			Title  string
			User   struct{ Login string }
			Base   struct{ Ref, SHA string }
			Head   struct{ Ref, SHA string }
		} `json:"pull_request"`
		Action string
		Number int
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return normalizedEvent{}, false, errors.New("WebHook JSON 无效")
	}
	repoName := payload.Repository.FullName
	if repoName == "" {
		repoName = payload.Repository.Name
	}
	repoURL := payload.Repository.CloneURL
	if repoURL == "" {
		repoURL = project.RepositoryURL
	}
	switch eventName {
	case "push", "push hook":
		if !project.PushEnabled || payload.Deleted {
			return normalizedEvent{}, true, nil
		}
		if !branchAllowed(project.BranchPattern, payload.Ref) {
			return normalizedEvent{}, true, nil
		}
		key := delivery
		if key == "" {
			key = "push:" + payload.After
		}
		author := payload.Pusher.Name
		if author == "" {
			author = payload.Sender.Login
		}
		return normalizedEvent{EventKey: key, EventType: "push", RepositoryName: repoName, RepositoryURL: repoURL, Ref: payload.Ref, BaseSHA: payload.Before, HeadSHA: payload.After, Author: author, Title: "Push " + payload.Ref}, false, validateSHAs(payload.After)
	case "pull_request", "merge_request", "pull request hook":
		if !project.PullRequestEnabled || !allowedPullRequestAction(payload.Action) {
			return normalizedEvent{}, true, nil
		}
		key := delivery
		if key == "" {
			key = fmt.Sprintf("pr:%d:%s", payload.Number, payload.PullRequest.Head.SHA)
		}
		return normalizedEvent{EventKey: key, EventType: "pull_request", RepositoryName: repoName, RepositoryURL: repoURL, Ref: payload.PullRequest.Head.Ref, BaseSHA: payload.PullRequest.Base.SHA, HeadSHA: payload.PullRequest.Head.SHA, Author: payload.PullRequest.User.Login, Title: payload.PullRequest.Title}, false, validateSHAs(payload.PullRequest.Head.SHA)
	default:
		return normalizedEvent{}, true, nil
	}
}

func parseGitLabWebhook(project reviewModel.Project, c *fiber.Ctx, body []byte) (normalizedEvent, bool, error) {
	eventName := strings.ToLower(strings.TrimSpace(c.Get("X-Gitlab-Event")))
	delivery := strings.TrimSpace(c.Get("X-Gitlab-Event-UUID"))
	var payload struct {
		Before, After, Ref string
		UserName           string `json:"user_name"`
		Project            struct {
			PathWithNamespace string `json:"path_with_namespace"`
			GitHTTPURL        string `json:"git_http_url"`
			WebURL            string `json:"web_url"`
		}
		ObjectAttributes struct {
			IID              int
			Title            string
			Action           string
			SourceBranch     string              `json:"source_branch"`
			LastCommit       string              `json:"last_commit_sha"`
			LastCommitObject struct{ ID string } `json:"last_commit"`
			DiffRefs         struct {
				BaseSHA string `json:"base_sha"`
				HeadSHA string `json:"head_sha"`
			} `json:"diff_refs"`
		} `json:"object_attributes"`
		ObjectKind string `json:"object_kind"`
		User       struct{ Username, Name string }
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return normalizedEvent{}, false, errors.New("WebHook JSON 无效")
	}
	repoURL := payload.Project.GitHTTPURL
	if repoURL == "" {
		repoURL = project.RepositoryURL
	}
	if strings.Contains(eventName, "push") || payload.ObjectKind == "push" {
		if !project.PushEnabled || !branchAllowed(project.BranchPattern, payload.Ref) {
			return normalizedEvent{}, true, nil
		}
		key := delivery
		if key == "" {
			key = "push:" + payload.After
		}
		return normalizedEvent{EventKey: key, EventType: "push", RepositoryName: payload.Project.PathWithNamespace, RepositoryURL: repoURL, Ref: payload.Ref, BaseSHA: payload.Before, HeadSHA: payload.After, Author: payload.UserName, Title: "Push " + payload.Ref}, false, validateSHAs(payload.After)
	}
	if strings.Contains(eventName, "merge request") || payload.ObjectKind == "merge_request" {
		if !project.PullRequestEnabled || !allowedPullRequestAction(payload.ObjectAttributes.Action) {
			return normalizedEvent{}, true, nil
		}
		head := payload.ObjectAttributes.DiffRefs.HeadSHA
		if head == "" {
			head = payload.ObjectAttributes.LastCommitObject.ID
		}
		if head == "" {
			head = payload.ObjectAttributes.LastCommit
		}
		key := delivery
		if key == "" {
			key = fmt.Sprintf("mr:%d:%s", payload.ObjectAttributes.IID, head)
		}
		author := payload.User.Username
		if author == "" {
			author = payload.User.Name
		}
		return normalizedEvent{EventKey: key, EventType: "pull_request", RepositoryName: payload.Project.PathWithNamespace, RepositoryURL: repoURL, Ref: payload.ObjectAttributes.SourceBranch, BaseSHA: payload.ObjectAttributes.DiffRefs.BaseSHA, HeadSHA: head, Author: author, Title: payload.ObjectAttributes.Title}, false, validateSHAs(head)
	}
	return normalizedEvent{}, true, nil
}

func allowedPullRequestAction(action string) bool {
	switch strings.ToLower(action) {
	case "open", "opened", "update", "synchronize", "reopen", "reopened":
		return true
	}
	return false
}
func branchAllowed(patterns, ref string) bool {
	patterns = strings.TrimSpace(patterns)
	if patterns == "" {
		return true
	}
	branch := strings.TrimPrefix(ref, "refs/heads/")
	for _, item := range strings.Split(patterns, ",") {
		item = strings.TrimSpace(item)
		if item == "*" || item == branch || item == ref {
			return true
		}
		if strings.HasSuffix(item, "*") && strings.HasPrefix(branch, strings.TrimSuffix(item, "*")) {
			return true
		}
	}
	return false
}
func validateSHAs(head string) error {
	if strings.TrimSpace(head) == "" || strings.Trim(head, "0") == "" {
		return errors.New("WebHook 缺少有效 head SHA")
	}
	return nil
}
