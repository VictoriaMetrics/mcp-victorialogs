package tools

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

// receivedRequest is what the fake VictoriaLogs server got from a tool.
type receivedRequest struct {
	method     string
	requestURI string
	header     http.Header
	body       string
}

// TestSelectToolsSendArgsInBody checks that every tool calling the /select/logsql/* API sends its
// arguments in a form-encoded POST body, leaving them out of the URL, and keeps the request headers.
func TestSelectToolsSendArgsInBody(t *testing.T) {
	received := make(chan receivedRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
		}
		received <- receivedRequest{method: r.Method, requestURI: r.RequestURI, header: r.Header.Clone(), body: string(body)}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("VL_INSTANCE_ENTRYPOINT", srv.URL)
	t.Setenv("VL_INSTANCE_CA_CERT", "")
	t.Setenv("VL_INSTANCE_CA_CERT_FILE", "")
	t.Setenv("VMC_API_KEY", "")
	t.Setenv("VL_INSTANCE_BEARER_TOKEN", "test-token")
	t.Setenv("VL_INSTANCE_HEADERS", "X-Custom-Header=custom-value")
	t.Setenv("MCP_PASSTHROUGH_HEADERS", "X-Passthrough-Header")
	t.Setenv("VL_DEFAULT_TENANT_ID", "")
	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	testCases := []struct {
		name    string
		handler func(context.Context, *config.Config, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		path    string
		args    map[string]any
		want    url.Values
	}{
		{
			name:    "query",
			handler: toolQueryHandler,
			path:    "query",
			args:    map[string]any{"query": "error | limit 5", "start": "2026-01-01T00:00:00Z", "end": "now", "limit": float64(10), "timeout": "5s"},
			want:    url.Values{"query": {"error | limit 5"}, "start": {"2026-01-01T00:00:00Z"}, "end": {"now"}, "limit": {"10"}, "timeout": {"5s"}},
		},
		{
			name:    "hits",
			handler: toolHitsHandler,
			path:    "hits",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now", "step": "5m", "field": []any{"level", "host"}, "fields_limit": float64(3)},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}, "step": {"5m"}, "field": {"level", "host"}, "fields_limit": {"3"}},
		},
		{
			name:    "facets",
			handler: toolFacetsHandler,
			path:    "facets",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now", "limit": float64(3), "max_values_per_field": float64(100), "max_value_len": float64(50), "keep_const_fields": true},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}, "limit": {"3"}, "max_values_per_field": {"100"}, "max_value_len": {"50"}, "keep_const_fields": {"1"}},
		},
		{
			name:    "stats_query",
			handler: toolStatsQueryHandler,
			path:    "stats_query",
			args:    map[string]any{"query": "* | stats count() logs", "time": "now"},
			want:    url.Values{"query": {"* | stats count() logs"}, "time": {"now"}},
		},
		{
			name:    "stats_query_range",
			handler: toolStatsQueryRangeHandler,
			path:    "stats_query_range",
			args:    map[string]any{"query": "* | stats count() logs", "start": "1d", "end": "now", "step": "1h"},
			want:    url.Values{"query": {"* | stats count() logs"}, "start": {"1d"}, "end": {"now"}, "step": {"1h"}},
		},
		{
			name:    "streams",
			handler: toolStreamsHandler,
			path:    "streams",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now"},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}},
		},
		{
			name:    "stream_ids",
			handler: toolStreamIDsHandler,
			path:    "stream_ids",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now"},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}},
		},
		{
			name:    "stream_field_names",
			handler: toolStreamFieldNamesHandler,
			path:    "stream_field_names",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now"},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}},
		},
		{
			name:    "stream_field_values",
			handler: toolStreamFieldValuesHandler,
			path:    "stream_field_values",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now", "field": "host"},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}, "field": {"host"}},
		},
		{
			name:    "field_names",
			handler: toolFieldNamesHandler,
			path:    "field_names",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now"},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}},
		},
		{
			name:    "field_values",
			handler: toolFieldValuesHandler,
			path:    "field_values",
			args:    map[string]any{"query": "error", "start": "1h", "end": "now", "field": "level", "limit": float64(5)},
			want:    url.Values{"query": {"error"}, "start": {"1h"}, "end": {"now"}, "field": {"level"}, "limit": {"5"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := map[string]any{"tenant": "12:34"}
			for k, v := range tc.args {
				args[k] = v
			}
			tcr := newToolRequest(args)
			tcr.Header.Set("X-Passthrough-Header", "passthrough-value")

			result, err := tc.handler(context.Background(), cfg, tcr)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result.IsError {
				t.Fatalf("Expected no error result, got: %v", result.Content)
			}

			var got receivedRequest
			select {
			case got = <-received:
			case <-time.After(5 * time.Second):
				t.Fatal("Expected a request to the VictoriaLogs server, got none")
			}

			wantURI := "/select/logsql/" + tc.path
			if got.method != http.MethodPost || got.requestURI != wantURI {
				t.Errorf("Expected POST %s, got %s %s", wantURI, got.method, got.requestURI)
			}
			if want := "application/x-www-form-urlencoded"; got.header.Get("Content-Type") != want {
				t.Errorf("Expected Content-Type %q, got %q", want, got.header.Get("Content-Type"))
			}
			if want := tc.want.Encode(); got.body != want {
				t.Errorf("Expected body %q, got %q", want, got.body)
			}
			for name, want := range map[string]string{
				"AccountID":            "12",
				"ProjectID":            "34",
				"Authorization":        "Bearer test-token",
				"X-Custom-Header":      "custom-value",
				"X-Passthrough-Header": "passthrough-value",
			} {
				if got.header.Get(name) != want {
					t.Errorf("Expected header %s %q, got %q", name, want, got.header.Get(name))
				}
			}
			if values, ok := got.header["Idempotency-Key"]; ok {
				t.Errorf("Expected no Idempotency-Key header on the wire, got %q", values)
			}
		})
	}
}

// TestCreateSelectRequestRetriesClosedConnection checks that a select request is retried on a new
// connection when the server closes a reused keep-alive connection after reading the request,
// the way net/http retries GET requests.
func TestCreateSelectRequestRetriesClosedConnection(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				r := bufio.NewReader(c)
				for i := 0; ; i++ {
					req, err := http.ReadRequest(r)
					if err != nil {
						return
					}
					_, _ = io.Copy(io.Discard, req.Body)
					// Answer the first request on the connection and close it on the second one.
					if i > 0 {
						return
					}
					_, _ = io.WriteString(c, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\n{}")
				}
			}(conn)
		}
	}()

	t.Setenv("VL_INSTANCE_ENTRYPOINT", "http://"+ln.Addr().String())
	t.Setenv("VL_INSTANCE_CA_CERT", "")
	t.Setenv("VL_INSTANCE_CA_CERT_FILE", "")
	t.Setenv("VMC_API_KEY", "")
	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	for i := 1; i <= 2; i++ {
		req, err := CreateSelectRequest(context.Background(), cfg, newToolRequest(map[string]any{}), url.Values{"query": {"error"}}, "query")
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		if result := GetTextBodyForRequest(req, cfg); result.IsError {
			t.Fatalf("Expected call %d to succeed, got: %v", i, result.Content)
		}
	}
}
