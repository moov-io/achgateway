// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"bytes"
	"os"
	"strings"
	"text/template"
	"time"
)

type FilenameData struct {
	RoutingNumber string

	// GPG is true if the file has been encrypted with GPG
	GPG bool

	// Index is the Nth file uploaded for a shard during a cutoff time
	Index int

	// ShardName is the name of a shard uploading this file
	ShardName string
}

var filenameFunctions template.FuncMap = map[string]interface{}{
	"date": func(pattern string) string {
		return time.Now().Format(pattern)
	},
	"env": func(name string) string {
		return os.Getenv(name)
	},
	"lower": func(s string) string {
		return strings.ToLower(s)
	},
	"upper": func(s string) string {
		return strings.ToUpper(s)
	},
}

func RenderACHFilename(raw string, data FilenameData) (string, error) {
	t, err := template.New(data.RoutingNumber).Funcs(filenameFunctions).Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
