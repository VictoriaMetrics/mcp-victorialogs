package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	vmcloud "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const (
	deploymentVLOK          = "00000000-0000-4000-8000-000000000001"
	deploymentVM            = "00000000-0000-4000-8000-000000000002"
	deploymentVLWriteOnly   = "00000000-0000-4000-8000-000000000003"
	deploymentVLShared      = "00000000-0000-4000-8000-000000000004"
	deploymentVLReadWrite   = "00000000-0000-4000-8000-000000000005"
	deploymentVLEmptySecret = "00000000-0000-4000-8000-000000000006"
	deploymentMissing       = "00000000-0000-4000-8000-000000000009"
)

// fakeCloudAPI serves the VictoriaMetrics Cloud API endpoints used to resolve a deployment.
type fakeCloudAPI struct {
	deployments map[string]vmcloud.DeploymentInfo
	tokens      []vmcloud.AccessToken
	secrets     map[string]string
	// ruleFiles maps a rules file name to its content.
	ruleFiles map[string]string
	calls     atomic.Int64

	mu sync.Mutex
	// apiKeys records the API key of every request.
	apiKeys []string
}

func (f *fakeCloudAPI) requestAPIKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.apiKeys)
}

func (f *fakeCloudAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)
	f.mu.Lock()
	f.apiKeys = append(f.apiKeys, r.Header.Get(vmcloud.AccessTokenHeader))
	f.mu.Unlock()
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/deployments"), "/"), "/")
	var resp any
	switch {
	case len(parts) == 1 && parts[0] != "":
		d, ok := f.deployments[parts[0]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		resp = d
	case len(parts) == 3 && parts[1] == "rule-sets" && parts[2] == "files":
		names := make([]string, 0, len(f.ruleFiles))
		for name := range f.ruleFiles {
			names = append(names, name)
		}
		slices.Sort(names)
		resp = names
	case len(parts) == 4 && parts[1] == "rule-sets" && parts[2] == "files":
		content, ok := f.ruleFiles[parts[3]]
		if !ok {
			http.Error(w, `{"error": "file not found"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write([]byte(content))
		return
	case len(parts) == 2 && parts[1] == "access_tokens":
		resp = f.tokens
	case len(parts) == 3 && parts[1] == "access_tokens":
		for _, t := range f.tokens {
			if t.ID == parts[2] {
				t.Secret = f.secrets[t.ID]
				resp = t
			}
		}
		if resp == nil {
			http.NotFound(w, r)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func newCloudConfig(t *testing.T, apiKey string, api *fakeCloudAPI) *config.Config {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "")
	t.Setenv("VL_INSTANCE_CA_CERT", "")
	t.Setenv("VL_INSTANCE_CA_CERT_FILE", "")
	t.Setenv("VMC_API_KEY", apiKey)
	t.Setenv("VMC_API_BASE_URL", srv.URL)
	t.Setenv("MCP_SERVER_MODE", "http")
	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}
	if !cfg.IsCloud() {
		t.Fatal("Expected IsCloud() to be true")
	}
	return cfg
}

func newToolRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Header: http.Header{},
		Params: mcp.CallToolParams{Arguments: args},
	}
}

func TestCreateSelectRequestCloud(t *testing.T) {
	api := &fakeCloudAPI{
		deployments: map[string]vmcloud.DeploymentInfo{
			deploymentVLOK: {ID: deploymentVLOK, Type: vmcloud.DeploymentTypeVLogs, AccessEndpoint: "https://vl-ok.cloud.example.com"},
			deploymentVM:   {ID: deploymentVM, Type: vmcloud.DeploymentTypeSingleNode, AccessEndpoint: "https://vm-one.cloud.example.com"},
		},
		tokens: []vmcloud.AccessToken{
			{ID: "write", Type: vmcloud.AccessModeWrite},
			{ID: "readwrite", Type: vmcloud.AccessModeReadWrite},
			{ID: "tenant", Type: vmcloud.AccessModeRead, TenantID: "1:2"},
			{ID: "read", Type: vmcloud.AccessModeRead},
		},
		secrets: map[string]string{"write": "write-secret", "readwrite": "readwrite-secret", "tenant": "tenant-secret", "read": "read-secret"},
	}
	cfg := newCloudConfig(t, "test-key", api)

	tcr := newToolRequest(map[string]any{"deployment_id": deploymentVLOK})
	req, err := CreateSelectRequest(context.Background(), cfg, tcr, "query")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if got, want := req.URL.String(), "https://vl-ok.cloud.example.com/select/logsql/query"; got != want {
		t.Errorf("Expected URL %q, got %q", want, got)
	}
	if got, want := req.Header.Get("Authorization"), "Bearer read-secret"; got != want {
		t.Errorf("Expected Authorization %q, got %q", want, got)
	}

	req, err = CreateAdminRequest(context.Background(), cfg, tcr, "flags")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if got, want := req.URL.String(), "https://vl-ok.cloud.example.com/flags"; got != want {
		t.Errorf("Expected URL %q, got %q", want, got)
	}

	// The endpoint and the token are cached, so repeated requests do not hit the API.
	calls := api.calls.Load()
	if _, err := CreateSelectRequest(context.Background(), cfg, tcr, "query"); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if api.calls.Load() != calls {
		t.Errorf("Expected cached deployment info and token, got %d more API calls", api.calls.Load()-calls)
	}

	for _, key := range api.requestAPIKeys() {
		if key != "test-key" {
			t.Errorf("Expected API key %q in every Cloud API request, got %q", "test-key", key)
		}
	}

	testCases := []struct {
		name          string
		args          map[string]any
		expectedError string
	}{
		{
			name:          "missing deployment_id",
			args:          map[string]any{},
			expectedError: "deployment_id param is required",
		},
		{
			name:          "empty deployment_id",
			args:          map[string]any{"deployment_id": ""},
			expectedError: "deployment_id parameter is required for cloud mode",
		},
		{
			name:          "VictoriaMetrics deployment",
			args:          map[string]any{"deployment_id": deploymentVM},
			expectedError: "unsupported deployment type single_node for deployment " + deploymentVM,
		},
		{
			name:          "unknown deployment",
			args:          map[string]any{"deployment_id": deploymentMissing},
			expectedError: "failed to get deployment details",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CreateSelectRequest(context.Background(), cfg, newToolRequest(tc.args), "query")
			if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
				t.Fatalf("Expected error containing %q, got: %v", tc.expectedError, err)
			}
		})
	}
}

func TestCreateSelectRequestCloudNoReadToken(t *testing.T) {
	api := &fakeCloudAPI{
		deployments: map[string]vmcloud.DeploymentInfo{
			deploymentVLWriteOnly: {ID: deploymentVLWriteOnly, Type: vmcloud.DeploymentTypeVLogs, AccessEndpoint: "https://vl.cloud.example.com"},
		},
		tokens:  []vmcloud.AccessToken{{ID: "write", Type: vmcloud.AccessModeWrite}},
		secrets: map[string]string{"write": "write-secret"},
	}
	cfg := newCloudConfig(t, "test-key", api)

	tcr := newToolRequest(map[string]any{"deployment_id": deploymentVLWriteOnly})
	_, err := CreateSelectRequest(context.Background(), cfg, tcr, "query")
	if err == nil || !strings.Contains(err.Error(), "no read access tokens found for deployment "+deploymentVLWriteOnly) {
		t.Fatalf("Expected no read access tokens error, got: %v", err)
	}
}

func TestCreateSelectRequestCloudReadWriteToken(t *testing.T) {
	api := &fakeCloudAPI{
		deployments: map[string]vmcloud.DeploymentInfo{
			deploymentVLReadWrite: {ID: deploymentVLReadWrite, Type: vmcloud.DeploymentTypeVLogs, AccessEndpoint: "https://vl.cloud.example.com"},
		},
		tokens: []vmcloud.AccessToken{
			{ID: "write", Type: vmcloud.AccessModeWrite},
			{ID: "tenant", Type: vmcloud.AccessModeReadWrite, TenantID: "1:2"},
			{ID: "readwrite", Type: vmcloud.AccessModeReadWrite},
		},
		secrets: map[string]string{"write": "write-secret", "tenant": "tenant-secret", "readwrite": "readwrite-secret"},
	}
	cfg := newCloudConfig(t, "test-key", api)

	tcr := newToolRequest(map[string]any{"deployment_id": deploymentVLReadWrite})
	req, err := CreateSelectRequest(context.Background(), cfg, tcr, "query")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if got, want := req.Header.Get("Authorization"), "Bearer readwrite-secret"; got != want {
		t.Errorf("Expected Authorization %q, got %q", want, got)
	}
}

func TestCloudSharedInstanceMissingHeader(t *testing.T) {
	cfg := newCloudConfig(t, vmcloud.DynamicAPIKey, &fakeCloudAPI{})
	const expectedError = "X-VM-Cloud-Access header with a VictoriaMetrics Cloud API key is required"

	_, err := CreateSelectRequest(context.Background(), cfg, newToolRequest(map[string]any{"deployment_id": deploymentVLOK}), "query")
	if err == nil || !strings.Contains(err.Error(), expectedError) {
		t.Fatalf("Expected missing header error, got: %v", err)
	}

	r, err := toolDeploymentsHandler(context.Background(), cfg, newToolRequest(map[string]any{}))
	if err != nil {
		t.Fatalf("Expected a tool error result, got: %v", err)
	}
	if !r.IsError || !strings.Contains(r.Content[0].(mcp.TextContent).Text, expectedError) {
		t.Fatalf("Expected missing header error, got: %v", r.Content)
	}
}

func TestCreateSelectRequestCloudSharedInstance(t *testing.T) {
	api := &fakeCloudAPI{
		deployments: map[string]vmcloud.DeploymentInfo{
			deploymentVLShared: {ID: deploymentVLShared, Type: vmcloud.DeploymentTypeVLogs, AccessEndpoint: "https://vl-shared.cloud.example.com"},
		},
		tokens:  []vmcloud.AccessToken{{ID: "read", Type: vmcloud.AccessModeRead}},
		secrets: map[string]string{"read": "read-secret"},
	}
	cfg := newCloudConfig(t, vmcloud.DynamicAPIKey, api)
	if !cfg.IsCloudSharedInstance() {
		t.Fatal("Expected IsCloudSharedInstance() to be true")
	}

	tcr := newToolRequest(map[string]any{"deployment_id": deploymentVLShared})
	tcr.Header.Set(vmcloud.AccessTokenHeader, "user-key")
	if _, err := CreateSelectRequest(context.Background(), cfg, tcr, "query"); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(api.requestAPIKeys()) == 0 {
		t.Fatal("Expected Cloud API requests")
	}
	for _, key := range api.requestAPIKeys() {
		if key != "user-key" {
			t.Errorf("Expected API key from the request header in every Cloud API request, got %q", key)
		}
	}

	// Another user does not get the token cached for the first one.
	calls := api.calls.Load()
	tcr.Header.Set(vmcloud.AccessTokenHeader, "other-user-key")
	if _, err := CreateSelectRequest(context.Background(), cfg, tcr, "query"); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if api.calls.Load() == calls {
		t.Error("Expected Cloud API requests for another API key, got cached results")
	}
}

func TestWithCloudParams(t *testing.T) {
	t.Setenv("VMC_API_KEY", "")
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
	t.Setenv("VL_INSTANCE_CA_CERT", "")
	t.Setenv("VL_INSTANCE_CA_CERT_FILE", "")
	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}
	tool := withCloudParams(cfg, toolQuery)
	if _, ok := tool.InputSchema.Properties["deployment_id"]; ok {
		t.Error("Expected no deployment_id argument without VMC_API_KEY")
	}

	cloudCfg := newCloudConfig(t, "test-key", &fakeCloudAPI{})
	tool = withCloudParams(cloudCfg, toolQuery)
	if _, ok := tool.InputSchema.Properties["deployment_id"]; !ok {
		t.Error("Expected deployment_id argument with VMC_API_KEY")
	}
	if !slices.Contains(tool.InputSchema.Required, "deployment_id") {
		t.Error("Expected deployment_id argument to be required")
	}
	if _, ok := toolQuery.InputSchema.Properties["deployment_id"]; ok {
		t.Error("Expected the package-level tool definition to stay untouched")
	}
	if slices.Contains(toolQuery.InputSchema.Required, "deployment_id") {
		t.Error("Expected the package-level required arguments to stay untouched")
	}
}

func TestRuleFileTools(t *testing.T) {
	const rules = "groups:\n  - name: errors\n    type: vlogs\n    rules:\n      - alert: TooManyErrors\n        expr: 'error | stats count() as errors | filter errors:>10'\n"
	api := &fakeCloudAPI{ruleFiles: map[string]string{"errors.yml": rules, "a.yml": "groups: []\n"}}
	cfg := newCloudConfig(t, "test-key", api)

	r, err := toolRuleFilenamesHandler(context.Background(), cfg, newToolRequest(map[string]any{"deployment_id": deploymentVLOK}))
	if err != nil || r.IsError {
		t.Fatalf("Expected no error, got: %v %v", err, r.Content)
	}
	if got, want := r.Content[0].(mcp.TextContent).Text, `["a.yml","errors.yml"]`; got != want {
		t.Errorf("Expected rule filenames %s, got %s", want, got)
	}

	r, err = toolRuleFileHandler(context.Background(), cfg, newToolRequest(map[string]any{"deployment_id": deploymentVLOK, "filename": "errors.yml"}))
	if err != nil || r.IsError {
		t.Fatalf("Expected no error, got: %v %v", err, r.Content)
	}
	// The YAML is returned as is, not as a JSON-encoded string.
	if got := r.Content[0].(mcp.TextContent).Text; got != rules {
		t.Errorf("Expected rules file content %q, got %q", rules, got)
	}

	testCases := []struct {
		name          string
		args          map[string]any
		expectedError string
	}{
		{name: "missing filename", args: map[string]any{"deployment_id": deploymentVLOK}, expectedError: "filename param is required"},
		{name: "empty filename", args: map[string]any{"deployment_id": deploymentVLOK, "filename": ""}, expectedError: "filename parameter must not be empty"},
		{name: "missing deployment_id", args: map[string]any{"filename": "errors.yml"}, expectedError: "deployment_id param is required"},
		{name: "unknown file", args: map[string]any{"deployment_id": deploymentVLOK, "filename": "missing.yml"}, expectedError: `failed to get rule file "missing.yml"`},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := toolRuleFileHandler(context.Background(), cfg, newToolRequest(tc.args))
			if err != nil {
				t.Fatalf("Expected a tool error result, got: %v", err)
			}
			if !r.IsError || !strings.Contains(r.Content[0].(mcp.TextContent).Text, tc.expectedError) {
				t.Fatalf("Expected error containing %q, got: %v", tc.expectedError, r.Content)
			}
		})
	}

	for _, filename := range []string{
		"../../access_tokens/read",
		"../errors.yml",
		"a/../../access_tokens/read",
		"sub/errors.yml",
		`sub\errors.yml`,
		"..",
		".hidden.yml",
		"errors..yml",
		"errors.yml%2F..",
		strings.Repeat("a", 256),
	} {
		t.Run("invalid filename "+filename[:min(len(filename), 32)], func(t *testing.T) {
			calls := api.calls.Load()
			r, err := toolRuleFileHandler(context.Background(), cfg, newToolRequest(map[string]any{"deployment_id": deploymentVLOK, "filename": filename}))
			if err != nil {
				t.Fatalf("Expected a tool error result, got: %v", err)
			}
			if !r.IsError || !strings.Contains(r.Content[0].(mcp.TextContent).Text, "invalid filename") {
				t.Fatalf("Expected invalid filename error, got: %v", r.Content)
			}
			if api.calls.Load() != calls {
				t.Errorf("Expected no Cloud API calls for invalid filename %q", filename)
			}
		})
	}
}

func TestCreateSelectRequestCloudEmptySecret(t *testing.T) {
	api := &fakeCloudAPI{
		deployments: map[string]vmcloud.DeploymentInfo{
			deploymentVLEmptySecret: {ID: deploymentVLEmptySecret, Type: vmcloud.DeploymentTypeVLogs, AccessEndpoint: "https://vl.cloud.example.com"},
		},
		tokens:  []vmcloud.AccessToken{{ID: "read", Type: vmcloud.AccessModeRead}},
		secrets: map[string]string{},
	}
	cfg := newCloudConfig(t, "test-key", api)

	tcr := newToolRequest(map[string]any{"deployment_id": deploymentVLEmptySecret})
	for range 2 {
		_, err := CreateSelectRequest(context.Background(), cfg, tcr, "query")
		if err == nil || !strings.Contains(err.Error(), "empty secret for access token read") {
			t.Fatalf("Expected empty secret error, got: %v", err)
		}
	}
	cloudAccessTokenCacheMutex.RLock()
	_, cached := cloudAccessTokenCache["/"+deploymentVLEmptySecret]
	cloudAccessTokenCacheMutex.RUnlock()
	if cached {
		t.Error("Expected the empty secret not to be cached")
	}
}

func TestToolRegistrationByMode(t *testing.T) {
	register := func(cfg *config.Config) *server.MCPServer {
		s := server.NewMCPServer("test", "test", server.WithToolCapabilities(true))
		RegisterToolFlags(s, cfg)
		RegisterToolQuery(s, cfg)
		RegisterToolRuleFile(s, cfg)
		RegisterToolRuleFilenames(s, cfg)
		RegisterToolDeployments(s, cfg)
		return s
	}

	t.Setenv("VMC_API_KEY", "")
	t.Setenv("VL_INSTANCE_ENTRYPOINT", "http://example.com")
	t.Setenv("VL_INSTANCE_CA_CERT", "")
	t.Setenv("VL_INSTANCE_CA_CERT_FILE", "")
	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}
	s := register(cfg)
	for _, name := range []string{"flags", "query"} {
		if s.GetTool(name) == nil {
			t.Errorf("Expected tool %q without VMC_API_KEY", name)
		}
	}
	for _, name := range []string{"rule_file", "rule_filenames", "deployments"} {
		if s.GetTool(name) != nil {
			t.Errorf("Expected no tool %q without VMC_API_KEY", name)
		}
	}

	s = register(newCloudConfig(t, "test-key", &fakeCloudAPI{}))
	for _, name := range []string{"query", "rule_file", "rule_filenames", "deployments"} {
		if s.GetTool(name) == nil {
			t.Errorf("Expected tool %q with VMC_API_KEY", name)
		}
	}
	// Read access tokens of VictoriaLogs deployments in VictoriaMetrics Cloud cannot reach /flags.
	if s.GetTool("flags") != nil {
		t.Error("Expected no flags tool with VMC_API_KEY")
	}
}
