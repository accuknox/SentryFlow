package f5bigip

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"

	pb "github.com/accuknox/SentryFlow/protobuf/golang"
	"github.com/accuknox/SentryFlow/sentryflow/pkg/util"
	"go.uber.org/zap"
)

const (
	DELIM     = "__"
	HSL_START = "__HSL_START__ __HSL_START__ __HSL_START__ __HSL_START__ __HSL_START__ __HSL_START__"
	HSL_END   = "__HSL_END__"
	REQHS     = "__REQHS__"
	REQHE     = "__REQHE__"
	HEAN      = "__HEAN__"
	HEAV      = "__HEAV__"
	RESPHS    = "__RESPHS__"
	RESPHE    = "__RESPHE__"
	REQPS     = "__REQPS__"
	REQPE     = "__REQPE__"
)

var logger *zap.SugaredLogger

func Start(ctx context.Context, port uint16, apiEventsChan chan *pb.APIEvent) {
	logger := util.LoggerFromCtx(ctx)
	listener, err := net.Listen("TCP", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Errorf("error starting TCP server: %v", err)
		return
	}
	defer func() {
		listener.Close()
		logger.Info("stopping f5-big-ip receiver")
	}()
	logger.Info("f5-big-ip receiver listening on :5000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Errorf("Connection error:", err)
			continue
		}
		go handleConn(conn, apiEventsChan)
	}
}

func handleConn(conn net.Conn, apiEventsChan chan *pb.APIEvent) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Text()
		event := parseF5LogLine(line)
		if event != nil {
			apiEventsChan <- event
		}

	}
}

func parseF5LogLine(line string) *pb.APIEvent {

	// 1) Extract the part between HSL_START and HSL_END
	start := strings.Index(line, HSL_START)
	end := strings.Index(line, HSL_END)
	if start < 0 || end < 0 {
		logger.Error("missing HSL_START or HSL_END")
		return nil
	}

	payload := strings.TrimSpace(line[start+len(HSL_START) : end])
	parts := strings.Split(payload, " ")
	if len(parts) < 10 {
		logger.Errorf("too few fields: %v", parts)
		return nil
	}
	// Extract the fixed fields
	scheme := parts[0]
	path := parts[1]
	method := parts[2]
	query := parts[3]
	sourceIP := parts[4]
	sourcePortStr := parts[5]
	destIP := parts[6]
	destPortStr := parts[7]
	protocol := parts[8]
	responseStatusCode := parts[9]
	reqTimeStr := parts[10]
	respTimeStr := parts[11]

	// Convert numeric fields
	sourcePort, _ := strconv.Atoi(sourcePortStr)
	destPort, _ := strconv.Atoi(destPortStr)
	reqTime, _ := strconv.ParseInt(reqTimeStr, 10, 64)
	respTime, _ := strconv.ParseInt(respTimeStr, 10, 64)

	// Extract headers + bodies section
	rest := strings.Join(parts[12:], " ")

	// REQUEST HEADERS
	reqHeaders, rest, _ := extractHeaders(rest, REQHS, REQHE)
	reqHeaders[":scheme"] = scheme
	reqHeaders[":path"] = path
	reqHeaders[":method"] = method
	reqHeaders[":query"] = query

	// RESPONSE HEADERS
	respHeaders, rest, _ := extractHeaders(rest, RESPHS, RESPHE)
	respHeaders[":status"] = responseStatusCode

	// REQUEST PAYLOAD
	var reqBody string
	if idx := strings.Index(rest, REQPS); idx >= 0 {
		tmp := rest[idx+len(REQPS):]
		if i2 := strings.Index(tmp, REQPE); i2 >= 0 {
			b64 := strings.TrimSpace(tmp[:i2])
			raw, _ := base64.StdEncoding.DecodeString(b64)
			reqBody = string(raw)
		}
	}

	// RESPONSE PAYLOAD
	var respBody string
	if idx := strings.Index(rest, REQPE); idx >= 0 {
		tmp := rest[idx+len(REQPE):]
		// until end
		b64 := strings.TrimSpace(tmp)
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			logger.Errorf("error decoding base64 string %v", err)
		}
		respBody = string(raw)
	}
	// Build final proto
	ev := &pb.APIEvent{
		Metadata: &pb.Metadata{
			ContextId:       0,
			Timestamp:       uint64(reqTime),
			MeshId:          "",
			NodeName:        "",
			ReceiverName:    "f5-big-ip",
			ReceiverVersion: "16.1",
		},
		Source: &pb.Workload{
			Name:      "",
			Namespace: "",
			Ip:        sourceIP,
			Port:      int32(sourcePort),
		},
		Destination: &pb.Workload{
			Name:      "",
			Namespace: "",
			Ip:        destIP,
			Port:      int32(destPort),
		},
		Request: &pb.Request{
			Headers: reqHeaders,
			Body:    reqBody,
		},
		Response: &pb.Response{
			Headers:               respHeaders,
			Body:                  respBody,
			BackendLatencyInNanos: uint64((respTime - reqTime) * 1_000_000),
		},
		Protocol: protocol,
	}

	return ev
}

func extractHeaders(s, startTag, endTag string) (map[string]string, string, error) {
	out := make(map[string]string)

	startIdx := strings.Index(s, startTag)
	if startIdx < 0 {
		return out, s, nil
	}
	tmp := s[startIdx+len(startTag):]

	endIdx := strings.Index(tmp, endTag)
	if endIdx < 0 {
		return out, s, nil
	}
	headerPart := tmp[:endIdx]
	rest := tmp[endIdx+len(endTag):]

	// split by __HEAN__
	sections := strings.Split(headerPart, HEAN)
	for _, sec := range sections {
		if !strings.Contains(sec, HEAV) {
			continue
		}
		kv := strings.SplitN(sec, HEAV, 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		key = strings.Trim(key, "_")
		val = strings.Trim(val, "_")

		out[key] = val
	}

	return out, rest, nil
}
