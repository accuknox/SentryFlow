// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package config

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
)

const (
	DefaultConfigFilePath             = "config/default.yaml"
	SentryFlowDefaultFilterServerPort = 8081
)

type nameAndNamespace struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

type receivers struct {
	ServiceMeshes []*nameAndNamespace `json:"serviceMeshes,omitempty"`
	Others        []*nameAndNamespace `json:"other,omitempty"`
}

type envoyFilterConfig struct {
	Uri        string `json:"uri"`
	GatewayTag string `json:"gatewayTag"`
	SidecarTag string `json:"sidecarTag"`
}

type server struct {
	Port uint16 `json:"port"`
}

type HttpConfig struct {
	Enabled        bool            `json:"enabled"`
	TimeoutSeconds uint32          `json:"timeoutSeconds"`
	Webhooks       []WebhookConfig `json:"webhooks"`
}

type WebhookConfig struct {
	Name    string            `mapstructure:"name"`
	URL     string            `mapstructure:"url"`
	Method  string            `mapstructure:"method"`
	Headers map[string]string `mapstructure:"headers"`

	TLS *WebhookTLSConfig `mapstructure:"tls,omitempty"`
}

type WebhookTLSConfig struct {
	InsecureSkipVerify bool   `mapstructure:"insecureSkipVerify"`
	CACertPath         string `mapstructure:"caCertPath"`
	ClientCertPath     string `mapstructure:"clientCertPath"`
	ClientKeyPath      string `mapstructure:"clientKeyPath"`
}

type nginxIngressConfig struct {
	DeploymentName             string `json:"deploymentName"`
	ConfigMapName              string `json:"configMapName"`
	SentryFlowNjsConfigMapName string `json:"sentryFlowNjsConfigMapName"`
}

type gcpConfig struct {
	ProjectID          string `json:"projectID"`
	SubscriptionID     string `json:"subscriptionID"`
	ServiceAccountJSON string `json:"serviceAccountJSON"` // Path to SA JSON key file
}

type filters struct {
	Envoy        *envoyFilterConfig  `json:"envoy,omitempty"`
	NginxIngress *nginxIngressConfig `json:"nginxIngress,omitempty"`
	GCP          *gcpConfig          `json:"gcp,omitempty"`
	Server       *server             `json:"server,omitempty"`
}

type ExporterConfig struct {
	Grpc *server     `json:"grpc"`
	HTTP *HttpConfig `json:"http"`
}

type Config struct {
	Filters   *filters        `json:"filters"`
	Receivers *receivers      `json:"receivers"`
	Exporter  *ExporterConfig `json:"exporter"`
}

func (c *Config) validate() error {
	if c.Filters == nil {
		return fmt.Errorf("no filter configuration provided")
	}
	if c.Filters.Envoy != nil {
		if c.Filters.Envoy.Uri == "" {
			return fmt.Errorf("no envoy filter URI provided")
		}
	}

	if c.Exporter == nil {
		return fmt.Errorf("no exporter configuration provided")
	}
	if c.Exporter.Grpc == nil {
		return fmt.Errorf("no exporter's gRPC configuration provided")
	}
	if c.Exporter.Grpc != nil && c.Exporter.Grpc.Port == 0 {
		return fmt.Errorf("no exporter's gRPC port provided")
	}

	if c.Receivers == nil {
		return fmt.Errorf("no receiver configuration provided")
	}

	for _, svcMesh := range c.Receivers.ServiceMeshes {
		if svcMesh.Name == "" {
			return fmt.Errorf("no service mesh name provided")
		}
		if svcMesh.Namespace == "" {
			return fmt.Errorf("no service mesh namespace provided")
		}
		if svcMesh.Name == util.ServiceMeshIstioSidecar && c.Filters.Envoy == nil {
			return fmt.Errorf("no envoy filter configuration provided for istio sidecar servicemesh")
		}
	}

	for _, other := range c.Receivers.Others {
		if other.Name == "" {
			return fmt.Errorf("no other receiver name provided")
		}
		if other.Name == util.NginxIncorporationIngressController {
			if other.Namespace == "" {
				return fmt.Errorf("no nginx-inc ingress controller namespace provided")
			}
			if c.Filters.NginxIngress == nil {
				return fmt.Errorf("no nginx-inc ingress configuration provided")
			}
			if c.Filters.NginxIngress.DeploymentName == "" {
				return fmt.Errorf("no nginx ingress deployment name provided")
			}
			if c.Filters.NginxIngress.ConfigMapName == "" {
				return fmt.Errorf("no nginx ingress configmap name provided")
			}
			if c.Filters.NginxIngress.SentryFlowNjsConfigMapName == "" {
				return fmt.Errorf("no sentryflow njs configmap name provided")
			}
		}
	}
	return nil
}

func New(configFilePath string, logger *zap.SugaredLogger) (*Config, error) {
	if configFilePath == "" {
		configFilePath = DefaultConfigFilePath
		logger.Warnf("Using default configfile path: %s", configFilePath)
	}

	viper.SetConfigFile(configFilePath)
	if err := viper.ReadInConfig(); err != nil {
		logger.Errorf("Failed to read config file: %v", err)
		return nil, err
	}

	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		logger.Errorf("Failed to unmarshal config file: %v", err)
		return nil, err
	}
	if config.Filters.Server == nil {
		config.Filters.Server = &server{}
	}
	if config.Filters.Server.Port == 0 {
		config.Filters.Server.Port = SentryFlowDefaultFilterServerPort
		logger.Warnf("Using default SentryFlow filter server port %d", config.Filters.Server.Port)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(config)
	if err != nil {
		logger.Errorf("Failed to marshal config file: %v", err)
	}
	logger.Debugf("Config: %s", string(bytes))

	return config, nil
}
