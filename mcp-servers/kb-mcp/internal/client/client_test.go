package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// articleEnvelope wraps a single Article in the store's envelope format.
func articleEnvelope(a *Article) map[string]interface{} {
	return map[string]interface{}{"data": a, "error": nil}
}

func listEnvelope(articles []*Article, total int) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"articles": articles,
			"total":    total,
			"limit":    50,
			"offset":   0,
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

// --- New ---

func TestNew(t *testing.T) {
	c := New("http://kb-store:8081")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "http://kb-store:8081" {
		t.Errorf("baseURL = %q", c.baseURL)
	}
}

// --- Search ---

func TestSearchReturnsArticles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/articles" {
			t.Errorf("path = %s, want /articles", r.URL.Path)
		}
		articles := []*Article{
			{ID: "KB-001", Title: "ECLSS Overview", Category: "life-support"},
		}
		writeJSON(w, listEnvelope(articles, 1))
	}))
	defer ts.Close()

	c := New(ts.URL)
	articles, total, err := c.Search("", "", "eclss", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(articles) != 1 || articles[0].ID != "KB-001" {
		t.Errorf("articles = %+v", articles)
	}
}

func TestSearchPassesQueryParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("category") != "life-support" {
			t.Errorf("category param = %q", q.Get("category"))
		}
		if q.Get("search") != "oxygen" {
			t.Errorf("search param = %q", q.Get("search"))
		}
		if q.Get("limit") != "10" {
			t.Errorf("limit param = %q", q.Get("limit"))
		}
		writeJSON(w, listEnvelope([]*Article{}, 0))
	}))
	defer ts.Close()

	c := New(ts.URL)
	c.Search("life-support", "", "oxygen", 10, 0)
}

func TestSearchOffsetParam(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "10" {
			t.Errorf("offset param = %q", r.URL.Query().Get("offset"))
		}
		writeJSON(w, listEnvelope([]*Article{}, 0))
	}))
	defer ts.Close()
	New(ts.URL).Search("", "", "", 50, 10)
}

func TestSearchAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, errorEnvelope("BAD_REQUEST", "invalid category"))
	}))
	defer ts.Close()

	c := New(ts.URL)
	_, _, err := c.Search("bad-cat", "", "", 50, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid category") {
		t.Errorf("error = %v", err)
	}
}

func TestSearchEmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, listEnvelope([]*Article{}, 0))
	}))
	defer ts.Close()

	c := New(ts.URL)
	articles, total, err := c.Search("", "", "nothing", 50, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 || len(articles) != 0 {
		t.Errorf("expected empty result, got total=%d articles=%v", total, articles)
	}
}

func TestSearchNetworkError(t *testing.T) {
	c := New("http://127.0.0.1:1") // nothing listening
	_, _, err := c.Search("", "", "", 50, 0)
	if err == nil {
		t.Fatal("expected network error")
	}
}

// --- GetArticle ---

func TestGetArticleSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/articles/KB-001" {
			t.Errorf("path = %s, want /articles/KB-001", r.URL.Path)
		}
		writeJSON(w, articleEnvelope(&Article{ID: "KB-001", Title: "ECLSS Overview", Body: "Details here."}))
	}))
	defer ts.Close()

	c := New(ts.URL)
	a, err := c.GetArticle("KB-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "KB-001" {
		t.Errorf("id = %q", a.ID)
	}
	if a.Title != "ECLSS Overview" {
		t.Errorf("title = %q", a.Title)
	}
}

func TestGetArticleNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, errorEnvelope("NOT_FOUND", "article KB-999 not found"))
	}))
	defer ts.Close()

	c := New(ts.URL)
	_, err := c.GetArticle("KB-999")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %v", err)
	}
}

func TestGetArticleNetworkError(t *testing.T) {
	c := New("http://127.0.0.1:1")
	_, err := c.GetArticle("KB-001")
	if err == nil {
		t.Fatal("expected network error")
	}
}

// --- CreateArticle ---

func TestCreateArticleSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/articles" {
			t.Errorf("path = %s, want /articles", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "New Title" {
			t.Errorf("title = %v", body["title"])
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, articleEnvelope(&Article{ID: "KB-031", Title: "New Title", Category: "medical"}))
	}))
	defer ts.Close()

	c := New(ts.URL)
	a, err := c.CreateArticle("New Title", "Body text", "medical", "gc-surgeon", []string{"tag1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID != "KB-031" {
		t.Errorf("id = %q", a.ID)
	}
}

func TestCreateArticleStoreError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, errorEnvelope("BAD_REQUEST", "invalid category"))
	}))
	defer ts.Close()

	c := New(ts.URL)
	_, err := c.CreateArticle("T", "B", "invalid-cat", "user", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateArticleNilTagsBecomesEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		tags, ok := body["tags"]
		if !ok {
			t.Error("tags field missing from request body")
		}
		if tags == nil {
			t.Error("tags should not be null")
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, articleEnvelope(&Article{ID: "KB-032", Title: "T"}))
	}))
	defer ts.Close()
	New(ts.URL).CreateArticle("T", "B", "medical", "user", nil)
}

// --- UpdateArticle ---

func TestUpdateArticleSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/articles/KB-001" {
			t.Errorf("path = %s", r.URL.Path)
		}
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "Updated Title" {
			t.Errorf("title = %v", body["title"])
		}
		writeJSON(w, articleEnvelope(&Article{ID: "KB-001", Title: "Updated Title"}))
	}))
	defer ts.Close()

	c := New(ts.URL)
	a, err := c.UpdateArticle("KB-001", map[string]interface{}{"title": "Updated Title"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Title != "Updated Title" {
		t.Errorf("title = %q", a.Title)
	}
}

func TestUpdateArticleNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, errorEnvelope("NOT_FOUND", "article KB-999 not found"))
	}))
	defer ts.Close()

	c := New(ts.URL)
	_, err := c.UpdateArticle("KB-999", map[string]interface{}{"title": "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- FormatArticle ---

func TestFormatArticle(t *testing.T) {
	score := 0.85
	dup := "KB-002"
	notes := "Needs review"
	a := &Article{
		ID:              "KB-001",
		Title:           "ECLSS Test",
		Body:            "Body content here.",
		Category:        "life-support",
		Tags:            []string{"eclss", "oxygen"},
		CreatedBy:       "gc-eclss",
		CreatedAt:       "2026-01-01T00:00:00Z",
		UpdatedAt:       "2026-01-02T00:00:00Z",
		ReferenceCount:  3,
		UsefulnessScore: &score,
		DuplicateOf:     &dup,
		CuratorNotes:    &notes,
		CuratorTags:     []string{"duplicate"},
	}

	text := FormatArticle(a)
	for _, want := range []string{"KB-001", "ECLSS Test", "life-support", "eclss", "Body content here.", "0.85", "KB-002", "Needs review"} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatArticle missing %q in output:\n%s", want, text)
		}
	}
}

func TestFormatArticleNoOptionals(t *testing.T) {
	a := &Article{
		ID:       "KB-002",
		Title:    "Basic Article",
		Body:     "Simple body.",
		Category: "navigation",
	}
	text := FormatArticle(a)
	if !strings.Contains(text, "KB-002") {
		t.Errorf("missing ID in output")
	}
	if !strings.Contains(text, "(none)") {
		t.Errorf("expected (none) for empty tags")
	}
}
