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
	logUploadBatchSize = 50
	logUploadQueueSize = 512
	maxLogChunkBytes   = 32 * 1024
)

type logUploader struct {
	client *Client
	taskID snowflake.ID
	queue  chan remoteReq.TaskLogEntry
	done   chan error
	mu     sync.Mutex
	next   int64
}

func newLogUploader(ctx context.Context, client *Client, taskID snowflake.ID, nextSequence int64) *logUploader {
	if nextSequence <= 0 {
		nextSequence = 1
	}
	uploader := &logUploader{
		client: client, taskID: taskID, queue: make(chan remoteReq.TaskLogEntry, logUploadQueueSize),
		done: make(chan error, 1), next: nextSequence,
	}
	go uploader.run(ctx)
	return uploader
}

func (u *logUploader) Append(stream string, content []byte) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for len(content) > 0 {
		end := logChunkEnd(content)
		entry := remoteReq.TaskLogEntry{Sequence: u.next, Stream: stream, Content: string(content[:end])}
		u.next++
		u.queue <- entry
		content = content[end:]
	}
}

func (u *logUploader) Close(ctx context.Context) error {
	u.mu.Lock()
	close(u.queue)
	u.mu.Unlock()
	select {
	case err := <-u.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (u *logUploader) run(ctx context.Context) {
	batch := make([]remoteReq.TaskLogEntry, 0, logUploadBatchSize)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case entry, ok := <-u.queue:
			if !ok {
				u.done <- u.flush(ctx, batch)
				return
			}
			batch = append(batch, entry)
			if len(batch) >= logUploadBatchSize {
				if err := u.flush(ctx, batch); err != nil {
					u.done <- err
					return
				}
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := u.flush(ctx, batch); err != nil {
					u.done <- err
					return
				}
				batch = batch[:0]
			}
		case <-ctx.Done():
			u.done <- ctx.Err()
			return
		}
	}
}

func (u *logUploader) flush(ctx context.Context, batch []remoteReq.TaskLogEntry) error {
	if len(batch) == 0 {
		return nil
	}
	request := remoteReq.TaskLogUploadRequest{TaskID: u.taskID, Logs: append([]remoteReq.TaskLogEntry(nil), batch...)}
	for {
		err := u.client.UploadLogs(ctx, request)
		if err == nil {
			return nil
		}
		if !sleepContext(ctx, 2*time.Second) {
			return errors.Join(err, ctx.Err())
		}
	}
}

func logChunkEnd(content []byte) int {
	if len(content) <= maxLogChunkBytes {
		return len(content)
	}
	end := maxLogChunkBytes
	for end > 0 && !utf8.RuneStart(content[end]) {
		end--
	}
	if end == 0 {
		return maxLogChunkBytes
	}
	return end
}
