from google.protobuf.internal import containers as _containers
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Iterable as _Iterable, Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ClientInfo(_message.Message):
    __slots__ = ()
    HOSTNAME_FIELD_NUMBER: _ClassVar[int]
    IPADDRESS_FIELD_NUMBER: _ClassVar[int]
    hostName: str
    IPAddress: str
    def __init__(self, hostName: _Optional[str] = ..., IPAddress: _Optional[str] = ...) -> None: ...

class APILog(_message.Message):
    __slots__ = ()
    class SrcLabelEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class DstLabelEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    SRCNAMESPACE_FIELD_NUMBER: _ClassVar[int]
    SRCNAME_FIELD_NUMBER: _ClassVar[int]
    SRCLABEL_FIELD_NUMBER: _ClassVar[int]
    SRCTYPE_FIELD_NUMBER: _ClassVar[int]
    SRCIP_FIELD_NUMBER: _ClassVar[int]
    SRCPORT_FIELD_NUMBER: _ClassVar[int]
    DSTNAMESPACE_FIELD_NUMBER: _ClassVar[int]
    DSTNAME_FIELD_NUMBER: _ClassVar[int]
    DSTLABEL_FIELD_NUMBER: _ClassVar[int]
    DSTTYPE_FIELD_NUMBER: _ClassVar[int]
    DSTIP_FIELD_NUMBER: _ClassVar[int]
    DSTPORT_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    RESPONSECODE_FIELD_NUMBER: _ClassVar[int]
    id: int
    timeStamp: str
    srcNamespace: str
    srcName: str
    srcLabel: _containers.ScalarMap[str, str]
    srcType: str
    srcIP: str
    srcPort: str
    dstNamespace: str
    dstName: str
    dstLabel: _containers.ScalarMap[str, str]
    dstType: str
    dstIP: str
    dstPort: str
    protocol: str
    method: str
    path: str
    responseCode: int
    def __init__(self, id: _Optional[int] = ..., timeStamp: _Optional[str] = ..., srcNamespace: _Optional[str] = ..., srcName: _Optional[str] = ..., srcLabel: _Optional[_Mapping[str, str]] = ..., srcType: _Optional[str] = ..., srcIP: _Optional[str] = ..., srcPort: _Optional[str] = ..., dstNamespace: _Optional[str] = ..., dstName: _Optional[str] = ..., dstLabel: _Optional[_Mapping[str, str]] = ..., dstType: _Optional[str] = ..., dstIP: _Optional[str] = ..., dstPort: _Optional[str] = ..., protocol: _Optional[str] = ..., method: _Optional[str] = ..., path: _Optional[str] = ..., responseCode: _Optional[int] = ...) -> None: ...

class APIEvent(_message.Message):
    __slots__ = ()
    METADATA_FIELD_NUMBER: _ClassVar[int]
    SOURCE_FIELD_NUMBER: _ClassVar[int]
    DESTINATION_FIELD_NUMBER: _ClassVar[int]
    REQUEST_FIELD_NUMBER: _ClassVar[int]
    RESPONSE_FIELD_NUMBER: _ClassVar[int]
    PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    LATENCY_NS_FIELD_NUMBER: _ClassVar[int]
    metadata: Metadata
    source: Workload
    destination: Workload
    request: Request
    response: Response
    protocol: str
    latency_ns: int
    def __init__(self, metadata: _Optional[_Union[Metadata, _Mapping]] = ..., source: _Optional[_Union[Workload, _Mapping]] = ..., destination: _Optional[_Union[Workload, _Mapping]] = ..., request: _Optional[_Union[Request, _Mapping]] = ..., response: _Optional[_Union[Response, _Mapping]] = ..., protocol: _Optional[str] = ..., latency_ns: _Optional[int] = ...) -> None: ...

