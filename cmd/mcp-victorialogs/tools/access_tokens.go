package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameAccessTokens = "access_tokens"

var (
	toolAccessTokens = mcp.NewTool(toolNameAccessTokens,
		mcp.WithDescription("List of access tokens of the VictoriaLogs deployment in VictoriaMetrics Cloud"),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of deployment access tokens",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
		withDeploymentIDParam(),
	)
)

func toolAccessTokensHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentID, err := getToolReqDeploymentID(tcr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if cfg.IsCloudSharedInstance() {
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	result, err := cfg.VMC().ListDeploymentAccessTokens(ctx, deploymentID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list access tokens: %v", err)), nil
	}
	data, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal access tokens: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// RegisterToolAccessTokens registers the access_tokens tool. It is available only with VMC_API_KEY.
func RegisterToolAccessTokens(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameAccessTokens) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolAccessTokens, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolAccessTokensHandler(ctx, c, request)
	})
}
