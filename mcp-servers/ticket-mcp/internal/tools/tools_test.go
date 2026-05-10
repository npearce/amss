package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/ticket-mcp/internal/client"
)

type mockTicketClient struct {
	searchFn  func(mission, severity, status, category, search string, limit, offset int) ([]*client.Ticket, int, error)
	getFn     func(id string) (*client.Ticket, error)
	createFn  func(title, description, severity, category, reportedBy, mission, assignedTo string, kbArticles []string) (*client.Ticket, error)
	updateFn  func(id string, updates map[string]interface{}) (*client.Ticket, error)
	commentFn func(ticketID, author, text string) (*client.Comment, error)
}

func (m *mockTicketClient) Search(mission, severity, status, category, search string, limit, offset int) ([]*client.Ticket, int, error) {
	if m.searchFn != nil {
		return m.searchFn(mission, severity, status, category, search, limit, offset)
	}
	return []*client.Ticket{}, 0, nil
}
func (m *mockTicketClient) GetTicket(id string) (*client.Ticket, error) {
	if m.getFn != nil {
		return m.getFn(id)
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockTicketClient) CreateTicket(title, description, severity, category, reportedBy, mission, assignedTo string, kbArticles []string) (*client.Ticket, error) {
	if m.createFn != nil {
		return m.createFn(title, description, severity, category, reportedBy, mission, assignedTo, kbArticles)
	}
	return nil, fmt.Errorf("create not implemented")
}
func (m *mockTicketClient) UpdateTicket(id string, updates map[string]interface{}) (*client.Ticket, error) {
	if m.updateFn != nil {
		return m.updateFn(id, updates)
	}
	return nil, fmt.Errorf("update not implemented")
}
func (m *mockTicketClient) AddComment(ticketID, author, text string) (*client.Comment, error) {
	if m.commentFn != nil {
		return m.commentFn(ticketID, author, text)
	}
	return nil, fmt.Errorf("comment not implemented")
}

func withMock(t *testing.T, mock *mockTicketClient) {
	t.Helper()
	SetClient(mock)
	t.Cleanup(func() { SetClient(nil) })
}

func callText[I, O any](t *testing.T, tool MCPTool[I, O], args I) (string, error) {
	t.Helper()
	result, err := tool.Handler(context.Background(), nil, &mcp.CallToolParamsFor[I]{Arguments: args})
	if err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", nil
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] is not *mcp.TextContent")
	}
	return tc.Text, nil
}

func sampleTicket() *client.Ticket {
	return &client.Ticket{
		ID:          "AMSS-001",
		Title:       "WCS Pressure Fault",
		Description: "Toilet pressure out of nominal range.",
		Severity:    "P2",
		Status:      "open",
		Category:    "life-support",
		Mission:     "artemis-ii",
		ReportedBy:  "wiseman-r",
		AssignedTo:  "gc-eclss",
	}
}

// --- search_tickets ---

func TestSearchTicketsSuccess(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			return []*client.Ticket{sampleTicket()}, 1, nil
		},
	})
	text, err := callText(t, SearchTickets(), SearchTicketsParams{Search: "pressure"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "AMSS-001") {
		t.Errorf("missing AMSS-001 in:\n%s", text)
	}
	if !strings.Contains(text, "WCS Pressure Fault") {
		t.Errorf("missing title in:\n%s", text)
	}
}

func TestSearchTicketsEmpty(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			return []*client.Ticket{}, 0, nil
		},
	})
	text, err := callText(t, SearchTickets(), SearchTicketsParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "No tickets found") {
		t.Errorf("expected 'No tickets found': %s", text)
	}
}

func TestSearchTicketsDefaultLimit(t *testing.T) {
	var capturedLimit int
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, limit, _ int) ([]*client.Ticket, int, error) {
			capturedLimit = limit
			return []*client.Ticket{}, 0, nil
		},
	})
	callText(t, SearchTickets(), SearchTicketsParams{})
	if capturedLimit != 50 {
		t.Errorf("default limit = %d, want 50", capturedLimit)
	}
}

