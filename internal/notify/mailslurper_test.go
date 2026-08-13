// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
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

	smtpPort := network.MustParsePort("1025/tcp")
	pool := dockertest.NewPoolT(t, "")
	container := pool.RunT(t, "oryd/mailslurper",
		dockertest.WithTag("latest-smtps"),
		dockertest.WithoutReuse(),
		// The image does not EXPOSE 1025, so PublishAllPorts will not bind it
		// unless we mark the port exposed the same way v3 ExposedPorts did.
		dockertest.WithContainerConfig(func(cfg *container.Config) {
			if cfg.ExposedPorts == nil {
				cfg.ExposedPorts = network.PortSet{}
			}
			cfg.ExposedPorts[smtpPort] = struct{}{}
		}),
	)

	dep := &mailslurpDeployment{
		container: container,
	}

	// Docker publishes the port before mailslurper is accepting SMTPS.
	// A TCP dial succeeds via the proxy and the later send then fails with EOF.
	err := pool.Retry(t.Context(), 2*time.Minute, func() error {
		port := dep.SMTPPort()
		if port == "" {
			return fmt.Errorf("smtp port not published")
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", port), 2*time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

		tlsConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS12,
			ServerName:         "localhost",
		})
		if err := tlsConn.Handshake(); err != nil {
			return err
		}
		buf := make([]byte, 64)
		n, err := tlsConn.Read(buf)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("empty smtp banner")
		}
		return tlsConn.Close()
	})
	require.NoError(t, err)

	return dep
}
