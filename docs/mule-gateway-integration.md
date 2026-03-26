# Mule Gateway Integration with SentryFlow

This guide covers the E2E integration of **Anypoint Mule Gateway** with SentryFlow: building
the custom policy, publishing it to Anypoint Exchange, setting up an API + proxy on your Anypoint
SaaS account, and verifying that traffic flows through to SentryFlow.

## Architecture Overview

```
[HTTP Client]
     │ sends request
     ▼
[Anypoint API Manager]
     │ applies SentryFlow policy (injected at proxy deploy time)
     ▼
[CloudHub / On-prem Mule Runtime + Mule Gateway]
     │  ① forwards request to backend
     │  ② BEFORE hook: captures method, path, headers, timestamp
     │  ③ AFTER hook: captures response code, headers, latency
     │       └─► HTTP POST (JSON) ──► SentryFlow :8081/api/v1/events/mule
     ▼
[Your Backend API]
```

Key insight: policies are **automatically injected** into every API proxy by Mule Gateway
when the proxy connects to Anypoint API Manager. No manual code changes in the Mule app.

---

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Java | 11+ | `sudo apt install openjdk-11-jdk` |
| Maven | 3.6+ | `sudo apt install maven` |
| Anypoint CLI | 4.x | already installed |
| Anypoint Platform account | — | [anypoint.mulesoft.com](https://anypoint.mulesoft.com) |

---

## Step 1 — Build the Policy JAR

```shell
cd filter/mule-gateway/policy-implementation
mvn clean package
```

This produces:
- `target/sentryflow-mule-policy-1.0.0-mule-policy.jar` — the policy binary
- You also need: `sentryflow-implementation-metadata.yaml` — already in the same directory

---

## Step 2 — Publish Policy Definition to Anypoint Exchange

> **What you saw in the UI was wrong** — "Custom" asset type is for generic files.
> Custom policies require the **"Policy"** asset type, and you upload only **one file**
> (the YAML definition) in this first step.

### 2.1 Publish the Policy Definition (UI)

1. Go to **Exchange → Publish new asset**
2. Fill in:
   - **Name:** `SentryFlow API Telemetry`
   - **Asset type:** `Policy` 
   - **File upload:** 
      - upload JSON Schema file `filter/mule-gateway/policy-definition/sentryflow-policy.json`
      - upload Metadata file `filter/mule-gateway/policy-definition/sentryflow-policy.yaml`
3. Click **Advanced** and set:
   - **Group ID:** your Anypoint org ID (auto-filled)
   - **Asset ID:** `sentryflow-mule-policy`
   - **Version:** `1.0.0`
4. **Lifecycle state:** `Stable`
5. Click **Publish**

You will see `Implementations: Pending` in the left sidebar — that's expected.

### 2.2 Upload the Policy Implementation (UI)

1. Open the newly created **SentryFlow API Telemetry** asset in Exchange
2. Click **Implementations** in the left sidebar
3. Click **Add implementation**
4. Fill in:
   - **Name:** `SentryFlow Mule 4 Policy`
   - **JAR file:** `target/sentryflow-mule-policy-1.0.0-mule-policy.jar`
   - **YAML file:** `sentryflow-implementation-metadata.yaml`
5. Click **Advanced** → set **version:** `1.0.0`
6. Click **Add implementation**

The policy is now available in Anypoint API Manager.

---

## Step 3 — E2E Test: Set Up an API + Proxy on Anypoint SaaS

To test the full flow, you need a Mule API with a proxy deployed on CloudHub. Here's how
to set this up from scratch using your existing Anypoint SaaS account.

### 3.1 Create an API Instance in API Manager

1. Go to **Anypoint Platform → API Manager**
2. Select your environment (e.g., `Sandbox`)
3. Click **Add API → Add new API**
4. Fill in:
   - **Runtime:** `Mule Gateway`
   - **Proxy type:** `Deploy a proxy application` (CloudHub managed)
   - **Proxy application name** (e.g. `sentryflow-mule-test`)
5. Click **Next**, then configure Create new api:
   - **API type:** `HTTP API`
   - add asset name 
6. Click **Next**, then configure Downstream:
   - Enable Manual approval 
7. Click **Next**, then configure Upsteam:
   - **Backend URL:** use a public test API, e.g. `https://jsonplaceholder.typicode.com`
6. Click **Save & Deploy**
7. Select your **CloudHub** target and set a **Proxy application name** (e.g. `sentryflow-mule-test`)
   > *Note: The **Proxy application name** is just a unique identifier for your proxy app on CloudHub. It will determine the final URL of your proxy (e.g. `sentryflow-mule-test.us-e2.cloudhub.io`). It must be globally unique across all of CloudHub.*
8. Click **Deploy**

> Mule Gateway will generate and deploy an **auto-generated API proxy** to CloudHub that
> forwards requests to `jsonplaceholder.typicode.com`.

### 3.2 Note your CloudHub Proxy URL

After deployment (takes ~2 minutes), find your proxy's public URL:

- Go to **Runtime Manager → Applications**
- Find `sentryflow-mule-test`
- Copy the **App URL**, e.g. `https://sentryflow-mule-test.cloudhub.io`

### 3.3 Apply the SentryFlow Policy

1. Back in **API Manager**
2. Click **Automated Policies** in the left sidebar
3. Click **Add Automated policy**
4. Search for `SentryFlow API Telemetry` 
5. Click **Apply** and configure:

| Field | Value |
|-------|-------|
| **SentryFlow Host** | `<your SentryFlow host>` (see section below) |
| **SentryFlow Events Path** | `/api/v1/events/mule` |
| **Forward Request Body** | `false` |
| **Forward Response Body** | `false` |
| **Timeout (ms)** | `5000` |

6. Click **Apply**

> The policy is now injected into the CloudHub proxy. Mule will automatically download
> and apply it within ~30 seconds — **no proxy restart required**.

---

## Step 4 — Make SentryFlow Reachable from CloudHub

CloudHub is a cloud runtime, so SentryFlow must be publicly reachable

### Option A — Quick local test with ngrok (recommended for dev)

```shell
# Run SentryFlow locally
cd sentryflow && go run main.go

# In a second terminal — expose port 8081 publicly
ngrok http 8081

# ngrok gives you a URL like: https://abc123.ngrok.io
# Use that as sentryflowHost in the policy config
```

> In API Manager policy config, set **SentryFlow Host** to `abc123.ngrok.io`.
> (The policy will default to communicating over port 80).

### Option B — Kubernetes with public LoadBalancer

```shell
# If SentryFlow is already on K8s, expose it:
kubectl expose deployment sentryflow \
  --type=LoadBalancer \
  --port=8081 \
  --target-port=8081 \
  -n sentryflow

# Get the external IP:
kubectl get svc -n sentryflow
```

Use the external IP as **SentryFlow Host** in the policy config.

---

## Step 5 — Deploy SentryFlow to Kubernetes

> **Skip this step** if SentryFlow is already running somewhere accessible.

The SentryFlow Mule image is published to Docker Hub. Deploy it into your cluster using Helm:

```shell
# From the root of the SentryFlow repository
helm upgrade --install sentryflow deployments/sentryflow \
  --namespace sentryflow \
  --create-namespace \
  --set image.repository=sanskardevops/sentryflow-mule \
  --set image.tag=0.0.3
```

Verify the pod is healthy:

```shell
kubectl -n sentryflow get pods
# NAME                         READY   STATUS    RESTARTS   AGE
# sentryflow-xxxxxxxxx-xxxxx   1/1     Running   0          30s

kubectl -n sentryflow logs deployment/sentryflow | grep -E "HTTP|mule|gRPC"
# {"level":"INFO","msg":"Mule Gateway receiver registered on shared HTTP server"}
# {"level":"INFO","msg":"HTTP server listening on port 8081"}
```

### Port-forward for local testing (when using ngrok)

If you are using ngrok to expose SentryFlow to CloudHub, port-forward the service first:

```shell
kubectl -n sentryflow port-forward svc/sentryflow 8081:8081
```

Then in a separate terminal start ngrok:

```shell
ngrok http 8081
```

Use the ngrok HTTP URL host (e.g. `abc123.ngrok.io`) as the **SentryFlow Host** in API Manager.
The policy will default to connecting over port 80.

---

## Step 6 — Configure SentryFlow

Update your SentryFlow `config/default.yaml` to enable the Mule Gateway receiver:

```yaml
filters:
  httpServer:
    port: 8081
  muleGateway:
    enabled: true
receivers:
  other:
    - name: mule-gateway

exporter:
  grpc:
    port: 8080
```

---

## Step 7 — E2E Verification

### 7.1 Verify SentryFlow receiver is listening

```shell
curl -s http://localhost:8081/api/v1/events/mule/health
# → {"status":"ok","receiver":"mule-gateway"}
```

### 7.2 Send a request through the Mule proxy

```shell
# Replace with your CloudHub proxy URL
curl -s https://sentryflow-mule-test.cloudhub.io/todos/1
```

Example response (forwarded from jsonplaceholder):
```json
{
  "userId": 1,
  "id": 1,
  "title": "delectus aut autem",
  "completed": false
}
```

### 7.3 Check SentryFlow logs for the event

```shell
# Local
./sentryflow 2>&1 | grep -i mule

# Docker
docker logs sentryflow 2>&1 | grep mule

# Kubernetes
kubectl -n sentryflow logs deployment/sentryflow --tail=50 | grep mule
```

Expected output:
```
{"level":"INFO","msg":"Mule Gateway receiver listening on :8081/api/v1/events/mule"}
```

### 7.4 Verify via Anypoint Analytics (optional)

In API Manager → your API → **Analytics**, you should see the request counted.
This is independent of SentryFlow and confirms the proxy+policy is working.

---

## Troubleshooting

### "Policy" type not visible in Exchange

The **Policy** asset type only appears if your Anypoint account has API Manager enabled.
In a trial account it may require activating the API Management entitlement.
Contact your Anypoint admin or check **Access Management → Entitlements**.

### Policy shows as "Pending" in Exchange

This means the implementation JAR has not been uploaded yet. Follow Step 2.2 above.

### CloudHub proxy not receiving requests

- Check **Runtime Manager → sentryflow-mule-test → Logs** for deploy errors
- The app URL format is `https://<app-name>.cloudhub.io` — no port needed (CloudHub uses 443)

### SentryFlow not receiving events from CloudHub

1. Verify the ngrok/LoadBalancer URL is correct in the policy config
2. From the Mule runtime logs (Runtime Manager → Logs), look for:
   ```
   [SentryFlow] Failed to send telemetry: ...
   ```
3. Test connectivity from inside CloudHub: you cannot do this directly, but verify by
   checking if `ngrok` shows incoming connections, or the LoadBalancer access logs

### Policy applied but no `SentryFlow API Telemetry` in policy list

- The policy definition was published as type "Custom" instead of "Policy"
- Delete the asset in Exchange and republish it with the correct **Policy** type

---

## File Reference

| File | Purpose |
|------|---------|
| `filter/mule-gateway/policy-definition/sentryflow-policy.yaml` | **Upload this** in Exchange Step 2.1 |
| `filter/mule-gateway/policy-definition/sentryflow-policy.json` | JSON schema (reference documentation) |
| `filter/mule-gateway/policy-implementation/pom.xml` | Maven build config |
| `filter/mule-gateway/policy-implementation/sentryflow-implementation-metadata.yaml` | **Upload this** in Exchange Step 2.2 (with JAR) |
| `filter/mule-gateway/policy-implementation/src/main/mule/sentryflow-mule-policy.xml` | Core policy logic |
| `sentryflow/pkg/receiver/other/mulegateway/mule.go` | SentryFlow HTTP receiver for Mule events |
