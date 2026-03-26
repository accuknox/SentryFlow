// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package util

import (
	"context"

	"go.uber.org/zap"
)

type LoggerContextKey struct{}

const (
	ServiceMeshIstioSidecar             = "istio-sidecar"
	ServiceMeshIstioGateway             = "istio-gateway"
	ServiceMeshIstioAmbient             = "istio-ambient"
	ServiceMeshKong                     = "kong"
	ServiceMeshConsul                   = "consul"
	ServiceMeshLinkerd                  = "linkerd"
	OpenTelemetry                       = "otel"
	NginxWebServer                      = "nginx-webserver"
	NginxIncorporationIngressController = "nginx-inc-ingress-controller" // https://github.com/nginxinc/kubernetes-ingress/
	KongGateway                         = "kong-gateway"                 // https://konghq.com/
	AzureAPIM                           = "Azure-APIM"
	AWSApiGateway                       = "aws-api-gateway"
	F5BigIp                             = "f5-big-ip"
	MuleGateway                         = "mule-gateway" // https://docs.mulesoft.com/mule-gateway/mule-gateway-capabilities-mule4
)

func LoggerFromCtx(ctx context.Context) *zap.SugaredLogger {
	logger, _ := ctx.Value(LoggerContextKey{}).(*zap.SugaredLogger)
	return logger
}
