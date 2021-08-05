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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moov-io/base"

	"github.com/stretchr/testify/require"
)

func TestMerging__getNonCanceledMatches(t *testing.T) {
	dir := t.TempDir()

	write := func(filename string) string {
		err := ioutil.WriteFile(filepath.Join(dir, filename), nil, 0600)
		if err != nil {
			t.Fatal(err)
		}
		return filename
	}

	transfer := write(fmt.Sprintf("%s.ach", base.ID()))
	canceled := write(fmt.Sprintf("%s.ach", base.ID()))
	canceled = write(fmt.Sprintf("%s.canceled", canceled))

	matches, err := getNonCanceledMatches(filepath.Join(dir, "*.ach"))
	require.NoError(t, err)

	if len(matches) != 1 {
		t.Errorf("got %d matches: %v", len(matches), matches)
	}
	if !strings.HasSuffix(matches[0], transfer) {
		t.Errorf("unexpected match: %v", matches[0])
	}
	if strings.Contains(matches[0], canceled) {
		t.Errorf("unexpected match: %v", matches[0])
	}
}
