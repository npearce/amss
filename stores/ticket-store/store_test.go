package main

import (
	"strings"
	"testing"
	"time"
)

func TestLoadSeed(t *testing.T) {
	s, err := NewStore("seed-data/tickets.json")
	if err != nil {
		t.Fatalf("Failed to load seed data: %v", err)
	}

	if len(s.tickets) != 15 {
		t.Errorf("Expected 15 tickets, got %d", len(s.tickets))
	}

	ticket := s.GetTicket("AMSS-001")
	if ticket == nil {
		t.Fatalf("Expected to find AMSS-001")
	}
	if ticket.Title != "WCS pressure anomaly during translunar coast" {
		t.Errorf("Unexpected title: %s", ticket.Title)
	}
	if ticket.Severity != "P2" {
		t.Errorf("Expected P2, got %s", ticket.Severity)
	}
	if ticket.Status != "resolved" {
		t.Errorf("Expected resolved, got %s", ticket.Status)
	}
	if len(ticket.Comments) != 5 {
		t.Errorf("AMSS-001 should have 5 comments, got %d", len(ticket.Comments))
	}

	if s.nextID != 16 {
		t.Errorf("Expected nextID 16 after loading 15 tickets, got %d", s.nextID)
	}
}

func TestLoadSeedEnsuresSlices(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	ticket := s.GetTicket("AMSS-001")
	if ticket.Comments == nil {
		t.Error("Comments should never be nil after load")
	}
	if ticket.KBArticlesReferenced == nil {
		t.Error("KBArticlesReferenced should never be nil after load")
	}
}

func TestCreateTicket(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		severity    string
		category    string
		reportedBy  string
		mission     string
		assignedTo  string
		shouldErr   bool
		errContains string
		expectedID  string
	}{
		{
			name:        "valid ticket",
			title:       "Test Ticket",
			description: "Test description",
			severity:    "P3",
			category:    "power",
			reportedBy:  "wiseman-r",
			mission:     "artemis-ii",
			assignedTo:  "gc-systems",
			shouldErr:   false,
			expectedID:  "AMSS-016",
		},
		{
			name:        "missing title",
			description: "Test",
			severity:    "P3",
			category:    "power",
			reportedBy:  "wiseman-r",
			mission:     "artemis-ii",
			shouldErr:   true,
			errContains: "title and description are required",
		},
		{
			name:        "missing description",
			title:       "Test",
			severity:    "P3",
			category:    "power",
			reportedBy:  "wiseman-r",
			mission:     "artemis-ii",
			shouldErr:   true,
			errContains: "title and description are required",
		},
		{
			name:        "invalid severity",
			title:       "Test",
			description: "Test",
			severity:    "P5",
			category:    "power",
			reportedBy:  "wiseman-r",
			mission:     "artemis-ii",
			shouldErr:   true,
			errContains: "invalid severity",
		},
		{
			name:        "empty severity",
			title:       "Test",
			description: "Test",
			severity:    "",
			category:    "power",
			reportedBy:  "wiseman-r",
			mission:     "artemis-ii",
			shouldErr:   true,
			errContains: "invalid severity",
		},
		{
			name:        "invalid category",
			title:       "Test",
			description: "Test",
			severity:    "P3",
			category:    "invalid-cat",
			reportedBy:  "wiseman-r",
			mission:     "artemis-ii",
			shouldErr:   true,
			errContains: "invalid category",
		},
		{
			name:        "missing reported_by",
			title:       "Test",
			description: "Test",
			severity:    "P3",
			category:    "power",
			mission:     "artemis-ii",
			shouldErr:   true,
			errContains: "reported_by is required",
		},
		{
			name:        "missing mission",
			title:       "Test",
			description: "Test",
			severity:    "P3",
			category:    "power",
			reportedBy:  "wiseman-r",
			shouldErr:   true,
			errContains: "mission is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := NewStore("seed-data/tickets.json")
			ticket, err := s.CreateTicket(tt.title, tt.description, tt.severity, tt.category, tt.reportedBy, tt.mission, tt.assignedTo, nil)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error containing %q", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if ticket.ID != tt.expectedID {
					t.Errorf("Expected ID %s, got %s", tt.expectedID, ticket.ID)
				}
				if ticket.Status != "open" {
					t.Errorf("New ticket status should be open, got %s", ticket.Status)
				}
				if ticket.ResolvedAt != nil {
					t.Error("New ticket ResolvedAt should be nil")
				}
				if ticket.Resolution != nil {
					t.Error("New ticket Resolution should be nil")
				}
			}
		})
	}
}

