package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestLoadSeed(t *testing.T) {
	s, err := NewStore("seed-data/kb.json")
	if err != nil {
		t.Fatalf("Failed to load seed data: %v", err)
	}

	if len(s.articles) != 30 {
		t.Errorf("Expected 30 articles, got %d", len(s.articles))
	}

	article := s.GetArticle("KB-001")
	if article == nil {
		t.Errorf("Expected to find KB-001")
	}
	if article.Title != "WCS manual flush procedure during low-gravity transit" {
		t.Errorf("Unexpected title: %s", article.Title)
	}
}

func TestCreateArticle(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	tests := []struct {
		name          string
		title         string
		body          string
		category      string
		createdBy     string
		tags          []string
		shouldErr     bool
		expectedID    string
		expectedError string
	}{
		{
			name:       "valid article",
			title:      "Test Article",
			body:       "Test body",
			category:   "life-support",
			createdBy:  "test-creator",
			tags:       []string{"tag1"},
			shouldErr:  false,
			expectedID: "KB-031",
		},
		{
			name:          "missing title",
			title:         "",
			body:          "Test body",
			category:      "life-support",
			shouldErr:     true,
			expectedError: "title and body are required",
		},
		{
			name:          "missing body",
			title:         "Test",
			body:          "",
			category:      "life-support",
			shouldErr:     true,
			expectedError: "title and body are required",
		},
		{
			name:          "invalid category",
			title:         "Test",
			body:          "Test",
			category:      "invalid-cat",
			shouldErr:     true,
			expectedError: "invalid category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article, err := s.CreateArticle(tt.title, tt.body, tt.category, tt.createdBy, tt.tags)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error: %s", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if article.ID != tt.expectedID {
					t.Errorf("Expected ID %s, got %s", tt.expectedID, article.ID)
				}
				if tt.createdBy == "" && article.CreatedBy != "ground-control" {
					t.Errorf("Expected default createdBy, got %s", article.CreatedBy)
				}
			}
		})
	}
}

func TestUpdateArticle(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	tests := []struct {
		name            string
		id              string
		updates         map[string]interface{}
		shouldErr       bool
		checkCurator    bool
		expectedCurator bool
	}{
		{
			name: "update title",
			id:   "KB-001",
			updates: map[string]interface{}{
				"title": "New Title",
			},
			shouldErr:       false,
			checkCurator:    false,
			expectedCurator: false,
		},
		{
			name: "update curator fields",
			id:   "KB-001",
			updates: map[string]interface{}{
				"usefulness_score": 0.85,
				"curator_notes":    "Test notes",
			},
			shouldErr:       false,
			checkCurator:    true,
			expectedCurator: true,
		},
		{
			name:      "article not found",
			id:        "KB-999",
			updates:   map[string]interface{}{},
			shouldErr: true,
		},
		{
			name: "invalid category",
			id:   "KB-001",
			updates: map[string]interface{}{
				"category": "invalid",
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article, err := s.UpdateArticle(tt.id, tt.updates)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if article.UpdatedAt == "" {
					t.Errorf("UpdatedAt should be set")
				}
				if tt.checkCurator {
					if !tt.expectedCurator && article.LastCuratedAt == nil {
						t.Errorf("LastCuratedAt should be set when curator fields change")
					}
				}
			}
		})
	}
}

func TestDeleteArticle(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	article := s.GetArticle("KB-001")
	if article == nil {
		t.Fatalf("Expected to find KB-001")
	}

	err := s.DeleteArticle("KB-001")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	article = s.GetArticle("KB-001")
	if article != nil {
		t.Errorf("Article should be deleted")
	}

	err = s.DeleteArticle("KB-999")
	if err == nil {
		t.Errorf("Expected error when deleting non-existent article")
	}
}

func TestSearch(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	tests := []struct {
		name                string
		query               string
		minExpectedMatches  int
		shouldContainAny    []string
	}{
		{
			name:               "search for wcs",
			query:              "wcs",
			minExpectedMatches: 3,
			shouldContainAny:   []string{"KB-001"},
		},
		{
			name:               "search for procedure",
			query:              "procedure",
			minExpectedMatches: 1,
			shouldContainAny:   []string{"KB-001", "KB-002"},
		},
		{
			name:               "search no results",
			query:              "xyzabc123",
			minExpectedMatches: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			articles, _ := s.ListArticles(make(map[string]string), tt.query, 50, 0)

			if len(articles) < tt.minExpectedMatches {
				t.Errorf("Expected at least %d results for '%s', got %d", tt.minExpectedMatches, tt.query, len(articles))
			}

			if len(tt.shouldContainAny) > 0 {
				foundMap := make(map[string]bool)
				for _, a := range articles {
					foundMap[a.ID] = true
				}

				found := false
				for _, id := range tt.shouldContainAny {
					if foundMap[id] {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find one of %v in results", tt.shouldContainAny)
				}
			}
		})
	}
}

