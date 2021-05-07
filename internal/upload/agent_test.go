// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"testing"
	"time"

	"github.com/moov-io/ach-conductor/internal/service"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	cfg := service.UploadAgents{
		Agents: []service.UploadAgent{
			{
				ID:   "mock",
				Mock: &service.MockAgent{},
			},
		},
	}
	agent, err := New(log.NewNopLogger(), cfg, "mock")
	require.NoError(t, err)

	if aa, ok := agent.(*MockAgent); !ok {
		t.Errorf("unexpected agent: %#v", aa)
	}

	// check Agent was registered
	require.Len(t, createdAgents.agents, 1)

	_, ok := createdAgents.agents[0].(*MockAgent)
	require.True(t, ok)

	// setup a second (retrying) agent
	cfg.Retry = &service.UploadRetry{
		Interval:   1 * time.Second,
		MaxRetries: 3,
	}
	agent, err = New(log.NewNopLogger(), cfg, "mock")
	require.NoError(t, err)

	if aa, ok := agent.(*RetryAgent); !ok {
		t.Errorf("unexpected agent: %#v", agent)
	} else {
		if aa, ok := aa.underlying.(*MockAgent); !ok {
			t.Errorf("unexpected agent: %#v", aa)
		}
	}

	// check Agent was registered
	require.Len(t, createdAgents.agents, 2)

	retr, ok := createdAgents.agents[1].(*RetryAgent)
	require.True(t, ok)

	_, ok = retr.underlying.(*MockAgent)
	require.True(t, ok)
}
