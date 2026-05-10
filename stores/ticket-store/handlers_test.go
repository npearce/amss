package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setupTestHandler() (*http.Handler, *Store) {
	s, _ := NewStore("seed-data/tickets.json")
	store = s

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /tickets", handleTickets)
	mux.HandleFunc("POST /tickets", handleTickets)
	mux.HandleFunc("GET /tickets/{id}", handleTicketByID)
	mux.HandleFunc("PUT /tickets/{id}", handleTicketByID)
	mux.HandleFunc("POST /tickets/{id}/comments", handleAddComment)
	mux.HandleFunc("POST /reset", handleReset)

	handler := http.Handler(mux)
	return &handler, s
}

// --- Health ---

func TestHealthEndpoint(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if env.Error != nil {
		t.Errorf("Expected no error")
	}
	if data, ok := env.Data.(map[string]interface{}); ok {
		if data["status"] != "ok" {
			t.Errorf("status = %v, want ok", data["status"])
		}
		if data["store"] != "ticket-store" {
			t.Errorf("store = %v, want ticket-store", data["store"])
		}
	} else {
		t.Error("Expected data object")
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestContentType(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// --- GET /tickets ---

func TestGetTickets(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		if data["total"].(float64) != 15 {
			t.Errorf("Expected 15 total tickets, got %v", data["total"])
		}
	} else {
		t.Error("Expected data object")
	}
}

func TestGetTicketsWithLimit(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?limit=5&offset=0", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		tickets := data["tickets"].([]interface{})
		if len(tickets) > 5 {
			t.Errorf("Expected at most 5 tickets, got %d", len(tickets))
		}
		if data["limit"].(float64) != 5 {
			t.Errorf("Expected limit 5, got %v", data["limit"])
		}
	}
}

func TestGetTicketsWithMaxLimit(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?limit=500", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		if data["limit"].(float64) != 200 {
			t.Errorf("Expected limit capped at 200, got %v", data["limit"])
		}
	}
}

func TestGetTicketsWithOffset(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?limit=5&offset=10", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		if data["offset"].(float64) != 10 {
			t.Errorf("Expected offset 10, got %v", data["offset"])
		}
		tickets := data["tickets"].([]interface{})
		if len(tickets) > 5 {
			t.Errorf("Expected at most 5 tickets with offset, got %d", len(tickets))
		}
	}
}

func TestGetTicketsFilterByMission(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?mission=artemis-iii", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		tickets := data["tickets"].([]interface{})
		if len(tickets) == 0 {
			t.Error("Expected artemis-iii tickets")
		}
		for _, raw := range tickets {
			ticket := raw.(map[string]interface{})
			if ticket["mission"] != "artemis-iii" {
				t.Errorf("Got mission %v, expected artemis-iii", ticket["mission"])
			}
		}
	}
}

func TestGetTicketsFilterBySeverity(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?severity=P1", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		if data["total"].(float64) != 2 {
			t.Errorf("Expected 2 P1 tickets, got %v", data["total"])
		}
		for _, raw := range data["tickets"].([]interface{}) {
			ticket := raw.(map[string]interface{})
			if ticket["severity"] != "P1" {
				t.Errorf("Got severity %v, expected P1", ticket["severity"])
			}
		}
	}
}

func TestGetTicketsFilterByStatus(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?status=open", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		tickets := data["tickets"].([]interface{})
		if len(tickets) == 0 {
			t.Error("Expected open tickets")
		}
		for _, raw := range tickets {
			ticket := raw.(map[string]interface{})
			if ticket["status"] != "open" {
				t.Errorf("Got status %v, expected open", ticket["status"])
			}
		}
	}
}

func TestGetTicketsFilterByCategory(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?category=life-support", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		tickets := data["tickets"].([]interface{})
		if len(tickets) == 0 {
			t.Error("Expected life-support tickets")
		}
		for _, raw := range tickets {
			ticket := raw.(map[string]interface{})
			if ticket["category"] != "life-support" {
				t.Errorf("Got category %v, expected life-support", ticket["category"])
			}
		}
	}
}

func TestGetTicketsWithSearch(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?search=WCS", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		tickets := data["tickets"].([]interface{})
		if len(tickets) == 0 {
			t.Error("Expected WCS search results")
		}
	}
}

