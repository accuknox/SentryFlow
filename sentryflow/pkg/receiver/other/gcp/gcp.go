package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	protobuf "github.com/accuknox/SentryFlow/protobuf/golang"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
)

// LogEntry represents the structure of a GCP Cloud Logging entry received via Pub/Sub.
type LogEntry struct {
	Timestamp   string      `json:"timestamp"`
	Severity    string      `json:"severity"`
	InsertId    string      `json:"insertId"`
	Resource    Resource    `json:"resource"`
	HttpRequest HttpRequest `json:"httpRequest"`
	JsonPayload JsonPayload `json:"jsonPayload"`
}

type Resource struct {
	Type   string            `json:"type"`
	Labels map[string]string `json:"labels"`
}

type HttpRequest struct {
	RequestMethod string `json:"requestMethod"`
	RequestUrl    string `json:"requestUrl"`
	RequestSize   string `json:"requestSize"`
	Status        int    `json:"status"`
	ResponseSize  string `json:"responseSize"`
	UserAgent     string `json:"userAgent"`
	RemoteIp      string `json:"remoteIp"`
	ServerIp      string `json:"serverIp"`
}

type JsonPayload struct {
	ApiName       string `json:"api_name"`
	ApiVersion    string `json:"api_version"`
	ApiMethod     string `json:"api_method"`
	Location      string `json:"location"`
	ProducerId    string `json:"producer_project_id"`
	ServiceConfig string `json:"service_config_id"`
}

func Start(ctx context.Context, cfg *config.Config, apiEvents chan *protobuf.APIEvent) {
	logger := util.LoggerFromCtx(ctx).Named("gcp-receiver")
	gcpConfig := cfg.Filters.GCP

	if gcpConfig == nil {
		logger.Error("GCP configuration is missing")
		return
	}

	if gcpConfig.ProjectID == "" || gcpConfig.SubscriptionID == "" {
		logger.Error("GCP ProjectID or SubscriptionID is missing")
		return
	}

	logger.Infof("Starting GCP Pub/Sub receiver for project: %s, subscription: %s", gcpConfig.ProjectID, gcpConfig.SubscriptionID)

	var client *pubsub.Client
	var err error

	opts := []option.ClientOption{}
	if gcpConfig.ServiceAccountJSON != "" {
		if len(gcpConfig.ServiceAccountJSON) > 0 && gcpConfig.ServiceAccountJSON[0] == '{' {
			opts = append(opts, option.WithCredentialsJSON([]byte(gcpConfig.ServiceAccountJSON)))
		} else {
			opts = append(opts, option.WithCredentialsFile(gcpConfig.ServiceAccountJSON))
		}
	}

	client, err = pubsub.NewClient(ctx, gcpConfig.ProjectID, opts...)
	if err != nil {
		logger.Errorf("Failed to create Pub/Sub client: %v", err)
		return
	}

	sub := client.Subscription(gcpConfig.SubscriptionID)
	sub.ReceiveSettings.MaxOutstandingMessages = 100

	logger.Info("Listening for GCP Pub/Sub messages...")
	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		processMessage(ctx, msg, apiEvents)
		msg.Ack()
	})
	if err != nil {
		logger.Errorf("GCP Pub/Sub Receive error: %v", err)
	}
	logger.Info("GCP Pub/Sub receiver stopped")
}

func processMessage(ctx context.Context, msg *pubsub.Message, apiEvents chan *protobuf.APIEvent) {
	logger := util.LoggerFromCtx(ctx)

	var entry LogEntry
	if err := json.Unmarshal(msg.Data, &entry); err != nil {
		logger.Warnf("Failed to unmarshal GCP log entry: %v", err)
		return
	}

	// Determine Hostname:
	// 1. Try resource.labels.service (e.g. "gateway-015...cloud.goog")
	// 2. Fallback to jsonPayload.api_name
	// 3. Fallback to generic "gcp-api-gateway"
	hostname := "gcp-api-gateway"
	if service, ok := entry.Resource.Labels["service"]; ok && service != "" {
		hostname = service
	} else if entry.JsonPayload.ApiName != "" {
		hostname = entry.JsonPayload.ApiName
	}

	// Map to SentryFlow APIEvent
	apiEvent := &protobuf.APIEvent{
		Metadata: &protobuf.Metadata{
			ReceiverName: util.GCPAPIGateway,
			Timestamp:    uint64(time.Now().Unix()),
		},
		Request: &protobuf.Request{
			Headers: map[string]string{
				":method":    entry.HttpRequest.RequestMethod,
				":authority": hostname,
				":path":      entry.HttpRequest.RequestUrl,
				"user-agent": entry.HttpRequest.UserAgent,
			},
		},
		Response: &protobuf.Response{
			Headers: map[string]string{
				":status": fmt.Sprintf("%d", entry.HttpRequest.Status),
			},
		},
		Source: &protobuf.Workload{
			Ip: entry.HttpRequest.RemoteIp,
		},
		Destination: &protobuf.Workload{
			Ip:   entry.HttpRequest.ServerIp,
			Name: entry.JsonPayload.ApiName, // Set destination name to API Name
		},
		Protocol: "HTTP",
	}

	// Enrich with extra context if available
	if entry.JsonPayload.Location != "" {
		apiEvent.Destination.Namespace = entry.JsonPayload.Location // Map location to namespace as a proxy for grouping
	}

	logger.Debug("Received GCP event: %s %s (Host: %s)", entry.HttpRequest.RequestMethod, entry.HttpRequest.RequestUrl, hostname)
	apiEvents <- apiEvent
}