func TestSearchTicketsPassesAllParams(t *testing.T) {
	var capMission, capSeverity, capStatus, capCategory, capSearch string
	withMock(t, &mockTicketClient{
		searchFn: func(mission, severity, status, category, search string, _, _ int) ([]*client.Ticket, int, error) {
			capMission, capSeverity, capStatus, capCategory, capSearch = mission, severity, status, category, search
			return []*client.Ticket{}, 0, nil
		},
	})
	callText(t, SearchTickets(), SearchTicketsParams{
		Mission: "artemis-ii", Severity: "P1", Status: "open", Category: "life-support", Search: "oxygen",
	})
	if capMission != "artemis-ii" {
		t.Errorf("mission = %q", capMission)
	}
	if capSeverity != "P1" {
		t.Errorf("severity = %q", capSeverity)
	}
	if capStatus != "open" {
		t.Errorf("status = %q", capStatus)
	}
	if capCategory != "life-support" {
		t.Errorf("category = %q", capCategory)
	}
	if capSearch != "oxygen" {
		t.Errorf("search = %q", capSearch)
	}
}

func TestSearchTicketsClientError(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			return nil, 0, fmt.Errorf("store unavailable")
		},
	})
	_, err := callText(t, SearchTickets(), SearchTicketsParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchTicketsNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, SearchTickets(), SearchTicketsParams{})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- read_ticket ---

func TestReadTicketSuccess(t *testing.T) {
	withMock(t, &mockTicketClient{
		getFn: func(id string) (*client.Ticket, error) {
			if id != "AMSS-001" {
				t.Errorf("id = %q", id)
			}
			return sampleTicket(), nil
		},
	})
	text, err := callText(t, ReadTicket(), ReadTicketParams{ID: "AMSS-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"AMSS-001", "WCS Pressure Fault", "P2", "life-support"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
}

func TestReadTicketMissingID(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, ReadTicket(), ReadTicketParams{ID: ""})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestReadTicketNotFound(t *testing.T) {
	withMock(t, &mockTicketClient{
		getFn: func(_ string) (*client.Ticket, error) {
			return nil, fmt.Errorf("NOT_FOUND: ticket AMSS-999 not found")
		},
	})
	_, err := callText(t, ReadTicket(), ReadTicketParams{ID: "AMSS-999"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadTicketNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, ReadTicket(), ReadTicketParams{ID: "AMSS-001"})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- create_ticket ---

func TestCreateTicketSuccess(t *testing.T) {
	withMock(t, &mockTicketClient{
		createFn: func(title, _, _, _, _, _, _ string, _ []string) (*client.Ticket, error) {
			return &client.Ticket{ID: "AMSS-016", Title: title, Severity: "P3", Status: "open"}, nil
		},
	})
	text, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "Nav glitch", Description: "Display flickering.", Severity: "P3",
		Category: "navigation", ReportedBy: "wiseman-r", Mission: "artemis-ii",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "AMSS-016") {
		t.Errorf("missing ticket ID:\n%s", text)
	}
	if !strings.Contains(text, "Created ticket") {
		t.Errorf("missing 'Created ticket':\n%s", text)
	}
}

func TestCreateTicketDefaultAssignedTo(t *testing.T) {
	var capturedAssignedTo string
	withMock(t, &mockTicketClient{
		createFn: func(_, _, _, _, _, _, assignedTo string, _ []string) (*client.Ticket, error) {
			capturedAssignedTo = assignedTo
			return &client.Ticket{ID: "AMSS-016"}, nil
		},
	})
	callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Severity: "P3",
		Category: "navigation", ReportedBy: "wiseman-r", Mission: "artemis-ii",
	})
	if capturedAssignedTo != "ground-control" {
		t.Errorf("assigned_to = %q, want ground-control", capturedAssignedTo)
	}
}

func TestCreateTicketMissingTitle(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Description: "D", Severity: "P3", Category: "navigation", ReportedBy: "wiseman-r", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestCreateTicketMissingDescription(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Severity: "P3", Category: "navigation", ReportedBy: "wiseman-r", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestCreateTicketMissingSeverity(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Category: "navigation", ReportedBy: "wiseman-r", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error for missing severity")
	}
}

func TestCreateTicketMissingCategory(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Severity: "P3", ReportedBy: "wiseman-r", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error for missing category")
	}
}

func TestCreateTicketMissingReportedBy(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Severity: "P3", Category: "navigation", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error for missing reported_by")
	}
}

