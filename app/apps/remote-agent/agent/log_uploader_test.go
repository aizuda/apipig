package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"apipig/toolkit/snowflake"

	"github.com/stretchr/testify/require"
)

func TestMessageUploaderStopsOnPermanentControllerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":"0","data":null,"msg":"assistant message is no longer streaming"}`))
	}))
	defer server.Close()
	client := &Client{
		baseURL: server.URL, httpClient: server.Client(), agentToken: "runtime-token",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	uploader := newMessageUploader(ctx, client, snowflake.ID(100), 1)

	uploader.Append([]byte("partial response"))
	err := uploader.Close(ctx)

	require.EqualError(t, err, "assistant message is no longer streaming")
}

func TestRetryableUploadErrorClassification(t *testing.T) {
	require.True(t, retryableUploadError(context.DeadlineExceeded))
	require.True(t, retryableUploadError(&APIError{Message: "invalid agent token", StatusCode: 200}))
	require.True(t, retryableUploadError(&APIError{Message: "busy", StatusCode: 503}))
	require.False(t, retryableUploadError(&APIError{Message: "invalid message chunk", StatusCode: 200}))
}