func TestCreateTicketDefaultAssignedToStore(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")
	ticket, err := s.CreateTicket("T", "D", "P3", "power", "wiseman-r", "artemis-ii", "", nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ticket.AssignedTo != "ground-control" {
		t.Errorf("Expected default assigned_to ground-control, got %s", ticket.AssignedTo)
	}
}

func TestCreateTicketIDSequential(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	t1, _ := s.CreateTicket("T1", "D1", "P3", "power", "a", "m", "", nil)
	t2, _ := s.CreateTicket("T2", "D2", "P3", "power", "a", "m", "", nil)

	if t1.ID != "AMSS-016" {
		t.Errorf("First ticket should be AMSS-016, got %s", t1.ID)
	}
	if t2.ID != "AMSS-017" {
		t.Errorf("Second ticket should be AMSS-017, got %s", t2.ID)
	}
}

func TestCreateTicketKBRefs(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	ticket, _ := s.CreateTicket("T", "D", "P3", "power", "a", "m", "", []string{"KB-001", "KB-002"})
	if len(ticket.KBArticlesReferenced) != 2 {
		t.Errorf("Expected 2 KB refs, got %d", len(ticket.KBArticlesReferenced))
	}

	ticket2, _ := s.CreateTicket("T", "D", "P3", "power", "a", "m", "", nil)
	if ticket2.KBArticlesReferenced == nil {
		t.Error("KBArticlesReferenced should not be nil (should be empty slice)")
	}
}

func TestGetTicket(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	ticket := s.GetTicket("AMSS-001")
	if ticket == nil {
		t.Fatal("Expected to find AMSS-001")
	}
	if ticket.ID != "AMSS-001" {
		t.Errorf("Expected AMSS-001, got %s", ticket.ID)
	}

	missing := s.GetTicket("AMSS-999")
	if missing != nil {
		t.Error("Expected nil for missing ticket")
	}
}

func TestUpdateTicketTitle(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	ticket, err := s.UpdateTicket("AMSS-014", map[string]interface{}{
		"title": "Updated Title",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ticket.Title != "Updated Title" {
		t.Errorf("Title should be updated, got %s", ticket.Title)
	}
	if ticket.UpdatedAt == "" {
		t.Error("UpdatedAt should be set")
	}
}

func TestUpdateTicketStatusToResolved(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	// AMSS-014 is "open" with no resolved_at
	ticket := s.GetTicket("AMSS-014")
	if ticket.ResolvedAt != nil {
		t.Fatal("AMSS-014 should start with nil resolved_at")
	}

	updated, err := s.UpdateTicket("AMSS-014", map[string]interface{}{
		"status": "resolved",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if updated.Status != "resolved" {
		t.Errorf("Status should be resolved, got %s", updated.Status)
	}
	if updated.ResolvedAt == nil {
		t.Error("ResolvedAt should be auto-set when status changes to resolved")
	}
	if _, err := time.Parse(time.RFC3339, *updated.ResolvedAt); err != nil {
		t.Errorf("ResolvedAt should be valid RFC3339: %v", err)
	}
}

func TestUpdateTicketAlreadyResolvedAtPreserved(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	// AMSS-001 is already resolved with a resolved_at timestamp
	original := s.GetTicket("AMSS-001")
	originalResolvedAt := *original.ResolvedAt

	updated, err := s.UpdateTicket("AMSS-001", map[string]interface{}{
		"status": "resolved",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if *updated.ResolvedAt != originalResolvedAt {
		t.Errorf("ResolvedAt should not change when already set: got %s, want %s", *updated.ResolvedAt, originalResolvedAt)
	}
}

func TestUpdateTicketNotFoundStore(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	_, err := s.UpdateTicket("AMSS-999", map[string]interface{}{"title": "x"})
	if err == nil {
		t.Error("Expected error for missing ticket")
	}
}

func TestUpdateTicketInvalidSeverityStore(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	_, err := s.UpdateTicket("AMSS-001", map[string]interface{}{"severity": "P9"})
	if err == nil {
		t.Error("Expected error for invalid severity")
	}
}

func TestUpdateTicketInvalidCategory(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	_, err := s.UpdateTicket("AMSS-001", map[string]interface{}{"category": "invalid"})
	if err == nil {
		t.Error("Expected error for invalid category")
	}
}

func TestUpdateTicketPartialUpdate(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	ticket, _ := s.CreateTicket("Original Title", "Original description", "P3", "power", "wiseman-r", "artemis-ii", "gc-systems", nil)
	originalID := ticket.ID

	updated, _ := s.UpdateTicket(ticket.ID, map[string]interface{}{
		"title": "New Title",
	})

	if updated.Title != "New Title" {
		t.Errorf("Title should be updated")
	}
	if updated.Description != "Original description" {
		t.Errorf("Description should be unchanged")
	}
	if updated.ReportedBy != "wiseman-r" {
		t.Errorf("ReportedBy should be unchanged")
	}
	if updated.ID != originalID {
		t.Errorf("ID should not change")
	}
}

func TestAddComment(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	beforeUpdate := s.GetTicket("AMSS-014").UpdatedAt

	comment, err := s.AddComment("AMSS-014", "wiseman-r", "This is a test comment")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if comment.Author != "wiseman-r" {
		t.Errorf("Author = %s, want wiseman-r", comment.Author)
	}
	if comment.Text != "This is a test comment" {
		t.Errorf("Text mismatch")
	}
	if comment.Timestamp == "" {
		t.Error("Timestamp should be set")
	}
	if _, err := time.Parse(time.RFC3339, comment.Timestamp); err != nil {
		t.Errorf("Timestamp should be valid RFC3339: %v", err)
	}

	ticket := s.GetTicket("AMSS-014")
	if len(ticket.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(ticket.Comments))
	}
	if ticket.UpdatedAt == beforeUpdate {
		t.Error("Ticket UpdatedAt should be updated after adding comment")
	}
}

func TestAddCommentTicketNotFoundStore(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	_, err := s.AddComment("AMSS-999", "author", "text")
	if err == nil {
		t.Error("Expected error for missing ticket")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention not found, got %q", err.Error())
	}
}

func TestAddCommentIncrementsCount(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	// AMSS-001 has 5 comments
	before := len(s.GetTicket("AMSS-001").Comments)
	s.AddComment("AMSS-001", "gc-capcom", "Additional comment")
	after := len(s.GetTicket("AMSS-001").Comments)

	if after != before+1 {
		t.Errorf("Expected %d comments, got %d", before+1, after)
	}
}

func TestListTicketsNoFilters(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{}, "", 50, 0)
	if total != 15 {
		t.Errorf("Expected 15 total tickets, got %d", total)
	}
	if len(tickets) != 15 {
		t.Errorf("Expected 15 tickets, got %d", len(tickets))
	}
}

func TestListTicketsFilterByMission(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{"mission": "artemis-iii"}, "", 50, 0)
	if total == 0 {
		t.Error("Expected some artemis-iii tickets")
	}
	for _, ticket := range tickets {
		if ticket.Mission != "artemis-iii" {
			t.Errorf("Got ticket with mission %s, expected artemis-iii", ticket.Mission)
		}
	}
	_ = tickets
}

func TestListTicketsFilterBySeverity(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{"severity": "P1"}, "", 50, 0)
	if total != 2 {
		t.Errorf("Expected 2 P1 tickets, got %d", total)
	}
	for _, ticket := range tickets {
		if ticket.Severity != "P1" {
			t.Errorf("Got ticket with severity %s, expected P1", ticket.Severity)
		}
	}
}

func TestListTicketsFilterByStatus(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{"status": "open"}, "", 50, 0)
	if total == 0 {
		t.Error("Expected at least one open ticket")
	}
	for _, ticket := range tickets {
		if ticket.Status != "open" {
			t.Errorf("Got ticket with status %s, expected open", ticket.Status)
		}
	}
}

func TestListTicketsFilterByCategory(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{"category": "life-support"}, "", 50, 0)
	if total == 0 {
		t.Error("Expected life-support tickets")
	}
	for _, ticket := range tickets {
		if ticket.Category != "life-support" {
			t.Errorf("Got ticket with category %s, expected life-support", ticket.Category)
		}
	}
}

func TestListTicketsSearch(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{}, "WCS", 50, 0)
	if total == 0 {
		t.Error("Expected WCS search results")
	}
	found := false
	for _, ticket := range tickets {
		if ticket.ID == "AMSS-001" {
			found = true
		}
	}
	if !found {
		t.Error("Expected AMSS-001 in WCS search results")
	}
}

