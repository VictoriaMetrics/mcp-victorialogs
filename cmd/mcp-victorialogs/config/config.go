package config

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
	"github.com/VictoriaMetrics/VictoriaMetrics/lib/promauth"
	vmcloud "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"
)

const defaultMaxResponseSize = 16 * 1024 * 1024

type Config struct {
	serverMode                 string
	listenAddr                 string
	entrypoint                 string
	bearerToken                string
	customHeaders              map[string]string
	passthroughHeaders         []string
	disabledTools              map[string]bool
	heartbeatInterval          time.Duration
	maxResponseSize            int64
	defaultTenantID            logstorage.TenantID
	stateless                  bool
	disableLocalhostProtection bool
	authToken                  string
	instructionsOverride       string
	apiKey                     string
	apiBaseURL                 string
	warnings                   []string

	entryPointURL *url.URL
	httpClient    *http.Client
	vmc           *vmcloud.VMCloudAPIClient

	// Logging configuration
	logFormat string
	logLevel  string
}

func InitConfig() (*Config, error) {
	disabledTools := os.Getenv("MCP_DISABLED_TOOLS")
	disabledToolsMap := make(map[string]bool)
	if disabledTools != "" {
		for _, tool := range strings.Split(disabledTools, ",") {
			tool = strings.Trim(tool, " ,")
			if tool != "" {
				disabledToolsMap[tool] = true
			}
		}
	}

	customHeaders := os.Getenv("VL_INSTANCE_HEADERS")
	customHeadersMap := make(map[string]string)
	if customHeaders != "" {
		for _, header := range strings.Split(customHeaders, ",") {
			header = strings.TrimSpace(header)
			if header != "" {
				parts := strings.SplitN(header, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					if key != "" && value != "" {
						customHeadersMap[key] = value
					}
				}
			}
		}
	}

	var passthroughHeaders []string
	passthroughHeadersStr := os.Getenv("MCP_PASSTHROUGH_HEADERS")
	if passthroughHeadersStr != "" {
		for _, h := range strings.Split(passthroughHeadersStr, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				passthroughHeaders = append(passthroughHeaders, h)
			}
		}
	}

	heartbeatInterval := 30 * time.Second
	heartbeatIntervalStr := os.Getenv("MCP_HEARTBEAT_INTERVAL")
	if heartbeatIntervalStr != "" {
		interval, err := time.ParseDuration(heartbeatIntervalStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP_HEARTBEAT_INTERVAL: %w", err)
		}
		if interval < 0 {
			return nil, fmt.Errorf("MCP_HEARTBEAT_INTERVAL must be a non-negative")
		}
		heartbeatInterval = interval
	}

	maxResponseSize := int64(defaultMaxResponseSize)
	maxResponseSizeStr := os.Getenv("MCP_MAX_RESPONSE_SIZE")
	if maxResponseSizeStr != "" {
		size, err := strconv.ParseInt(maxResponseSizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP_MAX_RESPONSE_SIZE: %w", err)
		}
		if size < 0 {
			return nil, fmt.Errorf("MCP_MAX_RESPONSE_SIZE must be non-negative")
		}
		maxResponseSize = size
	}

	logFormat := strings.ToLower(os.Getenv("MCP_LOG_FORMAT"))
	if logFormat == "" {
		logFormat = "text"
	}
	if logFormat != "text" && logFormat != "json" {
		return nil, fmt.Errorf("MCP_LOG_FORMAT must be 'text' or 'json'")
	}

	logLevel := strings.ToLower(os.Getenv("MCP_LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "info"
	}
	if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" {
		return nil, fmt.Errorf("MCP_LOG_LEVEL must be 'debug', 'info', 'warn' or 'error'")
	}

	stateless := false
	statelessStr := os.Getenv("MCP_STATELESS")
	if statelessStr != "" {
		v, err := strconv.ParseBool(statelessStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP_STATELESS: %w", err)
		}
		stateless = v
	}

	disableLocalhostProtection := false
	disableLocalhostStr := os.Getenv("MCP_DISABLE_LOCALHOST_PROTECTION")
	if disableLocalhostStr != "" {
		v, err := strconv.ParseBool(disableLocalhostStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MCP_DISABLE_LOCALHOST_PROTECTION: %w", err)
		}
		disableLocalhostProtection = v
	}

	result := &Config{
		serverMode:                 strings.ToLower(os.Getenv("MCP_SERVER_MODE")),
		listenAddr:                 os.Getenv("MCP_LISTEN_ADDR"),
		entrypoint:                 os.Getenv("VL_INSTANCE_ENTRYPOINT"),
		bearerToken:                os.Getenv("VL_INSTANCE_BEARER_TOKEN"),
		customHeaders:              customHeadersMap,
		passthroughHeaders:         passthroughHeaders,
		disabledTools:              disabledToolsMap,
		heartbeatInterval:          heartbeatInterval,
		maxResponseSize:            maxResponseSize,
		logFormat:                  logFormat,
		logLevel:                   logLevel,
		defaultTenantID:            logstorage.TenantID{AccountID: 0, ProjectID: 0},
		stateless:                  stateless,
		disableLocalhostProtection: disableLocalhostProtection,
		authToken:                  os.Getenv("MCP_AUTH_TOKEN"),
		instructionsOverride:       strings.TrimSpace(os.Getenv("MCP_INSTRUCTIONS_OVERRIDE")),
		apiKey:                     strings.TrimSpace(os.Getenv("VMC_API_KEY")),
		apiBaseURL:                 strings.TrimSpace(os.Getenv("VMC_API_BASE_URL")),
	}
	// Left for backward compatibility
	if result.listenAddr == "" {
		result.listenAddr = os.Getenv("MCP_SSE_ADDR")
	}
	if result.entrypoint == "" && result.apiKey == "" {
		return nil, fmt.Errorf("VL_INSTANCE_ENTRYPOINT or VMC_API_KEY is not set")
	}
	if result.entrypoint != "" && result.apiKey != "" {
		return nil, fmt.Errorf("VL_INSTANCE_ENTRYPOINT and VMC_API_KEY cannot be set at the same time")
	}
	if result.serverMode != "" && result.serverMode != "stdio" && result.serverMode != "sse" && result.serverMode != "http" {
		return nil, fmt.Errorf("MCP_SERVER_MODE must be 'stdio', 'sse' or 'http'")
	}
	if result.serverMode == "" {
		result.serverMode = "stdio"
	}
	if result.listenAddr == "" {
		result.listenAddr = "localhost:8081"
	}
	if result.apiKey == vmcloud.DynamicAPIKey && result.serverMode == "stdio" {
		return nil, fmt.Errorf("VMC_API_KEY=%s requires MCP_SERVER_MODE 'sse' or 'http', since the API key is taken from the %s header of every request", vmcloud.DynamicAPIKey, vmcloud.AccessTokenHeader)
	}
	if result.apiKey != "" {
		result.warnings = cloudWarnings(result)
	}

	defaultTenantID := strings.ToLower(os.Getenv("VL_DEFAULT_TENANT_ID"))
	if defaultTenantID != "" {
		tenantID, err := logstorage.ParseTenantID(defaultTenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse VL_DEFAULT_TENANT_ID %q: %w", defaultTenantID, err)
		}
		result.defaultTenantID = tenantID
	}

	var err error

	if result.apiKey == "" {
		result.entryPointURL, err = url.Parse(result.entrypoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL from VL_INSTANCE_ENTRYPOINT: %w", err)
		}
	}

	caCert := os.Getenv("VL_INSTANCE_CA_CERT")
	caCertFile := os.Getenv("VL_INSTANCE_CA_CERT_FILE")
	if caCert != "" && caCertFile != "" {
		return nil, fmt.Errorf("VL_INSTANCE_CA_CERT and VL_INSTANCE_CA_CERT_FILE cannot be set at the same time")
	}
	// promauth would fetch http(s) URLs with no timeout and, for http, no integrity protection.
	if u, err := url.Parse(caCertFile); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
		return nil, fmt.Errorf("VL_INSTANCE_CA_CERT_FILE must be a local file path, got URL %q", caCertFile)
	}
	result.httpClient = http.DefaultClient
	if caCert != "" || caCertFile != "" {
		result.httpClient, err = newHTTPClientWithCA(caCert, caCertFile)
		if err != nil {
			return nil, err
		}
	}

	if result.apiKey != "" {
		var vmcOptions []vmcloud.VMCloudAPIClientOption
		if result.apiBaseURL != "" {
			vmcOptions = append(vmcOptions, vmcloud.WithBaseURL(result.apiBaseURL))
		}
		vmcOptions = append(vmcOptions, vmcloud.WithHTTPClient(result.httpClient))
		result.vmc, err = vmcloud.New(result.apiKey, vmcOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to create VMCloud API client: %w", err)
		}
	}

	return result, nil
}

