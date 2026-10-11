package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitConfig(t *testing.T) {
	// Save original environment variables
	originalEntrypoint := os.Getenv("VL_INSTANCE_ENTRYPOINT")
	originalServerMode := os.Getenv("MCP_SERVER_MODE")
	originalSSEAddr := os.Getenv("MCP_SSE_ADDR")
	originalBearerToken := os.Getenv("VL_INSTANCE_BEARER_TOKEN")
	originalHeartbeatInterval := os.Getenv("MCP_HEARTBEAT_INTERVAL")
	originalDefaultTenantID := os.Getenv("VL_DEFAULT_TENANT_ID")
	originalPassthroughHeaders := os.Getenv("MCP_PASSTHROUGH_HEADERS")
	originalTLSInsecureSkipVerify := os.Getenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY")
	originalTLSCAFile := os.Getenv("VL_INSTANCE_TLS_CA_FILE")

	// Restore environment variables after test
	defer func() {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", originalEntrypoint)
		os.Setenv("MCP_SERVER_MODE", originalServerMode)
		os.Setenv("MCP_SSE_ADDR", originalSSEAddr)
		os.Setenv("VL_INSTANCE_BEARER_TOKEN", originalBearerToken)
		os.Setenv("MCP_HEARTBEAT_INTERVAL", originalHeartbeatInterval)
		os.Setenv("VL_DEFAULT_TENANT_ID", originalDefaultTenantID)
		os.Setenv("MCP_PASSTHROUGH_HEADERS", originalPassthroughHeaders)
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", originalTLSInsecureSkipVerify)
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", originalTLSCAFile)
	}()

	// Test case 1: Valid configuration
	t.Run("Valid configuration", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_SERVER_MODE", "stdio")
		os.Setenv("MCP_SSE_ADDR", "localhost:8080")
		os.Setenv("VL_INSTANCE_BEARER_TOKEN", "test-token")

		// Initialize config
		cfg, err := InitConfig()

		// Check for errors
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check config values
		if cfg.BearerToken() != "test-token" {
			t.Errorf("Expected bearer token 'test-token', got: %s", cfg.BearerToken())
		}
		if !cfg.IsStdio() {
			t.Error("Expected IsStdio() to be true")
		}
		if cfg.IsSSE() {
			t.Error("Expected IsSSE() to be false")
		}
		if cfg.ListenAddr() != "localhost:8080" {
			t.Errorf("Expected SSE address 'localhost:8080', got: %s", cfg.ListenAddr())
		}
		expectedURL, _ := url.Parse("http://example.com")
		if cfg.EntryPointURL().String() != expectedURL.String() {
			t.Errorf("Expected entrypoint URL 'http://example.com', got: %s", cfg.EntryPointURL().String())
		}
	})

	// Test case 2: Custom headers parsing
	t.Run("Custom headers parsing", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_INSTANCE_HEADERS", "CF-Access-Client-Id=test-client-id,CF-Access-Client-Secret=test-client-secret,Custom-Header=test-value")

		// Initialize config
		cfg, err := InitConfig()

		// Check for errors
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check custom headers
		headers := cfg.CustomHeaders()
		expectedHeaders := map[string]string{
			"CF-Access-Client-Id":     "test-client-id",
			"CF-Access-Client-Secret": "test-client-secret",
			"Custom-Header":           "test-value",
		}

		if len(headers) != len(expectedHeaders) {
			t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(headers))
		}

		for key, expectedValue := range expectedHeaders {
			if actualValue, exists := headers[key]; !exists {
				t.Errorf("Expected header %s to exist", key)
			} else if actualValue != expectedValue {
				t.Errorf("Expected header %s to have value %s, got %s", key, expectedValue, actualValue)
			}
		}
	})

	// Test case 3: Empty custom headers
	t.Run("Empty custom headers", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_INSTANCE_HEADERS", "")

		// Initialize config
		cfg, err := InitConfig()

		// Check for errors
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check custom headers
		headers := cfg.CustomHeaders()
		if len(headers) != 0 {
			t.Errorf("Expected 0 headers, got %d", len(headers))
		}
	})

	// Test case 4: Invalid header format (should be ignored)
	t.Run("Invalid header format", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_INSTANCE_HEADERS", "invalid-header,valid-header=value,another-invalid")

		// Initialize config
		cfg, err := InitConfig()

		// Check for errors
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check custom headers (only valid ones should be parsed)
		headers := cfg.CustomHeaders()
		expectedHeaders := map[string]string{
			"valid-header": "value",
		}

		if len(headers) != len(expectedHeaders) {
			t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(headers))
		}

		for key, expectedValue := range expectedHeaders {
			if actualValue, exists := headers[key]; !exists {
				t.Errorf("Expected header %s to exist", key)
			} else if actualValue != expectedValue {
				t.Errorf("Expected header %s to have value %s, got %s", key, expectedValue, actualValue)
			}
		}
	})

	// Test case 5: Missing entrypoint
	t.Run("Missing entrypoint", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "")

		// Initialize config
		_, err := InitConfig()

		// Check for errors
		if err == nil {
			t.Fatal("Expected error for missing entrypoint, got nil")
		}
	})

	// Test case 3: Invalid server mode
	t.Run("Invalid server mode", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_SERVER_MODE", "invalid")

		// Initialize config
		_, err := InitConfig()

		// Check for errors
		if err == nil {
			t.Fatal("Expected error for invalid server mode, got nil")
		}
	})

	// Test case 4: Default values
	t.Run("Default values", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_SERVER_MODE", "")
		os.Setenv("MCP_SSE_ADDR", "")

		// Initialize config
		cfg, err := InitConfig()

		// Check for errors
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check default values
		if !cfg.IsStdio() {
			t.Error("Expected default server mode to be stdio")
		}
		if cfg.ListenAddr() != "localhost:8081" {
			t.Errorf("Expected default SSE address 'localhost:8081', got: %s", cfg.ListenAddr())
		}
	})

	// Test case 5: Correct heartbeat interval
	t.Run("Correct heartbeat interval", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_SERVER_MODE", "stdio")
		os.Setenv("MCP_HEARTBEAT_INTERVAL", "30s")
		// Initialize config
		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		// Check values
		if cfg.HeartbeatInterval() != 30*time.Second {
			t.Errorf("Expected heartbeat interval to be 30 seconds, got: %d", cfg.HeartbeatInterval())
		}
	})

	// Test case 6: Incorrect heartbeat interval
	t.Run("Incorrect heartbeat interval", func(t *testing.T) {
		// Set environment variables
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_SERVER_MODE", "stdio")
		os.Setenv("MCP_HEARTBEAT_INTERVAL", "123")
		// Initialize config
		_, err := InitConfig()
		if err == nil || err.Error() != "failed to parse MCP_HEARTBEAT_INTERVAL: time: missing unit in duration \"123\"" {
			t.Errorf("Expected error 'failed to parse MCP_HEARTBEAT_INTERVAL: time: missing unit in duration \"123\"', got: %v", err)
		}
	})

	// Test case 7: Default tenant ID - valid format
	t.Run("Valid default tenant ID", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_SERVER_MODE", "stdio")
		os.Setenv("MCP_HEARTBEAT_INTERVAL", "")
		os.Setenv("VL_DEFAULT_TENANT_ID", "123:456")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		tenantID := cfg.DefaultTenantID()
		if tenantID.AccountID != 123 {
			t.Errorf("Expected AccountID 123, got: %d", tenantID.AccountID)
		}
		if tenantID.ProjectID != 456 {
			t.Errorf("Expected ProjectID 456, got: %d", tenantID.ProjectID)
		}
	})

	// Test case 8: Default tenant ID - account only
	t.Run("Default tenant ID - account only", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "789")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		tenantID := cfg.DefaultTenantID()
		if tenantID.AccountID != 789 {
			t.Errorf("Expected AccountID 789, got: %d", tenantID.AccountID)
		}
		if tenantID.ProjectID != 0 {
			t.Errorf("Expected ProjectID 0, got: %d", tenantID.ProjectID)
		}
	})

	// Test case 9: Default tenant ID - empty (should use 0:0)
	t.Run("Default tenant ID - empty", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		tenantID := cfg.DefaultTenantID()
		if tenantID.AccountID != 0 {
			t.Errorf("Expected AccountID 0, got: %d", tenantID.AccountID)
		}
		if tenantID.ProjectID != 0 {
			t.Errorf("Expected ProjectID 0, got: %d", tenantID.ProjectID)
		}
	})

	// Test case 10: Default tenant ID - invalid format
	t.Run("Default tenant ID - invalid format", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "invalid")

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for invalid tenant ID, got nil")
		}
	})

	// Test case 11: Default tenant ID - too many colons
	t.Run("Default tenant ID - too many colons", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "1:2:3")

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for invalid tenant ID format, got nil")
		}
	})

	// Test case: Passthrough headers parsing
	t.Run("Passthrough headers parsing", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "")
		os.Setenv("MCP_PASSTHROUGH_HEADERS", "Authorization,X-Custom-Token,X-Request-ID")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		headers := cfg.PassthroughHeaders()
		expected := []string{"Authorization", "X-Custom-Token", "X-Request-ID"}

		if len(headers) != len(expected) {
			t.Fatalf("Expected %d passthrough headers, got %d", len(expected), len(headers))
		}
		for i, h := range headers {
			if h != expected[i] {
				t.Errorf("Expected header %q at index %d, got %q", expected[i], i, h)
			}
		}
	})

	// Test case: Empty passthrough headers
	t.Run("Empty passthrough headers", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "")
		os.Setenv("MCP_PASSTHROUGH_HEADERS", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(cfg.PassthroughHeaders()) != 0 {
			t.Errorf("Expected 0 passthrough headers, got %d", len(cfg.PassthroughHeaders()))
		}
	})

	// Test case: Passthrough headers with whitespace and empty entries
	t.Run("Passthrough headers whitespace trimming", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("VL_DEFAULT_TENANT_ID", "")
		os.Setenv("MCP_PASSTHROUGH_HEADERS", " Authorization , , X-Custom-Token , ")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		headers := cfg.PassthroughHeaders()
		expected := []string{"Authorization", "X-Custom-Token"}

		if len(headers) != len(expected) {
			t.Fatalf("Expected %d passthrough headers, got %d", len(expected), len(headers))
		}
		for i, h := range headers {
			if h != expected[i] {
				t.Errorf("Expected header %q at index %d, got %q", expected[i], i, h)
			}
		}
	})

	// Test case: TLS settings are not configured
	t.Run("TLS default settings", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if cfg.TLSInsecureSkipVerify() {
			t.Error("Expected TLSInsecureSkipVerify() to be false by default")
		}
		if cfg.TLSCAFile() != "" {
			t.Errorf("Expected empty TLSCAFile(), got: %s", cfg.TLSCAFile())
		}
		if cfg.HTTPClient() != http.DefaultClient {
			t.Error("Expected HTTPClient() to fall back to http.DefaultClient")
		}
	})

	// Test case: TLS certificate verification disabled
	t.Run("TLS insecure skip verify enabled", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "true")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !cfg.TLSInsecureSkipVerify() {
			t.Error("Expected TLSInsecureSkipVerify() to be true")
		}
		if cfg.HTTPClient() == http.DefaultClient {
			t.Fatal("Expected a dedicated HTTP client, got http.DefaultClient")
		}

		transport, ok := cfg.HTTPClient().Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Expected *http.Transport, got: %T", cfg.HTTPClient().Transport)
		}
		if !transport.TLSClientConfig.InsecureSkipVerify {
			t.Error("Expected InsecureSkipVerify to be enabled on the transport")
		}
	})

	// Test case: boolean values other than 'true'/'false' are accepted
	t.Run("TLS insecure skip verify numeric value", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "1")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !cfg.TLSInsecureSkipVerify() {
			t.Error("Expected TLSInsecureSkipVerify() to be true for value '1'")
		}
	})

	// Test case: invalid boolean value
	t.Run("Invalid TLS insecure skip verify", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "yes")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", "")

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for invalid VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY, got nil")
		}
	})

	// Test case: custom CA bundle
	t.Run("TLS CA file", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", writeTestCAFile(t))

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cfg.HTTPClient() == http.DefaultClient {
			t.Fatal("Expected a dedicated HTTP client, got http.DefaultClient")
		}

		transport, ok := cfg.HTTPClient().Transport.(*http.Transport)
		if !ok {
			t.Fatalf("Expected *http.Transport, got: %T", cfg.HTTPClient().Transport)
		}
		if transport.TLSClientConfig.RootCAs == nil {
			t.Error("Expected RootCAs to be set on the transport")
		}
		if transport.TLSClientConfig.InsecureSkipVerify {
			t.Error("Expected InsecureSkipVerify to stay disabled")
		}
	})

	// Test case: CA bundle does not exist
	t.Run("Missing TLS CA file", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", filepath.Join(t.TempDir(), "missing-ca.pem"))

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for missing VL_INSTANCE_TLS_CA_FILE, got nil")
		}
	})

	// Test case: CA bundle without any certificate
	t.Run("TLS CA file without certificates", func(t *testing.T) {
		caFile := filepath.Join(t.TempDir(), "empty-ca.pem")
		if err := os.WriteFile(caFile, []byte("not a certificate"), 0o600); err != nil {
			t.Fatalf("Failed to write CA file: %v", err)
		}

		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", caFile)

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for CA file without certificates, got nil")
		}
	})

	// Test case: CA bundle and disabled verification are mutually exclusive
	t.Run("TLS CA file with insecure skip verify", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")
		os.Setenv("VL_INSTANCE_TLS_INSECURE_SKIP_VERIFY", "true")
		os.Setenv("VL_INSTANCE_TLS_CA_FILE", writeTestCAFile(t))

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for mutually exclusive TLS options, got nil")
		}
	})
}

// writeTestCAFile generates a self-signed CA certificate and writes it in PEM
// format to a temporary file, returning the path to it.
func writeTestCAFile(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mcp-victorialogs-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	return caFile
}