func TestGetTicketsFilterAndSearch(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets?category=life-support&search=WCS", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		tickets := data["tickets"].([]interface{})
		if len(tickets) == 0 {
			t.Error("Expected life-support WCS tickets")
		}
		for _, raw := range tickets {
			ticket := raw.(map[string]interface{})
			if ticket["category"] != "life-support" {
				t.Errorf("Filter should restrict to life-support, got %v", ticket["category"])
			}
		}
	}
}

// --- GET /tickets/{id} ---

func TestGetTicketByID(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets/AMSS-001", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if env.Error != nil {
		t.Errorf("Expected no error")
	}
	if ticket, ok := env.Data.(map[string]interface{}); ok {
		if ticket["id"] != "AMSS-001" {
			t.Errorf("Expected AMSS-001, got %v", ticket["id"])
		}
	}
}

func TestGetTicketNotFound(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets/AMSS-999", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if env.Data != nil {
		t.Error("Expected data to be null on 404")
	}
	if env.Error == nil || env.Error.Code != "NOT_FOUND" {
		t.Errorf("Expected NOT_FOUND error, got %v", env.Error)
	}
}

func TestGetTicketIncludesComments(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets/AMSS-001", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if ticket, ok := env.Data.(map[string]interface{}); ok {
		comments, ok := ticket["comments"].([]interface{})
		if !ok {
			t.Fatal("Expected comments array")
		}
		if len(comments) != 5 {
			t.Errorf("AMSS-001 should have 5 comments, got %d", len(comments))
		}
	}
}

func TestTicketFields(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets/AMSS-001", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if ticket, ok := env.Data.(map[string]interface{}); ok {
		required := []string{"id", "title", "description", "severity", "status", "category", "reported_by", "assigned_to", "mission", "created_at", "updated_at", "comments"}
		for _, field := range required {
			if _, ok := ticket[field]; !ok {
				t.Errorf("Missing required field: %s", field)
			}
		}
	}
}

// --- POST /tickets ---

