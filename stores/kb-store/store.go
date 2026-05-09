package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Article struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	Body                string    `json:"body"`
	Category            string    `json:"category"`
	Tags                []string  `json:"tags"`
	CreatedBy           string    `json:"created_by"`
	CreatedAt           string    `json:"created_at"`
	UpdatedAt           string    `json:"updated_at"`
	ReferencedByTickets []string  `json:"referenced_by_tickets"`
	ReferenceCount      int       `json:"reference_count"`
	UsefulnessScore     *float64  `json:"usefulness_score"`
	DuplicateOf         *string   `json:"duplicate_of"`
	CuratorNotes        *string   `json:"curator_notes"`
	CuratorTags         []string  `json:"curator_tags"`
	LastCuratedAt       *string   `json:"last_curated_at"`
}

type Store struct {
	mu              sync.RWMutex
	articles        map[string]*Article
	index           map[string]map[string]bool
	seedPath        string
	validCategories map[string]bool
	nextID          int
}

func NewStore(seedPath string) (*Store, error) {
	s := &Store{
		articles:   make(map[string]*Article),
		index:      make(map[string]map[string]bool),
		seedPath:   seedPath,
		nextID:     1,
		validCategories: map[string]bool{
			"life-support": true,
			"navigation":   true,
			"comms":        true,
			"power":        true,
			"propulsion":   true,
			"eva":          true,
			"medical":      true,
			"operations":   true,
			"thermal":      true,
			"structures":   true,
		},
	}

	if err := s.loadSeed(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) loadSeed() error {
	data, err := os.ReadFile(s.seedPath)
	if err != nil {
		return err
	}

	var seedData struct {
		Articles []*Article `json:"articles"`
	}

	if err := json.Unmarshal(data, &seedData); err != nil {
		return err
	}

	s.articles = make(map[string]*Article)
	s.index = make(map[string]map[string]bool)
	s.nextID = 1

	for _, article := range seedData.Articles {
		s.articles[article.ID] = article
		s.indexArticle(article)

		id := extractIDNumber(article.ID)
		if id >= s.nextID {
			s.nextID = id + 1
		}
	}

	return nil
}

func (s *Store) indexArticle(article *Article) {
	text := strings.ToLower(article.Title + " " + article.Body)
	tokens := tokenize(text)

	for _, token := range tokens {
		if s.index[token] == nil {
			s.index[token] = make(map[string]bool)
		}
		s.index[token][article.ID] = true
	}
}

func (s *Store) removeFromIndex(article *Article) {
	text := strings.ToLower(article.Title + " " + article.Body)
	tokens := tokenize(text)

	for _, token := range tokens {
		if s.index[token] != nil {
			delete(s.index[token], article.ID)
			if len(s.index[token]) == 0 {
				delete(s.index, token)
			}
		}
	}
}

func tokenize(text string) []string {
	re := regexp.MustCompile(`[^\w]+`)
	parts := re.Split(text, -1)

	var tokens []string
	for _, part := range parts {
		if part != "" && len(part) > 1 {
			tokens = append(tokens, part)
		}
	}
	return tokens
}

func (s *Store) search(query string) map[string]int {
	text := strings.ToLower(query)
	tokens := tokenize(text)

	results := make(map[string]int)

	for _, token := range tokens {
		if docs, exists := s.index[token]; exists {
			for docID := range docs {
				results[docID]++
			}
		}
	}

	return results
}

func (s *Store) GetArticle(id string) *Article {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.articles[id]
}

func (s *Store) ListArticles(filters map[string]string, search string, limit, offset int) ([]*Article, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var candidates []*Article

	for _, article := range s.articles {
		if !s.matchesFilters(article, filters) {
			continue
		}
		candidates = append(candidates, article)
	}

	if search != "" {
		searchResults := s.search(search)
		var filtered []*Article
		for _, article := range candidates {
			if _, matched := searchResults[article.ID]; matched {
				filtered = append(filtered, article)
			}
		}
		candidates = filtered

		sort.Slice(candidates, func(i, j int) bool {
			return searchResults[candidates[i].ID] > searchResults[candidates[j].ID]
		})
	} else {
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].ID < candidates[j].ID
		})
	}

	total := len(candidates)
	if offset > total {
		return []*Article{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return candidates[offset:end], total
}

func (s *Store) matchesFilters(article *Article, filters map[string]string) bool {
	if category, ok := filters["category"]; ok && article.Category != category {
		return false
	}

	if tags, ok := filters["tags"]; ok {
		tagList := strings.Split(tags, ",")
		if !s.matchesTags(article, tagList) {
			return false
		}
	}

	return true
}

func (s *Store) matchesTags(article *Article, filterTags []string) bool {
	allTags := append(article.Tags, article.CuratorTags...)
	filterMap := make(map[string]bool)
	for _, tag := range filterTags {
		filterMap[strings.TrimSpace(tag)] = true
	}

	for _, tag := range allTags {
		if filterMap[tag] {
			return true
		}
	}

	return len(filterTags) == 0
}

func (s *Store) CreateArticle(title, body, category, createdBy string, tags []string) (*Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if title == "" || body == "" {
		return nil, fmt.Errorf("title and body are required")
	}

	if !s.validCategories[category] {
		return nil, fmt.Errorf("invalid category: %s", category)
	}

	id := fmt.Sprintf("KB-%03d", s.nextID)
	s.nextID++

	if createdBy == "" {
		createdBy = "ground-control"
	}

	if tags == nil {
		tags = []string{}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	article := &Article{
		ID:                  id,
		Title:               title,
		Body:                body,
		Category:            category,
		Tags:                tags,
		CreatedBy:           createdBy,
		CreatedAt:           now,
		UpdatedAt:           now,
		ReferencedByTickets: []string{},
		ReferenceCount:      0,
		CuratorTags:         []string{},
	}

	s.articles[id] = article
	s.indexArticle(article)

	return article, nil
}

func (s *Store) UpdateArticle(id string, updates map[string]interface{}) (*Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	article, exists := s.articles[id]
	if !exists {
		return nil, fmt.Errorf("article not found")
	}

	s.removeFromIndex(article)

	curatorFieldsChanged := false

	if title, ok := updates["title"].(string); ok && title != "" {
		article.Title = title
	}
	if body, ok := updates["body"].(string); ok && body != "" {
		article.Body = body
	}
	if category, ok := updates["category"].(string); ok && category != "" {
		if !s.validCategories[category] {
			return nil, fmt.Errorf("invalid category: %s", category)
		}
		article.Category = category
	}
	if tags, ok := updates["tags"].([]interface{}); ok {
		article.Tags = interfaceSliceToStringSlice(tags)
	}
	if score, ok := updates["usefulness_score"].(float64); ok {
		article.UsefulnessScore = &score
		curatorFieldsChanged = true
	}
	if dupOf, ok := updates["duplicate_of"].(string); ok {
		if dupOf != "" {
			article.DuplicateOf = &dupOf
		} else {
			article.DuplicateOf = nil
		}
		curatorFieldsChanged = true
	}
	if notes, ok := updates["curator_notes"].(string); ok {
		if notes != "" {
			article.CuratorNotes = &notes
		} else {
			article.CuratorNotes = nil
		}
		curatorFieldsChanged = true
	}
	if curatorTags, ok := updates["curator_tags"].([]interface{}); ok {
		article.CuratorTags = interfaceSliceToStringSlice(curatorTags)
		if len(article.CuratorTags) > 0 {
			curatorFieldsChanged = true
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	article.UpdatedAt = now

	if curatorFieldsChanged {
		article.LastCuratedAt = &now
	}

	s.indexArticle(article)

	return article, nil
}

func (s *Store) DeleteArticle(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	article, exists := s.articles[id]
	if !exists {
		return fmt.Errorf("article not found")
	}

	s.removeFromIndex(article)
	delete(s.articles, id)

	return nil
}

func (s *Store) Reset() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadSeed(); err != nil {
		return 0, err
	}

	return len(s.articles), nil
}

func interfaceSliceToStringSlice(slice []interface{}) []string {
	var result []string
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func extractIDNumber(id string) int {
	parts := strings.Split(id, "-")
	if len(parts) < 2 {
		return 0
	}

	var num int
	fmt.Sscanf(parts[1], "%d", &num)
	return num
}
