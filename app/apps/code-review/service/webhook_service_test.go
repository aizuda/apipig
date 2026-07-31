package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	reviewModel "apipig/app/apps/code-review/model"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubPushWebhook(t *testing.T) {
	app := fiber.New()
	app.Post("/hook", func(c *fiber.Ctx) error {
		event, ignored, err := parseGitHubWebhook(reviewModel.Project{PushEnabled: true}, c, c.Body())
		require.NoError(t, err)
		require.False(t, ignored)
		require.Equal(t, "push", event.EventType)
		require.Equal(t, "acme/api", event.RepositoryName)
		require.Equal(t, "https://github.com/acme/api.git", event.RepositoryURL)
		require.Equal(t, "2222222", event.HeadSHA)
		return c.SendStatus(fiber.StatusOK)
	})
	req := newFiberRequest("/hook", `{"before":"1111111","after":"2222222","ref":"refs/heads/main","repository":{"full_name":"acme/api","clone_url":"https://github.com/acme/api.git"},"pusher":{"name":"alice"}}`)
	req.Header.Set("X-GitHub-Event", "push")
	response, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, response.StatusCode)
}

func TestVerifyGitHubWebhook(t *testing.T) {
	app := fiber.New()
	body := []byte(`{"ref":"refs/heads/main"}`)
	app.Post("/hook", func(c *fiber.Ctx) error {
		require.NoError(t, verifyWebhook("github", "secret", c, c.Body()))
		return c.SendStatus(fiber.StatusOK)
	})
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write(body)
	req := newFiberRequest("/hook", string(body))
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	response, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, response.StatusCode)
}

func TestBranchAllowed(t *testing.T) {
	require.True(t, branchAllowed("main,release/*", "refs/heads/main"))
	require.True(t, branchAllowed("main,release/*", "refs/heads/release/1.0"))
	require.False(t, branchAllowed("main,release/*", "refs/heads/feature/demo"))
}
