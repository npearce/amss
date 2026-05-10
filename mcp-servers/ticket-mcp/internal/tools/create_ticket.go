package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
)

func init() {
	registerTool(CreateTicket())
}

type CreateTicketParams struct {
	Title                string   `json:"title" description:"Ticket title (required)."`
	Description          string   `json:"description" description:"Ticket description (required)."`
	Severity             string   `json:"severity" description:"Severity: P1 (critical/safety), P2 (major), P3 (minor), or P4 (informational) (required)."`
	Category             string   `json:"category" description:"Category: life-support, navigation, comms, power, propulsion, eva, medical, operations, thermal, or structures (required)."`
	ReportedBy           string   `json:"reported_by" description:"Reporter's crew ID (required)."`
	Mission              string   `json:"mission" description:"Mission identifier, e.g. artemis-ii (required)."`
	AssignedTo           string   `json:"assigned_to" description:"Assignee crew ID (defaults to ground-control)."`
	KBArticlesReferenced []string `json:"kb_articles_referenced,omitempty" description:"Related KB article IDs."`
}

type CreateTicketResult struct {
	Text string `json:"text"`
}

func CreateTicket() MCPTool[CreateTicketParams, CreateTicketResult] {
	return MCPTool[CreateTicketParams, CreateTicketResult]{
		Name:        "create_ticket",
		Description: "Create a new mission support ticket. Status defaults to open. ID is auto-assigned.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CreateTicketParams]) (*mcp.CallToolResultFor[CreateTicketResult], error) {
			if ticketClient == nil {
				return nil, fmt.Errorf("ticket client not initialized")
			}
			p := params.Arguments
			if p.Title == "" || p.Description == "" || p.Severity == "" || p.Category == "" || p.ReportedBy == "" || p.Mission == "" {
				return nil, fmt.Errorf("title, description, severity, category, reported_by, and mission are required")
			}
			assignedTo := p.AssignedTo
			if assignedTo == "" {
				assignedTo = "ground-control"
			}

			ticket, err := ticketClient.CreateTicket(p.Title, p.Description, p.Severity, p.Category, p.ReportedBy, p.Mission, assignedTo, p.KBArticlesReferenced)
			if err != nil {
				return nil, fmt.Errorf("create failed: %w", err)
			}

			text := fmt.Sprintf("Created ticket %s:\n\n%s", ticket.ID, client.FormatTicket(ticket))
			return &mcp.CallToolResultFor[CreateTicketResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}
