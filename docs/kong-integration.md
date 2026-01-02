# Kong Gateway Integration with SentryFlow

This guide describes how to integrate Kong Gateway with SentryFlow for API security monitoring.

## Prerequisites

- Kubernetes cluster (e.g., kind, EKS, GKE)
- kubectl configured
- Helm 3.x installed
- AccuKnox agents installed

## Steps

### 1. Install Kong Gateway

```shell
helm repo add kong https://charts.konghq.com
helm repo update

# Install Kong with default settings (CRDs disabled for kind if preferred)
helm install kong kong/kong \
  --namespace kong \
  --create-namespace \
  --set ingressController.installCRDs=false \
  --set admin.enabled=true \
  --set admin.type=NodePort
```

### 2. Install SentryFlow Custom Plugin

We need to mount the `sentryflow-log` plugin into the Kong proxy container.

**Create ConfigMap from plugin files:**

```shell
# Create ConfigMap
kubectl create configmap sentryflow-log-plugin \
  --from-file=handler.lua=filter/kong/sentryflow-log/handler.lua \
  --from-file=schema.lua=filter/kong/sentryflow-log/schema.lua \
  -n kong
```

**Patch Kong Deployment:**

Mount the plugin into the **proxy container** (usually index 1) and the **ingress-controller** (index 0).

```shell
kubectl patch deployment kong-kong -n kong --type=json -p='[
  {
    "op": "add",
    "path": "/spec/template/spec/volumes/-",
    "value": {
      "name": "sentryflow-log-plugin",
      "configMap": {
        "name": "sentryflow-log-plugin"
      }
    }
  },
  {
    "op": "add",
    "path": "/spec/template/spec/containers/1/volumeMounts/-",
    "value": {
      "name": "sentryflow-log-plugin",
      "mountPath": "/usr/local/share/lua/5.1/kong/plugins/sentryflow-log",
      "readOnly": true
    }
  }
]'

# Enable the plugin in Kong
kubectl set env deployment/kong-kong -n kong KONG_PLUGINS=bundled,sentryflow-log
```

> **Note:** Ensure the `volumeMount` is applied to the `proxy` container.

### 3. Deploy SentryFlow

Deploy SentryFlow with Kong receiver enabled.

```shell
helm upgrade --install sentryflow \
  ./deployments/sentryflow \
  --namespace sentryflow \
  --create-namespace \
  --set image.repository=sanskardevops/sentryflow \
  --set image.tag=latest \
  --set config.receivers.kongGateway.enabled=true \
  --set config.receivers.kongGateway.namespace=kong \
  --set config.receivers.kongGateway.deploymentName=kong-kong
```

### 4. Enable sentryflow-log Plugin Globally

Create a `KongClusterPlugin` to enable logging for all routes.

```shell
kubectl apply -f - <<EOF
apiVersion: configuration.konghq.com/v1
kind: KongClusterPlugin
metadata:
  name: sentryflow-log
  annotations:
    kubernetes.io/ingress.class: kong
  labels:
    global: "true"
plugin: sentryflow-log
config:
  http_endpoint: "http://sentryflow.sentryflow:8081/api/v1/events"
  timeout: 10000
  keepalive: 60000
EOF
```

### 5. Patch Discovery Engine

Update the discovery-engine ConfigMap (`discovery-engine-sumengine`) to use SentryFlow and restart the deployment.

```shell
kubectl -n agents edit configmap discovery-engine-sumengine 
` ``

```yaml
  data:
  app.yaml: |
    ...
    summary-engine:
      sentryflow:
        cron-interval: 0h0m30s
        decode-jwt: true
        enabled: true
        include-bodies: true
        redact-sensitive-data: false
        sensitive-rules-files-path:
        - /var/lib/sumengine/sensitive-data-rules.yaml
        threshold: 10000
    watcher:
    ...
      sentryflow:
        enabled: true
        event-type:
          access-log: true
          metric: false
        service:
          enabled: true
          name: sentryflow
          port: "8080"
          url: "sentryflow.sentryflow"
```

```shell
kubectl -n agents rollout restart deployment/discovery-engine
```

## Verification Guide

### 1. Deploy Sample Application
Deploy the Google Microservices Demo `frontend` service (or any HTTP service).

```shell
# Verify frontend service exists
kubectl get svc
```

### 2. Create Ingress Resource
Create an Ingress to route traffic through Kong to your service.

```shell
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: frontend-ingress
  annotations:
    konghq.com/strip-path: "true"
spec:
  ingressClassName: kong
  rules:
    - http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 80
EOF
```

### 3. Generate Traffic
Port-forward the Kong proxy and send requests.

```shell
# Port forward Kong Proxy
kubectl port-forward -n kong svc/kong-kong-proxy 8000:80 &

# Send traffic
sleep 2
curl -s http://localhost:8000/ > /dev/null
curl -s http://localhost:8000/cart > /dev/null
```

### 4. Verify SentryFlow Logs
Check SentryFlow logs to confirm it received the events.

```shell
kubectl -n sentryflow logs deployment/sentryflow --tail=50
```

You should see logs indicating receipt of events:
```
{"level":"INFO",...,"msg":"Received API Event from kong"}
```
