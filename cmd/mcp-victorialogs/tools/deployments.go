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

const toolNameDeployments = "deployments"

var (
	toolDeployments = mcp.NewTool(toolNameDeployments,
		mcp.WithDescription("List of VictoriaLogs deployments in VictoriaMetrics Cloud. Use the `id` of a deployment as the `deployment_id` argument of other tools."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of deployments",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
	)
)

func toolDeploymentsHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cfg.IsCloudSharedInstance() {
		var err error
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	result, err := cfg.VMC().ListDeployments(ctx, vmcloud.WithDeploymentType(vmcloud.DeploymentTypeVLogs))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list deployments: %v", err)), nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal deployments: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// RegisterToolDeployments registers the deployments tool. It is available only with VMC_API_KEY.
func RegisterToolDeployments(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameDeployments) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolDeployments, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolDeploymentsHandler(ctx, c, request)
	})
}
