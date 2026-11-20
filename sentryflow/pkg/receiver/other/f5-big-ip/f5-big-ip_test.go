// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package f5bigip

import (
	"testing"

	"go.uber.org/zap"
)

func TestParseF5LogLineRejectsMissingBoundaryMarkers(t *testing.T) {
	logger := zap.NewNop().Sugar()

	if event := parseF5LogLine("an incomplete F5 log line", logger); event != nil {
		t.Fatalf("expected malformed line to be rejected, got %#v", event)
	}
}

func TestParseF5LogLineRejectsTooFewFixedFields(t *testing.T) {
	logger := zap.NewNop().Sugar()
	line := HSL_START + " http / GET query 10.0.0.1 1234 10.0.0.2 80 HTTP/1.1 200 " + HSL_END

	if event := parseF5LogLine(line, logger); event != nil {
		t.Fatalf("expected line with fewer than 12 fixed fields to be rejected, got %#v", event)
	}
}
