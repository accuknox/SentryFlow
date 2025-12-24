// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package exporter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	protobuf "github.com/accuknox/SentryFlow/protobuf/golang"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
	"go.uber.org/zap"
)

func TestHTTPExporter_HTTPWebhook(t *testing.T) {
	received := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-Test") != "true" {
			t.Fatalf("missing header")
		}
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Exporter: &config.ExporterConfig{
			HTTP: &config.HttpConfig{
				Enabled:        true,
				TimeoutSeconds: 2,
				Webhooks: []config.WebhookConfig{
					{
						Name:   "http-test",
						URL:    server.URL,
						Method: http.MethodPost,
						Headers: map[string]string{
							"X-Test": "true",
						},
					},
				},
			},
		},
	}

	events := make(chan *protobuf.APIEvent, 1)
	ctx := context.WithValue(context.Background(), util.LoggerContextKey{}, zap.NewNop().Sugar())

	var wg sync.WaitGroup
	if err := InitHTTPExporter(ctx, cfg, events, &wg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	events <- &protobuf.APIEvent{}

	select {
	case <-received:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("webhook not received")
	}
}

func TestHTTPExporter_HTTPS_Insecure(t *testing.T) {
	received := make(chan struct{}, 1)

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		Exporter: &config.ExporterConfig{
			HTTP: &config.HttpConfig{
				Enabled:        true,
				TimeoutSeconds: 2,
				Webhooks: []config.WebhookConfig{
					{
						Name:   "https-test",
						URL:    server.URL,
						Method: http.MethodPost,
						TLS: &config.WebhookTLSConfig{
							InsecureSkipVerify: true,
						},
					},
				},
			},
		},
	}

	events := make(chan *protobuf.APIEvent, 1)
	ctx := context.WithValue(context.Background(), util.LoggerContextKey{}, zap.NewNop().Sugar())

	var wg sync.WaitGroup
	if err := InitHTTPExporter(ctx, cfg, events, &wg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	events <- &protobuf.APIEvent{}

	select {
	case <-received:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("HTTPS webhook not called")
	}
}

func TestBuildHTTPClient_TLSConfig(t *testing.T) {
	cfg := &config.Config{
		Exporter: &config.ExporterConfig{
			HTTP: &config.HttpConfig{
				TimeoutSeconds: 5,
				Webhooks: []config.WebhookConfig{
					{
						URL: "https://example.com",
						TLS: &config.WebhookTLSConfig{
							InsecureSkipVerify: true,
						},
					},
				},
			},
		},
	}

	client, err := buildHTTPClient(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, ok := client.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil {
		t.Fatal("TLS config not applied")
	}

	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify not set")
	}
}

func TestHTTPExporter_ContextCancel(t *testing.T) {
	cfg := &config.Config{
		Exporter: &config.ExporterConfig{
			HTTP: &config.HttpConfig{
				Enabled: true,
			},
		},
	}

	events := make(chan *protobuf.APIEvent)
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, util.LoggerContextKey{}, zap.NewNop().Sugar())

	var wg sync.WaitGroup
	if err := InitHTTPExporter(ctx, cfg, events, &wg); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	cancel()
	wg.Wait()
}
