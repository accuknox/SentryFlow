// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

// Package mulegateway implements a SentryFlow receiver for Anypoint Mule Gateway.
// Mule Gateway is embedded in Mule Runtime Engine (Mule 4). A custom Mule policy
// (deployed via Anypoint API Manager) intercepts every API request/response and
// forwards structured JSON telemetry to the SentryFlow HTTP server.
package mulegateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	pb "github.com/accuknox/SentryFlow/protobuf/golang"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
)

const (
	receiverName    = "mule-gateway"
	receiverVersion = "4.x"

	EventsPath       = "/api/v1/events/mule"
	eventsHealthPath = "/api/v1/events/mule/health"
)

// MuleAPIEvent is the JSON payload sent by the SentryFlow Mule 4 custom policy.
// The policy POSTs this structure to SentryFlow's HTTP server after each API call.
type MuleAPIEvent struct {
	// Request fields
	Scheme   string `json:"scheme"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Query    string `json:"query,omitempty"`
	Protocol string `json:"protocol,omitempty"`

	// Network fields
	SourceIP   string `json:"sourceIp"`
	SourcePort int32  `json:"sourcePort,omitempty"`
	DestIP     string `json:"destIp,omitempty"`
	DestPort   int32  `json:"destPort,omitempty"`

	// Headers
	RequestHeaders  map[string]string `json:"requestHeaders,omitempty"`
	ResponseHeaders map[string]string `json:"responseHeaders,omitempty"`

	// Bodies (optional, may be empty if policy is configured to omit them)
	RequestBody  string `json:"requestBody,omitempty"`
	ResponseBody string `json:"responseBody,omitempty"`

	// Response status
	ResponseCode  int    `json:"responseCode"`
	ResponsePhase string `json:"responsePhase,omitempty"` // "client-response" or "error-response"

	// Timing (Unix milliseconds)
	RequestTimestamp  int64 `json:"requestTimestamp"`
	ResponseTimestamp int64 `json:"responseTimestamp"`

	// Mule metadata
	APIName    string `json:"apiName,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
	AppName    string `json:"appName,omitempty"`
}

// RegisterRoutes registers the Mule Gateway event and health-check handlers on the
// provided mux. It is called before the shared HTTP server starts so that no
// separate server or port is needed.
func RegisterRoutes(ctx context.Context, cfg *config.Config, mux *http.ServeMux, apiEvents chan *pb.APIEvent) error {
	logger := util.LoggerFromCtx(ctx).Named(receiverName)

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("mule gateway receiver config error: %w", err)
	}

	mux.HandleFunc(EventsPath, func(w http.ResponseWriter, r *http.Request) {
		handleEvent(w, r, apiEvents, logger)
	})

	mux.HandleFunc(eventsHealthPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","receiver":"mule-gateway"}`))
	})

	logger.Infof("Mule Gateway receiver registered at %s (shared HTTP server)", EventsPath)
	return nil
}

// handleEvent decodes the JSON payload from the Mule policy and pushes an APIEvent.
func handleEvent(w http.ResponseWriter, r *http.Request, apiEvents chan *pb.APIEvent, logger interface {
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
}) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024)) // 4 MB max
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		logger.Errorf("Mule Gateway: failed to read request body: %v", err)
		return
	}
	defer r.Body.Close()

	var ev MuleAPIEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		logger.Errorf("Mule Gateway: failed to parse event JSON: %v", err)
		return
	}

	logger.Infof("Mule Gateway receiver received API event for method=%s path=%s responseCode=%d", ev.Method, ev.Path, ev.ResponseCode)

	apiEvents <- toProto(&ev)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}

// toProto converts a MuleAPIEvent to the SentryFlow protobuf APIEvent.
func toProto(ev *MuleAPIEvent) *pb.APIEvent {
	// Build request headers map, injecting pseudo-headers for method/path/scheme
	reqHeaders := make(map[string]string)
	for k, v := range ev.RequestHeaders {
		reqHeaders[k] = v
	}
	if ev.Method != "" {
		reqHeaders[":method"] = ev.Method
	}
	if ev.Path != "" {
		reqHeaders[":path"] = ev.Path
	}
	if ev.Scheme != "" {
		reqHeaders[":scheme"] = ev.Scheme
	}
	if ev.Query != "" {
		reqHeaders[":query"] = ev.Query
	}

	// Build response headers map
	respHeaders := make(map[string]string)
	for k, v := range ev.ResponseHeaders {
		respHeaders[k] = v
	}
	if ev.ResponseCode != 0 {
		respHeaders[":status"] = fmt.Sprintf("%d", ev.ResponseCode)
	}

	// Calculate latency (ms → ns)
	var latencyNs uint64
	if ev.ResponseTimestamp > ev.RequestTimestamp {
		latencyNs = uint64((ev.ResponseTimestamp - ev.RequestTimestamp) * 1_000_000)
	}

	protocol := ev.Protocol
	if protocol == "" {
		protocol = "HTTP/1.1"
	}

	return &pb.APIEvent{
		Metadata: &pb.Metadata{
			ReceiverName:    receiverName,
			ReceiverVersion: receiverVersion,
			Timestamp:       uint64(ev.RequestTimestamp),
		},
		Source: &pb.Workload{
			Ip:   ev.SourceIP,
			Port: ev.SourcePort,
		},
		Destination: &pb.Workload{
			Ip:   ev.DestIP,
			Port: ev.DestPort,
		},
		Request: &pb.Request{
			Headers: reqHeaders,
			Body:    ev.RequestBody,
		},
		Response: &pb.Response{
			Headers:               respHeaders,
			Body:                  ev.ResponseBody,
			BackendLatencyInNanos: latencyNs,
		},
		Protocol: protocol,
	}
}

// validateConfig checks that required Mule Gateway configuration is present.
func validateConfig(cfg *config.Config) error {
	if cfg.Filters.MuleGateway == nil {
		return fmt.Errorf("no muleGateway filter configuration provided")
	}
	return nil
}
