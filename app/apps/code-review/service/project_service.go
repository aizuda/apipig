package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	aiService "apipig/app/ai/service"
	reviewModel "apipig/app/apps/code-review/model"
	reviewReq "apipig/app/apps/code-review/model/request"
	reviewResp "apipig/app/apps/code-review/model/response"
	"apipig/core/api/response"
	"apipig/core/db"
	"apipig/global"
	"apipig/toolkit/snowflake"

	"gorm.io/gorm"
)

type ProjectService struct{ vault aiService.CredentialVault }

func (s *ProjectService) Save(params *reviewReq.ProjectSaveParams) (reviewResp.ProjectSaveResult, error) {
	var result reviewResp.ProjectSaveResult
	if params == nil || params.Ctx == nil || params.Project == nil {
		return result, errors.New("项目参数不能为空")
	}
	project := params.Project
	project.Name = strings.TrimSpace(project.Name)
	project.Provider = strings.ToLower(strings.TrimSpace(project.Provider))
	project.RepositoryURL = strings.TrimSpace(project.RepositoryURL)
	project.DefaultBranch = strings.TrimSpace(project.DefaultBranch)
	project.Model = strings.TrimSpace(project.Model)
	if project.Provider == "" {
		project.Provider = "github"
	}
	if project.DefaultBranch == "" {
		project.DefaultBranch = "main"
	}
	if project.Name == "" || project.RepositoryURL == "" || project.AccessTokenID == 0 || project.Model == "" {
		return result, errors.New("名称、仓库地址、AI API 密钥和模型不能为空")
	}
	if project.Provider != "github" && project.Provider != "gitlab" && project.Provider != "gitee" {
		return result, errors.New("provider 仅支持 github、gitlab、gitee")
	}
	parsedURL, err := url.Parse(project.RepositoryURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return result, errors.New("仓库地址必须是有效的 HTTPS URL")
	}
	if project.Status == 0 {
		project.Status = reviewModel.ProjectStatusEnabled
	}
	project.Status = normalizedStatus(project.Status)
	var plaintextSecret string
	if project.ID == 0 {
		project.MODEL = db.NewModel(params.Ctx)
		project.WebhookKey, err = randomSecret(18)
		if err != nil {
			return result, err
		}
		plaintextSecret = strings.TrimSpace(params.WebhookSecret)
		if plaintextSecret == "" {
			plaintextSecret, err = randomSecret(32)
			if err != nil {
				return result, err
			}
		}
		project.WebhookSecret, err = s.vault.Encrypt(plaintextSecret)
		if err != nil {
			return result, err
		}
		if token := strings.TrimSpace(params.RepositoryToken); token != "" {
			project.RepositoryToken, err = s.vault.Encrypt(token)
			if err != nil {
				return result, err
			}
		}
		if err = global.DB.Create(project).Error; err != nil {
			return result, err
		}
	} else {
		var existing reviewModel.Project
		if err = global.DB.First(&existing, project.ID).Error; err != nil {
			return result, err
		}
		project.WebhookKey = existing.WebhookKey
		project.WebhookSecret = existing.WebhookSecret
		project.RepositoryToken = existing.RepositoryToken
		if params.RotateWebhookSecret || strings.TrimSpace(params.WebhookSecret) != "" {
			plaintextSecret = strings.TrimSpace(params.WebhookSecret)
			if plaintextSecret == "" {
				plaintextSecret, err = randomSecret(32)
				if err != nil {
					return result, err
				}
			}
			project.WebhookSecret, err = s.vault.Encrypt(plaintextSecret)
			if err != nil {
				return result, err
			}
		}
		if token := strings.TrimSpace(params.RepositoryToken); token != "" && token != "******" {
			project.RepositoryToken, err = s.vault.Encrypt(token)
			if err != nil {
				return result, err
			}
		}
		if err = global.DB.Model(&existing).Select("name", "provider", "repository_url", "default_branch", "webhook_secret", "repository_token", "access_token_id", "model", "review_prompt", "branch_pattern", "ignore_patterns", "push_enabled", "pull_request_enabled", "status", "remark").Updates(project).Error; err != nil {
			return result, err
		}
		if err = global.DB.First(project, project.ID).Error; err != nil {
			return result, err
		}
	}
	result.Project = sanitizeProject(*project)
	result.WebhookSecret = plaintextSecret
	result.WebhookURL = strings.TrimRight(params.Ctx.BaseURL(), "/") + "/v1/apps/code-review/webhook/" + project.WebhookKey
	return result, nil
}

func (s *ProjectService) Get(id snowflake.ID) (reviewModel.Project, error) {
	var project reviewModel.Project
	err := global.DB.First(&project, id).Error
	return sanitizeProject(project), err
}

func (s *ProjectService) Page(params *reviewReq.ProjectPageParams) (response.PageResult, error) {
	query := global.DB.Model(&reviewModel.Project{})
	if params != nil {
		if keyword := strings.TrimSpace(params.Keyword); keyword != "" {
			like := "%" + keyword + "%"
			query = query.Where("name LIKE ? OR repository_url LIKE ?", like, like)
		}
		if params.Provider != "" {
			query = query.Where("provider = ?", strings.ToLower(params.Provider))
		}
		if params.Status > 0 {
			query = query.Where("status = ?", params.Status)
		}
	}
	var projects []reviewModel.Project
	result, err := db.Page(query.Order("created_at DESC"), pageInfo(params), projects)
	if err == nil {
		if rows, ok := result.Records.([]reviewModel.Project); ok {
			for i := range rows {
				rows[i] = sanitizeProject(rows[i])
			}
			result.Records = rows
		}
	}
	return result, err
}

func (s *ProjectService) Delete(ids []snowflake.ID) (bool, error) {
	if len(ids) == 0 {
		return false, errors.New("请选择要删除的项目")
	}
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id IN ? AND status IN ?", ids, []string{reviewModel.TaskStatusQueued, reviewModel.TaskStatusRunning}).First(&reviewModel.Task{}).Error; err == nil {
			return errors.New("项目存在执行中任务，暂不能删除")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&reviewModel.Project{}).Error
	})
	return err == nil, err
}

func sanitizeProject(project reviewModel.Project) reviewModel.Project {
	project.WebhookSecret = ""
	project.RepositoryToken = ""
	return project
}
func normalizedStatus(status uint) uint {
	if status == reviewModel.ProjectStatusDisabled {
		return status
	}
	return reviewModel.ProjectStatusEnabled
}
func randomSecret(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
func webhookURL(baseURL, key string) string {
	return fmt.Sprintf("%s/v1/apps/code-review/webhook/%s", strings.TrimRight(baseURL, "/"), key)
}
