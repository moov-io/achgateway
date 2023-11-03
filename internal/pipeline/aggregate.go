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
	"io"
	"os"
	"strings"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/alerting"
	"github.com/moov-io/achgateway/internal/audittrail"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/notify"
	"github.com/moov-io/achgateway/internal/output"
	"github.com/moov-io/achgateway/internal/schedule"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/internal/transform"
	"github.com/moov-io/achgateway/internal/upload"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base"
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
	"github.com/moov-io/base/strx"
	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type aggregator struct {
	logger       log.Logger
	eventEmitter events.Emitter
	shard        service.Shard
	uploadAgents service.UploadAgents

	cutoffs       *schedule.CutoffTimes
	cutoffTrigger chan manuallyTriggeredCutoff
	merger        XferMerging

	auditStorage          audittrail.Storage
	preuploadTransformers []transform.PreUpload
	outputFormatter       output.Formatter
	alerters              alerting.Alerters
}

func newAggregator(logger log.Logger, eventEmitter events.Emitter, shard service.Shard, uploadAgents service.UploadAgents, errorAlerting service.ErrorAlerting) (*aggregator, error) {
	merger, err := NewMerging(logger, shard, uploadAgents)
	if err != nil {
		return nil, fmt.Errorf("error creating xfer merger: %v", err)
	}

	auditStorage, err := audittrail.NewStorage(shard.Audit)
	if err != nil {
		return nil, err
	}
	logger.Info().With(log.Fields{
		"shard": log.String(shard.Name),
	}).Logf("setup %T audit storage", auditStorage)

	preuploadTransformers, err := transform.Multi(logger, shard.PreUpload)
	if err != nil {
		return nil, err
	}
	logger.Info().With(log.Fields{
		"shard": log.String(shard.Name),
	}).Logf("setup %#v pre-upload transformers", preuploadTransformers)

	outputFormatter, err := output.NewFormatter(shard.Output)
	if err != nil {
		return nil, err
	}
	logger.Info().With(log.Fields{
		"shard": log.String(shard.Name),
	}).Logf("setup %T output formatter", outputFormatter)

	timeService := stime.NewSystemTimeService()
	cutoffs, err := schedule.ForCutoffTimes(timeService, shard.Cutoffs.Timezone, shard.Cutoffs.Windows)
	if err != nil {
		return nil, fmt.Errorf("error creating cutoffs: %v", err)
	}

	alerters, err := alerting.NewAlerters(errorAlerting)
	if err != nil {
		return nil, fmt.Errorf("error setting up alerters: %v", err)
	}

	return &aggregator{
		logger:                logger,
		eventEmitter:          eventEmitter,
		shard:                 shard,
		uploadAgents:          uploadAgents,
		cutoffs:               cutoffs,
		cutoffTrigger:         make(chan manuallyTriggeredCutoff, 1),
		merger:                merger,
		auditStorage:          auditStorage,
		preuploadTransformers: preuploadTransformers,
		outputFormatter:       outputFormatter,
		alerters:              alerters,
	}, nil
}

func (xfagg *aggregator) Start(ctx context.Context) {
	for {
		select {
		// process automated cutoff time triggering
		case day := <-xfagg.cutoffs.C:
			// Run our regular routines
			if day.IsBankingDay {
				if err := xfagg.withEachFile(day.Time); err != nil {
					err = xfagg.logger.Error().LogErrorf("merging files: %v", err).Err()
					xfagg.alertOnError(err)
				}
			}
			if day.IsHoliday && !day.IsWeekend {
				xfagg.notifyAboutHoliday(day)
			}

		// manually trigger cutoffs
		case waiter := <-xfagg.cutoffTrigger:
			xfagg.manualCutoff(waiter)

		case <-ctx.Done():
			xfagg.cutoffs.Stop()
			xfagg.Shutdown()
			return
		}
	}
}

