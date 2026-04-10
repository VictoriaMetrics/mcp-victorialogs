package prompts

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs/config"
)

var (
	promptDocumentation = mcp.NewPrompt("documentation",
		mcp.WithPromptDescription("Search by VictoriaLogs documentation or give context for answering questions"),
		mcp.WithArgument("query",
			mcp.RequiredArgument(),
			mcp.ArgumentDescription("What do you want to search in the VictoriaLogs documentation?"),
		),
	)
)

func promptDocumentationHandler(_ context.Context, gpr mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	query, err := GetPromptReqParam(gpr, "query", true)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %v", err)
	}
	return mcp.NewGetPromptResult(
		"",
		[]mcp.PromptMessage{
			{
				Role:    mcp.RoleUser,
				Content: mcp.NewTextContent(fmt.Sprintf(`Please tell me about "%v" by VictoriaLogs documentation`, query)),
			},
		},
	), nil
}

func RegisterPromptDocumentation(s *server.MCPServer, _ *config.Config) {
	s.AddPrompt(promptDocumentation, promptDocumentationHandler)
}
