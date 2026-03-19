use log::{debug, error, info, warn};
use proxy_wasm::traits::{Context, HttpContext, RootContext};
use proxy_wasm::types::{Action, ContextType, LogLevel};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::{Duration, UNIX_EPOCH};

const HEADER_X_SENTRYFLOW_TLS_VERSION: &str = "x-sentryflow-tls-version";
const MAX_BODY_SIZE: usize = 1_000_000; // 1 MB

// Add two accumulation buffers to Plugin struct
#[derive(Default)]
struct Plugin {
    _context_id: u32,
    config: PluginConfig,
    api_event: APIEvent,
    auth_token_id: u32,
    request_body_buffer: Vec<u8>,
    response_body_buffer: Vec<u8>,
}

#[derive(Deserialize, Clone, Default, Debug)]
struct PluginConfig {
    upstream_name: String,
    api_path: String,
    authority: String,
    #[cfg(feature = "rate-limit")]
    auth_upstream: String,
    #[cfg(feature = "rate-limit")]
    auth_authority: String,
    #[cfg(feature = "rate-limit")]
    auth_path: String,
}

#[derive(Serialize, Default, Clone)]
#[cfg(feature = "rate-limit")]
struct AuthRequest {
    id: String,
    source: String,
    method: String,
    path: String,
    headers: HashMap<String, String>,
}

#[derive(Serialize, Default, Clone)]
struct APIEvent {
    metadata: Metadata,
    request: Request,
    response: Response,
    source: Workload,
    destination: Workload,
    protocol: String,
}

#[derive(Serialize, Default, Clone)]
struct Metadata {
    context_id: u32,
    timestamp: u64,
    receiver_name: String,
    receiver_version: String,
    mesh_id: String,
    node_name: String,
}

#[derive(Serialize, Default, Clone)]
struct Workload {
    ip: String,
    port: u16,
}

#[derive(Serialize, Clone, Default)]
struct Request {
    headers: HashMap<String, String>,
    body: String,
}

#[derive(Serialize, Clone, Default, Debug)]
struct Response {
    headers: HashMap<String, String>,
    body: String,
    backend_latency_in_nanos: u64,
}

fn _start() {
    proxy_wasm::main! {{
        proxy_wasm::set_log_level(LogLevel::Warn);
        proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {Box::new(Plugin::default())});
    }}
}

#[cfg(feature = "rate-limit")]
impl Plugin {
    /// Dispatches the AuthRequest to the external Rate Limiter
    fn check_rate_limit(&mut self) -> Action {
        let auth_request = AuthRequest {
            id: self._context_id.to_string(),
            source: self.api_event.source.ip.clone(),
            method: self.api_event.request.headers.get(":method").cloned().unwrap_or_default(),
            path: self.api_event.request.headers.get(":path").cloned().unwrap_or_default(),
            headers: self.api_event.request.headers.clone(),
        };

        let body = match serde_json::to_vec(&auth_request) {
            Ok(b) => b,
            Err(e) => {
                error!("Failed to serialize AuthRequest: {:?}", e);
                return Action::Continue;
            }
        };

        let dispatch_headers = vec![
            (":method", "POST"),
            (":authority", &self.config.auth_authority),
            (":path", &self.config.auth_path),
            ("content-type", "application/json"),
        ];

        match self.dispatch_http_call(
            &self.config.auth_upstream,
            dispatch_headers,
            Some(&body),
            vec![],
            Duration::from_millis(500),
        ) {
            Ok(token_id) => {
                self.auth_token_id = token_id;
                Action::Pause
            }
            Err(e) => {
                error!("Auth dispatch failed: {:?}", e);
                Action::Continue
            }
        }
    }
}

