package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameRegions = "regions"

var (
	toolRegions = mcp.NewTool(toolNameRegions,
		mcp.WithDescription("List of regions in VictoriaMetrics Cloud"),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of regions",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
	)
)

func toolRegionsHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cfg.IsCloudSharedInstance() {
		var err error
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	result, err := cfg.VMC().ListRegions(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list regions: %v", err)), nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal regions: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// RegisterToolRegions registers the regions tool. It is available only with VMC_API_KEY.
func RegisterToolRegions(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameRegions) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolRegions, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolRegionsHandler(ctx, c, request)
	})
}
