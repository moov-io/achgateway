// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package output

import (
	"bytes"
	"fmt"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/transform"
)

type NACHA struct {
	lineEnding string
}

func (n *NACHA) Format(buf *bytes.Buffer, res *transform.Result) error {
	w := ach.NewWriter(buf)
	if n.lineEnding != "" {
		w.LineEnding = n.lineEnding
	}
	if err := w.Write(res.File); err != nil {
		return fmt.Errorf("unable to write Nacha file: %v", err)
	}
	return nil
}
