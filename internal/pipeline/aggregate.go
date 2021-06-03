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

package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/alerting"
	"github.com/moov-io/achgateway/internal/audittrail"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/notify"
	"github.com/moov-io/achgateway/internal/output"
	"github.com/moov-io/achgateway/internal/schedule"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/transform"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/base/log"
)

type aggregator struct {
	logger       log.Logger
	shard        service.Shard
	uploadAgents service.UploadAgents

	cutoffs *schedule.CutoffTimes
	merger  XferMerging

	auditStorage          audittrail.Storage
	preuploadTransformers []transform.PreUpload
	outputFormatter       output.Formatter
	alerter               alerting.Alerter
}

func newAggregator(logger log.Logger, shard service.Shard, uploadAgents service.UploadAgents) (*aggregator, error) {
	merger, err := NewMerging(logger, shard, uploadAgents)
	if err != nil {
		return nil, fmt.Errorf("error creating xfer merger: %v", err)
	}

	auditStorage, err := audittrail.NewStorage(shard.Audit)
	if err != nil {
		return nil, err
	}
	logger.Logf("setup %T audit storage", auditStorage)

	preuploadTransformers, err := transform.Multi(logger, shard.PreUpload)
	if err != nil {
		return nil, err
	}
	logger.Logf("setup %#v pre-upload transformers", preuploadTransformers)

	outputFormatter, err := output.NewFormatter(shard.Output)
	if err != nil {
		return nil, err
	}
	logger.Logf("setup %T output formatter", outputFormatter)

	cutoffs, err := schedule.ForCutoffTimes(shard.Cutoffs.Timezone, shard.Cutoffs.Windows)
	if err != nil {
		return nil, fmt.Errorf("error creating cutoffs: %v", err)
	}
	fmt.Printf("cutoffs=%#v\n", cutoffs)

	return &aggregator{
		logger:                logger,
		shard:                 shard,
		uploadAgents:          uploadAgents,
		cutoffs:               cutoffs,
		merger:                merger,
		auditStorage:          auditStorage,
		preuploadTransformers: preuploadTransformers,
		outputFormatter:       outputFormatter,
	}, nil
}

func (xfagg *aggregator) Start(ctx context.Context) {
	for {
		select {
		case tt := <-xfagg.cutoffs.C:
			// process automated cutoff time triggering
			if err := xfagg.withEachFile(tt); err != nil {
				err = xfagg.logger.LogErrorf("merging files: %v", err).Err()

				if xfagg.alerter != nil {
					if err := xfagg.alerter.AlertError(err); err != nil {
						xfagg.logger.LogErrorf("sending alert: %v", err)
					}
				}
			}

		// case waiter := <-xfagg.cutoffs.C: // TODO(adam): manual cutoff trigger
		// 	fmt.Printf("waiter=%#v\n", waiter)

		case <-ctx.Done():
			xfagg.cutoffs.Stop()
			xfagg.Shutdown()
			return
		}
	}
}

func (xfagg *aggregator) Shutdown() {
	xfagg.logger.Log("shutting down xfer aggregation")

	// if xfagg.auditStorage != nil {
	// 	xfagg.auditStorage.Close()
	// }
}

func (xfagg *aggregator) acceptFile(msg incoming.ACHFile) error {
	// TODO(adam): log?
	return xfagg.merger.HandleXfer(msg)
}

func (xfagg *aggregator) withEachFile(when time.Time) error {
	window := when.Format("15:04")
	tzname, _ := when.Zone()
	xfagg.logger.Logf("starting %s %s cutoff window processing", window, tzname)
	defer xfagg.logger.Logf("ended %s %s cutoff window processing", window, tzname)

	processed, err := xfagg.merger.WithEachMerged(xfagg.runTransformers)
	if err != nil {
		xfagg.logger.LogErrorf("ERROR inside WithEachMerged: %v", err)
		return fmt.Errorf("merging ACH files: %v", err)
	}

	fmt.Printf("PROCESSED\n   %#v\n", processed)

	// TODO(adam):
	// err = xfagg.service.MarkTransfersAsProcessed(processed.transferIDs)
	// if err != nil {
	// 	xfagg.logger.LogErrorf("ERROR marking %d transfers as processed: %v", len(processed.transferIDs), err)
	// 	return fmt.Errorf("marking transfers as processed: %v", err)
	// }

	return nil
}

func (xfagg *aggregator) runTransformers(agent upload.Agent, outgoing *ach.File) error {
	result, err := transform.ForUpload(outgoing, xfagg.preuploadTransformers)
	if err != nil {
		return err
	}
	return xfagg.uploadFile(agent, result)
}

func (xfagg *aggregator) uploadFile(agent upload.Agent, res *transform.Result) error {
	if res == nil || res.File == nil {
		return errors.New("uploadFile: nil Result / File")
	}

	data := upload.FilenameData{
		RoutingNumber: res.File.Header.ImmediateDestination,
		GPG:           len(res.Encrypted) > 0,
	}
	filename, err := upload.RenderACHFilename(xfagg.shard.FilenameTemplate(), data)
	if err != nil {
		uploadFilesErrors.With().Add(1)
		return fmt.Errorf("problem rendering filename template: %v", err)
	}

	var buf bytes.Buffer
	if err := xfagg.outputFormatter.Format(&buf, res); err != nil {
		uploadFilesErrors.With().Add(1)
		return fmt.Errorf("problem formatting output: %v", err)
	}

	// Record the file in our audit trail
	if err := xfagg.auditStorage.SaveFile(filename, res.File); err != nil {
		uploadFilesErrors.With().Add(1)
		return fmt.Errorf("problem saving file in audit record: %v", err)
	}

	// Upload our file
	err = agent.UploadFile(upload.File{
		Filename: filename,
		Contents: ioutil.NopCloser(&buf),
	})

	// Send Slack/PD or whatever notifications after the file is uploaded
	if err := xfagg.notifyAfterUpload(filename, res.File, agent, err); err != nil {
		xfagg.logger.LogError(err)
	}

	// record our upload metrics
	if err != nil {
		uploadFilesErrors.With().Add(1)
	} else {
		uploadedFilesCounter.With().Add(1)
	}

	return err
}

func (xfagg *aggregator) notifyAfterUpload(filename string, file *ach.File, agent upload.Agent, uploadErr error) error {
	msg := &notify.Message{
		Direction: notify.Upload,
		Filename:  filename,
		File:      file,
		Hostname:  agent.Hostname(),
	}

	uploadAgent := xfagg.uploadAgents.Find(agent.ID())

	if uploadAgent == nil {
		return fmt.Errorf("no uploadAgent found for id=%s", agent.ID())
	}

	notifier, err := notify.NewMultiSender(xfagg.logger, xfagg.shard.Notifications, uploadAgent.Notifications)
	if err != nil {
		return fmt.Errorf("notify: unable to create multi-sender: %v", err)
	}

	if uploadErr != nil {
		if err := notifier.Critical(msg); err != nil {
			return fmt.Errorf("problem sending critical notification for file=%s: %v", filename, err)
		}
	} else {
		if err := notifier.Info(msg); err != nil {
			return fmt.Errorf("problem sending info notification for file=%s: %v", filename, err)
		}
	}

	return nil
}
