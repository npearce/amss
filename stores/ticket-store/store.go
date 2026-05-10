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

var validSeverities = map[string]bool{
	"P1": true, "P2": true, "P3": true, "P4": true,
}

type Ticket struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	Severity             string    `json:"severity"`
	Status               string    `json:"status"`
	Category             string    `json:"category"`
	ReportedBy           string    `json:"reported_by"`
	AssignedTo           string    `json:"assigned_to"`
	Mission              string    `json:"mission"`
	CreatedAt            string    `json:"created_at"`
	UpdatedAt            string    `json:"updated_at"`
	ResolvedAt           *string   `json:"resolved_at"`
	KBArticlesReferenced []string  `json:"kb_articles_referenced"`
	Resolution           *string   `json:"resolution"`
	Comments             []Comment `json:"comments"`
}

type Comment struct {
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
	Text      string `json:"text"`
}

type Store struct {
	mu              sync.RWMutex
	tickets         map[string]*Ticket
	index           map[string]map[string]bool
	seedPath        string
	validCategories map[string]bool
	nextID          int
}

func NewStore(seedPath string) (*Store, error) {
	s := &Store{
		tickets:  make(map[string]*Ticket),
		index:    make(map[string]map[string]bool),
		seedPath: seedPath,
		nextID:   1,
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
		Tickets []*Ticket `json:"tickets"`
	}

	if err := json.Unmarshal(data, &seedData); err != nil {
		return err
	}

	s.tickets = make(map[string]*Ticket)
	s.index = make(map[string]map[string]bool)
	s.nextID = 1

	for _, ticket := range seedData.Tickets {
		if ticket.Comments == nil {
			ticket.Comments = []Comment{}
		}
		if ticket.KBArticlesReferenced == nil {
			ticket.KBArticlesReferenced = []string{}
		}
		s.tickets[ticket.ID] = ticket
		s.indexTicket(ticket)

		id := extractIDNumber(ticket.ID)
		if id >= s.nextID {
			s.nextID = id + 1
		}
	}

	return nil
}

func (s *Store) indexTicket(ticket *Ticket) {
	text := strings.ToLower(ticket.Title + " " + ticket.Description)
	tokens := tokenize(text)

	for _, token := range tokens {
		if s.index[token] == nil {
			s.index[token] = make(map[string]bool)
		}
		s.index[token][ticket.ID] = true
	}
}

func (s *Store) removeFromIndex(ticket *Ticket) {
	text := strings.ToLower(ticket.Title + " " + ticket.Description)
	tokens := tokenize(text)

	for _, token := range tokens {
		if s.index[token] != nil {
			delete(s.index[token], ticket.ID)
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

func (s *Store) GetTicket(id string) *Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tickets[id]
}

func (s *Store) ListTickets(filters map[string]string, search string, limit, offset int) ([]*Ticket, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var candidates []*Ticket

	for _, ticket := range s.tickets {
		if !s.matchesFilters(ticket, filters) {
			continue
		}
		candidates = append(candidates, ticket)
	}

	if search != "" {
		searchResults := s.search(search)
		var filtered []*Ticket
		for _, ticket := range candidates {
			if _, matched := searchResults[ticket.ID]; matched {
				filtered = append(filtered, ticket)
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
		return []*Ticket{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return candidates[offset:end], total
}

func (s *Store) matchesFilters(ticket *Ticket, filters map[string]string) bool {
	if mission, ok := filters["mission"]; ok && ticket.Mission != mission {
		return false
	}
	if severity, ok := filters["severity"]; ok && ticket.Severity != severity {
		return false
	}
	if status, ok := filters["status"]; ok && ticket.Status != status {
		return false
	}
	if category, ok := filters["category"]; ok && ticket.Category != category {
		return false
	}
	return true
}

func (s *Store) CreateTicket(title, description, severity, category, reportedBy, mission, assignedTo string, kbRefs []string) (*Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if title == "" || description == "" {
		return nil, fmt.Errorf("title and description are required")
	}
	if !validSeverities[severity] {
		return nil, fmt.Errorf("invalid severity: %s", severity)
	}
	if !s.validCategories[category] {
		return nil, fmt.Errorf("invalid category: %s", category)
	}
	if reportedBy == "" {
		return nil, fmt.Errorf("reported_by is required")
	}
	if mission == "" {
		return nil, fmt.Errorf("mission is required")
	}

	if assignedTo == "" {
		assignedTo = "ground-control"
	}
	if kbRefs == nil {
		kbRefs = []string{}
	}

	id := fmt.Sprintf("AMSS-%03d", s.nextID)
	s.nextID++

	now := time.Now().UTC().Format(time.RFC3339)

	ticket := &Ticket{
		ID:                   id,
		Title:                title,
		Description:          description,
		Severity:             severity,
		Status:               "open",
		Category:             category,
		ReportedBy:           reportedBy,
		AssignedTo:           assignedTo,
		Mission:              mission,
		CreatedAt:            now,
		UpdatedAt:            now,
		ResolvedAt:           nil,
		KBArticlesReferenced: kbRefs,
		Resolution:           nil,
		Comments:             []Comment{},
	}

	s.tickets[id] = ticket
	s.indexTicket(ticket)

	return ticket, nil
}

func (s *Store) UpdateTicket(id string, updates map[string]interface{}) (*Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, exists := s.tickets[id]
	if !exists {
		return nil, fmt.Errorf("ticket not found")
	}

	s.removeFromIndex(ticket)

	if title, ok := updates["title"].(string); ok && title != "" {
		ticket.Title = title
	}
	if description, ok := updates["description"].(string); ok && description != "" {
		ticket.Description = description
	}
	if severity, ok := updates["severity"].(string); ok && severity != "" {
		if !validSeverities[severity] {
			s.indexTicket(ticket)
			return nil, fmt.Errorf("invalid severity: %s", severity)
		}
		ticket.Severity = severity
	}
	if category, ok := updates["category"].(string); ok && category != "" {
		if !s.validCategories[category] {
			s.indexTicket(ticket)
			return nil, fmt.Errorf("invalid category: %s", category)
		}
		ticket.Category = category
	}
	if status, ok := updates["status"].(string); ok && status != "" {
		ticket.Status = status
	}
	if assignedTo, ok := updates["assigned_to"].(string); ok {
		ticket.AssignedTo = assignedTo
	}
	if mission, ok := updates["mission"].(string); ok && mission != "" {
		ticket.Mission = mission
	}
	if resolution, ok := updates["resolution"].(string); ok && resolution != "" {
		ticket.Resolution = &resolution
	}
	if kbRefs, ok := updates["kb_articles_referenced"].([]interface{}); ok {
		ticket.KBArticlesReferenced = interfaceSliceToStringSlice(kbRefs)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if status, ok := updates["status"].(string); ok && status == "resolved" && ticket.ResolvedAt == nil {
		resolvedAt := now
		ticket.ResolvedAt = &resolvedAt
	}

	ticket.UpdatedAt = now
	s.indexTicket(ticket)

	return ticket, nil
}

func (s *Store) AddComment(id, author, text string) (*Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, exists := s.tickets[id]
	if !exists {
		return nil, fmt.Errorf("ticket not found")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	comment := Comment{
		Author:    author,
		Timestamp: now,
		Text:      text,
	}

	ticket.Comments = append(ticket.Comments, comment)
	ticket.UpdatedAt = now

	return &comment, nil
}

func (s *Store) Reset() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadSeed(); err != nil {
		return 0, err
	}

	return len(s.tickets), nil
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
