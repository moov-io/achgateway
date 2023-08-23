// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/util"
	"github.com/moov-io/base"
	"github.com/moov-io/base/docker"
	"github.com/moov-io/base/log"

	"github.com/jlaffaye/ftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"goftp.io/server"
	"goftp.io/server/driver/file"
)

var (
	portSource = rand.NewSource(time.Now().Unix())

	rootFTPPath = filepath.Join("..", "..", "testdata", "ftp-server")
)

func port() int {
	return int(30000 + (portSource.Int63() % 9999))
}

func createTestFTPServer(t *testing.T) (*server.Server, error) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping due to -short")
	}

	// Create the outbound directory, this seems especially flakey in remote CI
	if err := os.MkdirAll(filepath.Join(rootFTPPath, "outbound"), 0777); err != nil {
		t.Fatal(err)
	}

	opts := &server.ServerOpts{
		Auth: &server.SimpleAuth{
			Name:     "moov",
			Password: "password",
		},
		Factory: &file.DriverFactory{
			RootPath: rootFTPPath,
			Perm:     server.NewSimplePerm("test", "test"),
		},
		Hostname: "localhost",
		Port:     port(),
		Logger:   &server.DiscardLogger{},
	}
	svc := server.NewServer(opts)
	if svc == nil {
		return nil, errors.New("nil FTP server")
	}
	if err := util.Timeout(func() error { return svc.ListenAndServe() }, 50*time.Millisecond); err != nil {
		if err == util.ErrTimeout {
			return svc, nil
		}
		return nil, err
	}
	return svc, nil
}

func TestFTPConfig__String(t *testing.T) {
	cfg := &service.FTP{
		Hostname: "host",
		Username: "user",
		Password: "pass",
	}
	if !strings.Contains(cfg.String(), "Password=p**s") {
		t.Error(cfg.String())
	}
}

func createTestFTPConnection(t *testing.T, svc *server.Server) (*ftp.ServerConn, error) {
	t.Helper()
	conn, err := ftp.Dial(
		fmt.Sprintf("localhost:%d", svc.Port),
		ftp.DialWithTimeout(10*time.Second),
	)
	require.NoError(t, err)
	if err := conn.Login("moov", "password"); err != nil {
		t.Fatal(err)
	}
	return conn, nil
}

func TestFTP(t *testing.T) {
	svc, err := createTestFTPServer(t)
	require.NoError(t, err)
	defer svc.Shutdown()

	conn, err := createTestFTPConnection(t, svc)
	require.NoError(t, err)
	defer conn.Quit()

	dir, err := conn.CurrentDir()
	require.NoError(t, err)
	if dir == "" {
		t.Error("empty current dir?!")
	}

	// Change directory
	if err := conn.ChangeDir("scratch"); err != nil {
		t.Error(err)
	}

	// Read a file we know should exist
	resp, err := conn.RetrFrom("existing-file", 0) // offset of 0
	if err != nil {
		t.Error(err)
	}
	bs, _ := io.ReadAll(resp)
	bs = bytes.TrimSpace(bs)
	if !bytes.Equal(bs, []byte("Hello, World!")) {
		t.Errorf("got %q", string(bs))
	}
}

func createTestFTPAgent(t *testing.T) (*server.Server, *FTPTransferAgent) {
	svc, err := createTestFTPServer(t)
	if err != nil {
		return nil, nil
	}

	auth, ok := svc.Auth.(*server.SimpleAuth)
	if !ok {
		t.Errorf("unknown svc.Auth: %T", svc.Auth)
	}
	cfg := &service.UploadAgent{ // these need to match paths at testdata/ftp-srever/
		FTP: &service.FTP{
			Hostname: fmt.Sprintf("%s:%d", svc.Hostname, svc.Port),
			Username: auth.Name,
			Password: auth.Password,
		},
		Paths: service.UploadPaths{
			Inbound:        "inbound",
			Outbound:       "outbound",
			Reconciliation: "reconciliation",
			Return:         "returned",
		},
	}
	agent, err := newFTPTransferAgent(log.NewTestLogger(), cfg)
	if err != nil {
		svc.Shutdown()
		t.Fatalf("problem creating Agent: %v", err)
		return nil, nil
	}
	require.NotNil(t, agent)
	return svc, agent
}

func TestFTPAgent(t *testing.T) {
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	assert.Equal(t, "inbound", agent.InboundPath())
	assert.Equal(t, "outbound", agent.OutboundPath())
	assert.Equal(t, "reconciliation", agent.ReconciliationPath())
	assert.Equal(t, "returned", agent.ReturnPath())
	assert.Contains(t, agent.Hostname(), "localhost:")
}

func TestFTPAgent_Hostname(t *testing.T) {
	tests := []struct {
		desc             string
		agent            Agent
		expectedHostname string
	}{
		{"no FTP config", &FTPTransferAgent{cfg: service.UploadAgent{}}, ""},
		{"returns expected hostname", &FTPTransferAgent{
			cfg: service.UploadAgent{
				FTP: &service.FTP{
					Hostname: "ftp.mybank.com:4302",
				},
			},
		}, "ftp.mybank.com:4302"},
		{"empty hostname", &FTPTransferAgent{
			cfg: service.UploadAgent{
				FTP: &service.FTP{
					Hostname: "",
				},
			},
		}, ""},
	}

	for _, test := range tests {
		assert.Equal(t, test.expectedHostname, test.agent.Hostname(), "Test: "+test.desc)
	}
}

