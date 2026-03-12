package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/VictoriaMetrics-Community/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

func CreateSelectRequest(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest, path ...string) (*http.Request, error) {
	accountID, projectID, err := GetToolReqTenant(tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %v", err)
	}

	environment, err := getToolEnvironment(cfg, tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve env: %v", err)
	}

	selectURL, err := getSelectURL(ctx, environment, path...)
	if err != nil {
		return nil, fmt.Errorf("failed to get select URL: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	bearerToken := environment.BearerToken()
	if bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	// Add custom headers from configuration
	for key, value := range environment.CustomHeaders() {
		req.Header.Set(key, value)
	}

	defaultTenantID := environment.DefaultTenantID()
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
	environment, err := getToolEnvironment(cfg, tcr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve env: %v", err)
	}

	selectURL, err := getRootURL(ctx, environment, path...)
	if err != nil {
		return nil, fmt.Errorf("failed to get select URL: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, selectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	bearerToken := environment.BearerToken()
	if bearerToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
	}

	// Add custom headers from configuration
	for key, value := range environment.CustomHeaders() {
		req.Header.Set(key, value)
	}

	return req, nil
}

func getRootURL(_ context.Context, environment *config.InstanceConfig, path ...string) (string, error) {
	return environment.EntryPointURL().JoinPath(path...).String(), nil
}

func getSelectURL(_ context.Context, environment *config.InstanceConfig, path ...string) (string, error) {
	return environment.EntryPointURL().JoinPath("select", "logsql").JoinPath(path...).String(), nil
}

func GetTextBodyForRequest(req *http.Request, _ *config.Config) *mcp.CallToolResult {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to do request: %v", err))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read response body: %v", err))
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("unexpected response status code %v: %s", resp.StatusCode, string(body)))
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

func GetToolReqEnv(tcr mcp.CallToolRequest) (string, error) {
	env, err := GetToolReqParam[string](tcr, "env", false)
	if err != nil {
		return "", fmt.Errorf("failed to get env: %v", err)
	}
	return strings.ToLower(strings.TrimSpace(env)), nil
}

func getToolEnvironment(cfg *config.Config, tcr mcp.CallToolRequest) (*config.InstanceConfig, error) {
	envName, err := GetToolReqEnv(tcr)
	if err != nil {
		return nil, err
	}
	return cfg.Environment(envName)
}

func withEnvironmentParam() mcp.ToolOption {
	return mcp.WithString("env",
		mcp.Title("Environment"),
		mcp.Description("Optional VictoriaLogs environment to target. If omitted, the server default environment is used."),
		mcp.Pattern(`^[A-Za-z0-9_-]+$`),
	)
}

func ptr[T any](v T) *T {
	return &v
}
