package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameFlags = "flags"

var (
	toolFlags = mcp.NewTool(toolNameFlags,
		mcp.WithDescription("List of non-default flags (parameters) of the VictoriaLogs instance. This tools uses `/flags` endpoint of VictoriaLogs API."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of non-default flags (parameters)",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
	)
)

func toolFlagsHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	req, err := CreateAdminRequest(ctx, cfg, tcr, "flags")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}
	return GetTextBodyForRequest(req, cfg), nil
}

func RegisterToolFlags(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameFlags) {
		return
	}
	// VictoriaMetrics Cloud routes only the /select/... APIs of a VictoriaLogs deployment
	// for read access tokens, so /flags is not reachable in cloud mode.
	if c.IsCloud() {
		return
	}
	s.AddTool(toolFlags, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolFlagsHandler(ctx, c, request)
	})
}