func TestFTP__getInboundFiles(t *testing.T) {
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	filenames, err := agent.GetInboundFiles()
	require.NoError(t, err)
	require.Len(t, filenames, 3)

	for i := range filenames {
		if filenames[i] == "inbound/iat-credit.ach" {
			file, err := agent.ReadFile(filenames[i])
			require.NoError(t, err)

			bs, _ := io.ReadAll(file.Contents)
			bs = bytes.TrimSpace(bs)
			if !strings.HasPrefix(string(bs), "101 121042882 2313801041812180000A094101Bank                   My Bank Name                   ") {
				t.Errorf("got %v", string(bs))
			}
		}
	}

	// make sure we perform the same call and get the same result
	filenames, err = agent.GetInboundFiles()
	require.NoError(t, err)
	require.Len(t, filenames, 3)
	require.ElementsMatch(t, filenames, []string{"inbound/iat-credit.ach", "inbound/cor-c01.ach", "inbound/prenote-ppd-debit.ach"})
}

func TestFTP__getReconciliationFiles(t *testing.T) {
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	filenames, err := agent.GetReconciliationFiles()
	require.NoError(t, err)
	require.Len(t, filenames, 1)
	require.ElementsMatch(t, filenames, []string{"reconciliation/ppd-debit.ach"})

	for i := range filenames {
		if filenames[i] == "reconciliation/ppd-debit.ach" {
			file, err := agent.ReadFile(filenames[i])
			require.NoError(t, err)

			bs, _ := io.ReadAll(file.Contents)
			bs = bytes.TrimSpace(bs)
			if !strings.HasPrefix(string(bs), "5225companyname                         origid    PPDCHECKPAYMT000002080730   1076401250000001") {
				t.Errorf("got %v", string(bs))
			}
		}
	}

	// make sure we perform the same call and get the same result
	filenames, err = agent.GetReconciliationFiles()
	require.NoError(t, err)
	require.ElementsMatch(t, filenames, []string{"reconciliation/ppd-debit.ach"})
}

func TestFTP__getReturnFiles(t *testing.T) {
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	filenames, err := agent.GetReturnFiles()
	require.NoError(t, err)
	require.Len(t, filenames, 1)
	require.Equal(t, "returned/return-WEB.ach", filenames[0])

	// read the returned file and verify its contents
	file, err := agent.ReadFile(filenames[0])
	require.NoError(t, err)

	bs, _ := io.ReadAll(file.Contents)
	bs = bytes.TrimSpace(bs)
	if !strings.HasPrefix(string(bs), "101 091400606 6910001341810170306A094101FIRST BANK & TRUST     ASF APPLICATION SUPERVI        ") {
		t.Errorf("got %v", string(bs))
	}

	// make sure we perform the same call and get the same result
	filenames, err = agent.GetReturnFiles()
	require.NoError(t, err)
	require.Len(t, filenames, 1)
	require.Equal(t, "returned/return-WEB.ach", filenames[0])
}

func TestFTP__uploadFile(t *testing.T) {
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	content := base.ID()
	f := File{
		Filepath: base.ID(),
		Contents: io.NopCloser(strings.NewReader(content)), // random file contents
	}

	// Create outbound directory
	parent := filepath.Join(rootFTPPath, agent.OutboundPath())
	if err := os.MkdirAll(parent, 0777); err != nil {
		t.Fatal(err)
	}

	if err := agent.UploadFile(f); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(agent.OutboundPath(), f.Filepath)

	// manually read file contents
	fd, err := agent.client.Reader(path)
	require.NoError(t, err)

	bs, err := io.ReadAll(fd)
	require.NoError(t, err)
	require.Equal(t, content, string(bs))

	// delete the file
	if err := agent.Delete(path); err != nil {
		t.Fatal(err)
	}

	// get an error with no FTP configs
	agent.cfg.FTP = nil
	if err := agent.UploadFile(f); err == nil {
		t.Error("expected error")
	}
}

func TestFTP__Issue494(t *testing.T) {
	// Issue 494 talks about how readFiles fails when directories exist inside of
	// the return/inbound directories. Let's make a directory inside and verify
	// downloads happen.
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	// Create extra directory
	path := filepath.Join(rootFTPPath, agent.ReturnPath(), "issue494")
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	// Read without an error
	files, err := agent.GetReturnFiles()
	if err != nil {
		t.Error(err)
	}
	if len(files) != 1 {
		t.Errorf("got %d files: %v", len(files), files)
	}
}

func TestFTP__DeleteMissing(t *testing.T) {
	svc, agent := createTestFTPAgent(t)
	defer agent.Close()
	defer svc.Shutdown()

	err := agent.Delete("/missing.txt")
	require.NoError(t, err)
}

func TestFTP_GetReconciliationFiles(t *testing.T) {
	if !docker.Enabled() {
		t.Skip("Docker not enabled")
	}
	if testing.Short() {
		t.Skip("skipping due to -short")
	}

	conf := &service.UploadAgent{
		FTP: &service.FTP{
			Hostname: "localhost:2121",
			Username: "admin",
			Password: "123456",
		},
		Paths: service.UploadPaths{
			Reconciliation: "reconciliation",
		},
	}
	logger := log.NewTestLogger()
	agent, err := newFTPTransferAgent(logger, conf)
	require.NoError(t, err)

	filepaths, err := agent.GetReconciliationFiles()
	require.NoError(t, err)
	require.ElementsMatch(t, filepaths, []string{"reconciliation/ppd-debit.ach"})
}
