package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	registerTool(AddTicketComment())
}

type AddTicketCommentParams struct {
	TicketID string `json:"ticket_id" description:"Ticket ID to comment on (e.g., AMSS-001)."`
	Author   string `json:"author" description:"Author's crew ID."`
	Text     string `json:"text" description:"Comment text."`
}

type AddTicketCommentResult struct {
	Text string `json:"text"`
}

func AddTicketComment() MCPTool[AddTicketCommentParams, AddTicketCommentResult] {
	return MCPTool[AddTicketCommentParams, AddTicketCommentResult]{
		Name:        "add_ticket_comment",
		Description: "Add a comment to an existing ticket. Updates the ticket's updated_at timestamp.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[AddTicketCommentParams]) (*mcp.CallToolResultFor[AddTicketCommentResult], error) {
			if ticketClient == nil {
				return nil, fmt.Errorf("ticket client not initialized")
			}
			p := params.Arguments
			if p.TicketID == "" {
				return nil, fmt.Errorf("ticket_id is required")
			}
			if p.Author == "" {
				return nil, fmt.Errorf("author is required")
			}
			if p.Text == "" {
				return nil, fmt.Errorf("text is required")
			}

			comment, err := ticketClient.AddComment(p.TicketID, p.Author, p.Text)
			if err != nil {
				return nil, fmt.Errorf("add comment failed: %w", err)
			}

			text := fmt.Sprintf("Comment added to %s:\n\n[%s] %s:\n%s",
				p.TicketID, comment.Timestamp, comment.Author, comment.Text)
			return &mcp.CallToolResultFor[AddTicketCommentResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}
