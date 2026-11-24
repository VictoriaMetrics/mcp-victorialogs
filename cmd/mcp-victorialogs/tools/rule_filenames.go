package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameRuleFilenames = "rule_filenames"

var (
	toolRuleFilenames = mcp.NewTool(toolNameRuleFilenames,
		mcp.WithDescription("List of alerting and recording rules files of the VictoriaLogs deployment in VictoriaMetrics Cloud. Rule groups of VictoriaLogs deployments use LogsQL expressions (`type: vlogs`). Use the `rule_file` tool to get the content of a file."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of alerting and recording rules files",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
		withDeploymentIDParam(),
	)
)

func toolRuleFilenamesHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentID, err := getToolReqDeploymentID(tcr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if cfg.IsCloudSharedInstance() {
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	filenames, err := cfg.VMC().ListDeploymentRuleFileNames(ctx, deploymentID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list rule filenames: %v", err)), nil
	}
	data, err := json.Marshal(filenames)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal rule filenames: %v", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

// RegisterToolRuleFilenames registers the rule_filenames tool. It is available only with VMC_API_KEY.
func RegisterToolRuleFilenames(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameRuleFilenames) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolRuleFilenames, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolRuleFilenamesHandler(ctx, c, request)
	})
}