class Metadata(_message.Message):
    __slots__ = ()
    CONTEXT_ID_FIELD_NUMBER: _ClassVar[int]
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    ISTIO_VERSION_FIELD_NUMBER: _ClassVar[int]
    MESH_ID_FIELD_NUMBER: _ClassVar[int]
    NODE_NAME_FIELD_NUMBER: _ClassVar[int]
    RECEIVER_NAME_FIELD_NUMBER: _ClassVar[int]
    RECEIVER_VERSION_FIELD_NUMBER: _ClassVar[int]
    context_id: int
    timestamp: int
    istio_version: str
    mesh_id: str
    node_name: str
    receiver_name: str
    receiver_version: str
    def __init__(self, context_id: _Optional[int] = ..., timestamp: _Optional[int] = ..., istio_version: _Optional[str] = ..., mesh_id: _Optional[str] = ..., node_name: _Optional[str] = ..., receiver_name: _Optional[str] = ..., receiver_version: _Optional[str] = ...) -> None: ...

class Workload(_message.Message):
    __slots__ = ()
    class LabelsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    NAME_FIELD_NUMBER: _ClassVar[int]
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    IP_FIELD_NUMBER: _ClassVar[int]
    PORT_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    KIND_FIELD_NUMBER: _ClassVar[int]
    name: str
    namespace: str
    ip: str
    port: int
    labels: _containers.ScalarMap[str, str]
    kind: str
    def __init__(self, name: _Optional[str] = ..., namespace: _Optional[str] = ..., ip: _Optional[str] = ..., port: _Optional[int] = ..., labels: _Optional[_Mapping[str, str]] = ..., kind: _Optional[str] = ...) -> None: ...

class Request(_message.Message):
    __slots__ = ()
    class HeadersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    METHOD_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    headers: _containers.ScalarMap[str, str]
    body: str
    method: str
    path: str
    def __init__(self, headers: _Optional[_Mapping[str, str]] = ..., body: _Optional[str] = ..., method: _Optional[str] = ..., path: _Optional[str] = ...) -> None: ...

class Response(_message.Message):
    __slots__ = ()
    class HeadersEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    HEADERS_FIELD_NUMBER: _ClassVar[int]
    BODY_FIELD_NUMBER: _ClassVar[int]
    BACKEND_LATENCY_IN_NANOS_FIELD_NUMBER: _ClassVar[int]
    STATUS_CODE_FIELD_NUMBER: _ClassVar[int]
    headers: _containers.ScalarMap[str, str]
    body: str
    backend_latency_in_nanos: int
    status_code: int
    def __init__(self, headers: _Optional[_Mapping[str, str]] = ..., body: _Optional[str] = ..., backend_latency_in_nanos: _Optional[int] = ..., status_code: _Optional[int] = ...) -> None: ...

class APIMetrics(_message.Message):
    __slots__ = ()
    class PerAPICountsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    PERAPICOUNTS_FIELD_NUMBER: _ClassVar[int]
    perAPICounts: _containers.ScalarMap[str, int]
    def __init__(self, perAPICounts: _Optional[_Mapping[str, int]] = ...) -> None: ...

class MetricValue(_message.Message):
    __slots__ = ()
    class ValueEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    VALUE_FIELD_NUMBER: _ClassVar[int]
    value: _containers.ScalarMap[str, str]
    def __init__(self, value: _Optional[_Mapping[str, str]] = ...) -> None: ...

class EnvoyMetrics(_message.Message):
    __slots__ = ()
    class LabelsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: str
        def __init__(self, key: _Optional[str] = ..., value: _Optional[str] = ...) -> None: ...
    class MetricsEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: MetricValue
        def __init__(self, key: _Optional[str] = ..., value: _Optional[_Union[MetricValue, _Mapping]] = ...) -> None: ...
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    NAME_FIELD_NUMBER: _ClassVar[int]
    IPADDRESS_FIELD_NUMBER: _ClassVar[int]
    LABELS_FIELD_NUMBER: _ClassVar[int]
    METRICS_FIELD_NUMBER: _ClassVar[int]
    timeStamp: str
    namespace: str
    name: str
    IPAddress: str
    labels: _containers.ScalarMap[str, str]
    metrics: _containers.MessageMap[str, MetricValue]
    def __init__(self, timeStamp: _Optional[str] = ..., namespace: _Optional[str] = ..., name: _Optional[str] = ..., IPAddress: _Optional[str] = ..., labels: _Optional[_Mapping[str, str]] = ..., metrics: _Optional[_Mapping[str, MetricValue]] = ...) -> None: ...

