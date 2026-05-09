package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestHandler() (*http.Handler, *Store) {
	s, _ := NewStore("seed-data/kb.json")
	store = s

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /articles", handleArticles)
	mux.HandleFunc("POST /articles", handleArticles)
	mux.HandleFunc("GET /articles/{id}", handleArticleByID)
	mux.HandleFunc("PUT /articles/{id}", handleArticleByID)
	mux.HandleFunc("DELETE /articles/{id}", handleArticleByID)
	mux.HandleFunc("POST /reset", handleReset)

	handler := http.Handler(mux)
	return &handler, s
}

func TestHealthEndpoint(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if envelope.Error != nil {
		t.Errorf("Expected no error")
	}

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		if data["status"] != "ok" || data["store"] != "kb-store" {
			t.Errorf("Unexpected health response")
		}
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestGetArticles(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if envelope.Error != nil {
		t.Errorf("Expected no error")
	}

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		if data["total"].(float64) != 30 {
			t.Errorf("Expected 30 total articles, got %v", data["total"])
		}
	}
}

func TestGetArticlesWithLimit(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles?limit=5&offset=0", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		articles := data["articles"].([]interface{})
		if len(articles) > 5 {
			t.Errorf("Expected at most 5 articles, got %d", len(articles))
		}
	}
}

func TestGetArticlesWithMaxLimit(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles?limit=500", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		if data["limit"].(float64) != 200 {
			t.Errorf("Expected limit capped at 200, got %v", data["limit"])
		}
	}
}

func TestGetArticlesWithSearch(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles?search=wcs", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		articles := data["articles"].([]interface{})
		if len(articles) == 0 {
			t.Errorf("Expected to find articles with 'wcs' in search")
		}
	}
}

func TestGetArticlesWithCategoryFilter(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles?category=life-support", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		articles := data["articles"].([]interface{})
		if len(articles) == 0 {
			t.Errorf("Expected to find life-support articles")
		}
	}
}

func TestGetArticleByID(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles/KB-001", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if envelope.Error != nil {
		t.Errorf("Expected no error")
	}

	if article, ok := envelope.Data.(map[string]interface{}); ok {
		if article["id"] != "KB-001" {
			t.Errorf("Expected KB-001, got %v", article["id"])
		}
	}
}

func TestGetArticleNotFound(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles/KB-999", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if envelope.Data != nil {
		t.Errorf("Expected data to be null on error")
	}

	if envelope.Error == nil || envelope.Error.Code != "NOT_FOUND" {
		t.Errorf("Expected NOT_FOUND error")
	}
}

func TestCreateArticleHandler(t *testing.T) {
	handler, _ := setupTestHandler()

	body := map[string]interface{}{
		"title":      "New Article",
		"body":       "New body",
		"category":   "life-support",
		"tags":       []string{"tag1"},
		"created_by": "test-creator",
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/articles", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if envelope.Error != nil {
		t.Errorf("Expected no error")
	}

	if article, ok := envelope.Data.(map[string]interface{}); ok {
		if _, hasID := article["id"]; !hasID {
			t.Errorf("Expected id in response")
		}
		if article["title"] != "New Article" {
			t.Errorf("Expected title to match")
		}
	}
}

func TestCreateArticleHandlerInvalidRequest(t *testing.T) {
	handler, _ := setupTestHandler()

	tests := []struct {
		name     string
		body     map[string]interface{}
		expectedCode string
	}{
		{
			name:         "missing title",
			body:         map[string]interface{}{"body": "test", "category": "life-support"},
			expectedCode: "BAD_REQUEST",
		},
		{
			name:         "missing body",
			body:         map[string]interface{}{"title": "test", "category": "life-support"},
			expectedCode: "BAD_REQUEST",
		},
		{
			name:         "missing category",
			body:         map[string]interface{}{"title": "test", "body": "test"},
			expectedCode: "BAD_REQUEST",
		},
		{
			name:         "invalid category",
			body:         map[string]interface{}{"title": "test", "body": "test", "category": "invalid"},
			expectedCode: "BAD_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/articles", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			(*handler).ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			var envelope Envelope
			json.NewDecoder(w.Body).Decode(&envelope)

			if envelope.Error == nil || envelope.Error.Code != tt.expectedCode {
				t.Errorf("Expected %s error", tt.expectedCode)
			}
		})
	}
}

