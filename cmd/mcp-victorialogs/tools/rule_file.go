package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameRuleFile = "rule_file"

var (
	toolRuleFile = mcp.NewTool(toolNameRuleFile,
		mcp.WithDescription("Content (YAML) of the alerting and recording rules file of the VictoriaLogs deployment in VictoriaMetrics Cloud. Rule groups of VictoriaLogs deployments use LogsQL expressions (`type: vlogs`)."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "Rules file content",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
		withDeploymentIDParam(),
		mcp.WithString("filename",
			mcp.Required(),
			mcp.Title("Rules filename"),
			mcp.Description("Name of the rules file to retrieve. This should be one of the files listed by the `rule_filenames` tool."),
			mcp.MinLength(1),
			mcp.MaxLength(255),
		),
	)
)

func toolRuleFileHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentID, err := getToolReqDeploymentID(tcr)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	filename, err := GetToolReqParam[string](tcr, "filename", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if filename == "" {
		return mcp.NewToolResultError("filename parameter must not be empty"), nil
	}
	if cfg.IsCloudSharedInstance() {
		if ctx, err = withCloudAccessKey(ctx, tcr); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}
	content, err := cfg.VMC().GetDeploymentRuleFileContent(ctx, deploymentID, filename)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get rule file %q: %v", filename, err)), nil
	}
	return mcp.NewToolResultText(content), nil
}

// RegisterToolRuleFile registers the rule_file tool. It is available only with VMC_API_KEY.
func RegisterToolRuleFile(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameRuleFile) {
		return
	}
	if !c.IsCloud() {
		return
	}
	s.AddTool(toolRuleFile, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolRuleFileHandler(ctx, c, request)
	})
}
