// Copyright 2020 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package upload

import (
	"net"
	"testing"

	"github.com/moov-io/achgateway/internal/service"
	"github.com/stretchr/testify/require"
)

func TestRejectOutboundIPRange(t *testing.T) {
	addrs, err := net.LookupIP("moov.io")
	require.NoError(t, err)

	var addr net.IP
	for i := range addrs {
		if a := addrs[i].To4(); a != nil {
			addr = a
			break
		}
	}

	cfg := &service.UploadAgent{AllowedIPs: addr.String()}

	// exact IP match
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err != nil {
		t.Error(err)
	}

	// multiple whitelisted, but exact IP match
	cfg.AllowedIPs = "127.0.0.1/24," + addr.String()
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err != nil {
		t.Error(err)
	}

	// multiple whitelisted, match range (convert IP to /24)
	cfg.AllowedIPs = addr.Mask(net.IPv4Mask(0xFF, 0xFF, 0xFF, 0x0)).String() + "/24"
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err != nil {
		t.Error(err)
	}

	// no match
	cfg.AllowedIPs = "8.8.8.0/24"
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err == nil {
		t.Error("expected error")
	}

	// empty whitelist, allow all
	cfg.AllowedIPs = ""
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err != nil {
		t.Errorf("expected no error: %v", err)
	}

	// error cases
	cfg.AllowedIPs = "afkjsafkjahfa"
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err == nil {
		t.Error("expected error")
	}
	cfg.AllowedIPs = "10.0.0.0/8"
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "lsjafkshfaksjfhas"); err == nil {
		t.Error("expected error")
	}
	cfg.AllowedIPs = "10...../8"
	if err := rejectOutboundIPRange(cfg.SplitAllowedIPs(), "moov.io"); err == nil {
		t.Error("expected error")
	}
}