func TestUpdateArticleHandler(t *testing.T) {
	handler, _ := setupTestHandler()

	updates := map[string]interface{}{
		"title": "Updated Title",
	}

	bodyBytes, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/articles/KB-001", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if article, ok := envelope.Data.(map[string]interface{}); ok {
		if article["title"] != "Updated Title" {
			t.Errorf("Expected updated title")
		}
	}
}

func TestUpdateArticleHandlerNotFound(t *testing.T) {
	handler, _ := setupTestHandler()

	updates := map[string]interface{}{"title": "Test"}
	bodyBytes, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/articles/KB-999", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestUpdateCuratorFields(t *testing.T) {
	handler, _ := setupTestHandler()

	updates := map[string]interface{}{
		"usefulness_score": 0.95,
		"curator_notes":    "Test notes",
	}

	bodyBytes, _ := json.Marshal(updates)
	req := httptest.NewRequest("PUT", "/articles/KB-001", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if article, ok := envelope.Data.(map[string]interface{}); ok {
		if article["last_curated_at"] == nil {
			t.Errorf("Expected last_curated_at to be set")
		}
	}
}

func TestDeleteArticleHandler(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("DELETE", "/articles/KB-010", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		if data["deleted"] != "KB-010" {
			t.Errorf("Expected deleted field")
		}
	}

	req = httptest.NewRequest("GET", "/articles/KB-010", nil)
	w = httptest.NewRecorder()
	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Article should be deleted")
	}
}

func TestDeleteArticleHandlerNotFound(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("DELETE", "/articles/KB-999", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestResetEndpoint(t *testing.T) {
	handler, _ := setupTestHandler()

	store.CreateArticle("Test", "Test", "life-support", "", nil)

	req := httptest.NewRequest("POST", "/reset", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		if data["article_count"].(float64) != 30 {
			t.Errorf("Expected 30 articles after reset")
		}
	}
}

func TestContentType(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestEnvelopeFormatSuccess(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles/KB-001", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["data"] == nil {
		t.Errorf("Expected data field")
	}
	if result["error"] != nil {
		t.Errorf("Expected error to be null on success")
	}
}

func TestEnvelopeFormatError(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles/KB-999", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["data"] != nil {
		t.Errorf("Expected data to be null on error")
	}
	if result["error"] == nil {
		t.Errorf("Expected error object")
	}
}

func TestPaginationOffset(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles?limit=10&offset=5", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		if data["offset"].(float64) != 5 {
			t.Errorf("Expected offset 5")
		}
	}
}

func TestSearchAndFilter(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles?category=life-support&search=wcs", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		articles := data["articles"].([]interface{})
		if len(articles) == 0 {
			t.Errorf("Expected to find WCS articles in life-support category")
		}
	}
}

func TestInvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("POST", "/articles", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestMethodNotAllowedOnArticles(t *testing.T) {
	handler, _ := setupTestHandler()

	methods := []string{"PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/articles/KB-001", nil)
		w := httptest.NewRecorder()

		(*handler).ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 for %s, got %d", method, w.Code)
		}
	}
}

func TestArticleFields(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles/KB-001", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if article, ok := envelope.Data.(map[string]interface{}); ok {
		requiredFields := []string{"id", "title", "body", "category", "tags", "created_by", "created_at", "updated_at"}
		for _, field := range requiredFields {
			if _, ok := article[field]; !ok {
				t.Errorf("Missing field: %s", field)
			}
		}
	}
}

func TestListResponseStructure(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/articles", nil)
	w := httptest.NewRecorder()

	(*handler).ServeHTTP(w, req)

	var envelope Envelope
	json.NewDecoder(w.Body).Decode(&envelope)

	if data, ok := envelope.Data.(map[string]interface{}); ok {
		fields := []string{"articles", "total", "limit", "offset"}
		for _, field := range fields {
			if _, ok := data[field]; !ok {
				t.Errorf("Missing field in list response: %s", field)
			}
		}
	}
}
