// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package receiver

import (
	"context"
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/receiver/other/nginx/nginxinc"
	istiogateway "github.com/accuknox/SentryFlow/sentryflow/pkg/receiver/svcmesh/istio/gateway"
	istiosidecar "github.com/accuknox/SentryFlow/sentryflow/pkg/receiver/svcmesh/istio/sidecar"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
)

// Init initializes the API event sources based on the provided configuration. It
// starts monitoring from configured sources and supports adding other sources in
// the future.
func Init(ctx context.Context, k8sClient client.Client, cfg *config.Config, wg *sync.WaitGroup, lock *sync.Mutex) error {
	logger := util.LoggerFromCtx(ctx).Named("receiver")

	for _, serviceMesh := range cfg.Receivers.ServiceMeshes {
		if serviceMesh.Name != "" {
			switch serviceMesh.Name {
			case util.ServiceMeshIstioSidecar:
				wg.Add(1)
				go func() {
					defer wg.Done()
					istiosidecar.StartMonitoring(ctx, cfg, k8sClient, lock)
				}()
			case util.ServiceMeshIstioGateway:
				wg.Add(1)
				go func() {
					defer wg.Done()
					istiogateway.StartMonitoring(ctx, cfg, k8sClient, lock)
				}()
			default:
				return fmt.Errorf("unsupported Service Mesh, %v", serviceMesh.Name)
			}
		}
	}

	for _, other := range cfg.Receivers.Others {
		if other.Name != "" {
			switch other.Name {
			case util.NginxWebServer:
				logger.Info("Started nginx webserver receiver")
			case util.AzureAPIM:
				logger.Info("Started Azure APIM receiver")
			case util.AWSApiGateway:
				logger.Info("Started AWS API Gateway receiver")
			case util.NginxIncorporationIngressController:
				wg.Add(1)
				go func() {
					defer wg.Done()
					nginxinc.Start(ctx, cfg, k8sClient)
				}()
			default:
				return fmt.Errorf("unsupported receiver, %v", other.Name)
			}
		}
	}

	return nil
}