// newHTTPClientWithCA returns an HTTP client that verifies server certificates
// against the PEM-encoded CA certificates from caCert or caCertFile
// instead of the system root CAs.
//
// caCertFile is re-read at most once per second, so a rotated CA is picked up
// without a restart.
func newHTTPClientWithCA(caCert, caCertFile string) (*http.Client, error) {
	// Some environments only accept single-line env var values,
	// so literal "\n" sequences are treated as newlines.
	if !strings.Contains(caCert, "\n") {
		caCert = strings.ReplaceAll(caCert, `\n`, "\n")
	}
	opts := &promauth.Options{
		TLSConfig: &promauth.TLSConfig{
			CA:     caCert,
			CAFile: caCertFile,
		},
	}
	ac, err := opts.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load VL_INSTANCE_CA_CERT: %w", err)
	}
	// Load the CA file once to fail at startup instead of on the first request.
	if _, err := ac.GetTLSConfig(); err != nil {
		return nil, fmt.Errorf("failed to load VL_INSTANCE_CA_CERT_FILE: %w", err)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return &http.Client{Transport: ac.NewRoundTripper(transport)}, nil
}

func (c *Config) IsStdio() bool {
	return c.serverMode == "stdio"
}

func (c *Config) IsSSE() bool {
	return c.serverMode == "sse"
}