func (xfagg *aggregator) Shutdown() {
	xfagg.logger.Info().With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	}).Log("shutting down xfer aggregation")

	if xfagg.auditStorage != nil {
		xfagg.auditStorage.Close()
	}
}

func (xfagg *aggregator) acceptFile(ctx context.Context, msg incoming.ACHFile) error {
	return xfagg.merger.HandleXfer(ctx, msg)
}

func (xfagg *aggregator) cancelFile(ctx context.Context, msg incoming.CancelACHFile) error {
	return xfagg.merger.HandleCancel(ctx, msg)
}

func (xfagg *aggregator) withEachFile(when time.Time) error {
	window := when.Format("15:04")
	tzname, _ := when.Zone()

	logger := xfagg.logger.With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	})
	logger.Info().Logf("starting %s %s cutoff window processing", window, tzname)

	defer logger.With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	}).Logf("ended %s %s cutoff window processing", window, tzname)

	ctx, span := telemetry.StartSpan(context.Background(), "automated-cutoff", trace.WithAttributes(
		attribute.String("shard", xfagg.shard.Name),
		attribute.String("timezone", tzname),
		attribute.String("window", window),
	))
	defer span.End()

	processed, err := xfagg.merger.WithEachMerged(ctx, xfagg.runTransformers)
	if err != nil {
		logger.LogErrorf("ERROR inside WithEachMerged: %v", err)
		return fmt.Errorf("merging ACH files: %v", err)
	}

	if err := xfagg.emitFilesUploaded(ctx, processed); err != nil {
		logger.LogErrorf("ERROR sending files uploaded event: %v", err)
	}

	return nil
}

func (xfagg *aggregator) manualCutoff(waiter manuallyTriggeredCutoff) {
	logger := xfagg.logger.With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	})
	logger.Info().Log("starting manual cutoff window processing")

	ctx, span := telemetry.StartSpan(context.Background(), "manual-cutoff", trace.WithAttributes(
		attribute.String("shard", xfagg.shard.Name),
	))
	defer span.End()

	if processed, err := xfagg.merger.WithEachMerged(ctx, xfagg.runTransformers); err != nil {
		logger.LogErrorf("ERROR inside manual WithEachMerged: %v", err)
		waiter.C <- err
	} else {
		// Publish event of File uploads
		if err := xfagg.emitFilesUploaded(ctx, processed); err != nil {
			logger.LogErrorf("ERROR sending manual files uploaded event: %v", err)
		}
		waiter.C <- err
	}

	logger.Info().With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	}).Log("ended manual cutoff window processing")
}

func (xfagg *aggregator) emitFilesUploaded(ctx context.Context, proc *processedFiles) error {
	var el base.ErrorList
	for i := range proc.fileIDs {
		err := xfagg.eventEmitter.Send(ctx, models.Event{
			Event: models.FileUploaded{
				FileID:     proc.fileIDs[i],
				ShardKey:   proc.shardKey,
				UploadedAt: time.Now(),
			},
		})
		if err != nil {
			el.Add(err)
		}
	}
	if el.Empty() {
		return nil
	}
	return el
}

func (xfagg *aggregator) runTransformers(ctx context.Context, index int, agent upload.Agent, outgoing *ach.File) error {
	result, err := transform.ForUpload(outgoing, xfagg.preuploadTransformers)
	if err != nil {
		return err
	}
	return xfagg.uploadFile(ctx, index, agent, result)
}

