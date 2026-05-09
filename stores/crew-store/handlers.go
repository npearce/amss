package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Envelope struct {
	Data  interface{} `json:"data"`
	Error *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HealthResponse struct {
	Status string `json:"status"`
	Store  string `json:"store"`
}

type ListCrewResponse struct {
	Crew   []*CrewMember `json:"crew"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type ActivityResponse struct {
	CrewID        string          `json:"crew_id"`
	Conversations []*Conversation `json:"conversations"`
	Total         int             `json:"total"`
}

type ListConversationsResponse struct {
	Conversations []*Conversation `json:"conversations"`
	Total         int             `json:"total"`
	Limit         int             `json:"limit"`
	Offset        int             `json:"offset"`
}

type ResetResponse struct {
	Message   string `json:"message"`
	CrewCount int    `json:"crew_count"`
}

var store *Store

func handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, &HealthResponse{Status: "ok", Store: "crew-store"})
}

func handleGetCrew(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil {
			limit = l
			if limit > 200 {
				limit = 200
			}
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil {
			offset = o
		}
	}

	mission := r.URL.Query().Get("mission")
	search := r.URL.Query().Get("search")

	crew, total := store.ListCrew(mission, search, limit, offset)
	if crew == nil {
		crew = []*CrewMember{}
	}

	respondJSON(w, http.StatusOK, &ListCrewResponse{
		Crew:   crew,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func handleGetCrewMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	member := store.GetCrewMember(id)
	if member == nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Crew member %s not found", id))
		return
	}
	respondJSON(w, http.StatusOK, member)
}

func handleGetCrewActivity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	limit := 20

	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	convs, total, err := store.GetCrewActivity(id, limit)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Crew member %s not found", id))
		return
	}
	if convs == nil {
		convs = []*Conversation{}
	}

	respondJSON(w, http.StatusOK, &ActivityResponse{
		CrewID:        id,
		Conversations: convs,
		Total:         total,
	})
}

func handleGetConversations(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil {
			limit = l
			if limit > 200 {
				limit = 200
			}
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if o, err := strconv.Atoi(v); err == nil {
			offset = o
		}
	}

	filters := make(map[string]string)
	if v := r.URL.Query().Get("crew_id"); v != "" {
		filters["crew_id"] = v
	}
	if v := r.URL.Query().Get("mission"); v != "" {
		filters["mission"] = v
	}
	if v := r.URL.Query().Get("session_id"); v != "" {
		filters["session_id"] = v
	}

	convs, total := store.ListConversations(filters, limit, offset)
	if convs == nil {
		convs = []*Conversation{}
	}

	respondJSON(w, http.StatusOK, &ListConversationsResponse{
		Conversations: convs,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	})
}

func handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CrewID               string   `json:"crew_id"`
		Mission              string   `json:"mission"`
		SessionID            string   `json:"session_id"`
		Query                string   `json:"query"`
		Response             string   `json:"response"`
		KBArticlesReferenced []string `json:"kb_articles_referenced"`
		TicketCreated        *string  `json:"ticket_created"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.CrewID == "" || req.Mission == "" || req.SessionID == "" || req.Query == "" || req.Response == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing required fields: crew_id, mission, session_id, query, response")
		return
	}

	conv, err := store.CreateConversation(req.CrewID, req.Mission, req.SessionID, req.Query, req.Response, req.KBArticlesReferenced, req.TicketCreated)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, conv)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	count, err := store.Reset()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, &ResetResponse{
		Message:   "Reset to seed data",
		CrewCount: count,
	})
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&Envelope{Data: data, Error: nil})
}

func respondError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&Envelope{Data: nil, Error: &ErrorInfo{Code: code, Message: message}})
}
