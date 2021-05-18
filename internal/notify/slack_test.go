// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package notify

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/service"

	"github.com/stretchr/testify/require"

	"github.com/gorilla/mux"
)

func TestSlack(t *testing.T) {
	handler := mux.NewRouter()
	handler.Methods("POST").Path("/webhook").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, _ := ioutil.ReadAll(r.Body)
		if bytes.Contains(bs, []byte(`"text"`)) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})
	svc := httptest.NewServer(handler)
	defer svc.Close()

	cfg := &service.Slack{
		ID:         "testing",
		WebhookURL: svc.URL + "/webhook",
	}
	slack, err := NewSlack(cfg)
	require.NoError(t, err)

	msg := &Message{
		Direction: Download,
		Filename:  "20200529-152259.ach",
		File:      ach.NewFile(),
	}

	if err := slack.Info(msg); err != nil {
		t.Fatal(err)
	}

	if err := slack.Critical(msg); err != nil {
		t.Fatal(err)
	}
}

func TestSlack__marshal(t *testing.T) {
	tests := []struct {
		desc          string
		status        uploadStatus
		msg           *Message
		shouldContain []string
	}{
		{
			desc:   "successful upload with hostname",
			status: success,
			msg: &Message{
				Direction: Upload,
				Filename:  "myfile.txt",
				Hostname:  "ftp.mybank.com:1234",
				File:      ach.NewFile(),
			},
			shouldContain: []string{
				"SUCCESSFUL upload of myfile.txt to ftp.mybank.com:1234",
				"0 entries | Debits: 0 | Credits: 0",
			},
		},
		{
			desc:   "failed upload with hostname",
			status: failed,
			msg: &Message{
				Direction: Upload,
				Filename:  "myfile.txt",
				Hostname:  "ftp.mybank.com:1234",
				File:      ach.NewFile(),
			},
			shouldContain: []string{
				"FAILED upload of myfile.txt to ftp.mybank.com:1234",
			},
		},
		{
			desc:   "successful download",
			status: success,
			msg: &Message{
				Direction: Download,
				Filename:  "myfile.txt",
				Hostname:  "ftp.mybank.com:1234",
				File:      ach.NewFile(),
			},
			shouldContain: []string{
				"SUCCESSFUL download of myfile.txt from ftp.mybank.com:1234 with ODFI server",
			},
		},
		{
			desc:   "failed download",
			status: failed,
			msg: &Message{
				Direction: Download,
				Filename:  "myfile.txt",
				Hostname:  "ftp.mybank.com:1234",
				File:      ach.NewFile(),
			},
			shouldContain: []string{
				"FAILED download of myfile.txt from ftp.mybank.com:1234 with ODFI server",
			},
		},
	}

	for _, test := range tests {
		actual := marshalSlackMessage(test.status, test.msg)
		for i := range test.shouldContain {
			require.Contains(t, actual, test.shouldContain[i])
		}
	}
}
