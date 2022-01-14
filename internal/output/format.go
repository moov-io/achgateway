// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package output

import (
	"bytes"
	"errors"
	"strings"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/transform"
)

// Formatter is a structure for encoding an encrypted or plaintext ACH file.
type Formatter interface {
	Format(buf *bytes.Buffer, res *transform.Result) error
}

func NewFormatter(cfg *service.Output) (Formatter, error) {
	if cfg == nil || cfg.Format == "" {
		return &NACHA{}, nil
	}

	format := strings.ToLower(cfg.Format)
	lineEnding := "\n"
	if strings.HasSuffix(format, "-crlf") {
		lineEnding = "\r\n"
	}

	switch {
	case strings.EqualFold(format, "encrypted-bytes"):
		return &Encrypted{}, nil

	case strings.HasPrefix(format, "base64"):
		return &Base64{
			lineEnding: lineEnding,
		}, nil

	case strings.HasPrefix(format, "nacha"):
		return &NACHA{
			lineEnding: lineEnding,
		}, nil
	}
	return nil, errors.New("unknown output format")
}
