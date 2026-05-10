package tools

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	registerTool(ListKBCategories())
}

type ListKBCategoriesParams struct{}

type ListKBCategoriesResult struct {
	Text string `json:"text"`
}

var validCategories = []string{
	"life-support",
	"navigation",
	"comms",
	"power",
	"propulsion",
	"eva",
	"medical",
	"operations",
	"thermal",
	"structures",
}

func ListKBCategories() MCPTool[ListKBCategoriesParams, ListKBCategoriesResult] {
	return MCPTool[ListKBCategoriesParams, ListKBCategoriesResult]{
		Name:        "list_kb_categories",
		Description: "List the 10 valid KB article categories. Use these values for the category field in search, create, and update calls.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListKBCategoriesParams]) (*mcp.CallToolResultFor[ListKBCategoriesResult], error) {
			text := "Valid KB article categories:\n\n" + strings.Join(validCategories, "\n")
			return &mcp.CallToolResultFor[ListKBCategoriesResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}