impl Context for Plugin {
    fn on_http_call_response(&mut self, token_id: u32, _num_headers: usize, _body_size: usize, _num_trailers: usize) {
        if token_id != self.auth_token_id {
            return;
        }
        let headers = self.get_http_call_response_headers_bytes();
        if headers.is_empty() {
            warn!("Rate limiter returned no headers. Resuming anyway.");
            self.resume_http_request();
            return;
        }
        let status = headers.iter()
            .find(|(k, _)| k.to_lowercase() == ":status")
            .map(|(_, v)| String::from_utf8_lossy(v).into_owned())
            .unwrap_or_else(|| "500".to_string());

        if status == "429" {
            info!("Context {}: Rate limit EXCEEDED (429). Blocking.", self._context_id);
            self.send_http_response(
                429,
                vec![("content-type", "text/plain")],
                Some(b"Rate limit exceeded."),
            );
        } else {
            if status != "200" {
                warn!("Context {}: Rate limiter returned unexpected status {}. Failing open.", self._context_id, status);
            }
            self.resume_http_request();
        }
    }

    fn on_done(&mut self) -> bool {
        // Flush buffers on abrupt disconnect
        // end_of_stream may never fire if the client drops the connection mid-stream
        // Promote whatever was accumulated.
        if self.api_event.request.body.is_empty() && !self.request_body_buffer.is_empty() {
            self.api_event.request.body =
                String::from_utf8_lossy(&self.request_body_buffer).into_owned();
        }
        if self.api_event.response.body.is_empty() && !self.response_body_buffer.is_empty() {
            self.api_event.response.body =
                String::from_utf8_lossy(&self.response_body_buffer).into_owned();
        }

        info!("Context {} finished, dispatching telemetry", self._context_id);
        dispatch_http_call_to_upstream(self);
        true
    }
}

