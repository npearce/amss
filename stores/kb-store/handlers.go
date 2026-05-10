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

type DeleteResponse struct {
	Deleted string `json:"deleted"`
}

type ResetResponse struct {
	Message       string `json:"message"`
	ArticleCount  int    `json:"article_count"`
}

type ListResponse struct {
	Articles []*Article `json:"articles"`
	Total    int        `json:"total"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
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

	respondJSON(w, http.StatusOK, &HealthResponse{Status: "ok", Store: "kb-store"})
}

func handleArticles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetArticles(w, r)
	case http.MethodPost:
		handleCreateArticle(w, r)
	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
}

func handleGetArticles(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	search := ""

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

	search = r.URL.Query().Get("search")

	filters := make(map[string]string)
	if category := r.URL.Query().Get("category"); category != "" {
		filters["category"] = category
	}
	if tags := r.URL.Query().Get("tags"); tags != "" {
		filters["tags"] = tags
	}

	articles, total := store.ListArticles(filters, search, limit, offset)
	if articles == nil {
		articles = []*Article{}
	}

	response := &ListResponse{
		Articles: articles,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}

	respondJSON(w, http.StatusOK, response)
}

func handleCreateArticle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Category  string   `json:"category"`
		Tags      []string `json:"tags"`
		CreatedBy string   `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Title == "" || req.Body == "" || req.Category == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Missing required fields: title, body, category")
		return
	}

	article, err := store.CreateArticle(req.Title, req.Body, req.Category, req.CreatedBy, req.Tags)
	if err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, article)
}

func handleArticleByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	switch r.Method {
	case http.MethodGet:
		article := store.GetArticle(id)
		if article == nil {
			respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Article %s not found", id))
			return
		}
		respondJSON(w, http.StatusOK, article)

	case http.MethodPut:
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
			return
		}

		article, err := store.UpdateArticle(id, updates)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Article %s not found", id))
			} else {
				respondError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			}
			return
		}

		respondJSON(w, http.StatusOK, article)

	case http.MethodDelete:
		if err := store.DeleteArticle(id); err != nil {
			respondError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("Article %s not found", id))
			return
		}

		respondJSON(w, http.StatusOK, &DeleteResponse{Deleted: id})

	default:
		http.Error(w, "", http.StatusMethodNotAllowed)
	}
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
		Message:      "Reset to seed data",
		ArticleCount: count,
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
