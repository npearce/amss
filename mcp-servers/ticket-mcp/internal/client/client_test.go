package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ticketEnvelope(t *Ticket) map[string]interface{} {
	return map[string]interface{}{"data": t, "error": nil}
}

func commentEnvelope(c *Comment) map[string]interface{} {
	return map[string]interface{}{"data": c, "error": nil}
}

func listEnvelope(tickets []*Ticket, total int) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"tickets": tickets,
			"total":   total,
			"limit":   50,
			"offset":  0,
		},
		"error": nil,
	}
}

func errorEnvelope(code, msg string) map[string]interface{} {
	return map[string]interface{}{
		"data":  nil,
		"error": map[string]interface{}{"code": code, "message": msg},
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func sampleTicket() *Ticket {
	return &Ticket{
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

// --- New ---

func TestNew(t *testing.T) {
	c := New("http://ticket-store:8082")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "http://ticket-store:8082" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

// --- Search ---

func TestSearchReturnsTickets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/tickets" {
			t.Errorf("path = %s, want /tickets", r.URL.Path)
		}
		writeJSON(w, listEnvelope([]*Ticket{sampleTicket()}, 1))
	}))
	defer ts.Close()

	c := New(ts.URL)
	tickets, total, err := c.Search("", "", "", "", "pressure", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(tickets) != 1 || tickets[0].ID != "AMSS-001" {
		t.Errorf("unexpected tickets: %+v", tickets)
	}
}

func TestSearchPassesQueryParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("mission") != "artemis-ii" {
			t.Errorf("mission = %q", q.Get("mission"))
		}
		if q.Get("severity") != "P1" {
			t.Errorf("severity = %q", q.Get("severity"))
		}
		if q.Get("status") != "open" {
			t.Errorf("status = %q", q.Get("status"))
		}
		if q.Get("category") != "life-support" {
			t.Errorf("category = %q", q.Get("category"))
		}
		if q.Get("search") != "oxygen" {
			t.Errorf("search = %q", q.Get("search"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit = %q", q.Get("limit"))
		}
		writeJSON(w, listEnvelope([]*Ticket{}, 0))
	}))
	defer ts.Close()

	New(ts.URL).Search("artemis-ii", "P1", "open", "life-support", "oxygen", 10, 0)
}

func TestSearchOffsetParam(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "5" {
			t.Errorf("offset = %q", r.URL.Query().Get("offset"))
		}
		writeJSON(w, listEnvelope([]*Ticket{}, 0))
	}))
	defer ts.Close()
	New(ts.URL).Search("", "", "", "", "", 50, 5)
}

func TestSearchAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, errorEnvelope("BAD_REQUEST", "invalid severity"))
	}))
	defer ts.Close()

	_, _, err := New(ts.URL).Search("", "XX", "", "", "", 50, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid severity") {
		t.Errorf("error = %v", err)
	}
}

func TestSearchEmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, listEnvelope([]*Ticket{}, 0))
	}))
	defer ts.Close()

	tickets, total, err := New(ts.URL).Search("", "", "", "", "nothing", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(tickets) != 0 {
		t.Errorf("expected empty, got total=%d tickets=%v", total, tickets)
	}
}

func TestSearchNetworkError(t *testing.T) {
	_, _, err := New("http://127.0.0.1:1").Search("", "", "", "", "", 50, 0)
	if err == nil {
		t.Fatal("expected network error")
	}
}

// --- GetTicket ---

func TestGetTicketSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tickets/AMSS-001" {
			t.Errorf("path = %s", r.URL.Path)
		}
		writeJSON(w, ticketEnvelope(sampleTicket()))
	}))
	defer ts.Close()

	ticket, err := New(ts.URL).GetTicket("AMSS-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != "AMSS-001" {
		t.Errorf("id = %q", ticket.ID)
	}
	if ticket.Title != "WCS Pressure Fault" {
		t.Errorf("title = %q", ticket.Title)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, errorEnvelope("NOT_FOUND", "ticket AMSS-999 not found"))
	}))
	defer ts.Close()

	_, err := New(ts.URL).GetTicket("AMSS-999")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v", err)
	}
}

func TestGetTicketNetworkError(t *testing.T) {
	_, err := New("http://127.0.0.1:1").GetTicket("AMSS-001")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestGetTicketIncludesComments(t *testing.T) {
	ticket := sampleTicket()
	ticket.Comments = []Comment{
		{Author: "gc-eclss", Timestamp: "2026-01-01T10:00:00Z", Text: "Investigating."},
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, ticketEnvelope(ticket))
	}))
	defer ts.Close()

	result, err := New(ts.URL).GetTicket("AMSS-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Comments) != 1 {
		t.Errorf("comments len = %d, want 1", len(result.Comments))
	}
}

// --- CreateTicket ---

func TestCreateTicketSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/tickets" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "New Fault" {
			t.Errorf("title = %v", body["title"])
		}
		if body["severity"] != "P3" {
			t.Errorf("severity = %v", body["severity"])
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, ticketEnvelope(&Ticket{ID: "AMSS-016", Title: "New Fault", Severity: "P3"}))
	}))
	defer ts.Close()

	ticket, err := New(ts.URL).CreateTicket("New Fault", "Desc", "P3", "navigation", "wiseman-r", "artemis-ii", "gc-gnc", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.ID != "AMSS-016" {
		t.Errorf("id = %q", ticket.ID)
	}
}

func TestCreateTicketStoreError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, errorEnvelope("BAD_REQUEST", "missing required fields"))
	}))
	defer ts.Close()

	_, err := New(ts.URL).CreateTicket("", "", "P1", "navigation", "wiseman-r", "artemis-ii", "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateTicketNilKBArticlesBecomesEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		refs, ok := body["kb_articles_referenced"]
		if !ok {
			t.Error("kb_articles_referenced missing from body")
		}
		if refs == nil {
			t.Error("kb_articles_referenced should not be null")
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, ticketEnvelope(&Ticket{ID: "AMSS-016"}))
	}))
	defer ts.Close()

	New(ts.URL).CreateTicket("T", "D", "P3", "navigation", "user", "artemis-ii", "gc", nil)
}

// --- UpdateTicket ---

func TestUpdateTicketSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/tickets/AMSS-001" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "resolved" {
			t.Errorf("status = %v", body["status"])
		}
		writeJSON(w, ticketEnvelope(&Ticket{ID: "AMSS-001", Status: "resolved"}))
	}))
	defer ts.Close()

	ticket, err := New(ts.URL).UpdateTicket("AMSS-001", map[string]interface{}{"status": "resolved"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ticket.Status != "resolved" {
		t.Errorf("status = %q", ticket.Status)
	}
}

func TestUpdateTicketNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, errorEnvelope("NOT_FOUND", "ticket AMSS-999 not found"))
	}))
	defer ts.Close()

	_, err := New(ts.URL).UpdateTicket("AMSS-999", map[string]interface{}{"status": "closed"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- AddComment ---

func TestAddCommentSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/tickets/AMSS-001/comments" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["author"] != "gc-eclss" {
			t.Errorf("author = %v", body["author"])
		}
		if body["text"] != "Investigating now." {
			t.Errorf("text = %v", body["text"])
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, commentEnvelope(&Comment{
			Author:    "gc-eclss",
			Timestamp: "2026-05-09T10:00:00Z",
			Text:      "Investigating now.",
		}))
	}))
	defer ts.Close()

	comment, err := New(ts.URL).AddComment("AMSS-001", "gc-eclss", "Investigating now.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comment.Author != "gc-eclss" {
		t.Errorf("author = %q", comment.Author)
	}
	if comment.Text != "Investigating now." {
		t.Errorf("text = %q", comment.Text)
	}
}

func TestAddCommentNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, errorEnvelope("NOT_FOUND", "ticket AMSS-999 not found"))
	}))
	defer ts.Close()

	_, err := New(ts.URL).AddComment("AMSS-999", "gc-flight", "note")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAddCommentNetworkError(t *testing.T) {
	_, err := New("http://127.0.0.1:1").AddComment("AMSS-001", "gc-flight", "note")
	if err == nil {
		t.Fatal("expected network error")
	}
}

// --- FormatTicket ---

func TestFormatTicket(t *testing.T) {
	resolved := "2026-01-02T00:00:00Z"
	resolution := "Replaced valve."
	ticket := &Ticket{
		ID:                   "AMSS-001",
		Title:                "WCS Pressure Fault",
		Description:          "Toilet pressure abnormal.",
		Severity:             "P2",
		Status:               "resolved",
		Category:             "life-support",
		Mission:              "artemis-ii",
		ReportedBy:           "wiseman-r",
		AssignedTo:           "gc-eclss",
		CreatedAt:            "2026-01-01T00:00:00Z",
		UpdatedAt:            "2026-01-02T00:00:00Z",
		ResolvedAt:           &resolved,
		Resolution:           &resolution,
		KBArticlesReferenced: []string{"KB-001"},
		Comments: []Comment{
			{Author: "gc-eclss", Timestamp: "2026-01-01T12:00:00Z", Text: "On it."},
		},
	}

	text := FormatTicket(ticket)
	for _, want := range []string{
		"AMSS-001", "WCS Pressure Fault", "P2", "resolved",
		"life-support", "artemis-ii", "Replaced valve.",
		"KB-001", "gc-eclss", "On it.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatTicket missing %q in:\n%s", want, text)
		}
	}
}

func TestFormatTicketNoOptionals(t *testing.T) {
	ticket := &Ticket{
		ID:       "AMSS-002",
		Title:    "Nav Display Glitch",
		Severity: "P3",
		Status:   "open",
		Category: "navigation",
	}
	text := FormatTicket(ticket)
	if !strings.Contains(text, "AMSS-002") {
		t.Errorf("missing ID in output")
	}
	if !strings.Contains(text, "(none)") {
		t.Errorf("expected (none) for empty KB refs")
	}
}
