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

package service

// TLSConfig specifies filepaths where a TLS certificate chain and private key can be found.
type TLSConfig struct {
	// CertFile points to a filename containing an X.509 certificate chain usable to
	// wrap HTTP connections with TLS.
	CertFile string

	// KeyFile points to a filename containing a matching private key for encrypting
	// and signing TLS connections found in CertFile.
	KeyFile string
}
