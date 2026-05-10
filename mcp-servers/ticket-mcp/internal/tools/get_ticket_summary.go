package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func init() {
	registerTool(GetTicketSummary())
}

type GetTicketSummaryParams struct{}

type GetTicketSummaryResult struct {
	Text string `json:"text"`
}

func GetTicketSummary() MCPTool[GetTicketSummaryParams, GetTicketSummaryResult] {
	return MCPTool[GetTicketSummaryParams, GetTicketSummaryResult]{
		Name:        "get_ticket_summary",
		Description: "Get an aggregate summary of all tickets by severity and status. Useful for situational awareness.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[GetTicketSummaryParams]) (*mcp.CallToolResultFor[GetTicketSummaryResult], error) {
			if ticketClient == nil {
				return nil, fmt.Errorf("ticket client not initialized")
			}

			tickets, total, err := ticketClient.Search("", "", "", "", "", 200, 0)
			if err != nil {
				return nil, fmt.Errorf("summary fetch failed: %w", err)
			}

			if total == 0 {
				return &mcp.CallToolResultFor[GetTicketSummaryResult]{
					Content: []mcp.Content{&mcp.TextContent{Text: "No tickets found."}},
				}, nil
			}

			bySeverity := map[string]int{"P1": 0, "P2": 0, "P3": 0, "P4": 0}
			byStatus := map[string]int{"open": 0, "in-progress": 0, "resolved": 0, "closed": 0}
			for _, t := range tickets {
				bySeverity[t.Severity]++
				byStatus[t.Status]++
			}

			var sb strings.Builder
			fmt.Fprintf(&sb, "Ticket Summary (%d total)\n\n", total)
			sb.WriteString("By Severity:\n")
			fmt.Fprintf(&sb, "  P1 — Critical:      %d\n", bySeverity["P1"])
			fmt.Fprintf(&sb, "  P2 — Major:         %d\n", bySeverity["P2"])
			fmt.Fprintf(&sb, "  P3 — Minor:         %d\n", bySeverity["P3"])
			fmt.Fprintf(&sb, "  P4 — Informational: %d\n", bySeverity["P4"])
			sb.WriteString("\nBy Status:\n")
			fmt.Fprintf(&sb, "  open:        %d\n", byStatus["open"])
			fmt.Fprintf(&sb, "  in-progress: %d\n", byStatus["in-progress"])
			fmt.Fprintf(&sb, "  resolved:    %d\n", byStatus["resolved"])
			fmt.Fprintf(&sb, "  closed:      %d\n", byStatus["closed"])

			return &mcp.CallToolResultFor[GetTicketSummaryResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
			}, nil
		},
	}
}
