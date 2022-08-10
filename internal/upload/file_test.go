// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"io"
	"strings"
	"testing"
)

func TestFile__close(t *testing.T) {
	var f File
	if err := f.Close(); err != nil {
		t.Error(err)
	}

	f.Contents = io.NopCloser(strings.NewReader("test"))
	if err := f.Close(); err != nil {
		t.Error(err)
	}
}
