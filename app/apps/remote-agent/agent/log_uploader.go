package agent

import (
	"context"
	"errors"
	"sync"
	"time"
	"unicode/utf8"

	remoteReq "apipig/app/apps/remote-agent/model/request"
	"apipig/toolkit/snowflake"
)

const (
	chunkUploadBatchSize = 50
	chunkUploadQueueSize = 512
	maxMessageChunkBytes = 32 * 1024
)

type messageUploader struct {
	client    *Client
	messageID snowflake.ID
	queue     chan remoteReq.MessageChunkEntry
	ctx       context.Context
	stopped   chan struct{}
	mu        sync.Mutex
	queueOnce sync.Once
	stopOnce  sync.Once
	runErr    error
	next      int64
}

func newMessageUploader(ctx context.Context, client *Client, messageID snowflake.ID, nextSequence int64) *messageUploader {
	if nextSequence <= 0 {
		nextSequence = 1
	}
	uploader := &messageUploader{
		client: client, messageID: messageID, queue: make(chan remoteReq.MessageChunkEntry, chunkUploadQueueSize),
		ctx: ctx, stopped: make(chan struct{}), next: nextSequence,
	}
	go uploader.run(ctx)
	return uploader
}

func (u *messageUploader) Append(content []byte) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for len(content) > 0 {
		end := messageChunkEnd(content)
		entry := remoteReq.MessageChunkEntry{Sequence: u.next, Content: string(content[:end])}
		select {
		case u.queue <- entry:
		case <-u.stopped:
			return
		case <-u.ctx.Done():
			return
		}
		u.next++
		content = content[end:]
	}
}

func (u *messageUploader) Close(ctx context.Context) error {
	u.mu.Lock()
	u.queueOnce.Do(func() { close(u.queue) })
	u.mu.Unlock()
	select {
	case <-u.stopped:
		return u.runErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (u *messageUploader) run(ctx context.Context) {
	batch := make([]remoteReq.MessageChunkEntry, 0, chunkUploadBatchSize)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case entry, ok := <-u.queue:
			if !ok {
				u.finish(u.flush(ctx, batch))
				return
			}
			batch = append(batch, entry)
			if len(batch) >= chunkUploadBatchSize {
				if err := u.flush(ctx, batch); err != nil {
					u.finish(err)
					return
				}
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := u.flush(ctx, batch); err != nil {
					u.finish(err)
					return
				}
				batch = batch[:0]
			}
		case <-ctx.Done():
			u.finish(ctx.Err())
			return
		}
	}
}

func (u *messageUploader) flush(ctx context.Context, batch []remoteReq.MessageChunkEntry) error {
	if len(batch) == 0 {
		return nil
	}
	request := remoteReq.MessageChunkUploadRequest{MessageID: u.messageID, Chunks: append([]remoteReq.MessageChunkEntry(nil), batch...)}
	for {
		err := u.client.UploadChunks(ctx, request)
		if err == nil {
			return nil
		}
		if !retryableUploadError(err) {
			return err
		}
		if !sleepContext(ctx, 2*time.Second) {
			return errors.Join(err, ctx.Err())
		}
	}
}

func (u *messageUploader) finish(err error) {
	u.stopOnce.Do(func() {
		u.runErr = err
		close(u.stopped)
	})
}

func retryableUploadError(err error) bool {
	var apiError *APIError
	if !errors.As(err, &apiError) {
		return true
	}
	if isAuthenticationError(err) {
		return true
	}
	return apiError.StatusCode == 429 || apiError.StatusCode >= 500
}

func messageChunkEnd(content []byte) int {
	if len(content) <= maxMessageChunkBytes {
		return len(content)
	}
	end := maxMessageChunkBytes
	for end > 0 && !utf8.RuneStart(content[end]) {
		end--
	}
	if end == 0 {
		return maxMessageChunkBytes
	}
	return end
}
