package tools

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	vmcloud "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

// defaultStep is the `step` value advertised by the hits and stats_query_range schemas
// (mcp.DefaultString). mcp-go writes schema defaults for client hinting only - nothing applies
// them server-side - so the handlers apply this themselves.
const defaultStep = "1d"

// CreateSelectRequest creates a request to the /select/logsql/... API that sends args in a
// form-encoded POST body rather than in the URL, since proxies in front of VictoriaLogs
// write request URLs, LogsQL queries included, to their access logs.
func CreateSelectRequest(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest, args url.Values, path ...string) (*http.Request, error) {
	accountID, projectID, err := GetToolReqTenant(tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %v", err)
	}

	selectURL, err := getSelectURL(ctx, cfg, tcr, path...)
	if err != nil {
		return nil, fmt.Errorf("failed to get select URL: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, selectURL, strings.NewReader(args.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Select requests only read data. An empty Idempotency-Key marks them idempotent without being
	// sent, so net/http retries them on a reused connection the server has closed, as it does for GET.
	req.Header["Idempotency-Key"] = nil
	bearerToken, err := getBearerToken(ctx, cfg, tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %v", err)
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	// Add custom headers from configuration
	for key, value := range cfg.CustomHeaders() {
		req.Header.Set(key, value)
	}

	// Apply passthrough headers from the incoming MCP request
	for _, name := range cfg.PassthroughHeaders() {
		if value := tcr.Header.Get(name); value != "" {
			req.Header.Set(name, value)
		}
	}

	defaultTenantID := cfg.DefaultTenantID()
	if accountID == "" {
		accountID = strconv.FormatUint(uint64(defaultTenantID.AccountID), 10)
	}
	if projectID == "" {
		projectID = strconv.FormatUint(uint64(defaultTenantID.ProjectID), 10)
	}

	req.Header.Set("AccountID", accountID)
	req.Header.Set("ProjectID", projectID)

	return req, nil
}

func CreateAdminRequest(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest, path ...string) (*http.Request, error) {
	selectURL, err := getRootURL(ctx, cfg, tcr, path...)
	if err != nil {
		return nil, fmt.Errorf("failed to get select URL: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	bearerToken, err := getBearerToken(ctx, cfg, tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token: %v", err)
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	// Add custom headers from configuration
	for key, value := range cfg.CustomHeaders() {
		req.Header.Set(key, value)
	}

	// Apply passthrough headers from the incoming MCP request
	for _, name := range cfg.PassthroughHeaders() {
		if value := tcr.Header.Get(name); value != "" {
			req.Header.Set(name, value)
		}
	}

	return req, nil
}

// toolDeploymentIDParam is the name of the tool argument with the VictoriaMetrics Cloud deployment ID.
const toolDeploymentIDParam = "deployment_id"

// withCloudParams returns tool with the deployment_id argument added when the server works
// with VictoriaMetrics Cloud, and tool itself otherwise.
// The input schema is copied, so the package-level tool definitions stay untouched.
func withCloudParams(c *config.Config, tool mcp.Tool) mcp.Tool {
	if !c.IsCloud() {
		return tool
	}
	tool.InputSchema.Properties = maps.Clone(tool.InputSchema.Properties)
	if tool.InputSchema.Properties == nil {
		tool.InputSchema.Properties = make(map[string]any)
	}
	tool.InputSchema.Required = slices.Clone(tool.InputSchema.Required)
	withDeploymentIDParam()(&tool)
	return tool
}

func withDeploymentIDParam() mcp.ToolOption {
	return mcp.WithString(toolDeploymentIDParam,
		mcp.Required(),
		mcp.Title("Deployment ID"),
		mcp.Description("Unique identifier of the VictoriaLogs deployment in VictoriaMetrics Cloud"),
		mcp.Pattern(`^[a-zA-Z0-9\-_]+$`),
	)
}

func getToolReqDeploymentID(tcr mcp.CallToolRequest) (string, error) {
	deploymentID, err := GetToolReqParam[string](tcr, toolDeploymentIDParam, true)
	if err != nil {
		return "", fmt.Errorf("failed to get deployment_id parameter: %v", err)
	}
	if deploymentID == "" {
		return "", fmt.Errorf("deployment_id parameter is required for cloud mode")
	}
	return deploymentID, nil
}

var (
	cloudAccessTokenCacheMutex      = &sync.RWMutex{}
	cloudAccessTokenCache           = make(map[string]string)
	cloudDeploymentEndpointCacheMux = &sync.RWMutex{}
	cloudDeploymentEndpointCache    = make(map[string]*url.URL)
)

// cloudCacheKey returns the key for the per-deployment caches. It includes the API key of the request
// on a shared instance, so one user never gets data cached for another one.
func cloudCacheKey(cfg *config.Config, tcr mcp.CallToolRequest, deploymentID string) string {
	apiKey := ""
	if cfg.IsCloudSharedInstance() {
		apiKey = tcr.Header.Get(vmcloud.AccessTokenHeader)
	}
	return fmt.Sprintf("%s/%s", apiKey, deploymentID)
}

// getCloudDeploymentEndpoint returns the access endpoint of the VictoriaLogs deployment from the request.
func getCloudDeploymentEndpoint(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*url.URL, error) {
	deploymentID, err := getToolReqDeploymentID(tcr)
	if err != nil {
		return nil, err
	}
	if cfg.IsCloudSharedInstance() {
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return nil, err
		}
	}
	cacheKey := cloudCacheKey(cfg, tcr, deploymentID)

	cloudDeploymentEndpointCacheMux.RLock()
	endpoint, ok := cloudDeploymentEndpointCache[cacheKey]
	cloudDeploymentEndpointCacheMux.RUnlock()
	if ok {
		return endpoint, nil
	}

	dd, err := cfg.VMC().GetDeploymentDetails(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment details: %v", err)
	}
	if dd.Type != vmcloud.DeploymentTypeVLogs {
		return nil, fmt.Errorf("unsupported deployment type %s for deployment %s, only %s deployments are supported", dd.Type, deploymentID, vmcloud.DeploymentTypeVLogs)
	}
	if dd.AccessEndpoint == "" {
		return nil, fmt.Errorf("deployment %s has no access endpoint", deploymentID)
	}
	endpoint, err = url.Parse(dd.AccessEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment access endpoint: %v", err)
	}

	cloudDeploymentEndpointCacheMux.Lock()
	cloudDeploymentEndpointCache[cacheKey] = endpoint
	cloudDeploymentEndpointCacheMux.Unlock()

	return endpoint, nil
}

// getBearerToken returns VL_INSTANCE_BEARER_TOKEN, or in cloud mode the secret of an access token
// of the deployment from the request: a read-only one, or a read-write one when the deployment
// has no read-only tokens. Tokens limited to a tenant are skipped.
func getBearerToken(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (string, error) {
	if !cfg.IsCloud() {
		return cfg.BearerToken(), nil
	}

	deploymentID, err := getToolReqDeploymentID(tcr)
	if err != nil {
		return "", err
	}
	if cfg.IsCloudSharedInstance() {
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return "", err
		}
	}
	cacheKey := cloudCacheKey(cfg, tcr, deploymentID)

	cloudAccessTokenCacheMutex.RLock()
	result, ok := cloudAccessTokenCache[cacheKey]
	cloudAccessTokenCacheMutex.RUnlock()
	if ok {
		return result, nil
	}

	at, err := cfg.VMC().ListDeploymentAccessTokens(ctx, deploymentID)
	if err != nil {
		return "", fmt.Errorf("failed to list deployment access tokens: %v", err)
	}
	if len(at) == 0 {
		return "", fmt.Errorf("no access tokens found for deployment %s", deploymentID)
	}
	tokenID := ""
	for _, mode := range []vmcloud.AccessMode{vmcloud.AccessModeRead, vmcloud.AccessModeReadWrite} {
		for _, t := range at {
			if t.Type == mode && t.TenantID == "" {
				tokenID = t.ID
				break
			}
		}
		if tokenID != "" {
			break
		}
	}
	if tokenID == "" {
		for _, t := range at {
			if (t.Type == vmcloud.AccessModeRead || t.Type == vmcloud.AccessModeReadWrite) && t.TenantID != "" {
				return "", fmt.Errorf("deployment %s has only read access tokens limited to a tenant, which are not supported; create a read access token without a tenant", deploymentID)
			}
		}
		return "", fmt.Errorf("no read access tokens found for deployment %s", deploymentID)
	}
	token, err := cfg.VMC().RevealDeploymentAccessToken(ctx, deploymentID, tokenID)
	if err != nil {
		return "", fmt.Errorf("failed to reveal access token for deployment %s: %v", deploymentID, err)
	}
	if token.Secret == "" {
		return "", fmt.Errorf("VictoriaMetrics Cloud API returned an empty secret for access token %s of deployment %s", tokenID, deploymentID)
	}
	cloudAccessTokenCacheMutex.Lock()
	cloudAccessTokenCache[cacheKey] = token.Secret
	cloudAccessTokenCacheMutex.Unlock()
	return token.Secret, nil
}

func getEntryPointURL(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*url.URL, error) {
	if !cfg.IsCloud() {
		return cfg.EntryPointURL(), nil
	}
	endpoint, err := getCloudDeploymentEndpoint(ctx, cfg, tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud deployment endpoint: %v", err)
	}
	return endpoint, nil
}

func getRootURL(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest, path ...string) (string, error) {
	entryPointURL, err := getEntryPointURL(ctx, cfg, tcr)
	if err != nil {
		return "", err
	}
	return entryPointURL.JoinPath(path...).String(), nil
}

func getSelectURL(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest, path ...string) (string, error) {
	entryPointURL, err := getEntryPointURL(ctx, cfg, tcr)
	if err != nil {
		return "", err
	}
	return entryPointURL.JoinPath("select", "logsql").JoinPath(path...).String(), nil
}

// withCloudAccessKey puts the VictoriaMetrics Cloud API key from the request header into ctx
// for a shared instance (VMC_API_KEY=dynamic). The header is required there.
func withCloudAccessKey(ctx context.Context, tcr mcp.CallToolRequest) (context.Context, error) {
	accessKey := tcr.Header.Get(vmcloud.AccessTokenHeader)
	if accessKey == "" {
		return nil, fmt.Errorf("%s header with a VictoriaMetrics Cloud API key is required, since the server runs with VMC_API_KEY=%s", vmcloud.AccessTokenHeader, vmcloud.DynamicAPIKey)
	}
	return vmcloud.ContextWithDynamicAPIKey(ctx, accessKey), nil
}

func GetTextBodyForRequest(req *http.Request, cfg *config.Config) *mcp.CallToolResult {
	resp, err := cfg.HTTPClient().Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to do request: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()
	maxSize := cfg.MaxResponseSize()
	var bodyReader io.Reader = resp.Body
	if maxSize > 0 {
		// Read one byte past the limit to tell a body of exactly maxSize bytes from a larger one.
		bodyReader = io.LimitReader(resp.Body, maxSize+1)
	}
	body, err := io.ReadAll(bodyReader)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read response body: %v", err))
	}
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("unexpected response status code %v: %s", resp.StatusCode, string(body)))
	}
	if maxSize > 0 && int64(len(body)) > maxSize {
		return mcp.NewToolResultError(fmt.Sprintf("response is larger than %d bytes (MCP_MAX_RESPONSE_SIZE); narrow the query or add a `| limit N` pipe to reduce the result", maxSize))
	}
	return mcp.NewToolResultText(string(body))
}

type ToolReqParamType interface {
	string | float64 | bool | []string | []any
}

func GetToolReqParam[T ToolReqParamType](tcr mcp.CallToolRequest, param string, required bool) (T, error) {
	var value T
	matchArg, ok := tcr.GetArguments()[param]
	if ok {
		value, ok = matchArg.(T)
		if !ok {
			return value, fmt.Errorf("%s has wrong type: %T", param, matchArg)
		}
	} else if required {
		return value, fmt.Errorf("%s param is required", param)
	}
	return value, nil
}

func GetToolReqTenant(tcr mcp.CallToolRequest) (string, string, error) {
	tenant, err := GetToolReqParam[string](tcr, "tenant", false)
	if err != nil {
		return "", "", fmt.Errorf("failed to get tenant: %v", err)
	}
	if tenant == "" {
		return "", "", nil
	}
	tenantID, err := logstorage.ParseTenantID(tenant)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse tenant %q: %v", tenant, err)
	}
	accountID := strconv.FormatUint(uint64(tenantID.AccountID), 10)
	projectID := strconv.FormatUint(uint64(tenantID.ProjectID), 10)
	return accountID, projectID, nil
}
