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
	"io/ioutil"
	"net/http"

	"github.com/moov-io/ach"
	"github.com/moov-io/achgateway/internal/compliance"
	"github.com/moov-io/achgateway/internal/incoming"
	"github.com/moov-io/achgateway/internal/service"
	"github.com/moov-io/base/log"

	"github.com/gorilla/mux"
	"gocloud.dev/pubsub"
)

func NewFilesController(logger log.Logger, cfg service.HTTPConfig, pub *pubsub.Topic) *FilesController {
	return &FilesController{
		logger:    logger,
		cfg:       cfg,
		publisher: pub,
	}
}

type FilesController struct {
	logger    log.Logger
	cfg       service.HTTPConfig
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

func (c *FilesController) CreateFileHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shardKey, fileID := vars["shardKey"], vars["fileID"]
	if shardKey == "" || fileID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bs, err := c.readBody(r)
	if err != nil {
		c.logger.LogErrorf("error reading file: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file, err := ach.NewReader(bytes.NewReader(bs)).Read()
	if err != nil {
		// attempt JSON decode
		f, err := ach.FileFromJSON(bs)
		if f == nil || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		file = *f
	}

	if err := publishFile(c.publisher, shardKey, fileID, &file); err != nil {
		c.logger.LogErrorf("error publishing fileID=%s: %v", fileID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (c *FilesController) readBody(req *http.Request) ([]byte, error) {
	defer req.Body.Close()

	var reader io.Reader = req.Body
	if c.cfg.MaxBodyBytes > 0 {
		reader = io.LimitReader(reader, c.cfg.MaxBodyBytes)
	}
	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return compliance.Reveal(c.cfg.Transform, bs)
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

	meta := make(map[string]string)
	meta["fileID"] = fileID
	meta["shardKey"] = shardKey

	return pub.Send(context.Background(), &pubsub.Message{
		Body:     body.Bytes(),
		Metadata: meta,
	})
}
