package tools

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/VictoriaMetrics-Community/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

// TestGetTextBodyForRequest tests the GetTextBodyForRequest function
func TestGetTextBodyForRequest(t *testing.T) {
	// Create a mock config
	cfg := &config.Config{}

	// Save the original HTTP client
	originalClient := http.DefaultClient

	// Create a mock HTTP client
	http.DefaultClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("test response")),
			},
		},
	}
	defer func() { http.DefaultClient = originalClient }()

	// Create a test request
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Call the function
	result := GetTextBodyForRequest(req, cfg)

	// Check the result
	if result.IsError {
		t.Error("Expected no error, got an error result")
	}

	// Extract the text content from the result
	if len(result.Content) == 0 {
		t.Fatal("Expected content in result, got empty content")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent, got different content type")
	}

	if textContent.Text != "test response" {
		t.Errorf("Expected 'test response', got: %s", textContent.Text)
	}
}

// mockTransport is a mock implementation of http.RoundTripper
type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return m.response, m.err
}

// TestGetTextBodyForRequestError tests the error handling in GetTextBodyForRequest
func TestGetTextBodyForRequestError(t *testing.T) {
	// Create a mock config
	cfg := &config.Config{}

	// Save the original HTTP client
	originalClient := http.DefaultClient

	// Create a mock HTTP client that returns an error
	http.DefaultClient = &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString("error message")),
			},
		},
	}
	defer func() { http.DefaultClient = originalClient }()

	// Create a test request
	req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Call the function
	result := GetTextBodyForRequest(req, cfg)

	// Check the result
	if !result.IsError {
		t.Error("Expected an error result, got success")
	}
}

