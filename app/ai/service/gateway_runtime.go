package service

import (
	"sync"
	"sync/atomic"
	"time"

	"apipig/toolkit/snowflake"
)

// RateLimiter 定义网关请求和 Token 的限流能力。
type RateLimiter interface {
	Allow(key string, rpm, tpm, reservedTokens int) bool
	AddTokens(key string, tokens int)
}

// CircuitBreaker 定义渠道熔断能力。
type CircuitBreaker interface {
	Allow(channelID snowflake.ID) bool
	RecordFailure(channelID snowflake.ID)
	RecordSuccess(channelID snowflake.ID)
}

type atomicWindowCounter struct {
	window  atomic.Int64
	request atomic.Int64
	tokens  atomic.Int64
	resetMu sync.Mutex
}

type localRateLimiter struct {
	counters    sync.Map
	lastCleanup atomic.Int64
	now         func() time.Time
}

func newLocalRateLimiter() *localRateLimiter {
	return &localRateLimiter{now: time.Now}
}

func (l *localRateLimiter) Allow(key string, rpm, tpm, reservedTokens int) bool {
	if rpm <= 0 && tpm <= 0 {
		return true
	}
	nowWindow := l.now().Unix() / 60
	counter := l.counter(key, nowWindow)
	if rpm > 0 && !incrementWithinLimit(&counter.request, int64(rpm)) {
		return false
	}
	if tpm > 0 {
		if counter.tokens.Load() >= int64(tpm) || !addWithinLimit(&counter.tokens, int64(reservedTokens), int64(tpm)) {
			if rpm > 0 {
				counter.request.Add(-1)
			}
			return false
		}
	}
	l.cleanup(nowWindow)
	return true
}

func (l *localRateLimiter) AddTokens(key string, tokens int) {
	if tokens <= 0 {
		return
	}
	nowWindow := l.now().Unix() / 60
	l.counter(key, nowWindow).tokens.Add(int64(tokens))
	l.cleanup(nowWindow)
}

func (l *localRateLimiter) counter(key string, nowWindow int64) *atomicWindowCounter {
	value, _ := l.counters.LoadOrStore(key, &atomicWindowCounter{})
	counter := value.(*atomicWindowCounter)
	if counter.window.Load() == nowWindow {
		return counter
	}
	counter.resetMu.Lock()
	if counter.window.Load() != nowWindow {
		counter.request.Store(0)
		counter.tokens.Store(0)
		counter.window.Store(nowWindow)
	}
	counter.resetMu.Unlock()
	return counter
}

func (l *localRateLimiter) cleanup(nowWindow int64) {
	last := l.lastCleanup.Load()
	if last != 0 && nowWindow-last < 5 {
		return
	}
	if !l.lastCleanup.CompareAndSwap(last, nowWindow) {
		return
	}
	l.counters.Range(func(key, value any) bool {
		if value.(*atomicWindowCounter).window.Load() < nowWindow-1 {
			l.counters.Delete(key)
		}
		return true
	})
}

func incrementWithinLimit(value *atomic.Int64, limit int64) bool {
	for {
		current := value.Load()
		if current+1 > limit {
			return false
		}
		if value.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

func addWithinLimit(value *atomic.Int64, delta, limit int64) bool {
	if delta <= 0 {
		return value.Load() < limit
	}
	for {
		current := value.Load()
		if current+delta > limit {
			return false
		}
		if value.CompareAndSwap(current, current+delta) {
			return true
		}
	}
}

type localBreakerState struct {
	mu         sync.Mutex
	failures   int
	lastFailed time.Time
	openUntil  time.Time
}

type localCircuitBreaker struct {
	states sync.Map
	now    func() time.Time
}

func newLocalCircuitBreaker() *localCircuitBreaker {
	return &localCircuitBreaker{now: time.Now}
}

func (b *localCircuitBreaker) Allow(channelID snowflake.ID) bool {
	value, ok := b.states.Load(channelID)
	if !ok {
		return true
	}
	state := value.(*localBreakerState)
	state.mu.Lock()
	defer state.mu.Unlock()
	now := b.now()
	if state.openUntil.After(now) {
		return false
	}
	if !state.openUntil.IsZero() {
		b.states.Delete(channelID)
	}
	return true
}

func (b *localCircuitBreaker) RecordFailure(channelID snowflake.ID) {
	value, _ := b.states.LoadOrStore(channelID, &localBreakerState{})
	state := value.(*localBreakerState)
	state.mu.Lock()
	defer state.mu.Unlock()
	now := b.now()
	if !state.lastFailed.IsZero() && now.Sub(state.lastFailed) > time.Minute {
		state.failures = 0
	}
	state.failures++
	state.lastFailed = now
	if state.failures >= 5 {
		state.openUntil = now.Add(time.Minute)
	}
}

func (b *localCircuitBreaker) RecordSuccess(channelID snowflake.ID) {
	b.states.Delete(channelID)
}

func (s *GatewayService) ensureRuntimeComponents() {
	s.runtimeOnce.Do(func() {
		if s.limiter == nil {
			s.limiter = newLocalRateLimiter()
		}
		if s.breaker == nil {
			s.breaker = newLocalCircuitBreaker()
		}
	})
}

func (s *GatewayService) allowRate(key string, rpm, tpm, tokens int) bool {
	s.ensureRuntimeComponents()
	return s.limiter.Allow(key, rpm, tpm, tokens)
}

func (s *GatewayService) recordRateTokens(key string, tokens int) {
	s.ensureRuntimeComponents()
	s.limiter.AddTokens(key, tokens)
}

func (s *GatewayService) breakerOpen(channelID snowflake.ID) bool {
	s.ensureRuntimeComponents()
	return !s.breaker.Allow(channelID)
}

func (s *GatewayService) recordBreakerFailure(channelID snowflake.ID) {
	s.ensureRuntimeComponents()
	s.breaker.RecordFailure(channelID)
}

func (s *GatewayService) recordBreakerSuccess(channelID snowflake.ID) {
	s.ensureRuntimeComponents()
	s.breaker.RecordSuccess(channelID)
}
