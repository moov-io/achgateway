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

package odfi

import (
	"context"
	"testing"
	"time"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/events"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/achgateway/pkg/models"
	"github.com/moov-io/base/log"

	"github.com/stretchr/testify/require"
)

func makeReturnFile(t *testing.T, batches int) *ach.File {
	t.Helper()

	file := ach.NewFile()
	fh := ach.NewFileHeader()
	fh.ImmediateDestination = "231380104"
	fh.ImmediateOrigin = "121042882"
	fh.FileCreationDate = time.Now().Format("060102")
	fh.FileCreationTime = time.Now().Format("1504")
	fh.ImmediateDestinationName = "Federal Reserve Bank"
	fh.ImmediateOriginName = "My Bank Name"
	file.SetHeader(fh)

	for i := 0; i < batches; i++ {
		bh := ach.NewBatchHeader()
		bh.ServiceClassCode = ach.CreditsOnly
		bh.StandardEntryClassCode = ach.PPD
		bh.CompanyName = "ACME"
		bh.CompanyIdentification = "121042882"
		bh.CompanyEntryDescription = "RETURN"
		bh.EffectiveEntryDate = time.Now().AddDate(0, 0, 1).Format("060102")
		bh.ODFIIdentification = "12104288"
		bh.BatchNumber = i + 1

		batch := ach.NewBatchPPD(bh)
		entry := ach.NewEntryDetail()
		entry.TransactionCode = ach.CheckingCredit
		entry.SetRDFI("231380104")
		entry.DFIAccountNumber = "123456789"
		entry.Amount = 100
		entry.IndividualName = "Test"
		entry.SetTraceNumber(bh.ODFIIdentification, i+1)
		entry.AddendaRecordIndicator = 1
		entry.Category = ach.CategoryReturn
		entry.Addenda99 = ach.NewAddenda99()
		entry.Addenda99.ReturnCode = "R01"
		entry.Addenda99.OriginalTrace = entry.TraceNumber
		entry.Addenda99.OriginalDFI = "12104288"
		batch.AddEntry(entry)
		require.NoError(t, batch.Create())
		file.AddBatch(batch)
	}
	require.NoError(t, file.Create())
	require.NotEmpty(t, file.ReturnEntries)
	return file
}

func TestReturns(t *testing.T) {
	cfg := service.ODFIReturns{
		Enabled: true,
	}
	eventsService, err := events.NewEmitter(log.NewTestLogger(), &service.EventsConfig{
		Webhook: &service.WebhookConfig{
			Endpoint: "https://cb.moov.io/incoming",
		},
	})
	require.NoError(t, err)

	emitter := ReturnEmitter(cfg, eventsService)
	require.NotNil(t, emitter)
}

func TestIsReturnFile(t *testing.T) {
	require.False(t, isReturnFile(File{}))
	require.False(t, isReturnFile(File{ACHFile: ach.NewFile()}))

	file := ach.NewFile()
	file.ReturnEntries = []ach.Batcher{ach.NewBatchPPD(ach.NewBatchHeader())}
	require.True(t, isReturnFile(File{ACHFile: file}))
}

func TestReturnsChunking(t *testing.T) {
	emitter := &events.MockEmitter{}
	pc := ReturnEmitter(service.ODFIReturns{Enabled: true}, emitter)
	require.NotNil(t, pc)

	file := makeReturnFile(t, 250)

	err := pc.Handle(context.Background(), log.NewTestLogger(), File{
		Filepath: "returns/big-return.ach",
		ACHFile:  file,
	})
	require.NoError(t, err)

	sent := emitter.Sent()
	require.Len(t, sent, 3)

	var filenames []string
	var totalBatches int
	for _, evt := range sent {
		ret, ok := evt.Event.(*models.ReturnFile)
		require.True(t, ok)
		filenames = append(filenames, ret.Filename)
		totalBatches += len(ret.Returns)
	}
	require.Equal(t, 250, totalBatches)
	require.Equal(t, []string{
		"CHUNK-1-OF-3__big-return.ach",
		"CHUNK-2-OF-3__big-return.ach",
		"CHUNK-3-OF-3__big-return.ach",
	}, filenames)
}

func TestReturnsChunking_SingleChunkKeepsFilename(t *testing.T) {
	emitter := &events.MockEmitter{}
	pc := ReturnEmitter(service.ODFIReturns{Enabled: true}, emitter)
	require.NotNil(t, pc)

	file := makeReturnFile(t, 1)

	err := pc.Handle(context.Background(), log.NewTestLogger(), File{
		Filepath: "returns/one.ach",
		ACHFile:  file,
	})
	require.NoError(t, err)

	sent := emitter.Sent()
	require.Len(t, sent, 1)
	ret, ok := sent[0].Event.(*models.ReturnFile)
	require.True(t, ok)
	require.Equal(t, "one.ach", ret.Filename)
}

func TestReturns_NonReturnFileSkipped(t *testing.T) {
	emitter := &events.MockEmitter{}
	pc := ReturnEmitter(service.ODFIReturns{Enabled: true}, emitter)
	require.NotNil(t, pc)

	err := pc.Handle(context.Background(), log.NewTestLogger(), File{
		Filepath: "inbound/forward.ach",
		ACHFile:  ach.NewFile(),
	})
	require.NoError(t, err)
	require.Empty(t, emitter.Sent())
}
