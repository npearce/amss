package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/kb-mcp/internal/client"
)

func init() {
	registerTool(UpdateKBArticle())
}

type UpdateKBArticleParams struct {
	ID              string   `json:"id" description:"Article ID to update (required)."`
	Title           *string  `json:"title,omitempty" description:"New title."`
	Body            *string  `json:"body,omitempty" description:"New body (markdown)."`
	Category        *string  `json:"category,omitempty" description:"New category."`
	Tags            []string `json:"tags,omitempty" description:"New tags (replaces existing)."`
	UsefulnessScore *float64 `json:"usefulness_score,omitempty" description:"Usefulness score 0.0–1.0 (curator field)."`
	DuplicateOf     *string  `json:"duplicate_of,omitempty" description:"Mark as duplicate of this KB ID (curator field)."`
	CuratorNotes    *string  `json:"curator_notes,omitempty" description:"Curator notes (curator field)."`
	CuratorTags     []string `json:"curator_tags,omitempty" description:"Curator tags (curator field)."`
}

type UpdateKBArticleResult struct {
	Text string `json:"text"`
}

func UpdateKBArticle() MCPTool[UpdateKBArticleParams, UpdateKBArticleResult] {
	return MCPTool[UpdateKBArticleParams, UpdateKBArticleResult]{
		Name:        "update_kb_article",
		Description: "Partially update a KB article. Only supplied fields are changed. Curator fields: usefulness_score, duplicate_of, curator_notes, curator_tags.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[UpdateKBArticleParams]) (*mcp.CallToolResultFor[UpdateKBArticleResult], error) {
			if kbClient == nil {
				return nil, fmt.Errorf("KB client not initialized")
			}
			p := params.Arguments
			if p.ID == "" {
				return nil, fmt.Errorf("id is required")
			}

			updates := make(map[string]interface{})
			if p.Title != nil {
				updates["title"] = *p.Title
			}
			if p.Body != nil {
				updates["body"] = *p.Body
			}
			if p.Category != nil {
				updates["category"] = *p.Category
			}
			if p.Tags != nil {
				updates["tags"] = p.Tags
			}
			if p.UsefulnessScore != nil {
				updates["usefulness_score"] = *p.UsefulnessScore
			}
			if p.DuplicateOf != nil {
				updates["duplicate_of"] = *p.DuplicateOf
			}
			if p.CuratorNotes != nil {
				updates["curator_notes"] = *p.CuratorNotes
			}
			if p.CuratorTags != nil {
				updates["curator_tags"] = p.CuratorTags
			}

			if len(updates) == 0 {
				return nil, fmt.Errorf("no fields to update")
			}

			article, err := kbClient.UpdateArticle(p.ID, updates)
			if err != nil {
				return nil, fmt.Errorf("update failed: %w", err)
			}

			text := fmt.Sprintf("Updated article %s:\n\n%s", article.ID, client.FormatArticle(article))
			return &mcp.CallToolResultFor[UpdateKBArticleResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}
