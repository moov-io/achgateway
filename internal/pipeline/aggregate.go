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
	"path/filepath"
	"strings"
	"sync"
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
	"github.com/moov-io/base/log"
	"github.com/moov-io/base/stime"
	"github.com/moov-io/base/strx"
	"github.com/moov-io/base/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
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
	cleanupService        *CleanupService
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
	cutoffs, err := schedule.ForCutoffTimes(logger, timeService, shard.Cutoffs.Timezone, shard.Cutoffs.Windows)
	if err != nil {
		return nil, fmt.Errorf("error creating cutoffs: %v", err)
	}

	alerters, err := alerting.NewAlerters(errorAlerting)
	if err != nil {
		return nil, fmt.Errorf("error setting up alerters: %v", err)
	}

	// Create cleanup service if configured
	var cleanupService *CleanupService
	if uploadAgents.Merging.Cleanup != nil && uploadAgents.Merging.Cleanup.Enabled {
		// Get the storage from merger (it's a filesystemMerging which has storage)
		if fm, ok := merger.(*filesystemMerging); ok {
			cleanupService, err = NewCleanupService(logger, fm.storage, shard, uploadAgents.Merging.Cleanup)
			if err != nil {
				return nil, fmt.Errorf("error creating cleanup service: %v", err)
			}
			logger.Info().With(log.Fields{
				"shard": log.String(shard.Name),
			}).Log("setup cleanup service")
		}
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
		cleanupService:        cleanupService,
	}, nil
}

func (xfagg *aggregator) Start(ctx context.Context) {
	// Start cleanup service if configured
	if xfagg.cleanupService != nil {
		xfagg.cleanupService.Start(ctx)
	}

	for {
		select {
		// process automated cutoff time triggering
		case day := <-xfagg.cutoffs.C:
			// Run our regular routines
			if day.IsBankingDay || xfagg.shard.Cutoffs.On == "all-days" {
				if err := xfagg.withEachFile(day.Time); err != nil {
					err = xfagg.logger.Error().LogErrorf("merging files: %v", err).Err()
					xfagg.alertOnError(err)
				}
			} else if day.IsHoliday && !day.IsWeekend {
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

	if xfagg.cleanupService != nil {
		xfagg.cleanupService.Stop()
	}

	if xfagg.auditStorage != nil {
		xfagg.auditStorage.Close()
	}
}

func (xfagg *aggregator) acceptFile(ctx context.Context, msg incoming.ACHFile) (incoming.QueueACHFileResponse, error) {
	err := xfagg.merger.HandleXfer(ctx, msg)

	now := time.Now().In(time.UTC)
	resp := incoming.QueueACHFileResponse{
		FileID:     msg.FileID,
		ShardKey:   msg.ShardKey,
		AcceptedAt: &now,
	}
	if err != nil {
		resp.Error = err.Error()
		return resp, err
	}

	nextCutoff, cutoffErr := schedule.NextCutoff(xfagg.shard.Cutoffs.Timezone, xfagg.shard.Cutoffs.Windows, xfagg.shard.Cutoffs.On, now)
	if cutoffErr != nil {
		xfagg.logger.Warn().With(log.Fields{
			"fileID":   log.String(msg.FileID),
			"shard":    log.String(xfagg.shard.Name),
			"shardKey": log.String(msg.ShardKey),
		}).Logf("unable to predict next cutoff: %v", cutoffErr)
	} else if !nextCutoff.IsZero() {
		resp.NextCutoff = &nextCutoff
	}

	return resp, nil
}

func (xfagg *aggregator) cancelFile(ctx context.Context, msg incoming.CancelACHFile) (models.FileCancellationResponse, error) {
	response, err := xfagg.merger.HandleCancel(ctx, msg)
	if err != nil {
		return models.FileCancellationResponse{}, err
	}

	// Send the response back
	out := models.FileCancellationResponse(response)
	err = xfagg.eventEmitter.Send(ctx, models.Event{
		Event: out,
	})
	if err != nil {
		return out, fmt.Errorf("problem emitting file cancellation response: %w", err)
	}
	return out, nil
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
		attribute.String("achgateway.shard", xfagg.shard.Name),
		attribute.String("achgateway.timezone", tzname),
		attribute.String("achgateway.window", window),
	))
	defer span.End()

	// WithEachMerged may return partial results with an error (e.g. some files
	// uploaded/mapped and others unmapped). Always emit FileUploaded for what
	// mapped successfully, then surface the error so on-call is alerted.
	merged, mergeErr := xfagg.merger.WithEachMerged(ctx, xfagg.runTransformers)
	if mergeErr != nil {
		mergeErr = fmt.Errorf("WithEachMerged: %w", mergeErr)
		logger.Error().LogError(mergeErr)
		span.RecordError(mergeErr)
	}

	if err := xfagg.emitFilesUploaded(ctx, merged); err != nil {
		err = fmt.Errorf("sending FileUploaded events: %w", err)
		logger.Error().LogError(err)
		span.RecordError(err)
		// Join so emit failures are never dropped when merge already failed.
		mergeErr = errors.Join(mergeErr, err)
	}

	if mergeErr != nil {
		return fmt.Errorf("merging ACH files: %w", mergeErr)
	}
	return nil
}

func (xfagg *aggregator) manualCutoff(waiter manuallyTriggeredCutoff) {
	logger := xfagg.logger.With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	})
	logger.Info().Log("starting manual cutoff window processing")

	ctx, span := telemetry.StartSpan(context.Background(), "manual-cutoff", trace.WithAttributes(
		attribute.String("achgateway.shard", xfagg.shard.Name),
	))
	defer span.End()

	// Emit FileUploaded for any successfully mapped files even when merge
	// reports unmapped leftovers or partial upload failures.
	merged, mergeErr := xfagg.merger.WithEachMerged(ctx, xfagg.runTransformers)
	if mergeErr != nil {
		mergeErr = fmt.Errorf("manual WithEachMerged: %w", mergeErr)
		logger.Error().LogError(mergeErr)
		span.RecordError(mergeErr)
	}
	if err := xfagg.emitFilesUploaded(ctx, merged); err != nil {
		err = fmt.Errorf("sending manual FileUploaded events: %w", err)
		logger.Error().LogError(err)
		span.RecordError(err)
		// Join so emit failures are never dropped when merge already failed.
		mergeErr = errors.Join(mergeErr, err)
	}
	waiter.C <- mergeErr

	logger.Info().With(log.Fields{
		"shard": log.String(xfagg.shard.Name),
	}).Log("ended manual cutoff window processing")
}

