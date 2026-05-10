package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
)

func init() {
	registerTool(ReadTicket())
}

type ReadTicketParams struct {
	ID string `json:"id" description:"Ticket ID (e.g., AMSS-001)."`
}

type ReadTicketResult struct {
	Text string `json:"text"`
}

func ReadTicket() MCPTool[ReadTicketParams, ReadTicketResult] {
	return MCPTool[ReadTicketParams, ReadTicketResult]{
		Name:        "read_ticket",
		Description: "Read the full details of a ticket by ID, including all comments.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ReadTicketParams]) (*mcp.CallToolResultFor[ReadTicketResult], error) {
			if ticketClient == nil {
				return nil, fmt.Errorf("ticket client not initialized")
			}
			id := params.Arguments.ID
			if id == "" {
				return nil, fmt.Errorf("id is required")
			}

			ticket, err := ticketClient.GetTicket(id)
			if err != nil {
				return nil, fmt.Errorf("ticket not found: %w", err)
			}

			return &mcp.CallToolResultFor[ReadTicketResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: client.FormatTicket(ticket)}},
			}, nil
		},
	}
}