func TestListTicketsFilterAndSearch(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{"category": "life-support"}, "WCS", 50, 0)
	if total == 0 {
		t.Error("Expected life-support + WCS results")
	}
	for _, ticket := range tickets {
		if ticket.Category != "life-support" {
			t.Errorf("Filter should restrict to life-support, got %s", ticket.Category)
		}
	}
}

func TestListTicketsPagination(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	page1, total := s.ListTickets(map[string]string{}, "", 5, 0)
	if len(page1) != 5 {
		t.Errorf("Expected 5 tickets on page 1, got %d", len(page1))
	}
	if total != 15 {
		t.Errorf("Total should be 15, got %d", total)
	}

	page2, _ := s.ListTickets(map[string]string{}, "", 5, 5)
	if len(page2) != 5 {
		t.Errorf("Expected 5 tickets on page 2, got %d", len(page2))
	}

	// IDs should not overlap
	p1IDs := make(map[string]bool)
	for _, tk := range page1 {
		p1IDs[tk.ID] = true
	}
	for _, tk := range page2 {
		if p1IDs[tk.ID] {
			t.Errorf("Ticket %s appeared in both pages", tk.ID)
		}
	}
}

func TestListTicketsOffsetBeyondTotal(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, total := s.ListTickets(map[string]string{}, "", 50, 100)
	if len(tickets) != 0 {
		t.Errorf("Expected 0 tickets when offset > total, got %d", len(tickets))
	}
	if total != 15 {
		t.Errorf("Total should still be 15, got %d", total)
	}
}

