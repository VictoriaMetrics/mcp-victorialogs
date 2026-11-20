package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware(t *testing.T) {
	const token = "secret-token"
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(token)(next)

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
	}{
		{name: "mcp without header", path: "/mcp", wantStatus: http.StatusUnauthorized},
		{name: "mcp with valid token", path: "/mcp", authHeader: "Bearer " + token, wantStatus: http.StatusOK},
		{name: "mcp with lowercase scheme", path: "/mcp", authHeader: "bearer " + token, wantStatus: http.StatusOK},
		{name: "mcp with invalid token", path: "/mcp", authHeader: "Bearer wrong", wantStatus: http.StatusUnauthorized},
		{name: "mcp with token prefix", path: "/mcp", authHeader: "Bearer " + token[:3], wantStatus: http.StatusUnauthorized},
		{name: "mcp with basic auth", path: "/mcp", authHeader: "Basic " + token, wantStatus: http.StatusUnauthorized},
		{name: "mcp with scheme only", path: "/mcp", authHeader: "Bearer", wantStatus: http.StatusUnauthorized},
		{name: "mcp path with dot", path: "/mcp.json", wantStatus: http.StatusUnauthorized},
		{name: "sse without header", path: "/sse", wantStatus: http.StatusUnauthorized},
		{name: "message without header", path: "/message", wantStatus: http.StatusUnauthorized},
		{name: "sse with valid token", path: "/sse", authHeader: "Bearer " + token, wantStatus: http.StatusOK},
		{name: "root is public", path: "/", wantStatus: http.StatusOK},
		{name: "static asset is public", path: "/assets/index.js", wantStatus: http.StatusOK},
		{name: "liveness is public", path: "/health/liveness", wantStatus: http.StatusOK},
		{name: "readiness is public", path: "/health/readiness", wantStatus: http.StatusOK},
		{name: "metrics is public", path: "/metrics", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
			}
		})
	}
}
