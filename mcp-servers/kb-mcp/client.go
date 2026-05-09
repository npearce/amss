package main

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

type Envelope struct {
	Data  interface{} `json:"data"`
	Error *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type KBClient struct {
	baseURL string
}

func NewKBClient(baseURL string) *KBClient {
	return &KBClient{baseURL: baseURL}
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

	url := fmt.Sprintf("%s/articles?%s", c.baseURL, params.Encode())
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, 0, err
	}

	if envelope.Error != nil {
		return nil, 0, fmt.Errorf("API error: %s", envelope.Error.Message)
	}

	data, ok := envelope.Data.(map[string]interface{})
	if !ok {
		return nil, 0, fmt.Errorf("unexpected response format")
	}

	articlesRaw, _ := json.Marshal(data["articles"])
	var articles []*Article
	json.Unmarshal(articlesRaw, &articles)

	total := int(data["total"].(float64))

	return articles, total, nil
}

func (c *KBClient) GetArticle(id string) (*Article, error) {
	url := fmt.Sprintf("%s/articles/%s", c.baseURL, id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	if envelope.Error != nil {
		return nil, fmt.Errorf("article not found")
	}

	articleRaw, _ := json.Marshal(envelope.Data)
	var article Article
	json.Unmarshal(articleRaw, &article)

	return &article, nil
}

func (c *KBClient) CreateArticle(title, body, category, createdBy string, tags []string) (*Article, error) {
	payload := map[string]interface{}{
		"title":      title,
		"body":       body,
		"category":   category,
		"created_by": createdBy,
		"tags":       tags,
	}

	payloadBytes, _ := json.Marshal(payload)
	resp, err := http.Post(
		fmt.Sprintf("%s/articles", c.baseURL),
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var envelope Envelope
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		return nil, err
	}

	if envelope.Error != nil {
		return nil, fmt.Errorf("creation failed: %s", envelope.Error.Message)
	}

	articleRaw, _ := json.Marshal(envelope.Data)
	var article Article
	json.Unmarshal(articleRaw, &article)

	return &article, nil
}

func (c *KBClient) UpdateArticle(id string, updates map[string]interface{}) (*Article, error) {
	payloadBytes, _ := json.Marshal(updates)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/articles/%s", c.baseURL, id), bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var envelope Envelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}

	if envelope.Error != nil {
		return nil, fmt.Errorf("update failed: %s", envelope.Error.Message)
	}

	articleRaw, _ := json.Marshal(envelope.Data)
	var article Article
	json.Unmarshal(articleRaw, &article)

	return &article, nil
}

func formatArticle(a *Article) string {
	tags := strings.Join(a.Tags, ", ")
	if tags == "" {
		tags = "(none)"
	}
	curatorTags := strings.Join(a.CuratorTags, ", ")
	if curatorTags == "" {
		curatorTags = "(none)"
	}

	result := fmt.Sprintf("**%s: %s**\n\n", a.ID, a.Title)
	result += fmt.Sprintf("Category: %s\n", a.Category)
	result += fmt.Sprintf("Tags: %s\n", tags)
	result += fmt.Sprintf("Curator Tags: %s\n", curatorTags)
	result += fmt.Sprintf("Created: %s by %s\n", a.CreatedAt, a.CreatedBy)
	result += fmt.Sprintf("Updated: %s\n", a.UpdatedAt)
	result += fmt.Sprintf("Reference Count: %d\n\n", a.ReferenceCount)

	if a.UsefulnessScore != nil {
		result += fmt.Sprintf("Usefulness Score: %.2f\n", *a.UsefulnessScore)
	}
	if a.DuplicateOf != nil {
		result += fmt.Sprintf("Marked as duplicate of: %s\n", *a.DuplicateOf)
	}
	if a.CuratorNotes != nil {
		result += fmt.Sprintf("Curator Notes: %s\n", *a.CuratorNotes)
	}

	result += fmt.Sprintf("\n%s", a.Body)

	return result
}
