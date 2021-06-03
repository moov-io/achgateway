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

package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/base/log"

	"github.com/gorilla/mux"
	"gocloud.dev/pubsub"
)

func NewFilesController(logger log.Logger, pub *pubsub.Topic) *FilesController {
	return &FilesController{
		logger:    logger,
		publisher: pub,
	}
}

type FilesController struct {
	logger    log.Logger
	publisher *pubsub.Topic
}

func (c *FilesController) AppendRoutes(router *mux.Router) *mux.Router {
	router.
		Name("Files.create").
		Methods("POST").
		Path("/shards/{shardKey}/files/{fileID}").
		HandlerFunc(c.CreateFileHandler)

	return router
}

// Publish to a pubsub.Subscription (inmem)

func (c *FilesController) CreateFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shardKey, fileID := vars["shardKey"], vars["fileID"]
	if shardKey == "" || fileID == "" {
		// TODO(adam): error
	}
	fmt.Printf("shardKey=%s  fileID=%s\n", shardKey, fileID)

	var buf bytes.Buffer // secondary reader for json decode, if needed
	reader := io.TeeReader(r.Body, &buf)

	file, err := ach.NewReader(reader).Read()
	if err != nil {
		// attempt JSON decode
		f, err := ach.FileFromJSON(buf.Bytes())
		if err != nil {
			fmt.Printf("error=%v\n", err)
		}
		file = *f
	}

	fmt.Printf("file=%#v\n", file)
	fmt.Printf("file.Validate()=%v\n", file.Validate())

	if err := publishFile(c.publisher, shardKey, fileID, &file); err != nil {
		// TODO(adam): error
	}
}

func publishFile(pub *pubsub.Topic, shardKey, fileID string, file *ach.File) error {
	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(incoming.ACHFile{
		FileID:   fileID,
		ShardKey: shardKey,
		File:     file,
	})
	if err != nil {
		return fmt.Errorf("fileID=%s unable to encode ACH file: %v", fileID, err)
	}

	meta := make(map[string]string, 0)
	meta["fileID"] = fileID
	meta["shardKey"] = shardKey

	return pub.Send(context.Background(), &pubsub.Message{
		Body:     body.Bytes(),
		Metadata: meta,
	})
}
