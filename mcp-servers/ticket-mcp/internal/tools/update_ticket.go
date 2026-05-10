package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
)

func init() {
	registerTool(UpdateTicket())
}

type UpdateTicketParams struct {
	ID                   string   `json:"id" description:"Ticket ID to update (required)."`
	Status               *string  `json:"status,omitempty" description:"New status: open, in-progress, resolved, or closed."`
	Severity             *string  `json:"severity,omitempty" description:"New severity: P1, P2, P3, or P4."`
	AssignedTo           *string  `json:"assigned_to,omitempty" description:"New assignee crew ID."`
	Resolution           *string  `json:"resolution,omitempty" description:"Resolution text (set when resolving a ticket)."`
	KBArticlesReferenced []string `json:"kb_articles_referenced,omitempty" description:"Updated KB article references."`
	Title                *string  `json:"title,omitempty" description:"New title."`
	Description          *string  `json:"description,omitempty" description:"New description."`
	Category             *string  `json:"category,omitempty" description:"New category."`
	Mission              *string  `json:"mission,omitempty" description:"New mission."`
}

type UpdateTicketResult struct {
	Text string `json:"text"`
}

func UpdateTicket() MCPTool[UpdateTicketParams, UpdateTicketResult] {
	return MCPTool[UpdateTicketParams, UpdateTicketResult]{
		Name:        "update_ticket",
		Description: "Partially update a ticket. Only supplied fields are changed. Setting status to resolved automatically sets resolved_at.",
		Handler: func(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[UpdateTicketParams]) (*mcp.CallToolResultFor[UpdateTicketResult], error) {
			if ticketClient == nil {
				return nil, fmt.Errorf("ticket client not initialized")
			}
			p := params.Arguments
			if p.ID == "" {
				return nil, fmt.Errorf("id is required")
			}

			updates := make(map[string]interface{})
			if p.Status != nil {
				updates["status"] = *p.Status
			}
			if p.Severity != nil {
				updates["severity"] = *p.Severity
			}
			if p.AssignedTo != nil {
				updates["assigned_to"] = *p.AssignedTo
			}
			if p.Resolution != nil {
				updates["resolution"] = *p.Resolution
			}
			if p.KBArticlesReferenced != nil {
				updates["kb_articles_referenced"] = p.KBArticlesReferenced
			}
			if p.Title != nil {
				updates["title"] = *p.Title
			}
			if p.Description != nil {
				updates["description"] = *p.Description
			}
			if p.Category != nil {
				updates["category"] = *p.Category
			}
			if p.Mission != nil {
				updates["mission"] = *p.Mission
			}

			if len(updates) == 0 {
				return nil, fmt.Errorf("no fields to update")
			}

			ticket, err := ticketClient.UpdateTicket(p.ID, updates)
			if err != nil {
				return nil, fmt.Errorf("update failed: %w", err)
			}

			text := fmt.Sprintf("Updated ticket %s:\n\n%s", ticket.ID, client.FormatTicket(ticket))
			return &mcp.CallToolResultFor[UpdateTicketResult]{
				Content: []mcp.Content{&mcp.TextContent{Text: text}},
			}, nil
		},
	}
}
