package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"apipig/app/ai/model"
)

type AsyncLogOptions struct {
	QueueSize     int
	BatchSize     int
	FlushInterval time.Duration
}

// CallLogSink 定义调用日志写入和关闭行为。
type CallLogSink interface {
	Enqueue(record model.CallLog) error
	Close(ctx context.Context) error
}

type asyncCallLogSink struct {
	repository      GatewayRepository
	optionsProvider func() AsyncLogOptions
	onError         func(error)
	startOnce       sync.Once
	lifecycleMu     sync.RWMutex
	started         atomic.Bool
	closed          atomic.Bool
	queue           chan model.CallLog
	stop            chan struct{}
	done            chan struct{}
}

func newAsyncCallLogSink(repository GatewayRepository, optionsProvider func() AsyncLogOptions, onError func(error)) CallLogSink {
	return &asyncCallLogSink{repository: repository, optionsProvider: optionsProvider, onError: onError}
}

func (s *asyncCallLogSink) Enqueue(record model.CallLog) error {
	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	if s.closed.Load() {
		return errors.New("调用日志队列已关闭")
	}
	s.startOnce.Do(s.start)
	select {
	case s.queue <- record:
		return nil
	default:
		// 队列满时同步回退，保证审计日志不被静默丢弃。
		return s.repository.CreateCallLogs([]model.CallLog{record})
	}
}

func (s *asyncCallLogSink) Close(ctx context.Context) error {
	s.lifecycleMu.Lock()
	if !s.closed.CompareAndSwap(false, true) {
		s.lifecycleMu.Unlock()
		return nil
	}
	if !s.started.Load() {
		s.lifecycleMu.Unlock()
		return nil
	}
	close(s.stop)
	s.lifecycleMu.Unlock()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *asyncCallLogSink) start() {
	options := normalizeAsyncLogOptions(s.optionsProvider())
	s.queue = make(chan model.CallLog, options.QueueSize)
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	s.started.Store(true)
	go s.run(options)
}

func (s *asyncCallLogSink) run(options AsyncLogOptions) {
	defer close(s.done)
	ticker := time.NewTicker(options.FlushInterval)
	defer ticker.Stop()
	batch := make([]model.CallLog, 0, options.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := s.repository.CreateCallLogs(batch); err != nil {
			for _, record := range batch {
				if fallbackErr := s.repository.CreateCallLogs([]model.CallLog{record}); fallbackErr != nil && s.onError != nil {
					s.onError(fallbackErr)
				}
			}
		}
		batch = batch[:0]
	}
	for {
		select {
		case record := <-s.queue:
			batch = append(batch, record)
			if len(batch) >= options.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.stop:
			for {
				select {
				case record := <-s.queue:
					batch = append(batch, record)
				default:
					flush()
					return
				}
			}
		}
	}
}

func normalizeAsyncLogOptions(options AsyncLogOptions) AsyncLogOptions {
	if options.QueueSize <= 0 {
		options.QueueSize = 2048
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 100
	}
	if options.FlushInterval <= 0 {
		options.FlushInterval = time.Second
	}
	return options
}