// fileUploadedEmitLimit caps concurrent FileUploaded produces.
//
// This limit is about *our* side: avoid spawning one goroutine per input
// (10k–100k on large cutoffs) when the event producer is already the bottleneck.
// 128 is enough to keep the producer fed without a goroutine stampede.
const fileUploadedEmitLimit = 128

func (xfagg *aggregator) emitFilesUploaded(ctx context.Context, merged mergedFiles) error {
	ctx, span := telemetry.StartSpan(ctx, "emit-file-uploaded", trace.WithAttributes(
		attribute.String("achgateway.shard", xfagg.shard.Name),
	))
	defer span.End()

	var inputFilepaths int
	uploadedFilenames := make([]string, 0, len(merged))

	// Attempt every FileUploaded independently. One produce failure must not
	// prevent the remaining events from being sent. errgroup.Wait only returns
	// the first error, so we also collect every failure in sendErrs for ops.
	var (
		g        errgroup.Group
		mu       sync.Mutex
		sendErrs []error
	)
	g.SetLimit(fileUploadedEmitLimit)

	missingEmitter := xfagg.eventEmitter == nil
	if missingEmitter {
		err := fmt.Errorf("missing event emitter for FileUploaded on shard %v", xfagg.shard.Name)
		xfagg.logger.Error().LogError(err)
		span.RecordError(err)
		// Still walk inputs for metrics, but there is nothing to produce.
	}

	for i := range merged {
		inputFilepaths += len(merged[i].InputFilepaths)
		uploadedFilenames = append(uploadedFilenames, merged[i].UploadedFilename)

		for k := range merged[i].InputFilepaths {
			// FileID's don't have .ach suffix
			_, filename := filepath.Split(merged[i].InputFilepaths[k])
			filename = trimACHExtension(filename)

			if missingEmitter {
				continue
			}

			fileID := filename
			uploadedName := merged[i].UploadedFilename
			shardKey := merged[i].Shard

			g.Go(func() error {
				evt := models.Event{
					Event: models.FileUploaded{
						FileID:     fileID,
						ShardKey:   shardKey,
						Filename:   uploadedName,
						UploadedAt: time.Now(),
					},
				}

				err := xfagg.eventEmitter.Send(ctx, evt)
				if err != nil {
					err = fmt.Errorf("FileUploaded file_id=%s filename=%s: %w", fileID, uploadedName, err)
					mu.Lock()
					sendErrs = append(sendErrs, err)
					mu.Unlock()
					return err
				}
				return nil
			})
		}
	}

	// Block until every produce finishes. Discard Wait's first-error return —
	// sendErrs already holds the full multi-error set for the caller.
	_ = g.Wait() //nolint:errcheck // intentional: all per-goroutine errors are collected in sendErrs; Wait's first-error return would drop the rest

	span.SetAttributes(
		attribute.Int("achgateway.input_filepaths", inputFilepaths),
		attribute.Int("achgateway.file_uploaded_errors", len(sendErrs)),
		attribute.Int("achgateway.file_uploaded_emit_limit", fileUploadedEmitLimit),
		attribute.StringSlice("achgateway.uploaded_filename", uploadedFilenames),
	)

	if missingEmitter {
		return fmt.Errorf("missing event emitter for FileUploaded on shard %v (%d events skipped)", xfagg.shard.Name, inputFilepaths)
	}
	if len(sendErrs) == 0 {
		return nil
	}
	// Prefer a multi-error so operators see how many emits failed, not only the first.
	return errors.Join(sendErrs...)
}

