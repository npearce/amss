package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Envelope struct {
	Data  interface{} `json:"data"`
	Error *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResetResponse struct {
	Message     string `json:"message"`
	TicketCount int    `json:"ticket_count"`
}

type ListResponse struct {
	Tickets []*Ticket `json:"tickets"`
	Total   int       `json:"total"`
	Limit   int       `json:"limit"`
	Offset  int       `json:"offset"`
}

type HealthResponse struct {
	Status string `json:"status"`
	Store  string `json:"store"`
}

var store *Store

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	respondJSON(w, http.StatusOK, &HealthResponse{Status: "ok", Store: "ticket-store"})
}

func handleTickets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetTickets(w, r)
	case http.MethodPost:
		handleCreateTicket(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func handleGetTickets(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
			if limit > 200 {
				limit = 200
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	search := r.URL.Query().Get("search")

	filters := make(map[string]string)
	if mission := r.URL.Query().Get("mission"); mission != "" {
		filters["mission"] = mission
	}
	if severity := r.URL.Query().Get("severity"); severity != "" {
		filters["severity"] = severity
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}
	if category := r.URL.Query().Get("category"); category != "" {
		filters["category"] = category
	}

	tickets, total := store.ListTickets(filters, search, limit, offset)
	if tickets == nil {
		tickets = []*Ticket{}
	}

	respondJSON(w, http.StatusOK, &ListResponse{
		Tickets: tickets,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

func handleCreateTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title                string   `json:"title"`
		Description          string   `json:"description"`
		Severity             string   `json:"severity"`
		Category             string   `json:"category"`
		ReportedBy           string   `json:"reported_by"`
		Mission              string   `json:"mission"`
		AssignedTo           string   `json:"assigned_to"`
		KBArticlesReferenced []string `json:"kb_articles_referenced"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Title == "" || req.Description == "" || req.Severity == "" || req.Category == "" || req.ReportedBy == "" || req.Mission == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing required fields: title, description, severity, category, reported_by, mission")
		return
	}

	ticket, err := store.CreateTicket(req.Title, req.Description, req.Severity, req.Category, req.ReportedBy, req.Mission, req.AssignedTo, req.KBArticlesReferenced)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, ticket)
}

func handleTicketByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	switch r.Method {
	case http.MethodGet:
		ticket := store.GetTicket(id)
		if ticket == nil {
			respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Ticket %s not found", id))
			return
		}
		respondJSON(w, http.StatusOK, ticket)

	case http.MethodPut:
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
			return
		}

		ticket, err := store.UpdateTicket(id, updates)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Ticket %s not found", id))
			} else {
				respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			}
			return
		}

		respondJSON(w, http.StatusOK, ticket)

	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func handleAddComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Author string `json:"author"`
		Text   string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Author == "" || req.Text == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing required fields: author, text")
		return
	}

	comment, err := store.AddComment(id, req.Author, req.Text)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Ticket %s not found", id))
		} else {
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return
	}

	respondJSON(w, http.StatusCreated, comment)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}

	count, err := store.Reset()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, &ResetResponse{
		Message:     "Reset to seed data",
		TicketCount: count,
	})
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	envelope := &Envelope{
		Data:  data,
		Error: nil,
	}

	json.NewEncoder(w).Encode(envelope)
}

func respondError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	envelope := &Envelope{
		Data: nil,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(envelope)
}
