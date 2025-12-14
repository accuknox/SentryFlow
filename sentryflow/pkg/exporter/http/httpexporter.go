package http

import (
	"bytes"
	"context"
	"net/http"
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

func Init(ctx context.Context, cfg *config.Config, events chan *protobuf.APIEvent, wg *sync.WaitGroup) error {

	if !cfg.Exporter.HTTP.Enabled {
		return nil
	}

	logger := util.LoggerFromCtx(ctx).Named("http-exporter")

	exp := &Exporter{
		logger: logger,
		client: &http.Client{
			Timeout: time.Duration(cfg.Exporter.HTTP.TimeoutSeconds) * time.Second,
		},
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
			return

		case ev, ok := <-e.events:
			if !ok {
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

	req, err := http.NewRequest(wh.Method, wh.Url, bytes.NewBuffer(body))
	if err != nil {
		e.logger.Errorf("request creation failed: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

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
