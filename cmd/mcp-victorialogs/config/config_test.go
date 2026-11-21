package config

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
	originalStateless := os.Getenv("MCP_STATELESS")
	originalDisableLocalhost := os.Getenv("MCP_DISABLE_LOCALHOST_PROTECTION")
	originalAuthToken := os.Getenv("MCP_AUTH_TOKEN")

	// Restore environment variables after test
	defer func() {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", originalEntrypoint)
		os.Setenv("MCP_SERVER_MODE", originalServerMode)
		os.Setenv("MCP_SSE_ADDR", originalSSEAddr)
		os.Setenv("VL_INSTANCE_BEARER_TOKEN", originalBearerToken)
		os.Setenv("MCP_HEARTBEAT_INTERVAL", originalHeartbeatInterval)
		os.Setenv("VL_DEFAULT_TENANT_ID", originalDefaultTenantID)
		os.Setenv("MCP_PASSTHROUGH_HEADERS", originalPassthroughHeaders)
		os.Setenv("MCP_STATELESS", originalStateless)
		os.Setenv("MCP_DISABLE_LOCALHOST_PROTECTION", originalDisableLocalhost)
		os.Setenv("MCP_AUTH_TOKEN", originalAuthToken)
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

	// Test case: Stateless - default (unset) is false
	t.Run("Stateless default false", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_PASSTHROUGH_HEADERS", "")
		os.Setenv("MCP_STATELESS", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cfg.Stateless() {
			t.Error("Expected Stateless() to default to false")
		}
	})

	// Test case: Stateless - explicitly enabled
	t.Run("Stateless enabled", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_STATELESS", "true")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !cfg.Stateless() {
			t.Error("Expected Stateless() to be true")
		}
	})

	// Test case: Stateless - invalid value errors
	t.Run("Stateless invalid value", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_STATELESS", "notabool")

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for invalid MCP_STATELESS, got nil")
		}
	})

	// Test case: DisableLocalhostProtection - default (unset) is false
	t.Run("DisableLocalhostProtection default false", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_STATELESS", "")
		os.Setenv("MCP_DISABLE_LOCALHOST_PROTECTION", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cfg.DisableLocalhostProtection() {
			t.Error("Expected DisableLocalhostProtection() to default to false")
		}
	})

	// Test case: DisableLocalhostProtection - explicitly enabled
	t.Run("DisableLocalhostProtection enabled", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_DISABLE_LOCALHOST_PROTECTION", "true")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !cfg.DisableLocalhostProtection() {
			t.Error("Expected DisableLocalhostProtection() to be true")
		}
	})

	// Test case: DisableLocalhostProtection - invalid value errors
	t.Run("DisableLocalhostProtection invalid value", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_DISABLE_LOCALHOST_PROTECTION", "notabool")

		_, err := InitConfig()
		if err == nil {
			t.Fatal("Expected error for invalid MCP_DISABLE_LOCALHOST_PROTECTION, got nil")
		}
	})

	// Test case: AuthToken - default (unset) is empty
	t.Run("AuthToken default empty", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_DISABLE_LOCALHOST_PROTECTION", "")
		os.Setenv("MCP_AUTH_TOKEN", "")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cfg.AuthToken() != "" {
			t.Errorf("Expected empty AuthToken(), got: %q", cfg.AuthToken())
		}
	})

	// Test case: AuthToken - explicitly set
	t.Run("AuthToken set", func(t *testing.T) {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
		os.Setenv("MCP_AUTH_TOKEN", "mcp-secret")

		cfg, err := InitConfig()
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if cfg.AuthToken() != "mcp-secret" {
			t.Errorf("Expected AuthToken() 'mcp-secret', got: %q", cfg.AuthToken())
		}
	})
}

func TestInitConfigInstructionsOverride(t *testing.T) {
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")

	t.Setenv("MCP_INSTRUCTIONS_OVERRIDE", "")
	cfg, err := InitConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.InstructionsOverride() != "" {
		t.Errorf("Expected empty instructions override, got: %q", cfg.InstructionsOverride())
	}

	t.Setenv("MCP_INSTRUCTIONS_OVERRIDE", "  This instance stores production logs.\n")
	cfg, err = InitConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if expected := "This instance stores production logs."; cfg.InstructionsOverride() != expected {
		t.Errorf("Expected instructions override %q, got: %q", expected, cfg.InstructionsOverride())
	}
}

func TestInitConfigInsecureSkipVerify(t *testing.T) {
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "https://example.com")

	// Default (unset) keeps verification enabled and uses http.DefaultClient.
	t.Setenv("VL_INSTANCE_INSECURE_SKIP_VERIFY", "")
	t.Setenv("VL_INSTANCE_CA_CERT", "")
	t.Setenv("VL_INSTANCE_CA_CERT_FILE", "")
	cfg, err := InitConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if cfg.InsecureSkipVerify() {
		t.Error("Expected InsecureSkipVerify() to default to false")
	}
	if cfg.HTTPClient() != http.DefaultClient {
		t.Error("Expected HTTPClient() to fall back to http.DefaultClient")
	}

	// Enabled: a dedicated client that accepts a self-signed TLS certificate.
	t.Setenv("VL_INSTANCE_INSECURE_SKIP_VERIFY", "true")
	cfg, err = InitConfig()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !cfg.InsecureSkipVerify() {
		t.Error("Expected InsecureSkipVerify() to be true")
	}
	if cfg.HTTPClient() == http.DefaultClient {
		t.Fatal("Expected a dedicated HTTP client, got http.DefaultClient")
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	resp, err := cfg.HTTPClient().Get(ts.URL)
	if err != nil {
		t.Fatalf("Expected request to a self-signed server to succeed with skip-verify, got: %v", err)
	}
	_ = resp.Body.Close()

	// strconv.ParseBool accepts "1".
	t.Setenv("VL_INSTANCE_INSECURE_SKIP_VERIFY", "1")
	cfg, err = InitConfig()
	if err != nil {
		t.Fatalf("Expected no error for value '1', got: %v", err)
	}
	if !cfg.InsecureSkipVerify() {
		t.Error("Expected InsecureSkipVerify() to be true for value '1'")
	}

	// Invalid boolean value fails at startup.
	t.Setenv("VL_INSTANCE_INSECURE_SKIP_VERIFY", "notabool")
	if _, err = InitConfig(); err == nil {
		t.Fatal("Expected error for invalid VL_INSTANCE_INSECURE_SKIP_VERIFY, got nil")
	}

	// Mutually exclusive with the CA options.
	t.Setenv("VL_INSTANCE_INSECURE_SKIP_VERIFY", "true")
	t.Setenv("VL_INSTANCE_CA_CERT", "some-cert")
	if _, err = InitConfig(); err == nil {
		t.Fatal("Expected error for VL_INSTANCE_INSECURE_SKIP_VERIFY combined with VL_INSTANCE_CA_CERT, got nil")
	}
}