func (xfagg *aggregator) runTransformers(ctx context.Context, index int, agent upload.Agent, outgoing *ach.File) (string, error) {
	result, err := transform.ForUpload(outgoing, xfagg.preuploadTransformers)
	if err != nil {
		return "", err
	}
	return xfagg.uploadFile(ctx, index, agent, result)
}

func (xfagg *aggregator) uploadFile(ctx context.Context, index int, agent upload.Agent, res *transform.Result) (string, error) {
	if res == nil || res.File == nil {
		return "", errors.New("uploadFile: nil Result / File")
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
		return "", fmt.Errorf("problem rendering filename template: %v", err)
	}

	ctx, span := telemetry.StartSpan(ctx, "upload-file", trace.WithAttributes(
		attribute.String("achgateway.filename", filename),
		attribute.String("achgateway.shard", xfagg.shard.Name),
	))
	defer span.End()

	var buf bytes.Buffer
	if err := xfagg.outputFormatter.Format(&buf, res); err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
		return "", fmt.Errorf("problem formatting output: %v", err)
	}

	// Record the file in our audit trail
	basePath := "outbound"
	if xfagg.shard.Audit != nil {
		basePath = strx.Or(xfagg.shard.Audit.BasePath, basePath)
	}
	path := fmt.Sprintf("%s/%s/%s/%s", basePath, agent.Hostname(), time.Now().Format("2006-01-02"), filename)
	if err := xfagg.auditStorage.SaveFile(ctx, path, buf.Bytes()); err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
		return "", fmt.Errorf("problem saving file in audit record: %v", err)
	}

	// Upload our file
	err = agent.UploadFile(ctx, upload.File{
		Filepath: filename,
		Contents: io.NopCloser(&buf),
	})

	// Send Slack/PD or whatever notifications after the file is uploaded
	if err := xfagg.notifyAfterUpload(ctx, filename, res.File, agent, err); err != nil {
		xfagg.alertOnError(xfagg.logger.Error().LogError(err).Err())
	}

	// record our upload metrics
	if err != nil {
		recordFileUploadError(ctx, xfagg.shard.Name, err)
	} else {
		uploadedFilesCounter.With("shard", xfagg.shard.Name).Add(1)
	}

	return filename, err
}

func prepareShardName(shardName string) string {
	return strings.ToUpper(strings.ReplaceAll(shardName, " ", "-"))
}

func (xfagg *aggregator) notifyAfterUpload(ctx context.Context, filename string, file *ach.File, agent upload.Agent, uploadErr error) error {
	_, span := telemetry.StartSpan(ctx, "notify-after-upload", trace.WithAttributes(
		attribute.String("achgateway.filename", filename),
		attribute.String("achgateway.shard", xfagg.shard.Name),
	))
	defer span.End()

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
		if err := notifier.Critical(ctx, msg); err != nil {
			return fmt.Errorf("problem sending critical notification for file=%s: %v", filename, err)
		}
	} else {
		if err := notifier.Info(ctx, msg); err != nil {
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

	ctx, span := telemetry.StartSpan(context.Background(), "notify-about-holiday", trace.WithAttributes(
		attribute.String("achgateway.holiday", day.Holiday.Name),
	))
	defer span.End()

	if uploadAgent.Notifications != nil {
		slackConfigs := xfagg.shard.Notifications.FindSlacks(uploadAgent.Notifications.Slack)
		for i := range slackConfigs {
			ss, err := notify.NewSlack(&slackConfigs[i])
			if err != nil {
				logger.Error().LogErrorf("ERROR creating slack holiday notifier: %v", err)
				continue
			}

			err = ss.Info(ctx, &notify.Message{
				Contents: formatHolidayMessage(day, xfagg.shard.Name),
			})
			if err != nil {
				logger.Error().LogErrorf("ERROR sending holiday notification: %v", err)
			} else {
				logger.Info().Log("sent holiday notification")
			}
		}
	}
}

func formatHolidayMessage(day *schedule.Day, shardName string) string {
	name := "is a holiday"
	if day != nil && day.Holiday != nil {
		name = fmt.Sprintf("(%s) is a holiday", day.Holiday.Name)
	}

	hostname, _ := os.Hostname()

	if shardName != "" {
		shardName = "for " + shardName
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s so %s will skip processing %s", day.Time.Format("Jan 02"), name, hostname, shardName))
}

func (xfagg *aggregator) alertOnError(err error) {
	if xfagg == nil {
		return
	}
	if err == nil {
		return
	}

	if err := xfagg.alerters.AlertError(err); err != nil {
		xfagg.logger.Error().LogErrorf("ERROR sending alert: %v", err)
	}
}
