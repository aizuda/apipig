package service

import (
	"testing"
	"time"

	remoteModel "apipig/app/apps/remote-agent/model"
	remoteResp "apipig/app/apps/remote-agent/model/response"
	coreAPI "apipig/core/api"
	"apipig/toolkit/snowflake"
	"github.com/stretchr/testify/require"
)

func TestAgentEventBrokerPublishesToSubscribers(t *testing.T) {
	broker := newAgentEventBroker()
	events, unsubscribe := broker.subscribe()
	defer unsubscribe()

	want := remoteResp.AgentEvent{
		Action: "upsert",
		Agent:  remoteModel.Agent{AgentKey: "node", MODEL: coreAPI.MODEL{ID: snowflake.ID(101)}, Status: remoteModel.AgentStatusOnline},
	}
	broker.publish(want)

	select {
	case got := <-events:
		require.Equal(t, want, got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for agent event")
	}
}
