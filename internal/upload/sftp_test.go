// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

//go:build linux || darwin
// +build linux darwin

package upload

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base"
	"github.com/moov-io/base/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moov-io/base/log"
	"github.com/ory/dockertest/v3"
)

type sftpDeployment struct {
	res   *dockertest.Resource
	agent *SFTPTransferAgent

	dir string // temporary directory
}

// spawnSFTP launches an SFTP Docker image
//
// You can verify this container launches with an ssh command like:
//
//	$ ssh ssh://demo@127.0.0.1:33138 -s sftp
func spawnSFTP(t *testing.T) *sftpDeployment {
	t.Helper()

	if testing.Short() {
		t.Skip("-short flag enabled")
	}
	if !docker.Enabled() {
		t.Skip("Docker not enabled")
	}
	switch runtime.GOOS {
	case "darwin", "linux":
		// continue on with our test
	default:
		t.Skipf("we haven't coded test support for uid/gid extraction on %s", runtime.GOOS)
	}

	// Setup a temp directory for our SFTP instance
	dir, uid, gid := mkdir(t)

	// Start our Docker image
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "atmoz/sftp",
		Tag:        "latest",
		// set user and group to grant write permissions
		Cmd: []string{
			fmt.Sprintf("demo:password:%d:%d:upload", uid, gid),
		},
		Mounts: []string{
			dir + ":/home/demo/upload",
		},
	})
	// Force container to shutdown prior to checking if it failed
	t.Cleanup(func() {
		if resource != nil {
			require.NoError(t, resource.Close())
			pool.Purge(resource)
		}
	})
	require.NoError(t, err)

	addr := "localhost:" + resource.GetPort("22/tcp")

	var agent *SFTPTransferAgent
	for i := 0; i < 10; i++ {
		if agent != nil {
			break
		}
		agent, err = newAgent(addr, "demo", "password", "")
		// Retry after a short sleep
		if agent == nil {
			time.Sleep(250 * time.Millisecond)
		}
	}
	require.NotNil(t, agent)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, agent.Close())
	})

	err = pool.Retry(func() error {
		return agent.Ping()
	})
	require.NoError(t, err)

	return &sftpDeployment{res: resource, agent: agent, dir: dir}
}

func mkdir(t *testing.T) (string, uint32, uint32) {
	dir := t.TempDir()
	fd, err := os.Stat(dir)
	require.NoError(t, err)

	stat, ok := fd.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("unable to stat %s", fd.Name())
	}
	return dir, stat.Uid, stat.Gid
}

func newAgent(host, user, pass, passFile string) (*SFTPTransferAgent, error) {
	cfg := &service.UploadAgent{
		Paths: service.UploadPaths{
			// Our SFTP client inits into '/' with one folder, 'upload', so we need to
			// put files into /upload/ (as an absolute path).
			//
			// Currently it's assumed sub-directories would exist for inbound vs outbound files.
			Inbound:  "/upload/inbound/",
			Outbound: "/upload/",
			Return:   "/upload/returned/",
		},
		SFTP: &service.SFTP{
			Hostname: host,
			Username: user,
		},
	}
	if pass != "" {
		cfg.SFTP.Password = pass
	} else {
		cfg.SFTP.ClientPrivateKey = passFile
	}
	return newSFTPTransferAgent(log.NewTestLogger(), cfg)
}

func cp(from, to string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}
	t, err := os.Create(to)
	if err != nil {
		return err
	}
	_, err = io.Copy(t, f)
	return err
}