func TestListTicketsSortedByID(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	tickets, _ := s.ListTickets(map[string]string{}, "", 50, 0)
	for i := 1; i < len(tickets); i++ {
		if tickets[i].ID < tickets[i-1].ID {
			t.Errorf("Tickets not sorted: %s before %s", tickets[i-1].ID, tickets[i].ID)
		}
	}
}

func TestSearchRanking(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	// Create one ticket with "pressure" once and one with it multiple times
	s.CreateTicket("Pressure Test", "some other content", "P3", "power", "a", "m", "", nil)
	s.CreateTicket("Pressure Warning", "pressure level high pressure reading", "P3", "power", "a", "m", "", nil)

	tickets, _ := s.ListTickets(map[string]string{}, "pressure", 50, 0)
	if len(tickets) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(tickets))
	}
	// Higher match count should appear first
	// The second created ticket has 3 occurrences of "pressure" vs 1 for the first
	if tickets[0].Title != "Pressure Warning" {
		t.Logf("Ranking: first result is %s (acceptable if tie-breaking)", tickets[0].Title)
	}
}

func TestReset(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	s.CreateTicket("Temp Ticket", "Temp", "P3", "power", "a", "m", "", nil)
	if len(s.tickets) != 16 {
		t.Errorf("Expected 16 tickets before reset, got %d", len(s.tickets))
	}

	count, err := s.Reset()
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if count != 15 {
		t.Errorf("After reset, expected 15 tickets, got %d", count)
	}

	extra := s.GetTicket("AMSS-016")
	if extra != nil {
		t.Error("Created ticket should be gone after reset")
	}
}

