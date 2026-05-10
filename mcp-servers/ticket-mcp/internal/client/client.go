package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

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

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TicketClient struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *TicketClient {
	return &TicketClient{baseURL: baseURL, httpClient: http.DefaultClient}
}

func (c *TicketClient) Search(mission, severity, status, category, search string, limit, offset int) ([]*Ticket, int, error) {
	params := url.Values{}
	if mission != "" {
		params.Set("mission", mission)
	}
	if severity != "" {
		params.Set("severity", severity)
	}
	if status != "" {
		params.Set("status", status)
	}
	if category != "" {
		params.Set("category", category)
	}
	if search != "" {
		params.Set("search", search)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tickets?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var env struct {
		Data *struct {
			Tickets []*Ticket `json:"tickets"`
			Total   int       `json:"total"`
		} `json:"data"`
		Error *ErrorInfo `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, err
	}
	if env.Error != nil {
		return nil, 0, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
	}
	if env.Data == nil {
		return []*Ticket{}, 0, nil
	}
	tickets := env.Data.Tickets
	if tickets == nil {
		tickets = []*Ticket{}
	}
	return tickets, env.Data.Total, nil
}

func (c *TicketClient) GetTicket(id string) (*Ticket, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/tickets/%s", c.baseURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var env struct {
		Data  *Ticket    `json:"data"`
		Error *ErrorInfo `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Error != nil {
		return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
	}
	return env.Data, nil
}

func (c *TicketClient) CreateTicket(title, description, severity, category, reportedBy, mission, assignedTo string, kbArticles []string) (*Ticket, error) {
	if kbArticles == nil {
		kbArticles = []string{}
	}
	payload := map[string]interface{}{
		"title":                  title,
		"description":            description,
		"severity":               severity,
		"category":               category,
		"reported_by":            reportedBy,
		"mission":                mission,
		"assigned_to":            assignedTo,
		"kb_articles_referenced": kbArticles,
	}
	payloadBytes, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/tickets", c.baseURL),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var env struct {
		Data  *Ticket    `json:"data"`
		Error *ErrorInfo `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &env); err != nil {
		return nil, err
	}
	if env.Error != nil {
		return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
	}
	return env.Data, nil
}

func (c *TicketClient) UpdateTicket(id string, updates map[string]interface{}) (*Ticket, error) {
	payloadBytes, _ := json.Marshal(updates)
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/tickets/%s", c.baseURL, id), bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var env struct {
		Data  *Ticket    `json:"data"`
		Error *ErrorInfo `json:"error"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	if env.Error != nil {
		return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
	}
	return env.Data, nil
}

func (c *TicketClient) AddComment(ticketID, author, text string) (*Comment, error) {
	payload := map[string]interface{}{
		"author": author,
		"text":   text,
	}
	payloadBytes, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/tickets/%s/comments", c.baseURL, ticketID),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var env struct {
		Data  *Comment   `json:"data"`
		Error *ErrorInfo `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &env); err != nil {
		return nil, err
	}
	if env.Error != nil {
		return nil, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
	}
	return env.Data, nil
}

// FormatTicket renders a full ticket as a human-readable string.
func FormatTicket(t *Ticket) string {
	s := fmt.Sprintf("**%s: %s**\n\n", t.ID, t.Title)
	s += fmt.Sprintf("Severity: %s | Status: %s\n", t.Severity, t.Status)
	s += fmt.Sprintf("Category: %s | Mission: %s\n", t.Category, t.Mission)
	s += fmt.Sprintf("Reported by: %s | Assigned to: %s\n", t.ReportedBy, t.AssignedTo)
	s += fmt.Sprintf("Created: %s | Updated: %s\n", t.CreatedAt, t.UpdatedAt)

	if t.ResolvedAt != nil {
		s += fmt.Sprintf("Resolved: %s\n", *t.ResolvedAt)
	}
	if t.Resolution != nil {
		s += fmt.Sprintf("Resolution: %s\n", *t.Resolution)
	}

	kbRefs := strings.Join(t.KBArticlesReferenced, ", ")
	if kbRefs == "" {
		kbRefs = "(none)"
	}
	s += fmt.Sprintf("KB References: %s\n", kbRefs)
	s += fmt.Sprintf("\n%s\n", t.Description)

	if len(t.Comments) > 0 {
		s += fmt.Sprintf("\n--- Comments (%d) ---\n", len(t.Comments))
		for _, c := range t.Comments {
			s += fmt.Sprintf("\n[%s] %s:\n%s\n", c.Timestamp, c.Author, c.Text)
		}
	}
	return s
}
