package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

const toolNameStreams = "streams"

var (
	toolStreams = mcp.NewTool(toolNameStreams,
		mcp.WithDescription("Get log streams from results of the given <query> on the given [<start> ... <end>] time range. The response also contains the number of log results per every _stream. This tool uses `/select/logsql/streams` endpoint of VictoriaLogs API."),
		mcp.WithToolAnnotation(mcp.ToolAnnotation{
			Title:           "List of log streams",
			ReadOnlyHint:    new(true),
			DestructiveHint: new(false),
			OpenWorldHint:   new(true),
		}),
		mcp.WithString("tenant",
			mcp.Title("Tenant name (Account ID and Project ID)"),
			mcp.Description("Name of the tenant for which the data will be displayed (format AccountID:ProjectID)"),
			mcp.DefaultString("0:0"),
			mcp.Pattern(`^([0-9]+)(:[0-9]+)$`),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Title("LogsQL expression"),
			mcp.Description(`LogsQL expression for the query of the logs streams`),
		),
		mcp.WithString("start",
			mcp.Required(),
			mcp.Title("Start timestamp"),
			mcp.Description("Start timestamp in RFC3339"),
			mcp.Pattern(`^((?:(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2}(?:\.\d+)?))(Z|[\+-]\d{2}:\d{2})?)|([0-9]+)$`),
		),
		mcp.WithString("end",
			mcp.Title("End timestamp"),
			mcp.Description("End timestamp in RFC3339. If <end> is missing, then it equals to the maximum timestamp across logs stored in VictoriaLogs."),
			mcp.Pattern(`^((?:(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2}(?:\.\d+)?))(Z|[\+-]\d{2}:\d{2})?)|([0-9]+)$`),
		),
	)
)

func toolStreamsHandler(ctx context.Context, cfg *config.Config, tcr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := GetToolReqParam[string](tcr, "query", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	start, err := GetToolReqParam[string](tcr, "start", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	end, err := GetToolReqParam[string](tcr, "end", false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	q := url.Values{}
	q.Add("query", query)
	q.Add("start", start)
	if end != "" {
		q.Add("end", end)
	}

	req, err := CreateSelectRequest(ctx, cfg, tcr, q, "streams")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	return GetTextBodyForRequest(req, cfg), nil
}

func RegisterToolStreams(s *server.MCPServer, c *config.Config) {
	if c.IsToolDisabled(toolNameStreams) {
		return
	}
	s.AddTool(withCloudParams(c, toolStreams), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return toolStreamsHandler(ctx, c, request)
	})
}