// TestGetToolReqParam tests the GetToolReqParam function
func TestGetToolReqParam(t *testing.T) {
	// Test cases
	testCases := []struct {
		name          string
		args          map[string]any
		param         string
		required      bool
		expectedValue string
		expectError   bool
	}{
		{
			name:          "Valid string parameter",
			args:          map[string]any{"test": "value"},
			param:         "test",
			required:      true,
			expectedValue: "value",
			expectError:   false,
		},
		{
			name:          "Missing required parameter",
			args:          map[string]any{},
			param:         "test",
			required:      true,
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "Missing optional parameter",
			args:          map[string]any{},
			param:         "test",
			required:      false,
			expectedValue: "",
			expectError:   false,
		},
		{
			name:          "Wrong type parameter",
			args:          map[string]any{"test": 123},
			param:         "test",
			required:      true,
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock tool request
			tcr := mcp.CallToolRequest{}
			tcr.Params.Arguments = tc.args

			// Call the function
			value, err := GetToolReqParam[string](tcr, tc.param, tc.required)

			// Check the result
			if tc.expectError && err == nil {
				t.Error("Expected an error, got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
			if value != tc.expectedValue {
				t.Errorf("Expected '%s', got: '%s'", tc.expectedValue, value)
			}
		})
	}
}

// TestGetToolReqParamFloat tests the GetToolReqParam function with float64 type
func TestGetToolReqParamFloat(t *testing.T) {
	// Create a mock tool request
	tcr := mcp.CallToolRequest{}
	tcr.Params.Arguments = map[string]any{
		"float": 123.45,
	}

	// Call the function
	value, err := GetToolReqParam[float64](tcr, "float", true)

	// Check the result
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if value != 123.45 {
		t.Errorf("Expected 123.45, got: %f", value)
	}
}

// TestGetToolReqParamBool tests the GetToolReqParam function with bool type
func TestGetToolReqParamBool(t *testing.T) {
	// Create a mock tool request
	tcr := mcp.CallToolRequest{}
	tcr.Params.Arguments = map[string]any{
		"bool": true,
	}

	// Call the function
	value, err := GetToolReqParam[bool](tcr, "bool", true)

	// Check the result
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !value {
		t.Error("Expected true, got false")
	}
}

// TestGetToolReqParamStringSlice tests the GetToolReqParam function with []string type
func TestGetToolReqParamStringSlice(t *testing.T) {
	// Create a mock tool request
	tcr := mcp.CallToolRequest{}
	tcr.Params.Arguments = map[string]any{
		"slice": []string{"a", "b", "c"},
	}

	// Call the function
	value, err := GetToolReqParam[[]string](tcr, "slice", true)

	// Check the result
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if len(value) != 3 || value[0] != "a" || value[1] != "b" || value[2] != "c" {
		t.Errorf("Expected [a b c], got: %v", value)
	}
}

// TestGetToolReqTenant tests the GetToolReqTenant function
func TestGetToolReqTenant(t *testing.T) {
	testCases := []struct {
		name              string
		tenant            string
		expectedAccountID string
		expectedProjectID string
		expectError       bool
	}{
		{
			name:              "Empty tenant returns default 0:0",
			tenant:            "",
			expectedAccountID: "",
			expectedProjectID: "",
			expectError:       false,
		},
		{
			name:              "Single number tenant",
			tenant:            "123",
			expectedAccountID: "123",
			expectedProjectID: "0",
			expectError:       false,
		},
		{
			name:              "Full tenant format",
			tenant:            "123:456",
			expectedAccountID: "123",
			expectedProjectID: "456",
			expectError:       false,
		},
		{
			name:              "Zero tenant",
			tenant:            "0:0",
			expectedAccountID: "0",
			expectedProjectID: "0",
			expectError:       false,
		},
		{
			name:              "Account only with colon",
			tenant:            "123:",
			expectedAccountID: "123",
			expectedProjectID: "0",
			expectError:       false,
		},
		{
			name:              "Project only with colon",
			tenant:            ":456",
			expectedAccountID: "0",
			expectedProjectID: "456",
			expectError:       false,
		},
		{
			name:        "Invalid tenant format - too many colons",
			tenant:      "1:2:3",
			expectError: true,
		},
		{
			name:        "Invalid tenant format - non-numeric",
			tenant:      "abc:def",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tcr := mcp.CallToolRequest{}
			tcr.Params.Arguments = map[string]any{
				"tenant": tc.tenant,
			}

			accountID, projectID, err := GetToolReqTenant(tcr)

			if tc.expectError {
				if err == nil {
					t.Error("Expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
				return
			}

			if accountID != tc.expectedAccountID {
				t.Errorf("Expected accountID %q, got %q", tc.expectedAccountID, accountID)
			}
			if projectID != tc.expectedProjectID {
				t.Errorf("Expected projectID %q, got %q", tc.expectedProjectID, projectID)
			}
		})
	}
}

// TestCreateSelectRequest_DefaultTenant tests that CreateSelectRequest uses default tenant from config
func TestCreateSelectRequest_DefaultTenant(t *testing.T) {
	// Save original environment variables
	originalEntrypoint := os.Getenv("VL_INSTANCE_ENTRYPOINT")
	originalDefaultTenantID := os.Getenv("VL_DEFAULT_TENANT_ID")

	// Restore environment variables after test
	defer func() {
		os.Setenv("VL_INSTANCE_ENTRYPOINT", originalEntrypoint)
		os.Setenv("VL_DEFAULT_TENANT_ID", originalDefaultTenantID)
	}()

	testCases := []struct {
		name              string
		defaultTenantID   string
		requestTenant     string
		expectedAccountID string
		expectedProjectID string
	}{
		{
			name:              "Empty request tenant uses default from config",
			defaultTenantID:   "100:200",
			requestTenant:     "",
			expectedAccountID: "100",
			expectedProjectID: "200",
		},
		{
			name:              "Request tenant overrides default",
			defaultTenantID:   "100:200",
			requestTenant:     "300:400",
			expectedAccountID: "300",
			expectedProjectID: "400",
		},
		{
			name:              "Empty default tenant uses 0:0",
			defaultTenantID:   "",
			requestTenant:     "",
			expectedAccountID: "0",
			expectedProjectID: "0",
		},
		{
			name:              "Partial request tenant (account only) uses default project",
			defaultTenantID:   "100:200",
			requestTenant:     "500",
			expectedAccountID: "500",
			expectedProjectID: "0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			os.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
			os.Setenv("VL_DEFAULT_TENANT_ID", tc.defaultTenantID)

			// Initialize config
			cfg, err := config.InitConfig()
			if err != nil {
				t.Fatalf("Failed to init config: %v", err)
			}

			// Create tool request
			tcr := mcp.CallToolRequest{}
			tcr.Params.Arguments = map[string]any{
				"tenant": tc.requestTenant,
			}

			// Call CreateSelectRequest
			req, err := CreateSelectRequest(context.Background(), cfg, tcr, "query")
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Check headers
			accountID := req.Header.Get("AccountID")
			projectID := req.Header.Get("ProjectID")

			if accountID != tc.expectedAccountID {
				t.Errorf("Expected AccountID %q, got %q", tc.expectedAccountID, accountID)
			}
			if projectID != tc.expectedProjectID {
				t.Errorf("Expected ProjectID %q, got %q", tc.expectedProjectID, projectID)
			}
		})
	}
}

func TestGetToolReqEnv(t *testing.T) {
	testCases := []struct {
		name      string
		args      map[string]any
		expected  string
		expectErr bool
	}{
		{
			name:     "env parameter",
			args:     map[string]any{"env": "Prod"},
			expected: "prod",
		},
		{
			name:     "environment alias",
			args:     map[string]any{"environment": "staging"},
			expected: "staging",
		},
		{
			name:      "invalid env type",
			args:      map[string]any{"env": 123},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tcr := mcp.CallToolRequest{}
			tcr.Params.Arguments = tc.args

			env, err := GetToolReqEnv(tcr)
			if tc.expectErr {
				if err == nil {
					t.Fatal("Expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if env != tc.expected {
				t.Fatalf("Expected env %q, got %q", tc.expected, env)
			}
		})
	}
}

func TestCreateSelectRequest_UsesSelectedEnvironment(t *testing.T) {
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "")
	t.Setenv("VL_INSTANCE_BEARER_TOKEN", "")
	t.Setenv("VL_INSTANCE_HEADERS", "")
	t.Setenv("VL_DEFAULT_TENANT_ID", "")
	t.Setenv("VL_ENVIRONMENTS", "demo,prod")
	t.Setenv("VL_DEFAULT_ENVIRONMENT", "demo")
	t.Setenv("VL_INSTANCE_DEMO_ENTRYPOINT", "https://demo.example.com")
	t.Setenv("VL_INSTANCE_DEMO_BEARER_TOKEN", "demo-token")
	t.Setenv("VL_INSTANCE_DEMO_HEADERS", "X-Scope=demo")
	t.Setenv("VL_INSTANCE_DEMO_DEFAULT_TENANT_ID", "1:10")
	t.Setenv("VL_INSTANCE_PROD_ENTRYPOINT", "https://prod.example.com")
	t.Setenv("VL_INSTANCE_PROD_BEARER_TOKEN", "prod-token")
	t.Setenv("VL_INSTANCE_PROD_HEADERS", "X-Scope=prod")
	t.Setenv("VL_INSTANCE_PROD_DEFAULT_TENANT_ID", "2:20")

	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	tcr := mcp.CallToolRequest{}
	tcr.Params.Arguments = map[string]any{
		"env": "prod",
	}

	req, err := CreateSelectRequest(context.Background(), cfg, tcr, "query")
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if req.URL.String() != "https://prod.example.com/select/logsql/query" {
		t.Fatalf("Expected prod query URL, got %q", req.URL.String())
	}
	if req.Header.Get("Authorization") != "Bearer prod-token" {
		t.Fatalf("Expected prod bearer token, got %q", req.Header.Get("Authorization"))
	}
	if req.Header.Get("X-Scope") != "prod" {
		t.Fatalf("Expected prod custom header, got %q", req.Header.Get("X-Scope"))
	}
	if req.Header.Get("AccountID") != "2" || req.Header.Get("ProjectID") != "20" {
		t.Fatalf("Expected prod default tenant 2:20, got %s:%s", req.Header.Get("AccountID"), req.Header.Get("ProjectID"))
	}
}

func TestCreateAdminRequest_UsesEnvironmentAlias(t *testing.T) {
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "")
	t.Setenv("VL_INSTANCE_BEARER_TOKEN", "")
	t.Setenv("VL_INSTANCE_HEADERS", "")
	t.Setenv("VL_DEFAULT_TENANT_ID", "")
	t.Setenv("VL_ENVIRONMENTS", "demo,prod")
	t.Setenv("VL_DEFAULT_ENVIRONMENT", "demo")
	t.Setenv("VL_INSTANCE_DEMO_ENTRYPOINT", "https://demo.example.com")
	t.Setenv("VL_INSTANCE_PROD_ENTRYPOINT", "https://prod.example.com")
	t.Setenv("VL_INSTANCE_PROD_BEARER_TOKEN", "prod-token")
	t.Setenv("VL_INSTANCE_PROD_HEADERS", "X-Scope=prod")

	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	tcr := mcp.CallToolRequest{}
	tcr.Params.Arguments = map[string]any{
		"environment": "prod",
	}

	req, err := CreateAdminRequest(context.Background(), cfg, tcr, "flags")
	if err != nil {
		t.Fatalf("Failed to create admin request: %v", err)
	}

	if req.URL.String() != "https://prod.example.com/flags" {
		t.Fatalf("Expected prod flags URL, got %q", req.URL.String())
	}
	if req.Header.Get("Authorization") != "Bearer prod-token" {
		t.Fatalf("Expected prod bearer token, got %q", req.Header.Get("Authorization"))
	}
	if req.Header.Get("X-Scope") != "prod" {
		t.Fatalf("Expected prod custom header, got %q", req.Header.Get("X-Scope"))
	}
}

func TestCreateSelectRequest_UnknownEnvironment(t *testing.T) {
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "https://demo.example.com")
	t.Setenv("VL_ENVIRONMENTS", "")
	t.Setenv("VL_DEFAULT_ENVIRONMENT", "")

	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	tcr := mcp.CallToolRequest{}
	tcr.Params.Arguments = map[string]any{
		"env": "prod",
	}

	_, err = CreateSelectRequest(context.Background(), cfg, tcr, "query")
	if err == nil {
		t.Fatal("Expected unknown environment error, got nil")
	}
}
