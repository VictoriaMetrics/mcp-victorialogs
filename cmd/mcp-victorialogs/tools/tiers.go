package tools

import (
	"context"
	"encoding/json"
	"fmt"

	vmcloud "github.com/VictoriaMetrics/victoriametrics-cloud-api-go/v1"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameTiers = "tiers"

var (
	toolTiers = mcp.NewTool(toolNameTiers,
		mcp.WithDescription("List of VictoriaLogs deployment tiers in VictoriaMetrics Cloud"),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of tiers",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
	)
)

func toolTiersHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cfg.IsCloudSharedInstance() {
		var err error
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	result, err := cfg.VMC().ListTiers(ctx, vmcloud.WithDeploymentType(vmcloud.DeploymentTypeVLogs))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tiers: %v", err)), nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal tiers: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// RegisterToolTiers registers the tiers tool. It is available only with VMC_API_KEY.
func RegisterToolTiers(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameTiers) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolTiers, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolTiersHandler(ctx, c, request)
	})
}
