// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"net"
	"testing"

	"github.com/moov-io/base/docker"
	"github.com/ory/dockertest/v4"
	"github.com/stretchr/testify/require"
)

type mailslurpDeployment struct {
	container dockertest.Resource
}

func (dep *mailslurpDeployment) SMTPPort() string {
	return dep.container.GetPort("1025/tcp")
}

func spawnMailslurp(t *testing.T) *mailslurpDeployment {
	if testing.Short() || !docker.Enabled() {
		t.Skip("skipping docker test")
	}

	pool := dockertest.NewPoolT(t, "")
	container := pool.RunT(t, "oryd/mailslurper",
		dockertest.WithTag("latest-smtps"),
		dockertest.WithoutReuse(),
	)

	dep := &mailslurpDeployment{
		container: container,
	}

	err := pool.Retry(t.Context(), 0, func() error {
		conn, err := net.Dial("tcp", "localhost:"+dep.SMTPPort())
		if err != nil {
			return err
		}
		return conn.Close()
	})
	require.NoError(t, err)

	return dep
}
