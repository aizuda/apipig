package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	reviewModel "apipig/app/apps/code-review/model"
	"apipig/global"
)

type diffResult struct {
	Content                            string
	ChangedFiles, Additions, Deletions int
	Truncated                          bool
}

func prepareDiff(ctx context.Context, project reviewModel.Project, task reviewModel.Task, repositoryToken string, maxBytes, maxFiles int) (diffResult, error) {
	root := strings.TrimSpace(globalWorkspaceRoot())
	if root == "" {
		root = os.TempDir()
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return diffResult{}, err
	}
	workspace, err := os.MkdirTemp(root, "apipig-review-*")
	if err != nil {
		return diffResult{}, err
	}
	defer os.RemoveAll(workspace)
	if err = validateRepositoryRemote(ctx, project.RepositoryURL); err != nil {
		return diffResult{}, err
	}
	run := func(args ...string) (string, error) {
		return runGit(ctx, workspace, gitArgsWithAuth(project.Provider, repositoryToken, args...)...)
	}
	if _, err = run("init", "--quiet"); err != nil {
		return diffResult{}, err
	}
	if _, err = run("remote", "add", "origin", project.RepositoryURL); err != nil {
		return diffResult{}, err
	}
	if _, err = run("fetch", "--quiet", "--no-tags", "--force", "--depth=100", "origin", task.HeadSHA); err != nil {
		return diffResult{}, fmt.Errorf("拉取 head 提交失败: %w", err)
	}
	if validBaseSHA(task.BaseSHA) {
		_, _ = run("fetch", "--quiet", "--no-tags", "--force", "--depth=100", "origin", task.BaseSHA)
	}
	if _, err = run("checkout", "--quiet", "--detach", task.HeadSHA); err != nil {
		return diffResult{}, err
	}
	base := task.BaseSHA
	if !validBaseSHA(base) {
		if parent, parentErr := run("rev-parse", task.HeadSHA+"^"); parentErr == nil {
			base = strings.TrimSpace(parent)
		}
	}
	var diff, numstat, names string
	if validBaseSHA(base) {
		diff, err = run("diff", "--no-ext-diff", "--unified=40", base, task.HeadSHA, "--")
		numstat, _ = run("diff", "--numstat", base, task.HeadSHA, "--")
		names, _ = run("diff", "--name-only", base, task.HeadSHA, "--")
	} else {
		diff, err = run("show", "--format=", "--no-ext-diff", "--unified=40", task.HeadSHA, "--")
		numstat, _ = run("show", "--format=", "--numstat", task.HeadSHA, "--")
		names, _ = run("show", "--format=", "--name-only", task.HeadSHA, "--")
	}
	if err != nil {
		return diffResult{}, fmt.Errorf("生成代码差异失败: %w", err)
	}
	changedFiles := nonEmptyLineCount(names)
	additions, deletions := parseNumstat(numstat)
	truncated := false
	if maxFiles > 0 && changedFiles > maxFiles {
		return diffResult{}, fmt.Errorf("变更文件数 %d 超过限制 %d", changedFiles, maxFiles)
	}
	if maxBytes > 0 && len(diff) > maxBytes {
		diff = diff[:maxBytes]
		truncated = true
	}
	return diffResult{Content: diff, ChangedFiles: changedFiles, Additions: additions, Deletions: deletions, Truncated: truncated}, nil
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=http.followRedirects",
		"GIT_CONFIG_VALUE_0=false",
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(redactGitError(message))
	}
	return stdout.String(), nil
}

func gitArgsWithAuth(providerName, token string, args ...string) []string {
	if strings.TrimSpace(token) == "" {
		return args
	}
	header := "Authorization: Bearer " + token
	if providerName == "github" {
		header = "Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
	}
	return append([]string{"-c", "http.extraHeader=" + header}, args...)
}

func validBaseSHA(sha string) bool {
	return validGitObjectID(sha)
}
func parseNumstat(value string) (int, int) {
	additions, deletions := 0, 0
	for _, line := range strings.Split(value, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if n, err := strconv.Atoi(fields[0]); err == nil {
			additions += n
		}
		if n, err := strconv.Atoi(fields[1]); err == nil {
			deletions += n
		}
	}
	return additions, deletions
}
func nonEmptyLineCount(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
func redactGitError(message string) string {
	if len(message) > 1000 {
		return message[:1000]
	}
	return message
}
func globalWorkspaceRoot() string {
	value := strings.TrimSpace(global.CONFIG.CodeReview.WorkspaceRoot)
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}
