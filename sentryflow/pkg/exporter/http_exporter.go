// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package exporter

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	protobuf "github.com/accuknox/SentryFlow/protobuf/golang"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
	"go.uber.org/zap"
)

type Exporter struct {
	logger   *zap.SugaredLogger
	client   *http.Client
	webhooks []config.WebhookConfig
	events   chan *protobuf.APIEvent
}

func InitHTTPExporter(ctx context.Context, cfg *config.Config, events chan *protobuf.APIEvent, wg *sync.WaitGroup) error {
	if !cfg.Exporter.HTTP.Enabled {
		return nil
	}

	logger := util.LoggerFromCtx(ctx).Named("http-exporter")

	client, err := buildHTTPClient(cfg)
	if err != nil {
		return err
	}

	exp := &Exporter{
		logger:   logger,
		client:   client,
		webhooks: cfg.Exporter.HTTP.Webhooks,
		events:   events,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		exp.run(ctx)
	}()

	logger.Info("HTTP exporter started")
	return nil
}

func (e *Exporter) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			e.logger.Info("HTTP exporter context cancelled")
			return

		case ev, ok := <-e.events:
			if !ok {
				e.logger.Warn("HTTP exporter channel closed")
				return
			}
			e.dispatch(ev)
		}
	}
}

func (e *Exporter) dispatch(event *protobuf.APIEvent) {
	for _, wh := range e.webhooks {
		go e.send(wh, event)
	}
}

func (e *Exporter) send(wh config.WebhookConfig, event *protobuf.APIEvent) {
	body, err := protojson.Marshal(event)
	if err != nil {
		e.logger.Errorf("marshal failed: %v", err)
		return
	}

	req, err := http.NewRequest(wh.Method, wh.URL, bytes.NewBuffer(body))
	if err != nil {
		e.logger.Errorf("request creation failed: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	e.logger.Infow(
		"sending webhook",
		"name", wh.Name,
		"method", wh.Method,
		"url", wh.URL,
	)

	resp, err := e.client.Do(req)
	if err != nil {
		e.logger.Errorf("webhook %s failed: %v", wh.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		e.logger.Warnf("webhook %s returned status %d", wh.Name, resp.StatusCode)
	}
}

func buildHTTPClient(cfg *config.Config) (*http.Client, error) {
	timeout := time.Duration(cfg.Exporter.HTTP.TimeoutSeconds) * time.Second

	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
	}

	var tlsConfig *tls.Config

	for _, wh := range cfg.Exporter.HTTP.Webhooks {
		if !strings.HasPrefix(strings.ToLower(wh.URL), "https://") {
			continue
		}

		if wh.TLS == nil {
			continue // public CA → default Go TLS
		}

		if tlsConfig == nil {
			tlsConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}

		if wh.TLS.CACertPath != "" {
			caCert, err := os.ReadFile(wh.TLS.CACertPath)
			if err != nil {
				return nil, err
			}
			caPool := x509.NewCertPool()
			caPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caPool
		}

		if wh.TLS.ClientCertPath != "" && wh.TLS.ClientKeyPath != "" {
			cert, err := tls.LoadX509KeyPair(
				wh.TLS.ClientCertPath,
				wh.TLS.ClientKeyPath,
			)
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		if wh.TLS.InsecureSkipVerify {
			tlsConfig.InsecureSkipVerify = true
		}
	}

	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}
