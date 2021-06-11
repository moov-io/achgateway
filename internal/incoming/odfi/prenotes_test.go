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
	"path/filepath"
	"testing"

	"github.com/moov-io/ach"

	"github.com/stretchr/testify/require"
)

func TestPrenote__isPrenoteEntry(t *testing.T) {
	file, err := ach.ReadFile(filepath.Join("testdata", "prenote-ppd-debit.ach"))
	require.NoError(t, err)
	entries := file.Batches[0].GetEntries()
	if len(entries) != 1 {
		t.Fatalf("unexpected entries: %#v", entries)
	}
	for i := range entries {
		if ok, err := isPrenoteEntry(entries[i]); !ok || err != nil {
			t.Errorf("expected prenote entry: %#v", entries[i])
			t.Error(err)
		}
	}

	// non prenote file
	file, err = ach.ReadFile(filepath.Join("..", "..", "..", "testdata", "ppd-debit.ach"))
	require.NoError(t, err)
	entries = file.Batches[0].GetEntries()
	for i := range entries {
		if ok, err := isPrenoteEntry(entries[i]); ok || err != nil {
			t.Errorf("expected no prenote entry: %#v", entries[i])
			t.Error(err)
		}
	}
}

func TestPrenote__isPrenoteEntryErr(t *testing.T) {
	file, err := ach.ReadFile(filepath.Join("testdata", "prenote-ppd-debit.ach"))
	require.NoError(t, err)
	entries := file.Batches[0].GetEntries()
	if len(entries) != 1 {
		t.Fatalf("unexpected entries: %#v", entries)
	}

	entries[0].Amount = 125 // non-zero amount
	if exists, err := isPrenoteEntry(entries[0]); !exists || err == nil {
		t.Errorf("expected invalid prenote: %v", err)
	}
}
