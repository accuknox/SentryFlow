# GCP API Gateway Integration Guide

This guide details how to integrate GCP API Gateway with SentryFlow using Cloud Pub/Sub.

## Prerequisites

- GCP Project with billing enabled.
- `gcloud` CLI installed and authenticated.
- API Gateway set up and handling traffic.

## Architecture

API Gateway (Access Logs) -> Cloud Logging -> Log Sink -> Pub/Sub Topic -> SentryFlow (Receiver)

## Setup Steps

### 1. Enable APIs

```bash
gcloud services enable pubsub.googleapis.com logging.googleapis.com
```

### 2. Create Pub/Sub Topic and Subscription

Create a topic for logs:
```bash
gcloud pubsub topics create sentryflow-logs-topic
```

Create a Pull subscription for SentryFlow:
```bash
gcloud pubsub subscriptions create sentryflow-logs-sub --topic=sentryflow-logs-topic
```

### 3. Enable API Gateway Managed Service
**Critical**: After creating the Gateway, a specific Managed Service is created for it (format: `gateway-<id>...cloud.goog`). You **MUST** enable this service for logs and metrics to be generated.

1.  Get the Managed Service Name:
    ```bash
    gcloud api-gateway apis describe <your-api-id> --project=<your-project-id>
    ```
    Look for the `managedService` field in the output.

2.  Enable the Service:
    ```bash
    gcloud services enable <managed-service-name> --project=<your-project-id>
    ```

### 4. Create Service Account for SentryFlow

SentryFlow needs a Service Account (SA) with permission to pull messages.

```bash
# Create SA
gcloud iam service-accounts create sentryflow-sa --display-name="SentryFlow Service Account"

# Grant Pub/Sub Subscriber role
gcloud projects add-iam-policy-binding $(gcloud config get-value project) \
    --member="serviceAccount:sentryflow-sa@$(gcloud config get-value project).iam.gserviceaccount.com" \
    --role="roles/pubsub.subscriber"
```

### 4. Setup Cloud Logging Sink

Route API Gateway logs to the Pub/Sub topic:
1.  **Destinations**:
    *   Select **Cloud Pub/Sub topic**.
    *   Select the topic created earlier (`sentryflow-logs-topic`).
2.  **Choose logs to include in sink**:
    *   Enter the following filter (replace `<your-project-id>`):
        ```
        resource.type="api"
        resource.labels.service:"apigateway.<your-project-id>.cloud.goog"
        ```
3.  Click **Create Sink**.

**Important**: Grant the Logging Sink identity permission to publish to the topic.
The output of the previous command will show a `writerIdentity`. Grant it the publisher role:

```bash
# Capture writer identity (example: serviceAccount:cloud-logs@system.gserviceaccount.com)
WRITER_IDENTITY=$(gcloud logging sinks describe sentryflow-sink --format="value(writerIdentity)")

gcloud pubsub topics add-iam-policy-binding sentryflow-logs-topic \
    --member="$WRITER_IDENTITY" \
    --role="roles/pubsub.publisher"
```

**Important**: Grant the Service Account permission to access the SA.

```bash
gcloud iam service-accounts add-iam-policy-binding sentryflow-sa@$(gcloud config get-value project).iam.gserviceaccount.com \
    --member="user:$(gcloud config get-value account)" \
    --role="roles/iam.serviceAccountUser"
```

### 5. Generate Service Account Key

Forward the key to SentryFlow.

```bash
gcloud iam service-accounts keys create sentryflow-key.json \
    --iam-account=sentryflow-sa@$(gcloud config get-value project).iam.gserviceaccount.com
```

### 6. Configure SentryFlow

Update `values.yaml` or install with set flags:

```bash
# Path to your downloaded service account key
helm upgrade --install sentryflow ./deployments/sentryflow \
  --namespace sentryflow --create-namespace \
  --set config.receivers.gcp.enabled=true \
  --set config.receivers.gcp.projectID=$(gcloud config get-value project) \
  --set config.receivers.gcp.subscriptionID=sentryflow-logs-sub \
  --set-file secrets.receivers.gcp.serviceAccountJSON=sentryflow-key.json
```

**Note**: For Kubernetes deployment, it's safer to mount the secret. If running locally, the path in `serviceAccountJSON` should point to the file.
