// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moov-io/achgateway/internal/service"

	"github.com/stretchr/testify/require"

	"github.com/moov-io/ach"
)

func TestEmailSend(t *testing.T) {
	dep := spawnMailslurp(t)

	cfg := &service.Email{
		ID:   "testing",
		From: "noreply@moov.io",
		To: []string{
			"jane@company.com",
		},
		ConnectionURI: fmt.Sprintf("smtps://test:test@localhost:%s/?insecure_skip_verify=true", dep.SMTPPort()),
		CompanyName:   "Moov",
	}

	dialer, err := setupGoMailClient(cfg)
	require.NoError(t, err)
	// Enable SSL for our test container, this breaks if set for production SMTP server.
	// GMail fails to connect if we set this.
	dialer.SSL = strings.HasPrefix(cfg.ConnectionURI, "smtps://")

	msg := &Message{
		Direction: Upload,
		Filename:  "20200529-131400.ach",
		File:      ach.NewFile(),
	}

	body, err := marshalEmail(cfg, msg, true)
	require.NoError(t, err)

	ctx := context.Background()
	if err := sendEmail(ctx, cfg, dialer, msg.Filename, body); err != nil {
		t.Fatal(err)
	}

	dep.Close() // remove container after successful tests
}

func TestEmail__marshalSubject(t *testing.T) {
	cfg := &service.Email{CompanyName: "Moov"}
	msg := &Message{Direction: Upload}
	filename := "20200529-131400.ach"

	expected := "20200529-131400.ach uploaded by Moov"
	output := marshalSubject(cfg, msg, filename, true)
	require.Equal(t, expected, output)

	expected = "20200529-131400.ach FAILED upload by Moov"
	output = marshalSubject(cfg, msg, filename, false)
	require.Equal(t, expected, output)
}

func TestEmail__marshalDefaultTemplate(t *testing.T) {
	f, err := ach.ReadFile(filepath.Join("..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)

	tests := []struct {
		desc string
		msg  *Message

		successLine string
		failureLine string
	}{
		{
			desc: "upload with hostname",
			msg: &Message{
				Direction: Upload,
				File:      f,
				Filename:  "20200529-131400.ach",
				Hostname:  "ftp.bank.com:3294",
			},
			successLine: "A file has been uploaded to ftp.bank.com:3294 - 20200529-131400.ach",
			failureLine: "File upload of 20200529-131400.ach FAILED to upload to ftp.bank.com:3294",
		},
		{
			desc: "upload with no hostname",
			msg: &Message{
				Direction: Upload,
				File:      f,
				Filename:  "20200529-131400.ach",
			},
			successLine: "A file has been uploaded - 20200529-131400.ach",
			failureLine: "File upload of 20200529-131400.ach FAILED to upload",
		},
		{
			desc: "download with hostname",
			msg: &Message{
				Direction: Download,
				File:      f,
				Filename:  "20200529-131400.ach",
				Hostname:  "138.34.204.3",
			},
			successLine: "A file has been downloaded from 138.34.204.3 - 20200529-131400.ach",
			failureLine: "File upload of 20200529-131400.ach FAILED to download from 138.34.204.3",
		},
		{
			desc: "download",
			msg: &Message{
				Direction: Download,
				File:      f,
				Filename:  "20200529-131400.ach",
			},
			successLine: "A file has been downloaded - 20200529-131400.ach",
			failureLine: "File upload of 20200529-131400.ach FAILED to download",
		},
	}

	cfg := &service.Email{
		CompanyName: "Moov",
	}

	for _, test := range tests {
		// Simulate .Info()
		contents, err := marshalEmail(cfg, test.msg, true)
		if err != nil {
			t.Fatal(err)
		}

		if testing.Verbose() {
			t.Logf("Info:\n%s", contents)
		}

		require.Contains(t, contents, test.successLine, "Test: "+test.desc)
		require.Contains(t, contents, "Moov")
		require.Contains(t, contents, `Debits:  $105.00`, "Test: "+test.desc)
		require.Contains(t, contents, `Credits: $0.00`, "Test: "+test.desc)
		require.Contains(t, contents, `Batches: 1`, "Test: "+test.desc)
		require.Contains(t, contents, `Total Entries: 1`, "Test: "+test.desc)

		// Simulate .Critical()
		contents, err = marshalEmail(cfg, test.msg, false)
		if err != nil {
			t.Fatal(err)
		}

		if testing.Verbose() {
			t.Logf("Critical:\n%s", contents)
		}

		require.Contains(t, contents, test.failureLine, "Test: "+test.desc)
		require.Contains(t, contents, "Moov")
		require.Contains(t, contents, `Debits:  $105.00`, "Test: "+test.desc)
		require.Contains(t, contents, `Credits: $0.00`, "Test: "+test.desc)
		require.Contains(t, contents, `Batches: 1`, "Test: "+test.desc)
		require.Contains(t, contents, `Total Entries: 1`, "Test: "+test.desc)
	}
}
