# Nginx Incorporation Ingress Controller

## Description

This guide provides a step-by-step process to integrate SentryFlow
with [Nginx Inc.](https://docs.nginx.com/nginx-ingress-controller/) Ingress Controller, aimed at enhancing API
observability. It includes detailed commands for each step along with their explanations.

SentryFlow make use of following to provide visibility into API calls:

- [Nginx njs](https://nginx.org/en/docs/njs/) module.
- [Njs filter](../../../../../filter/nginx).

## Prerequisites

- Nginx Inc. Ingress Controller.
  Follow [this](https://docs.nginx.com/nginx-ingress-controller/installation/installing-nic/) to deploy it.

## How to

To Observe API calls of your workloads served by Nginx inc. ingress controller in Kubernetes environment, follow
the below
steps:

1. Create the following configmap in the same namespace as ingress controller.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: sentryflow-nginx-inc # Do not change this
  namespace: default
data:
  sentryflow.js: |
    const MAX_BODY_SIZE = 1_000_000; // 1 MB

    function getVar(r, name) {
        try {
            const v = r.variables[name];
            return (v === undefined || v === null) ? "" : v;
        } catch (err) {
            r.error(`getVar failed for ${name}: ${err}`);
            return "";
        }
    }

    function getIntSafely(r, name) {
        try {
            const v = r.variables[name];
            return (v === undefined || v === null) ? 0 : v;
        } catch (err) {
            r.error(`getIntSafely failed for ${name}: ${err}`);
            return 0;
        }
    }
    
    function safeGet(value) {
        return (value === undefined || value === null) ? "" : value;
    }

    function captureRequestBody(r) {
        try {
            let body = r.variables.request_body || "";
            if (body.length > MAX_BODY_SIZE) {
                r.log(`REQUEST BODY OVER 1 MB LIMIT, truncating`);
                body = body.slice(0, MAX_BODY_SIZE);
            }
            return body;
        } catch (err) {
            r.error(`Failed to get request body: ${err}`);
            return "";
        }
    }
    

    function responseHandler(r, data, flags) {
        try {
            
            r.sendBuffer(data, flags);

            if (!r._respInit) {
                r._respChunks = [];
                r._respBytes = 0;
                r._tooLarge = false;
                r._respInit = true;

            }
        
            if (data && !r._tooLarge) {
                const newSize = r._respBytes + data.length;
                

                if (newSize > MAX_BODY_SIZE) {
                    r.log(`RESPONSE BODY OVER 1 MB LIMIT, NOT CAPTURING RESPONSE BODY`);
                    r._tooLarge = true;
                    r._respChunks = null;
                    r._respBytes = 0;
                } else {
                    r._respChunks.push(data);
                    r._respBytes = newSize;
                }
            }
            
            if (!flags.last) return;

            let responseBody = ""
            
            if (r._respBytes > 0) {
                try {
                    const merged = Buffer.concat(r._respChunks);
                    responseBody = new TextDecoder("utf-8").decode(merged);
                } catch (err) {
                    r.error(`Failed to decode response body: ${err}`);
                    responseBody = "";
                }
            }

            // Safety check: final string shouldn't exceed max
            //
            if (responseBody.length > MAX_BODY_SIZE) {
                r.log(`FINAL STRING TOO LARGE: length=${responseBody.length} → cleared`);
                responseBody = "";
            }

            r._respChunks = null;
            r._respBytes = 0;

            let apiEvent = {
                "metadata": {
                    "timestamp": Date.parse(r.variables.time_iso8601.split("+")[0]) / 1000,
                    "receiver_name": "nginx",
                    "receiver_version": ngx.version,
                },
                "source": {
                    "ip": safeGet(r.remoteAddress),
                    "port": getIntSafely(r, "remote_port"),
                },
                "destination": {
                    "ip": getVar(r, "server_addr"),
                    "port": getIntSafely(r, "server_port"),
                },
                "request": {
                    "headers": {},
                    "body": getVar(r, "body_text"),
                },
                "response": {
                    "headers": {},
                    "body": responseBody,
                },
                "protocol": getVar(r, "server_protocol"),
            };

            const headersIn = r.headersIn || {};
            for (const header in headersIn) {
                const value = headersIn[header];
                apiEvent.request.headers[header] = Array.isArray(value) ? value.join(",") : value;
            }

            apiEvent.request.headers[":scheme"] = getVar(r, "scheme")
            apiEvent.request.headers[":path"] = safeGet(r.uri)
            apiEvent.request.headers[":method"] = getVar(r, "request_method")

            apiEvent.request.headers["body_bytes_sent"] = getVar(r, "body_bytes_sent")
            apiEvent.request.headers["request_length"] = getVar(r, "request_length")
            apiEvent.request.headers["request_time"] = getVar(r, "request_time")
            apiEvent.request.headers["query"] = getVar(r, "query_string")
            
            const headersOut = r.headersOut || {};
            for (const header in headersOut) {
                const value = headersOut[header];
                apiEvent.response.headers[header] = Array.isArray(value) ? value.join(",") : value;
            }
        
            apiEvent.response.headers[":status"] = getVar(r, "status")

            r.subrequest("/sentryflow", {
                method: "POST",
                body: JSON.stringify(apiEvent),
                detached: true,
            });
        } catch (err) {
            r.error(`responseHandler failed: ${err}`);
            // send minimal event for failure
            if (flags.last) {
                r.subrequest("/sentryflow", {
                    method: "POST",
                    body: JSON.stringify({ error: err.toString() }),
                    detached: true,
                });
            }
        }
    }

    export default {responseHandler, captureRequestBody};
EOF
```

2. Add the following volume and volume-mount in ingress controller deployment:

```yaml
...
volumes:
  - name: sentryflow-nginx-inc
    configMap:
      name: sentryflow-nginx-inc
...
...
volumeMounts:
  - mountPath: /etc/nginx/njs/sentryflow.js
    name: sentryflow-nginx-inc
    subPath: sentryflow.js
```

3. Update ingress controller configmap as follows:

```yaml
...
data:
  http-snippets: |
    js_path "/etc/nginx/njs/";
    subrequest_output_buffer_size 32k;
    js_import main from sentryflow.js;
  location-snippets: |
    js_body_filter main.responseHandler buffer_type=buffer;
    js_set $body_text main.captureRequestBody;
  server-snippets: |
    location /sentryflow {
      internal;
      # Update SentryFlow URL with path to ingest access logs if required.
      proxy_pass http://sentryflow.sentryflow:8081/api/v1/events;
      proxy_method      POST;
      proxy_set_header accept "application/json";
      proxy_set_header Content-Type "application/json";
    }
```

4. Install Sentryflow

Update the nginx deployment name, config map name, and namespace name before running the command. 

  ```shell
  helm upgrade --install sentryflow \
  oci://public.ecr.aws/k9v9d5v2/sentryflow-helm-charts \
  --version v0.1.4 \
  --namespace sentryflow \
  --create-namespace \
  --set config.receivers.nginxIngressController.enabled=true \
  --set config.receivers.nginxIngressController.deploymentName=<nginx deployment name> \
  --set config.receivers.nginxIngressController.configMapName=<nginx config map name> \
  --set config.receivers.nginxIngressController.namespace=<namespace where ingress controller is deployed> 
  ```

5. Modify discovery engine config map and restart the discovery engine deployment

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

7. Trigger API calls to generate traffic.

8. Use SentryFlow [log client](../../../../client) to see the API Events.
