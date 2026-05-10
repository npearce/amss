package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/kb-mcp/internal/client"
)

func init() {
	registerTool(CreateKBArticle())
}

type CreateKBArticleParams struct {
	Title     string   `json:"title" description:"Article title (required)."`
	Body      string   `json:"body" description:"Article body in markdown (required)."`
	Category  string   `json:"category" description:"Category: life-support, navigation, comms, power, propulsion, eva, medical, operations, thermal, or structures (required)."`
	Tags      []string `json:"tags" description:"Optional tags for the article."`
	CreatedBy string   `json:"created_by" description:"Creator ID (defaults to ground-control)."`
}

type CreateKBArticleResult struct {
	Text string `json:"text"`
}

func CreateKBArticle() MCPTool[CreateKBArticleParams, CreateKBArticleResult] {
	return MCPTool[CreateKBArticleParams, CreateKBArticleResult]{
		Name:        "create_kb_article",
		Description: "Create a new KB article. Requires title, body, and category.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CreateKBArticleParams]) (*mcp.CallToolResultFor[CreateKBArticleResult], error) {
			if kbClient == nil {
				return nil, fmt.Errorf("KB client not initialized")
			}
			p := params.Arguments
			if p.Title == "" || p.Body == "" || p.Category == "" {
				return nil, fmt.Errorf("title, body, and category are required")
			}
			createdBy := p.CreatedBy
			if createdBy == "" {
				createdBy = "ground-control"
			}

			article, err := kbClient.CreateArticle(p.Title, p.Body, p.Category, createdBy, p.Tags)
			if err != nil {
				return nil, fmt.Errorf("create failed: %w", err)
			}

			text := fmt.Sprintf("Created article %s:\n\n%s", article.ID, client.FormatArticle(article))
			return &mcp.CallToolResultFor[CreateKBArticleResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}
