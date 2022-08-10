// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package httptest

import (
	"bytes"
	"crypto/tls"
	"encoding/pem"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// GrabConnectionCertificates returns a filepath of certificate chain from a given address's
// server. This is useful for adding extra root CA's to network clients
func GrabConnectionCertificates(t *testing.T, addr string) (string, error) {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, nil)
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()

	fd, err := os.CreateTemp("", "conn-certs")
	require.NoError(t, err)

	// Write x509 certs to disk
	certs := conn.ConnectionState().PeerCertificates
	var buf bytes.Buffer
	for i := range certs {
		b := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certs[i].Raw,
		}
		if err := pem.Encode(&buf, b); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(fd.Name(), buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return fd.Name(), nil
}
