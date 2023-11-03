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
	"net/http"
	"path/filepath"
	"strings"

	"github.com/moov-io/base/log"
	"github.com/moov-io/base/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/gorilla/mux"
)

func (fr *FileReceiver) manuallyProduceFileUploaded() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := fr.logger.With(log.Fields{
			"route": log.String("manual-file-upload-produce"),
		})

		agg := fr.lookupAggregator(logger, r)
		if agg == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		m, ok := agg.merger.(*filesystemMerging)
		if !ok {
			logger.Error().Logf("unexpected %T merger", agg.merger)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		dir := mux.Vars(r)["isolatedDirectory"]
		if dir == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Reject paths which are trying to traverse the filesystem
		if strings.Contains(dir, "..") || filepath.IsAbs(dir) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		ctx, span := telemetry.StartSpan(r.Context(), "pipeline-manual-file-uploaded", trace.WithAttributes(
			attribute.String("dir", dir),
		))
		defer span.End()

		matches, err := m.getNonCanceledMatches(dir)
		if err != nil {
			logger.LogErrorf("problem listing matches: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		processed := newProcessedFiles(agg.shard.Name, matches)
		if len(matches) == 0 || len(processed.fileIDs) == 0 {
			logger.Logf("%s not found", dir)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		err = agg.emitFilesUploaded(ctx, processed)
		if err != nil {
			logger.LogErrorf("problem emitting FileUploaded: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
