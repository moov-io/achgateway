// Licensed to The Moov Authors under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package test

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/incoming/odfi"
	"github.com/moov-io/achgateway/internal/incoming/stream/streamtest"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/admin"
	"github.com/moov-io/base/database"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	downloadConf = &service.Config{
		Database: database.DatabaseConfig{
			DatabaseName: "achgateway",
			MySQL: &database.MySQLConfig{
				Address:  "tcp(127.0.0.1:3306)",
				User:     "root",
				Password: "root",
			},
		},
		Inbound: service.Inbound{
			ODFI: &service.ODFIFiles{
				Processors: service.ODFIProcessors{
					Corrections: service.ODFICorrections{
						Enabled: true,
					},
					Reconciliation: service.ODFIReconciliation{
						Enabled:     true,
						PathMatcher: "/reconciliation/",
					},
					Returns: service.ODFIReturns{
						Enabled: true,
					},
				},
				Interval:   10 * time.Minute,
				ShardNames: []string{"testing"},
				Storage: service.ODFIStorage{
					// Directory string // Configured in each test
					CleanupLocalDirectory: false,
					KeepRemoteFiles:       true,
					RemoveZeroByteFiles:   true,
				},
			},
		},
		Sharding: service.Sharding{
			Shards: []service.Shard{
				{
					Name: "testing",
					Cutoffs: service.Cutoffs{
						Timezone: "America/New_York",
						Windows:  []string{"14:30"},
					},
					UploadAgent: "sftp-test",
				},
			},
		},
		Upload: service.UploadAgents{
			Agents: []service.UploadAgent{
				{
					ID: "sftp-test",
					SFTP: &service.SFTP{
						Hostname: "127.0.0.1:2222",
						Username: "demo",
						Password: "password",
					},
					Paths: service.UploadPaths{
						Inbound:        "/inbound/",
						Outbound:       "/outbound/",
						Reconciliation: "/reconciliation/",
						Return:         "/returned/",
					},
				},
			},
		},
	}
)

func TestODFIDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test via -short")
	}

	logger := log.NewDefaultLogger()

	// Setup our config with test specific values
	odfiStorageDir := t.TempDir()
	downloadConf.Inbound.ODFI.Storage.Directory = odfiStorageDir
	auditDir := t.TempDir()
	downloadConf.Inbound.ODFI.Audit = &service.AuditTrail{
		BucketURI: fmt.Sprintf("file://%s", auditDir),
	}

	eventTopic := fmt.Sprintf("mem://%s", t.Name())
	downloadConf.Events = &service.EventsConfig{
		Stream: &service.EventsStream{
			InMem: &service.InMemory{
				URL: eventTopic,
			},
		},
	}
	_, eventSub := streamtest.InmemStream(t)

	// Setup the ODFI Scheduler and trigger it
	emitter, err := events.NewEmitter(logger, downloadConf.Events)
	require.NoError(t, err)

	processorConf := downloadConf.Inbound.ODFI.Processors
	processors := odfi.SetupProcessors(
		odfi.CorrectionEmitter(processorConf.Corrections, emitter),
		odfi.CreditReconciliationEmitter(processorConf.Reconciliation, emitter),
		odfi.ReturnEmitter(processorConf.Returns, emitter),
	)

	scheduler, err := odfi.NewPeriodicScheduler(logger, downloadConf, processors)
	require.NoError(t, err)
	go func() {
		require.NoError(t, scheduler.Start())
	}()
	t.Cleanup(func() { scheduler.Shutdown() })

	adminServer := admin.NewServer(":0")
	go adminServer.Listen()
	defer adminServer.Shutdown()
	scheduler.RegisterRoutes(adminServer)

	// Write empty files to reconciliation folder
	var buf bytes.Buffer
	emptyFilepath := filepath.Join("..", "..", "testdata", "download-test", "reconciliation", "empty.txt")
	info, err := os.Stat(filepath.Join(filepath.Dir(emptyFilepath), "ppd-debit.ach"))
	require.NoError(t, err)
	err = os.WriteFile(emptyFilepath, buf.Bytes(), info.Mode())
	require.NoError(t, err)

	// Trigger inbound processing
	body := strings.NewReader(`{"shardNames":["testing"]}`)
	req, _ := http.NewRequest("PUT", "http://"+adminServer.BindAddr()+"/trigger-inbound", body)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var corrections []models.CorrectionFile
	var reconciliations []models.ReconciliationFile
	var returns []models.ReturnFile
	for {
		ctx, cleanupFunc := context.WithTimeout(context.Background(), 5*time.Second)
		msg, err := eventSub.Receive(ctx)
		if msg != nil {
			msg.Ack()

			evt, err := models.ReadWithOpts(msg.Body, &ach.ValidateOpts{
				AllowMissingFileControl:    true,
				AllowMissingFileHeader:     true,
				AllowUnorderedBatchNumbers: true,
			})
			if err != nil {
				t.Logf("ERROR: %v", err)
			}

			switch v := evt.Event.(type) {
			case *models.CorrectionFile:
				if len(v.Corrections) > 0 {
					corrections = append(corrections, *v)
				}
			case *models.ReconciliationFile:
				if len(v.Reconciliations) > 0 {
					reconciliations = append(reconciliations, *v)
				}
			case *models.ReturnFile:
				if len(v.Returns) > 0 {
					returns = append(returns, *v)
				}
			default:
				t.Errorf("unexpected event %T", evt)
			}

		}
		if err != nil {
			if err == context.DeadlineExceeded {
				cleanupFunc()
				break
			}
			cleanupFunc()
			continue
		}
	}

	// Verify Corrections
	assert.Len(t, corrections, 1)
	if len(corrections) > 0 {
		assert.Equal(t, "cor-c01.ach", corrections[0].Filename)
		assert.Len(t, corrections[0].Corrections, 1)
	}

	// Verify Reconciliations
	assert.Len(t, reconciliations, 1)
	if len(reconciliations) > 0 {
		assert.Equal(t, "ppd-debit.ach", reconciliations[0].Filename)
		assert.Len(t, reconciliations[0].Reconciliations, 1)
	}

	// Verify Returns
	assert.Len(t, returns, 1)
	if len(returns) > 0 {
		assert.Equal(t, "return-WEB.ach", returns[0].Filename)
		assert.Len(t, returns[0].Returns, 2)
	}

	// Check what was downloaded
	filenames := make([]string, 0)
	err = filepath.Walk(odfiStorageDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		subpath, _ := strings.CutPrefix(path, odfiStorageDir)
		filenames = append(filenames, subpath)
		return nil
	})
	require.NoError(t, err)
	containsFilepaths(t, filenames, []string{
		"/inbound/cor-c01.ach", "/inbound/iat-credit.ach",
		"/reconciliation/empty.txt", "/reconciliation/ppd-debit.ach",
		"/returned/return-WEB.ach",
	})

	// Check what was saved in audit trail
	filenames = make([]string, 0)
	err = filepath.Walk(auditDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		subpath, _ := strings.CutPrefix(path, auditDir)
		filenames = append(filenames, subpath)
		return nil
	})
	require.NoError(t, err)

	yymmdd := time.Now().Format("2006-01-02")
	containsFilepaths(t, filenames, []string{
		fmt.Sprintf("/odfi/127.0.0.1:2222/inbound/%s/cor-c01.ach", yymmdd),
		fmt.Sprintf("/odfi/127.0.0.1:2222/inbound/%s/iat-credit.ach", yymmdd),
		fmt.Sprintf("/odfi/127.0.0.1:2222/reconciliation/%s/ppd-debit.ach", yymmdd),
		fmt.Sprintf("/odfi/127.0.0.1:2222/returned/%s/return-WEB.ach", yymmdd),
	})

	// Verify emptyFilepath is deleted
	_, err = os.Stat(emptyFilepath)
	require.ErrorContains(t, err, "no such file or directory")
}

func containsFilepaths(t *testing.T, all, expected []string) {
	t.Helper()

	for i := range all {
		matched := false
		for j := range expected {
			if strings.Contains(all[i], expected[j]) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("%q not found in %v", all[i], expected)
		}
	}
}

func TestContainsFilepath(t *testing.T) {
	containsFilepaths(t,
		[]string{"a/b/c.txt"},
		[]string{"a/b/c.txt", "b/c.txt", "c.txt"})
}