func TestSFTP__password(t *testing.T) {
	deployment := spawnSFTP(t)

	err := deployment.agent.Ping()
	require.NoError(t, err)

	ctx := context.Background()
	err = deployment.agent.UploadFile(ctx, File{
		Filepath: "upload.ach",
		Contents: io.NopCloser(strings.NewReader("test data")),
	})
	require.NoError(t, err)

	err = deployment.agent.Delete(ctx, deployment.agent.OutboundPath()+"upload.ach")
	require.NoError(t, err)

	// Inbound files (IAT in our testdata/sftp-server/)
	os.MkdirAll(filepath.Join(deployment.dir, "inbound"), 0777)
	err = cp(
		filepath.Join("..", "..", "testdata", "sftp-server", "inbound", "iat-credit.ach"),
		filepath.Join(deployment.dir, "inbound", "iat-credit.ach"),
	)
	require.NoError(t, err)

	filepaths, err := deployment.agent.GetInboundFiles(ctx)
	require.NoError(t, err)
	require.Len(t, filepaths, 1)
	require.Equal(t, "/upload/inbound/iat-credit.ach", filepaths[0])

	// Return files (WEB in our testdata/sftp-server/)
	os.MkdirAll(filepath.Join(deployment.dir, "returned"), 0777)
	err = cp(
		filepath.Join("..", "..", "testdata", "sftp-server", "returned", "return-WEB.ach"),
		filepath.Join(deployment.dir, "returned", "return-WEB.ach"),
	)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	filepaths, err = deployment.agent.GetReturnFiles(ctx)
	require.NoError(t, err)
	require.Len(t, filepaths, 1)
	require.Equal(t, "/upload/returned/return-WEB.ach", filepaths[0])
}

func TestSFTP__readFilesEmpty(t *testing.T) {
	deployment := spawnSFTP(t)

	err := deployment.agent.Ping()
	require.NoError(t, err)

	// Upload an empty file
	ctx := context.Background()
	filename := base.ID() + ".ach"
	err = deployment.agent.UploadFile(ctx, File{
		Filepath: filename,
		Contents: io.NopCloser(strings.NewReader("")),
	})
	require.NoError(t, err)

	// Read the empty file
	filepaths, err := deployment.agent.readFilepaths(ctx, deployment.agent.OutboundPath())
	require.NoError(t, err)
	require.Len(t, filepaths, 1)
	require.ElementsMatch(t, filepaths, []string{
		filepath.Join(deployment.agent.OutboundPath(), filename),
	})

	file, err := deployment.agent.ReadFile(ctx, filepaths[0])
	require.NoError(t, err)

	bs, err := io.ReadAll(file.Contents)
	require.NoError(t, err)
	require.Equal(t, "", string(bs))

	// read a non-existent directory
	filepaths, err = deployment.agent.readFilepaths(ctx, "/dev/null")
	require.NoError(t, err)
	require.Empty(t, filepaths)
}

func TestSFTP__uploadFile(t *testing.T) {
	deployment := spawnSFTP(t)

	ctx := context.Background()

	err := deployment.agent.Ping()
	require.NoError(t, err)

	// force out OutboundPath to create more directories
	deployment.agent.cfg.Paths.Outbound = filepath.Join("upload", "foo")
	err = deployment.agent.UploadFile(ctx, File{
		Filepath: "upload.ach",
		Contents: io.NopCloser(strings.NewReader("test data")),
	})
	require.NoError(t, err)

	// fail to create the OutboundPath
	deployment.agent.cfg.Paths.Outbound = string(os.PathSeparator) + filepath.Join("home", "bad-path")
	err = deployment.agent.UploadFile(ctx, File{
		Filepath: "upload.ach",
		Contents: io.NopCloser(strings.NewReader("test data")),
	})
	require.Error(t, err)
}

func TestSFTPAgent(t *testing.T) {
	agent := &SFTPTransferAgent{
		cfg: service.UploadAgent{
			Paths: service.UploadPaths{
				Inbound:        "inbound",
				Outbound:       "outbound",
				Reconciliation: "reconciliation",
				Return:         "return",
			},
			SFTP: &service.SFTP{
				Hostname: "sftp.bank.com",
			},
		},
	}

	assert.Equal(t, "inbound", agent.InboundPath())
	assert.Equal(t, "outbound", agent.OutboundPath())
	assert.Equal(t, "reconciliation", agent.ReconciliationPath())
	assert.Equal(t, "return", agent.ReturnPath())
	assert.Equal(t, "sftp.bank.com", agent.Hostname())
}