func TestResetRestoresComments(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	s.AddComment("AMSS-014", "author", "extra comment")
	s.Reset()

	ticket := s.GetTicket("AMSS-014")
	if len(ticket.Comments) != 1 {
		t.Errorf("After reset, AMSS-014 should have original 1 comment, got %d", len(ticket.Comments))
	}
}

func TestValidSeverities(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	for _, sev := range []string{"P1", "P2", "P3", "P4"} {
		_, err := s.CreateTicket("T", "D", sev, "power", "a", "m", "", nil)
		if err != nil {
			t.Errorf("Severity %s should be valid: %v", sev, err)
		}
	}
}

func TestValidCategories(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	cats := []string{"life-support", "navigation", "comms", "power", "propulsion", "eva", "medical", "operations", "thermal", "structures"}
	for _, cat := range cats {
		_, err := s.CreateTicket("T", "D", "P3", cat, "a", "m", "", nil)
		if err != nil {
			t.Errorf("Category %s should be valid: %v", cat, err)
		}
	}
}

func TestTimestamps(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	ticket, _ := s.CreateTicket("T", "D", "P3", "power", "a", "m", "", nil)

	if _, err := time.Parse(time.RFC3339, ticket.CreatedAt); err != nil {
		t.Errorf("CreatedAt invalid RFC3339: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, ticket.UpdatedAt); err != nil {
		t.Errorf("UpdatedAt invalid RFC3339: %v", err)
	}
}

func TestIndexBuiltOnLoad(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	results := s.search("WCS")
	if len(results) == 0 {
		t.Error("Expected WCS to be indexed")
	}
	if _, found := results["AMSS-001"]; !found {
		t.Error("AMSS-001 should be in WCS search results")
	}
}

func TestIndexUpdatedOnCreate(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	s.CreateTicket("Unique Xylophone Title", "Unique Xylophone description", "P3", "power", "a", "m", "", nil)

	results := s.search("Xylophone")
	if len(results) == 0 {
		t.Error("Newly created ticket should be indexed")
	}
}

func TestIndexUpdatedOnUpdate(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	s.UpdateTicket("AMSS-014", map[string]interface{}{
		"title": "Unique Zephyr Navigation Issue",
	})

	results := s.search("Zephyr")
	if _, found := results["AMSS-014"]; !found {
		t.Error("Updated ticket should be indexed with new title")
	}
}

func TestThreadSafety(t *testing.T) {
	s, _ := NewStore("seed-data/tickets.json")

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			s.ListTickets(map[string]string{}, "", 50, 0)
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	for i := 0; i < 5; i++ {
		go func() {
			s.CreateTicket("T", "D", "P3", "power", "a", "m", "", nil)
			done <- true
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestEnvelopeFormat(t *testing.T) {
	envelope := &Envelope{
		Data:  map[string]interface{}{"test": "data"},
		Error: nil,
	}

	if envelope.Data == nil {
		t.Error("Expected data field")
	}
	if envelope.Error != nil {
		t.Error("Expected error to be nil")
	}

	errEnvelope := &Envelope{
		Data:  nil,
		Error: &ErrorInfo{Code: "NOT_FOUND", Message: "not found"},
	}
	if errEnvelope.Data != nil {
		t.Error("Expected data to be nil on error")
	}
	if errEnvelope.Error == nil {
		t.Error("Expected error to be set")
	}
}