func (c *Config) ServerMode() string {
	return c.serverMode
}

func (c *Config) ListenAddr() string {
	return c.listenAddr
}

func (c *Config) BearerToken() string {
	return c.bearerToken
}

func (c *Config) EntryPointURL() *url.URL {
	return c.entryPointURL
}

func (c *Config) IsToolDisabled(toolName string) bool {
	if c.disabledTools == nil {
		return false
	}
	disabled, ok := c.disabledTools[toolName]
	return ok && disabled
}

func (c *Config) HeartbeatInterval() time.Duration {
	return c.heartbeatInterval
}

// HTTPClient returns the client for requests to VictoriaLogs and the VictoriaMetrics Cloud API.
// It trusts the CA certificates from VL_INSTANCE_CA_CERT or VL_INSTANCE_CA_CERT_FILE
// when one of them is set, and is http.DefaultClient otherwise.
func (c *Config) HTTPClient() *http.Client {
	return c.httpClient
}

// MaxResponseSize returns the maximum size in bytes of a VictoriaLogs response
// a tool reads. Zero means no limit.
func (c *Config) MaxResponseSize() int64 {
	return c.maxResponseSize
}

func (c *Config) CustomHeaders() map[string]string {
	return c.customHeaders
}

func (c *Config) PassthroughHeaders() []string {
	return c.passthroughHeaders
}

func (c *Config) LogFormat() string {
	return c.logFormat
}

func (c *Config) LogLevel() string {
	return c.logLevel
}

func (c *Config) DefaultTenantID() logstorage.TenantID {
	return c.defaultTenantID
}

func (c *Config) Stateless() bool {
	return c.stateless
}

func (c *Config) DisableLocalhostProtection() bool {
	return c.disableLocalhostProtection
}

// AuthToken returns the bearer token required on incoming MCP requests in sse/http modes.
// Empty means authentication is disabled.
func (c *Config) AuthToken() string {
	return c.authToken
}

// InstructionsOverride returns text that replaces the default MCP server instructions.
// Empty means the default instructions are used.
func (c *Config) InstructionsOverride() string {
	return c.instructionsOverride
}

// IsCloud reports whether the server works with VictoriaMetrics Cloud deployments (VMC_API_KEY is set)
// instead of a single VictoriaLogs instance from VL_INSTANCE_ENTRYPOINT.
func (c *Config) IsCloud() bool {
	return c.vmc != nil
}

// IsCloudSharedInstance reports whether the VictoriaMetrics Cloud API key is taken
// from every incoming MCP request (VMC_API_KEY=dynamic) instead of the configuration.
func (c *Config) IsCloudSharedInstance() bool {
	return c.apiKey == vmcloud.DynamicAPIKey
}

// VMC returns the VictoriaMetrics Cloud API client, nil unless IsCloud.
func (c *Config) VMC() *vmcloud.VMCloudAPIClient {
	return c.vmc
}

// Warnings returns messages about settings that are accepted but probably do not work as intended.
func (c *Config) Warnings() []string {
	return c.warnings
}

// cloudWarnings returns warnings about settings that conflict with the access tokens
// of VictoriaLogs deployments in VictoriaMetrics Cloud.
func cloudWarnings(c *Config) []string {
	var warnings []string
	if c.bearerToken != "" {
		warnings = append(warnings, "VL_INSTANCE_BEARER_TOKEN is ignored with VMC_API_KEY: requests to VictoriaLogs deployments use their access tokens")
	}
	for key := range c.customHeaders {
		if http.CanonicalHeaderKey(key) == "Authorization" {
			warnings = append(warnings, "VL_INSTANCE_HEADERS sets the Authorization header, which replaces the access token of VictoriaLogs deployments with VMC_API_KEY")
			break
		}
	}
	for _, name := range c.passthroughHeaders {
		if http.CanonicalHeaderKey(name) == "Authorization" {
			warnings = append(warnings, "MCP_PASSTHROUGH_HEADERS forwards the Authorization header, which replaces the access token of VictoriaLogs deployments with VMC_API_KEY when a request has it")
			break
		}
	}
	return warnings
}
