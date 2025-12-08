// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Authors of SentryFlow

package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	_struct "github.com/golang/protobuf/ptypes/struct"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"
	extensionsv1alpha1 "istio.io/api/extensions/v1alpha1"
	"istio.io/api/type/v1beta1"
	"istio.io/client-go/pkg/apis/extensions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/accuknox/SentryFlow/sentryflow/pkg/config"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
)

var istioRootNs = getIstioRootNamespaceFromConfig(getConfig())

func Test_createWasmPlugin(t *testing.T) {
	cfg := getConfig()
	ctx := context.WithValue(context.Background(), util.LoggerContextKey{}, zap.S())
	fakeClient := getFakeClient()

	t.Run("when wasm plugin doesn't exist should create it", func(t *testing.T) {
		// Given
		wasmPlugin := getWasmPlugin()
		want, _ := json.Marshal(wasmPlugin)

		defer func() {
			if err := fakeClient.Delete(ctx, wasmPlugin); err != nil {
				t.Errorf("createWasmPlugin() failed to delete plugin = %v", err)
			}
		}()

		// When
		if err := createWasmPlugin(ctx, cfg, fakeClient); err != nil {
			t.Errorf("createWasmPlugin() error = %v, wantErr = nil", err)
		}

		// Then
		latestWasmPlugin := &v1alpha1.WasmPlugin{}
		_ = fakeClient.Get(ctx, client.ObjectKeyFromObject(wasmPlugin), latestWasmPlugin)
		got, _ := json.Marshal(latestWasmPlugin)

		if string(got) != string(want) {
			t.Errorf("createWasmPlugin() got = %v, want = %v", string(got), string(want))
		}
	})

	t.Run("when wasm plugin already exist should not create new one", func(t *testing.T) {
		// Given
		wasmPlugin := getWasmPlugin()
		want, _ := json.Marshal(wasmPlugin)
		wasmPlugin.ResourceVersion = ""

		if err := fakeClient.Create(ctx, wasmPlugin); err != nil {
			t.Errorf("createWasmPlugin() failed to create error = %v, wantErr = nil", err)
		}

		defer func() {
			if err := fakeClient.Delete(ctx, wasmPlugin); err != nil {
				t.Errorf("createWasmPlugin() failed to delete plugin = %v", err)
			}
		}()

		// When
		if err := createWasmPlugin(ctx, cfg, fakeClient); err != nil {
			t.Errorf("createWasmPlugin() error = %v, wantErr = nil", err)
		}

		// Then
		latestWasmPlugin := &v1alpha1.WasmPlugin{}
		_ = fakeClient.Get(ctx, client.ObjectKeyFromObject(wasmPlugin), latestWasmPlugin)
		got, _ := json.Marshal(latestWasmPlugin)

		if string(got) != string(want) {
			t.Errorf("createWasmPlugin() got = %v, want = %v", string(got), string(want))
		}
	})
}

func Test_deleteWasmPlugin(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LoggerContextKey{}, zap.S())
	fakeClient := getFakeClient()

	t.Run("when wasm plugin exists should delete it and return no error", func(t *testing.T) {
		// Given
		wasmPlugin := getWasmPlugin()
		wasmPlugin.ResourceVersion = ""

		if err := fakeClient.Create(ctx, wasmPlugin); err != nil {
			t.Errorf("deleteWasmPlugin() failed to create wasm plugin error = %v, wantErr = nil", err)
		}

		// When & Then
		if err := deleteWasmPlugin(zap.S(), fakeClient, istioRootNs); err != nil {
			t.Errorf("deleteWasmPlugin() error = %v, wantErr = nil", err)
		}
	})

	t.Run("when wasm plugin doesn't exist should return error", func(t *testing.T) {
		// Given
		errMessage := `wasmplugins.extensions.istio.io "http-filter-gateway" not found`

		// When
		err := deleteWasmPlugin(zap.S(), fakeClient, istioRootNs)

		// Then
		if err == nil {
			t.Errorf("deleteWasmPlugin() error = nil, wantErr = %v", errMessage)
		}

		if err.Error() != errMessage {
			t.Errorf("deleteWasmPlugin() errorMessage = %v, wantErrMessage = %v", err, errMessage)
		}
	})
}

func Test_getIstioRootNamespaceFromConfig(t *testing.T) {
	t.Run("with valid istio-gateway receiver config should return its namespace", func(t *testing.T) {
		if got := getIstioRootNamespaceFromConfig(getConfig()); got != "istio-system" {
			t.Errorf("getIstioRootNamespaceFromConfig() got = %v, want %v", got, "istio-system")
		}
	})
}

func getConfig() *config.Config {
	configFilePath, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "config", "test-configs", "default-config.yaml"))
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path of config file: %v", err))
	}

	cfg, err := config.New(configFilePath, zap.S())
	if err != nil {
		panic(fmt.Errorf("failed to create config: %v", err))
	}

	return cfg
}

func getFakeClient() client.WithWatch {
	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	return fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
}

func getWasmPlugin() *v1alpha1.WasmPlugin {
	cfg := getConfig()

	return &v1alpha1.WasmPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WasmPlugin",
			APIVersion: "extensions.istio.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            FilterName,
			Namespace:       istioRootNs,
			ResourceVersion: "1",
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "sentryflow",
			},
		},
		Spec: extensionsv1alpha1.WasmPlugin{
			Url: fmt.Sprintf("%s:%s", cfg.Filters.Envoy.Uri, cfg.Filters.Envoy.GatewayTag),
			Selector: &v1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					"istio": "ingressgateway",
				},
			},
			PluginConfig: &_struct.Struct{
				Fields: map[string]*_struct.Value{
					"upstream_name": {
						Kind: &_struct.Value_StringValue{
							StringValue: UpstreamAndClusterName,
						},
					},
					"authority": {
						Kind: &_struct.Value_StringValue{
							StringValue: UpstreamAndClusterName,
						},
					},
					"api_path": {
						Kind: &_struct.Value_StringValue{
							StringValue: ApiPath,
						},
					},
				},
			},
			Priority:     &wrapperspb.Int32Value{Value: 100},
			FailStrategy: extensionsv1alpha1.FailStrategy_FAIL_OPEN,
		},
	}
}