class APIEventFilter(_message.Message):
    __slots__ = ()
    NAMESPACE_FIELD_NUMBER: _ClassVar[int]
    POD_NAME_FIELD_NUMBER: _ClassVar[int]
    PROTOCOLS_FIELD_NUMBER: _ClassVar[int]
    METHODS_FIELD_NUMBER: _ClassVar[int]
    STATUS_PATTERNS_FIELD_NUMBER: _ClassVar[int]
    MIN_DURATION_MS_FIELD_NUMBER: _ClassVar[int]
    namespace: str
    pod_name: str
    protocols: _containers.RepeatedScalarFieldContainer[str]
    methods: _containers.RepeatedScalarFieldContainer[str]
    status_patterns: _containers.RepeatedScalarFieldContainer[str]
    min_duration_ms: int
    def __init__(self, namespace: _Optional[str] = ..., pod_name: _Optional[str] = ..., protocols: _Optional[_Iterable[str]] = ..., methods: _Optional[_Iterable[str]] = ..., status_patterns: _Optional[_Iterable[str]] = ..., min_duration_ms: _Optional[int] = ...) -> None: ...

class MetricsRequest(_message.Message):
    __slots__ = ()
    TIME_RANGE_SECONDS_FIELD_NUMBER: _ClassVar[int]
    time_range_seconds: int
    def __init__(self, time_range_seconds: _Optional[int] = ...) -> None: ...

class EndpointMetric(_message.Message):
    __slots__ = ()
    METHOD_FIELD_NUMBER: _ClassVar[int]
    PATH_FIELD_NUMBER: _ClassVar[int]
    REQUEST_COUNT_FIELD_NUMBER: _ClassVar[int]
    AVG_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    P95_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    P99_LATENCY_MS_FIELD_NUMBER: _ClassVar[int]
    method: str
    path: str
    request_count: int
    avg_latency_ms: float
    p95_latency_ms: float
    p99_latency_ms: float
    def __init__(self, method: _Optional[str] = ..., path: _Optional[str] = ..., request_count: _Optional[int] = ..., avg_latency_ms: _Optional[float] = ..., p95_latency_ms: _Optional[float] = ..., p99_latency_ms: _Optional[float] = ...) -> None: ...

class APIObserverMetrics(_message.Message):
    __slots__ = ()
    class EventsByProtocolEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class EventsByStatusEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: int
        def __init__(self, key: _Optional[str] = ..., value: _Optional[int] = ...) -> None: ...
    class AvgLatencyByEndpointEntry(_message.Message):
        __slots__ = ()
        KEY_FIELD_NUMBER: _ClassVar[int]
        VALUE_FIELD_NUMBER: _ClassVar[int]
        key: str
        value: float
        def __init__(self, key: _Optional[str] = ..., value: _Optional[float] = ...) -> None: ...
    TIMESTAMP_FIELD_NUMBER: _ClassVar[int]
    TOTAL_EVENTS_FIELD_NUMBER: _ClassVar[int]
    EVENTS_BY_PROTOCOL_FIELD_NUMBER: _ClassVar[int]
    EVENTS_BY_STATUS_FIELD_NUMBER: _ClassVar[int]
    TOP_ENDPOINTS_FIELD_NUMBER: _ClassVar[int]
    AVG_LATENCY_BY_ENDPOINT_FIELD_NUMBER: _ClassVar[int]
    ERROR_RATE_FIELD_NUMBER: _ClassVar[int]
    timestamp: int
    total_events: int
    events_by_protocol: _containers.ScalarMap[str, int]
    events_by_status: _containers.ScalarMap[str, int]
    top_endpoints: _containers.RepeatedCompositeFieldContainer[EndpointMetric]
    avg_latency_by_endpoint: _containers.ScalarMap[str, float]
    error_rate: float
    def __init__(self, timestamp: _Optional[int] = ..., total_events: _Optional[int] = ..., events_by_protocol: _Optional[_Mapping[str, int]] = ..., events_by_status: _Optional[_Mapping[str, int]] = ..., top_endpoints: _Optional[_Iterable[_Union[EndpointMetric, _Mapping]]] = ..., avg_latency_by_endpoint: _Optional[_Mapping[str, float]] = ..., error_rate: _Optional[float] = ...) -> None: ...
