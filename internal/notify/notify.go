// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"

	"github.com/moov-io/ach"
)

type Direction string

const (
	Upload   Direction = "upload"
	Download Direction = "download"
)

type Message struct {
	Direction Direction
	Filename  string
	File      *ach.File
	Hostname  string

	// Contents will be used instead of the above fields
	Contents string
}

type Sender interface {
	Info(ctx context.Context, msg *Message) error
	Critical(ctx context.Context, msg *Message) error
}