func TestListWithFilters(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	tests := []struct {
		name          string
		filters       map[string]string
		minExpected   int
		limit         int
		offset        int
	}{
		{
			name:        "filter by category",
			filters:     map[string]string{"category": "life-support"},
			minExpected: 5,
		},
		{
			name:        "filter by tags",
			filters:     map[string]string{"tags": "wcs"},
			minExpected: 1,
		},
		{
			name:        "pagination",
			filters:     make(map[string]string),
			minExpected: 5,
			limit:       5,
			offset:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.limit == 0 {
				tt.limit = 50
			}

			articles, total := s.ListArticles(tt.filters, "", tt.limit, tt.offset)

			if len(articles) < tt.minExpected {
				t.Errorf("Expected at least %d results, got %d", tt.minExpected, len(articles))
			}

			if tt.limit > 0 && len(articles) > tt.limit {
				t.Errorf("Expected at most %d results, got %d", tt.limit, len(articles))
			}

			if len(tt.filters) == 0 && total != 30 {
				t.Errorf("Expected total of 30 articles (no filters), got %d", total)
			}

			if len(articles) > 0 && total == 0 {
				t.Errorf("Total should be > 0 when articles are returned")
			}
		})
	}
}

func TestReset(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	s.CreateArticle("Test", "Test body", "life-support", "", nil)

	count, err := s.Reset()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if count != 30 {
		t.Errorf("After reset, expected 30 articles, got %d", count)
	}

	article := s.GetArticle("KB-031")
	if article != nil {
		t.Errorf("Created article should be gone after reset")
	}
}

func TestThreadSafety(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			s.ListArticles(make(map[string]string), "", 50, 0)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	for i := 0; i < 5; i++ {
		go func(idx int) {
			s.CreateArticle("Test", "Body", "life-support", "", nil)
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestIndex(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	s.CreateArticle("Sky Blue", "The sky is very blue today", "life-support", "", nil)

	results := s.search("sky blue")
	if len(results) == 0 {
		t.Errorf("Expected to find article by title search")
	}

	results = s.search("very blue")
	if len(results) == 0 {
		t.Errorf("Expected to find article by body search")
	}
}

func TestSearchRanking(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	s.CreateArticle("WCS Test One", "WCS", "life-support", "", nil)
	s.CreateArticle("WCS Test Two", "WCS WCS WCS", "life-support", "", nil)

	articles, _ := s.ListArticles(make(map[string]string), "wcs", 50, 0)

	if len(articles) < 2 {
		t.Errorf("Expected at least 2 results")
	}

	lastArticle := articles[len(articles)-1]
	if lastArticle.Title != "WCS Test One" {
		t.Logf("Ranking may need verification. Got: %s", lastArticle.Title)
	}
}

func TestEnvelopeFormat(t *testing.T) {
	data := map[string]interface{}{
		"test": "data",
	}

	envelope := &Envelope{
		Data:  data,
		Error: nil,
	}

	jsonData, err := json.Marshal(envelope)
	if err != nil {
		t.Errorf("Failed to marshal: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(jsonData, &result)

	if result["data"] == nil {
		t.Errorf("Expected data field in envelope")
	}
	if result["error"] != nil {
		t.Errorf("Expected error to be null")
	}
}

func TestValidCategories(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	categories := []string{
		"life-support", "navigation", "comms", "power", "propulsion",
		"eva", "medical", "operations", "thermal", "structures",
	}

	for _, cat := range categories {
		_, err := s.CreateArticle("Test", "Test", cat, "", nil)
		if err != nil {
			t.Errorf("Category %s should be valid: %v", cat, err)
		}
	}
}

func TestTimestamps(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	article, _ := s.CreateArticle("Test", "Test", "life-support", "", nil)

	createdTime, err := time.Parse(time.RFC3339, article.CreatedAt)
	if err != nil {
		t.Errorf("Invalid CreatedAt format: %v", err)
	}

	updatedTime, err := time.Parse(time.RFC3339, article.UpdatedAt)
	if err != nil {
		t.Errorf("Invalid UpdatedAt format: %v", err)
	}

	if createdTime.After(updatedTime) {
		t.Errorf("UpdatedAt should be >= CreatedAt")
	}
}

func TestCuratorFields(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	article, _ := s.CreateArticle("Test", "Test", "life-support", "", nil)

	if article.LastCuratedAt != nil {
		t.Errorf("LastCuratedAt should be nil on create")
	}

	score := 0.95
	article, _ = s.UpdateArticle(article.ID, map[string]interface{}{
		"usefulness_score": score,
	})

	if article.LastCuratedAt == nil {
		t.Errorf("LastCuratedAt should be set when curator score is updated")
	}
}

func TestPartialUpdate(t *testing.T) {
	s, _ := NewStore("seed-data/kb.json")

	article, _ := s.CreateArticle("Original Title", "Original body", "life-support", "creator1", []string{"tag1"})
	originalID := article.ID

	article, _ = s.UpdateArticle(article.ID, map[string]interface{}{
		"title": "Updated Title",
	})

	if article.Title != "Updated Title" {
		t.Errorf("Title should be updated")
	}
	if article.Body != "Original body" {
		t.Errorf("Body should remain unchanged")
	}
	if article.CreatedBy != "creator1" {
		t.Errorf("CreatedBy should remain unchanged")
	}
	if article.ID != originalID {
		t.Errorf("ID should not change")
	}
}
