package service

import (
	"sync"

	remoteResp "apipig/app/apps/remote-agent/model/response"
)

const agentEventBufferSize = 32

type agentEventBroker struct {
	mu          sync.RWMutex
	nextID      uint64
	subscribers map[uint64]chan remoteResp.AgentEvent
}

func newAgentEventBroker() *agentEventBroker {
	return &agentEventBroker{subscribers: make(map[uint64]chan remoteResp.AgentEvent)}
}

func (b *agentEventBroker) subscribe() (<-chan remoteResp.AgentEvent, func()) {
	b.mu.Lock()
	b.nextID++
	id := b.nextID
	channel := make(chan remoteResp.AgentEvent, agentEventBufferSize)
	b.subscribers[id] = channel
	b.mu.Unlock()

	var once sync.Once
	return channel, func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, id)
			b.mu.Unlock()
		})
	}
}

func (b *agentEventBroker) publish(event remoteResp.AgentEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, channel := range b.subscribers {
		select {
		case channel <- event:
		default:
			// A slow client only needs the newest state; never block agent heartbeats.
			select {
			case <-channel:
			default:
			}
			select {
			case channel <- event:
			default:
			}
		}
	}
}
