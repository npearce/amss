package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
)

func init() {
	registerTool(SearchTickets())
}

type SearchTicketsParams struct {
	Mission  string `json:"mission" description:"Filter by mission (e.g., artemis-ii)."`
	Severity string `json:"severity" description:"Filter by severity: P1, P2, P3, or P4."`
	Status   string `json:"status" description:"Filter by status: open, in-progress, resolved, or closed."`
	Category string `json:"category" description:"Filter by category."`
	Search   string `json:"search" description:"Full-text search over title and description."`
	Limit    int    `json:"limit" description:"Maximum results (default 50, max 200)."`
	Offset   int    `json:"offset" description:"Pagination offset (default 0)."`
}

type SearchTicketsResult struct {
	Text string `json:"text"`
}

func SearchTickets() MCPTool[SearchTicketsParams, SearchTicketsResult] {
	return MCPTool[SearchTicketsParams, SearchTicketsResult]{
		Name:        "search_tickets",
		Description: "Search and filter mission support tickets by mission, severity, status, category, or text. Returns matching ticket summaries.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[SearchTicketsParams]) (*mcp.CallToolResultFor[SearchTicketsResult], error) {
			if ticketClient == nil {
				return nil, fmt.Errorf("ticket client not initialized")
			}
			p := params.Arguments
			limit := p.Limit
			if limit <= 0 {
				limit = 50
			}

			tickets, total, err := ticketClient.Search(p.Mission, p.Severity, p.Status, p.Category, p.Search, limit, p.Offset)
			if err != nil {
				return nil, fmt.Errorf("search failed: %w", err)
			}

			text := formatSearchResults(tickets, total, p.Offset)
			return &mcp.CallToolResultFor[SearchTicketsResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}

func formatSearchResults(tickets []*client.Ticket, total, offset int) string {
	if total == 0 {
		return "No tickets found."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d ticket(s) (showing %d", total, len(tickets))
	if offset > 0 {
		fmt.Fprintf(&sb, " from offset %d", offset)
	}
	sb.WriteString("):\n\n")
	for i, t := range tickets {
		fmt.Fprintf(&sb, "%d. **%s** [%s/%s] — %s\n   Mission: %s | Assigned: %s\n",
			i+1+offset, t.ID, t.Severity, t.Status, t.Title, t.Mission, t.AssignedTo)
	}
	return sb.String()
}
