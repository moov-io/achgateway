// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package output

import (
	"testing"

	"github.com/moov-io/ach-conductor/internal/service"
)

func TestFormatter(t *testing.T) {
	cfg := &service.Output{
		Format: "other",
	}
	enc, err := NewFormatter(cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if enc != nil {
		t.Errorf("unexpected Formatter: %#v", enc)
	}
}
