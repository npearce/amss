package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/kb-mcp/internal/client"
)

func init() {
	registerTool(ReadKBArticle())
}

type ReadKBArticleParams struct {
	ID string `json:"id" description:"Article ID (e.g., KB-001)."`
}

type ReadKBArticleResult struct {
	Text string `json:"text"`
}

func ReadKBArticle() MCPTool[ReadKBArticleParams, ReadKBArticleResult] {
	return MCPTool[ReadKBArticleParams, ReadKBArticleResult]{
		Name:        "read_kb_article",
		Description: "Read the full content of a KB article by ID.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ReadKBArticleParams]) (*mcp.CallToolResultFor[ReadKBArticleResult], error) {
			if kbClient == nil {
				return nil, fmt.Errorf("KB client not initialized")
			}
			id := params.Arguments.ID
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}

			article, err := kbClient.GetArticle(id)
			if err != nil {
				return nil, fmt.Errorf("article not found: %w", err)
			}

			return &mcp.CallToolResultFor[ReadKBArticleResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: client.FormatArticle(article)}},
			}, nil
		},
	}
}
