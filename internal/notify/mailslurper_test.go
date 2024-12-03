// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"net"
	"testing"
	"time"

	"github.com/moov-io/base/docker"
	"github.com/stretchr/testify/require"

	"github.com/ory/dockertest/v3"
)

type mailslurpDeployment struct {
	container *dockertest.Resource
}

func (dep *mailslurpDeployment) SMTPPort() string {
	return dep.container.GetPort("1025/tcp")
}

func (dep *mailslurpDeployment) Close() error {
	return dep.container.Close()
}

func spawnMailslurp(t *testing.T) *mailslurpDeployment {
	if testing.Short() || !docker.Enabled() {
		t.Skip("skipping docker test")
	}

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository:   "oryd/mailslurper",
		Tag:          "latest-smtps",
		ExposedPorts: []string{"1025"},
	})
	require.NoError(t, err)

	dep := &mailslurpDeployment{
		container: container,
	}

	err = pool.Retry(func() error {
		time.Sleep(1 * time.Second)

		conn, err := net.Dial("tcp", "localhost:"+dep.SMTPPort())
		if err != nil {
			return err
		}
		return conn.Close()
	})
	require.NoError(t, err)

	return dep
}
