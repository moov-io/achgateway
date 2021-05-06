// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package httptest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrabConnectionCertificates(t *testing.T) {
	if testing.Short() {
		return
	}

	path, err := GrabConnectionCertificates(t, "google.com:443")
	require.NoError(t, err)
	defer os.Remove(path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	if info.Size() == 0 {
		t.Fatalf("%s is an empty file", info.Name())
	}
}
