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
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/moov-io/achgateway/internal/storage"
)

type listShardsResponse struct {
	Shards []shard `json:"shards"`
}

type shard struct {
	Name string `json:"name"`
}

func (fr *FileReceiver) listShards() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var shards []shard

		for name := range fr.shardAggregators {
			shards = append(shards, shard{
				Name: name,
			})
		}

		json.NewEncoder(w).Encode(listShardsResponse{
			Shards: shards,
		})
	}
}

type listShardFilesResponse struct {
	Files []listFileResponse `json:"files"`
}

type listFileResponse struct {
	Filename string
	Path     string
	ModTime  time.Time
}

func (fr *FileReceiver) listShardFiles() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agg := fr.lookupAggregator(r)
		if agg == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		chest := fr.getStorage(agg)
		if chest == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		matches, err := chest.Glob("*")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var wrapper []listFileResponse
		for i := range matches {
			_, filename := filepath.Split(matches[i].RelativePath)

			wrapper = append(wrapper, listFileResponse{
				Filename: filename,
				Path:     matches[i].RelativePath,
				ModTime:  matches[i].ModTime,
			})
		}
		json.NewEncoder(w).Encode(&listShardFilesResponse{
			Files: wrapper,
		})
	}
}

type getFileResponse struct {
	Filename string
	Contents string
	Valid    error
	ModTime  time.Time
}

func (fr *FileReceiver) getShardFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agg := fr.lookupAggregator(r)
		if agg == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		chest := fr.getStorage(agg)
		if chest == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		path := mux.Vars(r)["filepath"] // TODO(adam): need to trim off r.URL
		file, err := chest.Open(path)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(getFileResponse{
			Filename: file.Filename(),
			Contents: "TODO",
			Valid:    errors.New("TODO - .Validate()"),
			ModTime:  time.Now(),
		})
	}
}

func (fr *FileReceiver) lookupAggregator(r *http.Request) *aggregator {
	shardName := mux.Vars(r)["shardName"]
	if shardName == "" {
		return nil
	}
	agg, exists := fr.shardAggregators[shardName]
	if !exists {
		return nil
	}
	return agg
}

func (fr *FileReceiver) getStorage(agg *aggregator) storage.Chest {
	mm, ok := agg.merger.(*filesystemMerging)
	if !ok {
		return nil
	}
	return mm.storage
}