func TestCreateTicketHandler(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{
		"title":       "New Test Ticket",
		"description": "Test description",
		"severity":    "P3",
		"category":    "power",
		"reported_by": "wiseman-r",
		"mission":     "artemis-ii",
		"assigned_to": "gc-systems",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if env.Error != nil {
		t.Errorf("Expected no error, got %v", env.Error)
	}
	if ticket, ok := env.Data.(map[string]interface{}); ok {
		if _, hasID := ticket["id"]; !hasID {
			t.Error("Expected id in response")
		}
		if ticket["title"] != "New Test Ticket" {
			t.Errorf("Title mismatch")
		}
	}
}

func TestCreateTicketDefaultStatus(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{
		"title": "T", "description": "D", "severity": "P3",
		"category": "power", "reported_by": "a", "mission": "m",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if ticket, ok := env.Data.(map[string]interface{}); ok {
		if ticket["status"] != "open" {
			t.Errorf("Default status should be open, got %v", ticket["status"])
		}
	}
}

func TestCreateTicketDefaultAssignedTo(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{
		"title": "T", "description": "D", "severity": "P3",
		"category": "power", "reported_by": "a", "mission": "m",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if ticket, ok := env.Data.(map[string]interface{}); ok {
		if ticket["assigned_to"] != "ground-control" {
			t.Errorf("Default assigned_to should be ground-control, got %v", ticket["assigned_to"])
		}
	}
}

func TestCreateTicketMissingFields(t *testing.T) {
	handler, _ := setupTestHandler()

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{"missing title", map[string]interface{}{"description": "D", "severity": "P3", "category": "power", "reported_by": "a", "mission": "m"}},
		{"missing description", map[string]interface{}{"title": "T", "severity": "P3", "category": "power", "reported_by": "a", "mission": "m"}},
		{"missing severity", map[string]interface{}{"title": "T", "description": "D", "category": "power", "reported_by": "a", "mission": "m"}},
		{"missing category", map[string]interface{}{"title": "T", "description": "D", "severity": "P3", "reported_by": "a", "mission": "m"}},
		{"missing reported_by", map[string]interface{}{"title": "T", "description": "D", "severity": "P3", "category": "power", "mission": "m"}},
		{"missing mission", map[string]interface{}{"title": "T", "description": "D", "severity": "P3", "category": "power", "reported_by": "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/tickets", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()
			(*handler).ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400, got %d", w.Code)
			}
			var env Envelope
			json.NewDecoder(w.Body).Decode(&env)
			if env.Error == nil || env.Error.Code != "BAD_REQUEST" {
				t.Errorf("Expected BAD_REQUEST error, got %v", env.Error)
			}
		})
	}
}

func TestCreateTicketInvalidSeverity(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{
		"title": "T", "description": "D", "severity": "P5",
		"category": "power", "reported_by": "a", "mission": "m",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid severity, got %d", w.Code)
	}
}

func TestCreateTicketInvalidCategory(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{
		"title": "T", "description": "D", "severity": "P3",
		"category": "invalid-cat", "reported_by": "a", "mission": "m",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid category, got %d", w.Code)
	}
}

func TestCreateTicketInvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("POST", "/tickets", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

// --- PUT /tickets/{id} ---

func TestUpdateTicketHandler(t *testing.T) {
	handler, _ := setupTestHandler()

	updates := map[string]interface{}{"assigned_to": "gc-gnc"}
	bodyBytes, _ := json.Marshal(updates)

	req := httptest.NewRequest("PUT", "/tickets/AMSS-014", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if ticket, ok := env.Data.(map[string]interface{}); ok {
		if ticket["assigned_to"] != "gc-gnc" {
			t.Errorf("assigned_to should be updated, got %v", ticket["assigned_to"])
		}
	}
}

func TestUpdateTicketResolvedAt(t *testing.T) {
	handler, _ := setupTestHandler()

	// AMSS-014 is open with no resolved_at
	updates := map[string]interface{}{"status": "resolved"}
	bodyBytes, _ := json.Marshal(updates)

	req := httptest.NewRequest("PUT", "/tickets/AMSS-014", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if ticket, ok := env.Data.(map[string]interface{}); ok {
		if ticket["resolved_at"] == nil {
			t.Error("resolved_at should be set when status changes to resolved")
		}
		if _, err := time.Parse(time.RFC3339, ticket["resolved_at"].(string)); err != nil {
			t.Errorf("resolved_at should be valid RFC3339: %v", err)
		}
	}
}

func TestUpdateTicketAlreadyResolvedPreservesTimestamp(t *testing.T) {
	handler, _ := setupTestHandler()

	// Get original resolved_at for AMSS-001
	req := httptest.NewRequest("GET", "/tickets/AMSS-001", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var getEnv Envelope
	json.NewDecoder(w.Body).Decode(&getEnv)
	origTicket := getEnv.Data.(map[string]interface{})
	origResolvedAt := origTicket["resolved_at"].(string)

	// Update again with status resolved
	updates := map[string]interface{}{"status": "resolved"}
	bodyBytes, _ := json.Marshal(updates)
	req2 := httptest.NewRequest("PUT", "/tickets/AMSS-001", bytes.NewReader(bodyBytes))
	w2 := httptest.NewRecorder()
	(*handler).ServeHTTP(w2, req2)

	var putEnv Envelope
	json.NewDecoder(w2.Body).Decode(&putEnv)

	if ticket, ok := putEnv.Data.(map[string]interface{}); ok {
		if ticket["resolved_at"].(string) != origResolvedAt {
			t.Errorf("resolved_at changed: got %v, want %v", ticket["resolved_at"], origResolvedAt)
		}
	}
}

func TestUpdateTicketNotFound(t *testing.T) {
	handler, _ := setupTestHandler()

	updates := map[string]interface{}{"title": "x"}
	bodyBytes, _ := json.Marshal(updates)

	req := httptest.NewRequest("PUT", "/tickets/AMSS-999", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestUpdateTicketInvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("PUT", "/tickets/AMSS-001", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestUpdateTicketInvalidSeverity(t *testing.T) {
	handler, _ := setupTestHandler()

	updates := map[string]interface{}{"severity": "P9"}
	bodyBytes, _ := json.Marshal(updates)

	req := httptest.NewRequest("PUT", "/tickets/AMSS-001", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid severity, got %d", w.Code)
	}
}

// --- POST /tickets/{id}/comments ---

func TestAddCommentHandler(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{"author": "wiseman-r", "text": "Test comment"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets/AMSS-014/comments", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if env.Error != nil {
		t.Errorf("Expected no error, got %v", env.Error)
	}
	if comment, ok := env.Data.(map[string]interface{}); ok {
		if comment["author"] != "wiseman-r" {
			t.Errorf("author = %v, want wiseman-r", comment["author"])
		}
		if comment["text"] != "Test comment" {
			t.Errorf("text mismatch")
		}
		if comment["timestamp"] == nil {
			t.Error("timestamp should be set")
		}
	}
}

func TestAddCommentUpdatesTicketTimestamp(t *testing.T) {
	handler, _ := setupTestHandler()

	// Get original updated_at
	req := httptest.NewRequest("GET", "/tickets/AMSS-014", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)
	var getEnv Envelope
	json.NewDecoder(w.Body).Decode(&getEnv)
	origTicket := getEnv.Data.(map[string]interface{})
	origUpdatedAt := origTicket["updated_at"].(string)

	// Add comment
	body := map[string]interface{}{"author": "gc-systems", "text": "Update"}
	bodyBytes, _ := json.Marshal(body)
	req2 := httptest.NewRequest("POST", "/tickets/AMSS-014/comments", bytes.NewReader(bodyBytes))
	w2 := httptest.NewRecorder()
	(*handler).ServeHTTP(w2, req2)

	// Get updated ticket
	req3 := httptest.NewRequest("GET", "/tickets/AMSS-014", nil)
	w3 := httptest.NewRecorder()
	(*handler).ServeHTTP(w3, req3)
	var getEnv2 Envelope
	json.NewDecoder(w3.Body).Decode(&getEnv2)
	updatedTicket := getEnv2.Data.(map[string]interface{})

	if updatedTicket["updated_at"].(string) == origUpdatedAt {
		t.Error("Ticket updated_at should change after adding comment")
	}
}

func TestAddCommentMissingFields(t *testing.T) {
	handler, _ := setupTestHandler()

	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{"missing author", map[string]interface{}{"text": "hello"}},
		{"missing text", map[string]interface{}{"author": "wiseman-r"}},
		{"empty body", map[string]interface{}{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/tickets/AMSS-014/comments", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()
			(*handler).ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400, got %d", w.Code)
			}
			var env Envelope
			json.NewDecoder(w.Body).Decode(&env)
			if env.Error == nil || env.Error.Code != "BAD_REQUEST" {
				t.Errorf("Expected BAD_REQUEST error, got %v", env.Error)
			}
		})
	}
}

func TestAddCommentTicketNotFound(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{"author": "a", "text": "t"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/tickets/AMSS-999/comments", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestAddCommentInvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("POST", "/tickets/AMSS-001/comments", bytes.NewReader([]byte("bad json")))
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

// --- POST /reset ---

func TestResetEndpoint(t *testing.T) {
	handler, _ := setupTestHandler()

	store.CreateTicket("Temp", "Temp", "P3", "power", "a", "m", "", nil)

	req := httptest.NewRequest("POST", "/reset", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		if data["ticket_count"].(float64) != 15 {
			t.Errorf("Expected 15 tickets after reset, got %v", data["ticket_count"])
		}
	}
}

// --- Envelope format ---

func TestEnvelopeFormatSuccess(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets/AMSS-001", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["data"] == nil {
		t.Error("Expected data field on success")
	}
	if result["error"] != nil {
		t.Error("Expected error to be null on success")
	}
}

func TestEnvelopeFormatError(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets/AMSS-999", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["data"] != nil {
		t.Error("Expected data to be null on error")
	}
	if result["error"] == nil {
		t.Error("Expected error object on failure")
	}
}

func TestListResponseStructure(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/tickets", nil)
	w := httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)

	if data, ok := env.Data.(map[string]interface{}); ok {
		for _, field := range []string{"tickets", "total", "limit", "offset"} {
			if _, ok := data[field]; !ok {
				t.Errorf("Missing field in list response: %s", field)
			}
		}
	} else {
		t.Error("Expected data object")
	}
}