func TestCreateTicketMissingMission(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Severity: "P3", Category: "navigation", ReportedBy: "wiseman-r",
	})
	if err == nil {
		t.Fatal("expected error for missing mission")
	}
}

func TestCreateTicketClientError(t *testing.T) {
	withMock(t, &mockTicketClient{
		createFn: func(_, _, _, _, _, _, _ string, _ []string) (*client.Ticket, error) {
			return nil, fmt.Errorf("BAD_REQUEST: invalid severity")
		},
	})
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Severity: "XX", Category: "navigation", ReportedBy: "user", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateTicketNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, CreateTicket(), CreateTicketParams{
		Title: "T", Description: "D", Severity: "P3", Category: "navigation", ReportedBy: "user", Mission: "artemis-ii",
	})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- update_ticket ---

func TestUpdateTicketSuccess(t *testing.T) {
	withMock(t, &mockTicketClient{
		updateFn: func(id string, _ map[string]interface{}) (*client.Ticket, error) {
			return &client.Ticket{ID: id, Status: "resolved"}, nil
		},
	})
	status := "resolved"
	text, err := callText(t, UpdateTicket(), UpdateTicketParams{ID: "AMSS-001", Status: &status})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Updated ticket AMSS-001") {
		t.Errorf("result:\n%s", text)
	}
}

func TestUpdateTicketMissingID(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, UpdateTicket(), UpdateTicketParams{})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestUpdateTicketNoFields(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, UpdateTicket(), UpdateTicketParams{ID: "AMSS-001"})
	if err == nil {
		t.Fatal("expected error when no fields to update")
	}
}

func TestUpdateTicketOnlySuppliedFields(t *testing.T) {
	var capturedUpdates map[string]interface{}
	withMock(t, &mockTicketClient{
		updateFn: func(_ string, updates map[string]interface{}) (*client.Ticket, error) {
			capturedUpdates = updates
			return &client.Ticket{ID: "AMSS-001"}, nil
		},
	})
	status := "in-progress"
	callText(t, UpdateTicket(), UpdateTicketParams{ID: "AMSS-001", Status: &status})
	if capturedUpdates["status"] != "in-progress" {
		t.Errorf("status = %v", capturedUpdates["status"])
	}
	if _, ok := capturedUpdates["severity"]; ok {
		t.Error("severity should not be in updates when not supplied")
	}
}

func TestUpdateTicketClientError(t *testing.T) {
	withMock(t, &mockTicketClient{
		updateFn: func(_ string, _ map[string]interface{}) (*client.Ticket, error) {
			return nil, fmt.Errorf("NOT_FOUND: ticket AMSS-999 not found")
		},
	})
	status := "closed"
	_, err := callText(t, UpdateTicket(), UpdateTicketParams{ID: "AMSS-999", Status: &status})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateTicketNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	status := "closed"
	_, err := callText(t, UpdateTicket(), UpdateTicketParams{ID: "AMSS-001", Status: &status})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- add_ticket_comment ---

func TestAddTicketCommentSuccess(t *testing.T) {
	withMock(t, &mockTicketClient{
		commentFn: func(ticketID, author, text string) (*client.Comment, error) {
			return &client.Comment{
				Author:    author,
				Timestamp: "2026-05-09T10:00:00Z",
				Text:      text,
			}, nil
		},
	})
	text, err := callText(t, AddTicketComment(), AddTicketCommentParams{
		TicketID: "AMSS-001", Author: "gc-eclss", Text: "Investigating.",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Comment added to AMSS-001") {
		t.Errorf("result:\n%s", text)
	}
	if !strings.Contains(text, "gc-eclss") {
		t.Errorf("missing author:\n%s", text)
	}
	if !strings.Contains(text, "Investigating.") {
		t.Errorf("missing text:\n%s", text)
	}
}

func TestAddTicketCommentMissingTicketID(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, AddTicketComment(), AddTicketCommentParams{Author: "gc-eclss", Text: "note"})
	if err == nil {
		t.Fatal("expected error for missing ticket_id")
	}
}

func TestAddTicketCommentMissingAuthor(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, AddTicketComment(), AddTicketCommentParams{TicketID: "AMSS-001", Text: "note"})
	if err == nil {
		t.Fatal("expected error for missing author")
	}
}

