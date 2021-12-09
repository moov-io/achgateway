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
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

var (
	incomingHTTPFiles = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "incoming_http_files",
		Help: "Counter of ACH files submitted through the http interface",
	}, nil)
	incomingStreamFiles = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "incoming_stream_files",
		Help: "Counter of ACH files received through stream interface",
	}, nil)

	httpFileProcessingErrors = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "http_file_processing_errors",
		Help: "Counter of http submitted ACH files that failed processing",
	}, nil)
	streamFileProcessingErrors = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "stream_file_processing_errors",
		Help: "Counter of stream submitted ACH files that failed processing",
	}, nil)

	pendingFiles = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "pending_files",
		Help: "Counter of ACH files waiting to be uploaded",
	}, []string{"shard"})
	filesMissingShardAggregators = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "files_missing_shard_aggregators",
		Help: "Counter of ACH files unable to be matched with a shard aggregator",
	}, nil)

	uploadedFilesCounter = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "ach_uploaded_files",
		Help: "Counter of ACH files uploaded through the pipeline to the ODFI",
	}, nil)
	uploadFilesErrors = prometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Name: "ach_upload_errors",
		Help: "Counter of errors encountered when attempting ACH files upload",
	}, nil)
)

func init() {
	incomingHTTPFiles.With().Add(0)
	incomingStreamFiles.With().Add(0)

	httpFileProcessingErrors.With().Add(0)
	streamFileProcessingErrors.With().Add(0)

	filesMissingShardAggregators.With().Add(0)

	uploadedFilesCounter.With().Add(0)
	uploadFilesErrors.With().Add(0)
}
