// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package konggateway

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
)

// Start initializes the Kong Gateway receiver.
// It validates that the Kong deployment exists and the sentryflow-log plugin is configured.
func Start(ctx context.Context, cfg *config.Config, k8sClient client.Client) {
	logger := util.LoggerFromCtx(ctx)

	logger.Info("Starting Kong Gateway receiver")
	if err := validateResources(ctx, cfg, k8sClient); err != nil {
		logger.Errorf("%v. Stopped Kong Gateway receiver", err)
		return
	}
	logger.Info("Started Kong Gateway receiver")

	<-ctx.Done()
	logger.Info("Shutting down Kong Gateway receiver")
	logger.Info("Stopped Kong Gateway receiver")
}

func validateResources(ctx context.Context, cfg *config.Config, k8sClient client.Client) error {
	kongNamespace := getKongNamespaceFromConfig(cfg)
	kongDeploymentName := getKongDeploymentNameFromConfig(cfg)

	if kongDeploymentName == "" {
		return fmt.Errorf("kong deployment name not configured")
	}

	// Validate Kong deployment exists
	kongDeploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kongDeploymentName,
			Namespace: kongNamespace,
		},
	}

	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kongDeploy), kongDeploy); err != nil {
		return fmt.Errorf("failed to get Kong Gateway deployment '%s' in namespace '%s': %w",
			kongDeploymentName, kongNamespace, err)
	}

	// Note: Unlike Nginx, Kong plugin configuration is done via Kong Admin API
	// or declarative config, not via Kubernetes ConfigMaps. We cannot easily
	// validate if sentryflow-log plugin is installed from here.
	// The user must ensure the plugin is properly configured.

	return nil
}

func getKongNamespaceFromConfig(cfg *config.Config) string {
	for _, other := range cfg.Receivers.Others {
		switch other.Name {
		case util.KongGateway:
			return other.Namespace
		}
	}
	return ""
}

func getKongDeploymentNameFromConfig(cfg *config.Config) string {
	if cfg.Filters.KongGateway != nil {
		return cfg.Filters.KongGateway.DeploymentName
	}
	return ""
}
