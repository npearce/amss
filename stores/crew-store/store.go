package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var tokenizer = regexp.MustCompile(`[^\w]+`)

type CrewMember struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Mission   string `json:"mission"`
	Specialty string `json:"specialty"`
	Status    string `json:"status"`
	Persona   string `json:"persona"`
}

type Conversation struct {
	ID                   string   `json:"id"`
	CrewID               string   `json:"crew_id"`
	Mission              string   `json:"mission"`
	SessionID            string   `json:"session_id"`
	Timestamp            string   `json:"timestamp"`
	Query                string   `json:"query"`
	Response             string   `json:"response"`
	KBArticlesReferenced []string `json:"kb_articles_referenced"`
	TicketCreated        *string  `json:"ticket_created"`
}

type Store struct {
	mu            sync.RWMutex
	crew          map[string]*CrewMember
	conversations map[string]*Conversation
	crewIndex     map[string]map[string]bool
	convIndex     map[string]map[string]bool
	seedPath      string
	nextConvID    int
}

type seedData struct {
	Crew          []CrewMember   `json:"crew"`
	Conversations []Conversation `json:"conversations"`
}

func NewStore(seedPath string) (*Store, error) {
	s := &Store{seedPath: seedPath}
	if err := s.loadSeed(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) loadSeed() error {
	data, err := os.ReadFile(s.seedPath)
	if err != nil {
		return fmt.Errorf("reading seed file: %w", err)
	}

	var seed seedData
	if err := json.Unmarshal(data, &seed); err != nil {
		return fmt.Errorf("parsing seed file: %w", err)
	}

	s.crew = make(map[string]*CrewMember)
	s.conversations = make(map[string]*Conversation)
	s.crewIndex = make(map[string]map[string]bool)
	s.convIndex = make(map[string]map[string]bool)
	s.nextConvID = 1

	for i := range seed.Crew {
		m := seed.Crew[i]
		s.crew[m.ID] = &m
		s.indexCrewMember(&m)
	}

	for i := range seed.Conversations {
		c := seed.Conversations[i]
		if c.KBArticlesReferenced == nil {
			c.KBArticlesReferenced = []string{}
		}
		s.conversations[c.ID] = &c
		s.indexConversation(&c)
		if n := extractConvNumber(c.ID); n >= s.nextConvID {
			s.nextConvID = n + 1
		}
	}

	return nil
}

func (s *Store) indexCrewMember(m *CrewMember) {
	for _, word := range tokenizeText(m.Name + " " + m.Role + " " + m.Specialty) {
		if s.crewIndex[word] == nil {
			s.crewIndex[word] = make(map[string]bool)
		}
		s.crewIndex[word][m.ID] = true
	}
}

func (s *Store) indexConversation(c *Conversation) {
	for _, word := range tokenizeText(c.Query + " " + c.Response) {
		if s.convIndex[word] == nil {
			s.convIndex[word] = make(map[string]bool)
		}
		s.convIndex[word][c.ID] = true
	}
}

func tokenizeText(text string) []string {
	parts := tokenizer.Split(strings.ToLower(text), -1)
	result := parts[:0]
	for _, p := range parts {
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func extractConvNumber(id string) int {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(parts[1])
	return n
}

func matchesMissionFilter(m *CrewMember, mission string) bool {
	return m.Mission == mission || m.Mission == "all"
}

func (s *Store) GetCrewMember(id string) *CrewMember {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.crew[id]
}

func (s *Store) ListCrew(mission, search string, limit, offset int) ([]*CrewMember, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*CrewMember
	for _, m := range s.crew {
		if mission != "" && !matchesMissionFilter(m, mission) {
			continue
		}
		filtered = append(filtered, m)
	}

	if search != "" {
		tokens := tokenizeText(search)
		scores := make(map[string]int)
		for _, tok := range tokens {
			for id := range s.crewIndex[tok] {
				scores[id]++
			}
		}
		var searched []*CrewMember
		for _, m := range filtered {
			if scores[m.ID] > 0 {
				searched = append(searched, m)
			}
		}
		sort.Slice(searched, func(i, j int) bool {
			si, sj := scores[searched[i].ID], scores[searched[j].ID]
			if si != sj {
				return si > sj
			}
			return searched[i].ID < searched[j].ID
		})
		filtered = searched
	} else {
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].ID < filtered[j].ID
		})
	}

	total := len(filtered)
	if offset >= total {
		return []*CrewMember{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

func (s *Store) GetCrewActivity(crewID string, limit int) ([]*Conversation, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.crew[crewID]; !ok {
		return nil, 0, fmt.Errorf("crew member %s not found", crewID)
	}

	var convs []*Conversation
	for _, c := range s.conversations {
		if c.CrewID == crewID {
			convs = append(convs, c)
		}
	}

	sort.Slice(convs, func(i, j int) bool {
		return convs[i].Timestamp > convs[j].Timestamp
	})

	total := len(convs)
	if limit > total {
		limit = total
	}
	return convs[:limit], total, nil
}

func (s *Store) ListConversations(filters map[string]string, limit, offset int) ([]*Conversation, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*Conversation
	for _, c := range s.conversations {
		if crewID := filters["crew_id"]; crewID != "" && c.CrewID != crewID {
			continue
		}
		if mission := filters["mission"]; mission != "" && c.Mission != mission {
			continue
		}
		if sessionID := filters["session_id"]; sessionID != "" && c.SessionID != sessionID {
			continue
		}
		filtered = append(filtered, c)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID < filtered[j].ID
	})

	total := len(filtered)
	if offset >= total {
		return []*Conversation{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total
}

func (s *Store) CreateConversation(crewID, mission, sessionID, query, response string, kbArticles []string, ticketCreated *string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.crew[crewID]; !ok {
		return nil, fmt.Errorf("crew member %s not found", crewID)
	}

	if kbArticles == nil {
		kbArticles = []string{}
	}

	id := fmt.Sprintf("conv-%04d", s.nextConvID)
	s.nextConvID++

	c := &Conversation{
		ID:                   id,
		CrewID:               crewID,
		Mission:              mission,
		SessionID:            sessionID,
		Timestamp:            time.Now().UTC().Format(time.RFC3339),
		Query:                query,
		Response:             response,
		KBArticlesReferenced: kbArticles,
		TicketCreated:        ticketCreated,
	}

	s.conversations[id] = c
	s.indexConversation(c)
	return c, nil
}

func (s *Store) Reset() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadSeed(); err != nil {
		return 0, err
	}
	return len(s.crew), nil
}