func (xfagg *aggregator) uploadFile(ctx context.Context, index int, agent upload.Agent, res *transform.Result) error {
	if res == nil || res.File == nil {
		return errors.New("uploadFile: nil Result / File")
	}

	data := upload.FilenameData{
		RoutingNumber: res.File.Header.ImmediateDestination,
		GPG:           len(res.Encrypted) > 0,
		ShardName:     prepareShardName(xfagg.shard.Name),
		Index:         index,
	}
	filename, err := upload.RenderACHFilename(xfagg.shard.FilenameTemplate(), data)
	if err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
		return fmt.Errorf("problem rendering filename template: %v", err)
	}

	telemetry.AddEvent(ctx, "prepare-file", trace.WithAttributes(
		attribute.String("filename", filename),
		attribute.String("shard", xfagg.shard.Name),
	))

	var buf bytes.Buffer
	if err := xfagg.outputFormatter.Format(&buf, res); err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
		return fmt.Errorf("problem formatting output: %v", err)
	}

	// Record the file in our audit trail
	basePath := "outbound"
	if xfagg.shard.Audit != nil {
		basePath = strx.Or(xfagg.shard.Audit.BasePath, basePath)
	}
	path := fmt.Sprintf("%s/%s/%s/%s", basePath, agent.Hostname(), time.Now().Format("2006-01-02"), filename)
	if err := xfagg.auditStorage.SaveFile(ctx, path, buf.Bytes()); err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
		return fmt.Errorf("problem saving file in audit record: %v", err)
	}

	// Upload our file
	err = agent.UploadFile(ctx, upload.File{
		Filepath: filename,
		Contents: io.NopCloser(&buf),
	})

	telemetry.AddEvent(ctx, "uploaded-file", trace.WithAttributes(
		attribute.String("filename", filename),
		attribute.String("shard", xfagg.shard.Name),
	))

	// Send Slack/PD or whatever notifications after the file is uploaded
	if err := xfagg.notifyAfterUpload(filename, res.File, agent, err); err != nil {
		xfagg.alertOnError(xfagg.logger.LogError(err).Err())
	}

	// record our upload metrics
	if err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
	} else {
		uploadedFilesCounter.With("shard", xfagg.shard.Name).Add(1)
	}

	return err
}

func prepareShardName(shardName string) string {
	return strings.ToUpper(strings.ReplaceAll(shardName, " ", "-"))
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

	logger := xfagg.logger.With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	})

	notifier, err := notify.NewMultiSender(logger, xfagg.shard.Notifications, uploadAgent.Notifications)
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

func (xfagg *aggregator) notifyAboutHoliday(day *schedule.Day) {
	logger := xfagg.logger.With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	})

	if !day.FirstWindow {
		logger.Info().Log("skipping holiday notification")
		return
	}

	uploadAgent := xfagg.uploadAgents.Find(xfagg.shard.UploadAgent)
	if uploadAgent == nil {
		logger.Warn().Logf("skipping holiday log for %v", day.Time.Format("2006-01-02"))
		return
	}

	if uploadAgent.Notifications != nil {
		slackConfigs := xfagg.shard.Notifications.FindSlacks(uploadAgent.Notifications.Slack)
		for i := range slackConfigs {
			ss, err := notify.NewSlack(&slackConfigs[i])
			if err != nil {
				logger.Error().LogErrorf("ERROR creating slack holiday notifier: %v", err)
				continue
			}

			err = ss.Info(&notify.Message{
				Contents: formatHolidayMessage(day),
			})
			if err != nil {
				logger.Error().LogErrorf("ERROR sending holiday notification: %v", err)
			} else {
				logger.Info().Log("sent holiday notification")
			}
		}
	}
}

func formatHolidayMessage(day *schedule.Day) string {
	name := "is a holiday"
	if day != nil && day.Holiday != nil {
		name = fmt.Sprintf("(%s) is a holiday", day.Holiday.Name)
	}

	hostname, _ := os.Hostname()

	return fmt.Sprintf("%s %s so %s will skip processing", day.Time.Format("Jan 02"), name, hostname)
}

func (xfagg *aggregator) alertOnError(err error) {
	if xfagg == nil {
		return
	}
	if err == nil {
		return
	}

	if err := xfagg.alerters.AlertError(err); err != nil {
		xfagg.logger.LogErrorf("ERROR sending alert: %v", err)
	}
}
