package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/kb-mcp/internal/client"
)

func init() {
	registerTool(SearchKB())
}

type SearchKBParams struct {
	Category string `json:"category" description:"Filter by category (life-support, navigation, comms, power, propulsion, eva, medical, operations, thermal, structures)."`
	Tags     string `json:"tags" description:"Comma-separated tags to filter by."`
	Search   string `json:"search" description:"Full-text search over article titles and bodies."`
	Limit    int    `json:"limit" description:"Maximum results (default 50, max 200)."`
	Offset   int    `json:"offset" description:"Pagination offset (default 0)."`
}

type SearchKBResult struct {
	Text string `json:"text"`
}

func SearchKB() MCPTool[SearchKBParams, SearchKBResult] {
	return MCPTool[SearchKBParams, SearchKBResult]{
		Name:        "search_kb",
		Description: "Search knowledge base articles by text, category, or tags. Returns matching article summaries.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[SearchKBParams]) (*mcp.CallToolResultFor[SearchKBResult], error) {
			if kbClient == nil {
				return nil, fmt.Errorf("KB client not initialized")
			}
			p := params.Arguments
			limit := p.Limit
			if limit <= 0 {
				limit = 50
			}

			articles, total, err := kbClient.Search(p.Category, p.Tags, p.Search, limit, p.Offset)
			if err != nil {
				return nil, fmt.Errorf("search failed: %w", err)
			}

			text := formatSearchResults(articles, total, p.Offset)
			return &mcp.CallToolResultFor[SearchKBResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}

func formatSearchResults(articles []*client.Article, total, offset int) string {
	if total == 0 {
		return "No articles found."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d article(s) (showing %d", total, len(articles))
	if offset > 0 {
		fmt.Fprintf(&sb, " from offset %d", offset)
	}
	sb.WriteString("):\n\n")
	for i, a := range articles {
		tags := strings.Join(a.Tags, ", ")
		if tags == "" {
			tags = "(none)"
		}
		fmt.Fprintf(&sb, "%d. **%s** — %s\n   Category: %s | Tags: %s\n",
			i+1+offset, a.ID, a.Title, a.Category, tags)
	}
	return sb.String()
}
