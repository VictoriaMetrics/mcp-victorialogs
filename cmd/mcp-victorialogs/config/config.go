package config

import (
	"fmt"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/VictoriaMetrics/VictoriaLogs/lib/logstorage"
)

const defaultEnvironmentName = "default"

type InstanceConfig struct {
	name            string
	bearerToken     string
	customHeaders   map[string]string
	defaultTenantID logstorage.TenantID

	entryPointURL *url.URL
}

type Config struct {
	serverMode         string
	listenAddr         string
	disabledTools      map[string]bool
	heartbeatInterval  time.Duration
	environments       map[string]*InstanceConfig
	environmentOrder   []string
	defaultEnvironment string

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

	result := &Config{
		serverMode:        strings.ToLower(os.Getenv("MCP_SERVER_MODE")),
		listenAddr:        os.Getenv("MCP_LISTEN_ADDR"),
		disabledTools:     disabledToolsMap,
		heartbeatInterval: heartbeatInterval,
		logFormat:         logFormat,
		logLevel:          logLevel,
	}
	// Left for backward compatibility
	if result.listenAddr == "" {
		result.listenAddr = os.Getenv("MCP_SSE_ADDR")
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

	environments, environmentOrder, defaultEnvironment, err := initEnvironmentConfigs()
	if err != nil {
		return nil, err
	}
	result.environments = environments
	result.environmentOrder = environmentOrder
	result.defaultEnvironment = defaultEnvironment

	return result, nil
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
	env, err := c.Environment("")
	if err != nil {
		return ""
	}
	return env.BearerToken()
}

func (c *Config) EntryPointURL() *url.URL {
	env, err := c.Environment("")
	if err != nil {
		return nil
	}
	return env.EntryPointURL()
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

func (c *Config) CustomHeaders() map[string]string {
	env, err := c.Environment("")
	if err != nil {
		return nil
	}
	return env.CustomHeaders()
}

func (c *Config) LogFormat() string {
	return c.logFormat
}

func (c *Config) LogLevel() string {
	return c.logLevel
}

func (c *Config) DefaultTenantID() logstorage.TenantID {
	env, err := c.Environment("")
	if err != nil {
		return logstorage.TenantID{AccountID: 0, ProjectID: 0}
	}
	return env.DefaultTenantID()
}

func (c *Config) DefaultEnvironment() string {
	return c.defaultEnvironment
}

func (c *Config) EnvironmentNames() []string {
	return slices.Clone(c.environmentOrder)
}

func (c *Config) Environment(name string) (*InstanceConfig, error) {
	if len(c.environments) == 0 {
		return nil, fmt.Errorf("no VictoriaLogs environments configured")
	}

	resolvedName := strings.TrimSpace(strings.ToLower(name))
	if resolvedName == "" {
		resolvedName = c.defaultEnvironment
	}

	env, ok := c.environments[resolvedName]
	if !ok {
		return nil, fmt.Errorf("unknown VictoriaLogs env %q; available envs: %s", resolvedName, strings.Join(c.environmentOrder, ", "))
	}
	return env, nil
}

func (c *InstanceConfig) Name() string {
	return c.name
}

func (c *InstanceConfig) BearerToken() string {
	return c.bearerToken
}

func (c *InstanceConfig) CustomHeaders() map[string]string {
	return c.customHeaders
}

func (c *InstanceConfig) DefaultTenantID() logstorage.TenantID {
	return c.defaultTenantID
}

func (c *InstanceConfig) EntryPointURL() *url.URL {
	return c.entryPointURL
}

func initEnvironmentConfigs() (map[string]*InstanceConfig, []string, string, error) {
	if envNamesValue := os.Getenv("VL_ENVIRONMENTS"); envNamesValue != "" {
		if err := validateNoLegacyInstanceConfig(); err != nil {
			return nil, nil, "", err
		}

		envNames, err := parseEnvironmentNames(envNamesValue)
		if err != nil {
			return nil, nil, "", err
		}

		defaultEnvironment := strings.TrimSpace(strings.ToLower(os.Getenv("VL_DEFAULT_ENVIRONMENT")))
		if defaultEnvironment == "" {
			defaultEnvironment = envNames[0]
		}
		if !slices.Contains(envNames, defaultEnvironment) {
			return nil, nil, "", fmt.Errorf("VL_DEFAULT_ENVIRONMENT %q is not listed in VL_ENVIRONMENTS", defaultEnvironment)
		}

		environments := make(map[string]*InstanceConfig, len(envNames))
		for _, envName := range envNames {
			prefix := environmentVarPrefix(envName)
			instance, err := newInstanceConfig(
				envName,
				os.Getenv(prefix+"ENTRYPOINT"),
				os.Getenv(prefix+"BEARER_TOKEN"),
				parseHeaders(os.Getenv(prefix+"HEADERS")),
				os.Getenv(prefix+"DEFAULT_TENANT_ID"),
				prefix+"DEFAULT_TENANT_ID",
			)
			if err != nil {
				return nil, nil, "", err
			}
			environments[envName] = instance
		}

		return environments, envNames, defaultEnvironment, nil
	}

	instance, err := newInstanceConfig(
		defaultEnvironmentName,
		os.Getenv("VL_INSTANCE_ENTRYPOINT"),
		os.Getenv("VL_INSTANCE_BEARER_TOKEN"),
		parseHeaders(os.Getenv("VL_INSTANCE_HEADERS")),
		os.Getenv("VL_DEFAULT_TENANT_ID"),
		"VL_DEFAULT_TENANT_ID",
	)
	if err != nil {
		return nil, nil, "", err
	}

	return map[string]*InstanceConfig{defaultEnvironmentName: instance}, []string{defaultEnvironmentName}, defaultEnvironmentName, nil
}

func validateNoLegacyInstanceConfig() error {
	for _, envVar := range []string{
		"VL_INSTANCE_ENTRYPOINT",
		"VL_INSTANCE_BEARER_TOKEN",
		"VL_INSTANCE_HEADERS",
		"VL_DEFAULT_TENANT_ID",
	} {
		if os.Getenv(envVar) != "" {
			return fmt.Errorf("%s cannot be combined with VL_ENVIRONMENTS; use per-environment variables instead", envVar)
		}
	}
	return nil
}

func parseEnvironmentNames(value string) ([]string, error) {
	names := make([]string, 0)
	seenNames := make(map[string]struct{})
	seenPrefixes := make(map[string]string)

	for _, rawName := range strings.Split(value, ",") {
		name := strings.TrimSpace(strings.ToLower(rawName))
		if name == "" {
			continue
		}
		if !isValidEnvironmentName(name) {
			return nil, fmt.Errorf("VL_ENVIRONMENTS contains invalid env name %q; only letters, numbers, dashes, and underscores are allowed", rawName)
		}
		if _, ok := seenNames[name]; ok {
			return nil, fmt.Errorf("VL_ENVIRONMENTS contains duplicate env name %q", name)
		}

		prefix := environmentVarPrefix(name)
		if existingName, ok := seenPrefixes[prefix]; ok {
			return nil, fmt.Errorf("VL_ENVIRONMENTS names %q and %q map to the same environment variable prefix %q", existingName, name, prefix)
		}

		seenNames[name] = struct{}{}
		seenPrefixes[prefix] = name
		names = append(names, name)
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("VL_ENVIRONMENTS is set but does not contain any env names")
	}

	return names, nil
}

func isValidEnvironmentName(value string) bool {
	for _, r := range value {
		if isASCIIAlphaNumeric(r) || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func environmentVarPrefix(name string) string {
	var b strings.Builder
	b.WriteString("VL_INSTANCE_")
	for _, r := range strings.ToUpper(name) {
		if isASCIIAlphaNumeric(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	b.WriteByte('_')
	return b.String()
}

func isASCIIAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func newInstanceConfig(name, entrypoint, bearerToken string, customHeaders map[string]string, defaultTenantID, defaultTenantEnvVar string) (*InstanceConfig, error) {
	if strings.TrimSpace(entrypoint) == "" {
		if name == defaultEnvironmentName {
			return nil, fmt.Errorf("VL_INSTANCE_ENTRYPOINT is not set")
		}
		return nil, fmt.Errorf("%sENTRYPOINT is not set", environmentVarPrefix(name))
	}

	tenantID := logstorage.TenantID{AccountID: 0, ProjectID: 0}
	if defaultTenantID != "" {
		parsedTenantID, err := logstorage.ParseTenantID(strings.ToLower(defaultTenantID))
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s %q: %w", defaultTenantEnvVar, defaultTenantID, err)
		}
		tenantID = parsedTenantID
	}

	entryPointURL, err := url.Parse(entrypoint)
	if err != nil {
		if name == defaultEnvironmentName {
			return nil, fmt.Errorf("failed to parse URL from VL_INSTANCE_ENTRYPOINT: %w", err)
		}
		return nil, fmt.Errorf("failed to parse URL from %sENTRYPOINT: %w", environmentVarPrefix(name), err)
	}

	return &InstanceConfig{
		name:            name,
		bearerToken:     bearerToken,
		customHeaders:   customHeaders,
		defaultTenantID: tenantID,
		entryPointURL:   entryPointURL,
	}, nil
}

func parseHeaders(value string) map[string]string {
	headers := make(map[string]string)
	if value == "" {
		return headers
	}

	for _, header := range strings.Split(value, ",") {
		header = strings.TrimSpace(header)
		if header == "" {
			continue
		}

		parts := strings.SplitN(header, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		headerValue := strings.TrimSpace(parts[1])
		if key == "" || headerValue == "" {
			continue
		}
		headers[key] = headerValue
	}

	return headers
}