impl RootContext for Plugin {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            if let Ok(config) = serde_json::from_slice::<PluginConfig>(&config_bytes) {
                self.config = config;
            } else {
                error!("Failed to parse plugin config");
            }
        } else {
            error!("No plugin config found");
        }
        true
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        debug!("Creating HTTP context {}", _context_id);
        Some(Box::new(Plugin {
            _context_id,
            config: self.config.clone(),
            api_event: Default::default(),
            auth_token_id: 0,
            // initialize buffers per-context
            request_body_buffer: Vec::new(),
            response_body_buffer: Vec::new(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

impl HttpContext for Plugin {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Fetch source address once, reuse — was fetched twice
        let raw_src = String::from_utf8(
            self.get_property(vec!["source", "address"]).unwrap_or_default(),
        )
        .unwrap_or_default();

        info!("Source address: {}", raw_src);

        let (mut src_ip, src_port) = get_url_and_port(raw_src);

        debug!(
            "Processing request headers for context {}: src={}:{}",
            self._context_id, src_ip, src_port
        );

        let req_headers = self.get_http_request_headers();
        let mut headers: HashMap<String, String> = HashMap::with_capacity(req_headers.len());
        for (key, value) in req_headers {
            // Don't include Envoy's pseudo headers
            // https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#id13
            if !key.starts_with("x-envoy") {
                headers.insert(key.clone(), value.clone());
            }
            // Case-insensitive x-forwarded-for; avoid Vec alloc
            if key.to_lowercase() == "x-forwarded-for" {
                if let Some(first_ip) = value.split(',').next() {
                    src_ip = first_ip.trim().to_string();
                }
            }
        }

        headers.insert(
            "query".to_string(),
            String::from_utf8(
                self.get_property(vec!["request", "query"]).unwrap_or_default(),
            )
            .unwrap_or_default(),
        );
        headers.insert(
            ":path".to_string(),
            String::from_utf8(
                self.get_property(vec!["request", "url_path"]).unwrap_or_default(),
            )
            .unwrap_or_default(),
        );

        // https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes#connection-attributes
        if let Some(tls_version) = self.get_property(vec!["connection", "tls_version"]) {
            headers.insert(
                HEADER_X_SENTRYFLOW_TLS_VERSION.to_string(),
                String::from_utf8(tls_version).unwrap_or_default(),
            );
        }

        self.api_event.metadata.timestamp = self
            .get_current_time()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();
        self.api_event.metadata.context_id = self._context_id;
        self.api_event.request.headers = headers;

        let protocol = String::from_utf8(
            self.get_property(vec!["request", "protocol"]).unwrap_or_default(),
        )
        .unwrap_or_default();
        self.api_event.protocol = protocol;

        self.api_event.source.ip = src_ip;
        self.api_event.source.port = src_port;

        #[cfg(feature = "rate-limit")]
        {
            return self.check_rate_limit();
        }
        Action::Continue
    }

    fn on_http_request_body(&mut self, _body_size: usize, _end_of_stream: bool) -> Action {
        // Accumulate chunks instead of overwriting
        // Envoy flushes the body buffer after Action::Continue, so each call
        // only contains the current chunk starting at offset 0.
        // We append to our own buffer and finalize only when end_of_stream fires.
        if let Some(chunk) = self.get_http_request_body(0, _body_size) {
            if self.request_body_buffer.len() + chunk.len() <= MAX_BODY_SIZE {
                self.request_body_buffer.extend_from_slice(&chunk);
            } else {
                info!(
                    "Context {}: Request body exceeded MAX_BODY_SIZE ({} bytes), capping capture",
                    self._context_id,
                    self.request_body_buffer.len() + chunk.len()
                );
            }
        }

        if _end_of_stream && !self.request_body_buffer.is_empty() {
            self.api_event.request.body =
                String::from_utf8_lossy(&self.request_body_buffer).into_owned();
            debug!(
                "Context {}: Request body fully captured ({} bytes)",
                self._context_id,
                self.api_event.request.body.len()
            );
        }

        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes#upstream-attributes
        let (mut dest_ip, mut dest_port) = get_url_and_port(
            String::from_utf8(
                self.get_property(vec!["upstream", "address"]).unwrap_or_default(),
            )
            .unwrap_or_default(),
        );

        // For `OPTIONS` requests, the upstream connection remote address is often unavailable.
        // This is due to browser CORS preflight behavior, which may prevent Envoy from
        // establishing a full upstream connection.
        // In such cases, fallback to the downstream local address, which represents
        // the IP address of the Envoy listener handling the client connection.
        // https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/advanced/attributes#connection-attributes
        if dest_ip.is_empty() || dest_port == 0 {
            (dest_ip, dest_port) = get_url_and_port(
                String::from_utf8(
                    self.get_property(vec!["destination", "address"]).unwrap_or_default(),
                )
                .unwrap_or_default(),
            );
        }

        debug!("Processing response headers: dest={}:{}", dest_ip, dest_port);

        let res_headers = self.get_http_response_headers();
        let mut headers: HashMap<String, String> = HashMap::with_capacity(res_headers.len());
        for res_header in res_headers {
            // Don't include Envoy's pseudo headers
            // https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers#id13
            if !res_header.0.starts_with("x-envoy") {
                headers.insert(res_header.0, res_header.1);
            }
        }

        if let Some(tls_version) = self.get_property(vec!["upstream", "tls_version"]) {
            headers.insert(
                HEADER_X_SENTRYFLOW_TLS_VERSION.to_string(),
                String::from_utf8(tls_version).unwrap_or_default(),
            );
        }

        self.api_event.response.headers = headers;
        self.api_event.destination.ip = dest_ip;
        self.api_event.destination.port = dest_port;
        Action::Continue
    }

    fn on_http_response_body(&mut self, _body_size: usize, _end_of_stream: bool) -> Action {
        // Same accumulation pattern as request body
        // For SSE / LLM streaming, this fires once per DATA frame.
        // The terminal frame often has _body_size=0 and _end_of_stream=true,
        // so we finalize from our buffer, not from _body_size.
        if let Some(chunk) = self.get_http_response_body(0, _body_size) {
            if self.response_body_buffer.len() + chunk.len() <= MAX_BODY_SIZE {
                self.response_body_buffer.extend_from_slice(&chunk);
            } else {
                info!(
                    "Context {}: Response body exceeded MAX_BODY_SIZE ({} bytes), capping capture",
                    self._context_id,
                    self.response_body_buffer.len() + chunk.len()
                );
            }
        }

        if _end_of_stream {
            if !self.response_body_buffer.is_empty() {
                self.api_event.response.body =
                    String::from_utf8_lossy(&self.response_body_buffer).into_owned();
                debug!(
                    "Context {}: Response body fully captured ({} bytes)",
                    self._context_id,
                    self.api_event.response.body.len()
                );
            }

            // Read latency at end_of_stream only
            // backend_latency is only populated by Envoy after the full upstream
            // response is received — reading it mid-stream returns stale/zero data.
            if let Some(value) = self.get_property(vec!["response", "backend_latency"]) {
                if value.len() >= 8 {
                    self.api_event.response.backend_latency_in_nanos =
                        u64::from_ne_bytes(value[..8].try_into().unwrap_or_default());
                    debug!(
                        "Context {}: Backend latency: {} ns",
                        self._context_id, self.api_event.response.backend_latency_in_nanos
                    );
                }
            }
        }

        Action::Continue
    }
}

fn dispatch_http_call_to_upstream(obj: &mut Plugin) {
    update_metadata(obj);
    let telemetry_json = match serde_json::to_string(&obj.api_event) {
        Ok(json) => json,
        Err(e) => {
            error!("Failed to serialize telemetry: {:?}", e);
            return;
        }
    };

    info!(
        "Dispatching telemetry to upstream '{}' at '{}{}' ({} bytes)",
        &obj.config.upstream_name,
        &obj.config.authority,
        &obj.config.api_path,
        telemetry_json.len()
    );

    let headers = vec![
        (":method", "POST"),
        (":authority", &obj.config.authority),
        (":path", &obj.config.api_path),
        ("accept", "*/*"),
        ("Content-Type", "application/json"),
    ];

    // Raise timeout from 1s -> 10s for large LLM payloads
    let http_call_res = obj.dispatch_http_call(
        &obj.config.upstream_name,
        headers,
        Some(telemetry_json.as_bytes()),
        vec![],
        Duration::from_secs(10),
    );

    if http_call_res.is_err() {
        error!(
            "Failed to dispatch HTTP call, to '{}' status: {http_call_res:#?}",
            &obj.config.upstream_name,
        );
    } else {
        debug!("HTTP call dispatched successfully");
    }
}

fn update_metadata(obj: &mut Plugin) {
    obj.api_event.metadata.node_name = String::from_utf8(
        obj.get_property(vec!["node", "metadata", "NODE_NAME"]).unwrap_or_default(),
    )
    .unwrap_or_default();
    obj.api_event.metadata.mesh_id = String::from_utf8(
        obj.get_property(vec!["node", "metadata", "MESH_ID"]).unwrap_or_default(),
    )
    .unwrap_or_default();

    let istio_version: String = String::from_utf8(
        obj.get_property(vec!["node", "metadata", "ISTIO_VERSION"]).unwrap_or_default(),
    )
    .unwrap_or_default();

    #[cfg(feature = "gateway")]
    let proxy_type = "Gateway";

    #[cfg(feature = "sidecar")]
    let proxy_type = "Sidecar";

    obj.api_event.metadata.receiver_name = format!("Istio-{}", proxy_type);
    obj.api_event.metadata.receiver_version = istio_version;

    info!(
        "Metadata - type: {}, receiver: {}",
        proxy_type, obj.api_event.metadata.receiver_name,
    );
}

// Use rfind(':') so IPv6 addresses like [fd00::1]:8080 parse correctly
fn get_url_and_port(address: String) -> (String, u16) {
    if address.is_empty() {
        return (String::new(), 0);
    }

    if let Some(colon_pos) = address.rfind(':') {
        let host = address[..colon_pos]
            .trim_matches(|c| c == '[' || c == ']')
            .to_string();
        let port = address[colon_pos + 1..].parse::<u16>().unwrap_or(0);
        (host, port)
    } else {
        error!("Invalid address format: '{}'", address);
        (String::new(), 0)
    }
}