func TestAddTicketCommentMissingText(t *testing.T) {
	withMock(t, &mockTicketClient{})
	_, err := callText(t, AddTicketComment(), AddTicketCommentParams{TicketID: "AMSS-001", Author: "gc-eclss"})
	if err == nil {
		t.Fatal("expected error for missing text")
	}
}

func TestAddTicketCommentClientError(t *testing.T) {
	withMock(t, &mockTicketClient{
		commentFn: func(_, _, _ string) (*client.Comment, error) {
			return nil, fmt.Errorf("NOT_FOUND: ticket AMSS-999 not found")
		},
	})
	_, err := callText(t, AddTicketComment(), AddTicketCommentParams{
		TicketID: "AMSS-999", Author: "gc-flight", Text: "note",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddTicketCommentNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, AddTicketComment(), AddTicketCommentParams{
		TicketID: "AMSS-001", Author: "gc-flight", Text: "note",
	})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- get_ticket_summary ---

func TestGetTicketSummarySuccess(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			return []*client.Ticket{
				{ID: "AMSS-001", Severity: "P1", Status: "open"},
				{ID: "AMSS-002", Severity: "P2", Status: "in-progress"},
				{ID: "AMSS-003", Severity: "P2", Status: "resolved"},
				{ID: "AMSS-004", Severity: "P3", Status: "open"},
				{ID: "AMSS-005", Severity: "P3", Status: "closed"},
			}, 5, nil
		},
	})
	text, err := callText(t, GetTicketSummary(), GetTicketSummaryParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "5 total") {
		t.Errorf("missing total count:\n%s", text)
	}
	for _, want := range []string{"P1", "P2", "P3", "P4", "open", "in-progress", "resolved", "closed"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in summary:\n%s", want, text)
		}
	}
}

func TestGetTicketSummaryEmpty(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			return []*client.Ticket{}, 0, nil
		},
	})
	text, err := callText(t, GetTicketSummary(), GetTicketSummaryParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "No tickets found") {
		t.Errorf("expected 'No tickets found':\n%s", text)
	}
}

func TestGetTicketSummaryClientError(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			return nil, 0, fmt.Errorf("store unreachable")
		},
	})
	_, err := callText(t, GetTicketSummary(), GetTicketSummaryParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetTicketSummaryNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, GetTicketSummary(), GetTicketSummaryParams{})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

func TestGetTicketSummaryCounts(t *testing.T) {
	withMock(t, &mockTicketClient{
		searchFn: func(_, _, _, _, _ string, _, _ int) ([]*client.Ticket, int, error) {
			tickets := []*client.Ticket{
				{Severity: "P1", Status: "open"},
				{Severity: "P1", Status: "open"},
				{Severity: "P2", Status: "in-progress"},
				{Severity: "P3", Status: "open"},
				{Severity: "P3", Status: "resolved"},
				{Severity: "P4", Status: "closed"},
			}
			return tickets, len(tickets), nil
		},
	})
	text, err := callText(t, GetTicketSummary(), GetTicketSummaryParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// P1 count should be 2
	if !strings.Contains(text, "P1") {
		t.Errorf("missing P1:\n%s", text)
	}
	// Should show 6 total
	if !strings.Contains(text, "6 total") {
		t.Errorf("missing 6 total:\n%s", text)
	}
}
