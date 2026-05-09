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

type Article struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Body                string   `json:"body"`
	Category            string   `json:"category"`
	Tags                []string `json:"tags"`
	CreatedBy           string   `json:"created_by"`
	CreatedAt           string   `json:"created_at"`
	UpdatedAt           string   `json:"updated_at"`
	ReferencedByTickets []string `json:"referenced_by_tickets"`
	ReferenceCount      int      `json:"reference_count"`
	UsefulnessScore     *float64 `json:"usefulness_score"`
	DuplicateOf         *string  `json:"duplicate_of"`
	CuratorNotes        *string  `json:"curator_notes"`
	CuratorTags         []string `json:"curator_tags"`
	LastCuratedAt       *string  `json:"last_curated_at"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type KBClient struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string) *KBClient {
	return &KBClient{baseURL: baseURL, httpClient: http.DefaultClient}
}

func (c *KBClient) Search(category, tags, search string, limit, offset int) ([]*Article, int, error) {
	params := url.Values{}
	if category != "" {
		params.Set("category", category)
	}
	if tags != "" {
		params.Set("tags", tags)
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

	resp, err := c.httpClient.Get(fmt.Sprintf("%s/articles?%s", c.baseURL, params.Encode()))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var env struct {
		Data *struct {
			Articles []*Article `json:"articles"`
			Total    int        `json:"total"`
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
		return []*Article{}, 0, nil
	}
	articles := env.Data.Articles
	if articles == nil {
		articles = []*Article{}
	}
	return articles, env.Data.Total, nil
}

func (c *KBClient) GetArticle(id string) (*Article, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/articles/%s", c.baseURL, id))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var env struct {
		Data  *Article   `json:"data"`
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

func (c *KBClient) CreateArticle(title, body, category, createdBy string, tags []string) (*Article, error) {
	if tags == nil {
		tags = []string{}
	}
	payload := map[string]interface{}{
		"title":      title,
		"body":       body,
		"category":   category,
		"created_by": createdBy,
		"tags":       tags,
	}
	payloadBytes, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/articles", c.baseURL),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var env struct {
		Data  *Article   `json:"data"`
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

func (c *KBClient) UpdateArticle(id string, updates map[string]interface{}) (*Article, error) {
	payloadBytes, _ := json.Marshal(updates)
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/articles/%s", c.baseURL, id), bytes.NewReader(payloadBytes))
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
		Data  *Article   `json:"data"`
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

// FormatArticle renders a full article as a human-readable markdown-ish string.
func FormatArticle(a *Article) string {
	tags := strings.Join(a.Tags, ", ")
	if tags == "" {
		tags = "(none)"
	}
	curatorTags := strings.Join(a.CuratorTags, ", ")
	if curatorTags == "" {
		curatorTags = "(none)"
	}

	s := fmt.Sprintf("**%s: %s**\n\n", a.ID, a.Title)
	s += fmt.Sprintf("Category: %s\n", a.Category)
	s += fmt.Sprintf("Tags: %s\n", tags)
	s += fmt.Sprintf("Curator Tags: %s\n", curatorTags)
	s += fmt.Sprintf("Created: %s by %s\n", a.CreatedAt, a.CreatedBy)
	s += fmt.Sprintf("Updated: %s\n", a.UpdatedAt)
	s += fmt.Sprintf("Reference Count: %d\n", a.ReferenceCount)

	if a.UsefulnessScore != nil {
		s += fmt.Sprintf("Usefulness Score: %.2f\n", *a.UsefulnessScore)
	}
	if a.DuplicateOf != nil {
		s += fmt.Sprintf("Duplicate of: %s\n", *a.DuplicateOf)
	}
	if a.CuratorNotes != nil {
		s += fmt.Sprintf("Curator Notes: %s\n", *a.CuratorNotes)
	}
	s += fmt.Sprintf("\n%s", a.Body)
	return s
}