func TestSFTPAgent_Hostname(t *testing.T) {
	tests := []struct {
		desc             string
		agent            Agent
		expectedHostname string
	}{
		{"no SFTP config", &SFTPTransferAgent{cfg: service.UploadAgent{}}, ""},
		{"returns expected hostname", &SFTPTransferAgent{
			cfg: service.UploadAgent{
				SFTP: &service.SFTP{
					Hostname: "sftp.mybank.com:4302",
				},
			},
		}, "sftp.mybank.com:4302"},
		{"empty hostname", &SFTPTransferAgent{
			cfg: service.UploadAgent{
				SFTP: &service.SFTP{
					Hostname: "",
				},
			},
		}, ""},
	}

	for _, test := range tests {
		assert.Equal(t, test.expectedHostname, test.agent.Hostname(), "Test: "+test.desc)
	}
}

func TestSFTPConfig__String(t *testing.T) {
	cfg := &service.SFTP{
		Hostname:         "host",
		Username:         "user",
		Password:         "pass",
		ClientPrivateKey: "clientPriv",
		HostPublicKey:    "hostPub",
	}
	if !strings.Contains(cfg.String(), "Password=p**s") {
		t.Error(cfg.String())
	}
}

func TestSFTP__Issue494(t *testing.T) {
	// Issue 494 talks about how readFiles fails when directories exist inside of
	// the return/inbound directories. Let's make a directory inside and verify
	// downloads happen.
	deploy := spawnSFTP(t)

	// Create extra directory
	path := filepath.Join(deploy.dir, "returned", "issue494")
	if err := os.MkdirAll(path, 0777); err != nil {
		t.Fatal(err)
	}

	// Verify that dir exists
	_, err := deploy.agent.client.ListFiles(filepath.Join(deploy.agent.ReturnPath(), "issue494"))
	require.NoError(t, err)

	// Read without an error
	ctx := context.Background()
	files, err := deploy.agent.GetReturnFiles(ctx)
	if err != nil {
		t.Error(err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files", len(files))
	}
}

func TestSFTP__DeleteMissing(t *testing.T) {
	deploy := spawnSFTP(t)

	ctx := context.Background()
	err := deploy.agent.Delete(ctx, "/missing.txt")
	require.NoError(t, err)
}

func TestSFTP_GetReconciliationFiles(t *testing.T) {
	if !docker.Enabled() {
		t.Skip("Docker not enabled")
	}
	if testing.Short() {
		t.Skip("skipping due to -short")
	}

	conf := &service.UploadAgent{
		SFTP: &service.SFTP{
			Hostname: "localhost:2222",
			Username: "demo",
			Password: "password",
		},
		Paths: service.UploadPaths{
			Reconciliation: "reconciliation",
		},
	}

	ctx := context.Background()
	logger := log.NewTestLogger()

	t.Run("relative path", func(t *testing.T) {
		agent, err := newSFTPTransferAgent(logger, conf)
		require.NoError(t, err)

		filepaths, err := agent.GetReconciliationFiles(ctx)
		require.NoError(t, err)
		require.ElementsMatch(t, filepaths, []string{"reconciliation/ppd-debit.ach"})
	})

	t.Run("relative path with trailing slash", func(t *testing.T) {
		conf.Paths.Reconciliation = "reconciliation/"

		agent, err := newSFTPTransferAgent(logger, conf)
		require.NoError(t, err)

		filepaths, err := agent.GetReconciliationFiles(ctx)
		require.NoError(t, err)
		require.ElementsMatch(t, filepaths, []string{"reconciliation/ppd-debit.ach"})
	})

	t.Run("root path", func(t *testing.T) {
		conf.Paths.Reconciliation = "/reconciliation"

		agent, err := newSFTPTransferAgent(logger, conf)
		require.NoError(t, err)

		filepaths, err := agent.GetReconciliationFiles(ctx)
		require.NoError(t, err)
		require.ElementsMatch(t, filepaths, []string{"/reconciliation/ppd-debit.ach"})
	})

	t.Run("root path with trailing slash", func(t *testing.T) {
		conf.Paths.Reconciliation = "/reconciliation/"

		agent, err := newSFTPTransferAgent(logger, conf)
		require.NoError(t, err)

		filepaths, err := agent.GetReconciliationFiles(ctx)
		require.NoError(t, err)
		require.ElementsMatch(t, filepaths, []string{"/reconciliation/ppd-debit.ach"})
	})
}
