package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameCloudProviders = "cloud_providers"

var (
	toolCloudProviders = mcp.NewTool(toolNameCloudProviders,
		mcp.WithDescription("List of cloud providers in VictoriaMetrics Cloud"),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of cloud providers",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
	)
)

func toolCloudProvidersHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cfg.IsCloudSharedInstance() {
		var err error
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	result, err := cfg.VMC().ListCloudProviders(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list cloud providers: %v", err)), nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal cloud providers: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// RegisterToolCloudProviders registers the cloud_providers tool. It is available only with VMC_API_KEY.
func RegisterToolCloudProviders(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameCloudProviders) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolCloudProviders, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolCloudProvidersHandler(ctx, c, request)
	})
}
