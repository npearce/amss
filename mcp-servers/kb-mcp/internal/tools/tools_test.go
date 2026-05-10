package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/npearce/amss/mcp-servers/kb-mcp/internal/client"
)

// mockKBClient satisfies ClientInterface with injectable functions.
type mockKBClient struct {
	searchFn  func(category, tags, search string, limit, offset int) ([]*client.Article, int, error)
	getFn     func(id string) (*client.Article, error)
	createFn  func(title, body, category, createdBy string, tags []string) (*client.Article, error)
	updateFn  func(id string, updates map[string]interface{}) (*client.Article, error)
}

func (m *mockKBClient) Search(category, tags, search string, limit, offset int) ([]*client.Article, int, error) {
	if m.searchFn != nil {
		return m.searchFn(category, tags, search, limit, offset)
	}
	return []*client.Article{}, 0, nil
}
func (m *mockKBClient) GetArticle(id string) (*client.Article, error) {
	if m.getFn != nil {
		return m.getFn(id)
	}
	return nil, fmt.Errorf("not found")
}
func (m *mockKBClient) CreateArticle(title, body, category, createdBy string, tags []string) (*client.Article, error) {
	if m.createFn != nil {
		return m.createFn(title, body, category, createdBy, tags)
	}
	return nil, fmt.Errorf("create not implemented")
}
func (m *mockKBClient) UpdateArticle(id string, updates map[string]interface{}) (*client.Article, error) {
	if m.updateFn != nil {
		return m.updateFn(id, updates)
	}
	return nil, fmt.Errorf("update not implemented")
}

func withMock(t *testing.T, mock *mockKBClient) {
	t.Helper()
	SetClient(mock)
	t.Cleanup(func() { SetClient(nil) })
}

func callText[I, O any](t *testing.T, tool MCPTool[I, O], args I) (string, error) {
	t.Helper()
	result, err := tool.Handler(context.Background(), nil, &mcp.CallToolParamsFor[I]{Arguments: args})
	if err != nil {
		return "", err
	}
	if len(result.Content) == 0 {
		return "", nil
	}
	tc, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0] is not *mcp.TextContent")
	}
	return tc.Text, nil
}

// --- search_kb ---

func TestSearchKBSuccess(t *testing.T) {
	withMock(t, &mockKBClient{
		searchFn: func(category, tags, search string, limit, offset int) ([]*client.Article, int, error) {
			return []*client.Article{
				{ID: "KB-001", Title: "ECLSS Overview", Category: "life-support", Tags: []string{"eclss"}},
				{ID: "KB-002", Title: "CO2 Scrubbers", Category: "life-support"},
			}, 2, nil
		},
	})
	text, err := callText(t, SearchKB(), SearchKBParams{Search: "eclss"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "KB-001") {
		t.Errorf("missing KB-001 in result:\n%s", text)
	}
	if !strings.Contains(text, "KB-002") {
		t.Errorf("missing KB-002 in result:\n%s", text)
	}
	if !strings.Contains(text, "2") {
		t.Errorf("missing total count in result:\n%s", text)
	}
}

func TestSearchKBEmpty(t *testing.T) {
	withMock(t, &mockKBClient{
		searchFn: func(_, _, _ string, _, _ int) ([]*client.Article, int, error) {
			return []*client.Article{}, 0, nil
		},
	})
	text, err := callText(t, SearchKB(), SearchKBParams{Search: "nothing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "No articles found") {
		t.Errorf("expected 'No articles found': %s", text)
	}
}

func TestSearchKBDefaultLimit(t *testing.T) {
	var capturedLimit int
	withMock(t, &mockKBClient{
		searchFn: func(_, _, _ string, limit, _ int) ([]*client.Article, int, error) {
			capturedLimit = limit
			return []*client.Article{}, 0, nil
		},
	})
	callText(t, SearchKB(), SearchKBParams{}) // no limit set
	if capturedLimit != 50 {
		t.Errorf("default limit = %d, want 50", capturedLimit)
	}
}

func TestSearchKBPassesParams(t *testing.T) {
	var capturedCategory, capturedSearch string
	withMock(t, &mockKBClient{
		searchFn: func(category, _, search string, _, _ int) ([]*client.Article, int, error) {
			capturedCategory = category
			capturedSearch = search
			return []*client.Article{}, 0, nil
		},
	})
	callText(t, SearchKB(), SearchKBParams{Category: "medical", Search: "surgery"})
	if capturedCategory != "medical" {
		t.Errorf("category = %q, want medical", capturedCategory)
	}
	if capturedSearch != "surgery" {
		t.Errorf("search = %q, want surgery", capturedSearch)
	}
}

func TestSearchKBClientError(t *testing.T) {
	withMock(t, &mockKBClient{
		searchFn: func(_, _, _ string, _, _ int) ([]*client.Article, int, error) {
			return nil, 0, fmt.Errorf("store unavailable")
		},
	})
	_, err := callText(t, SearchKB(), SearchKBParams{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "store unavailable") {
		t.Errorf("error = %v", err)
	}
}

func TestSearchKBNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, SearchKB(), SearchKBParams{})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- read_kb_article ---

func TestReadKBArticleSuccess(t *testing.T) {
	withMock(t, &mockKBClient{
		getFn: func(id string) (*client.Article, error) {
			if id != "KB-001" {
				t.Errorf("id = %q, want KB-001", id)
			}
			return &client.Article{
				ID:       "KB-001",
				Title:    "ECLSS Overview",
				Body:     "Full body content.",
				Category: "life-support",
			}, nil
		},
	})
	text, err := callText(t, ReadKBArticle(), ReadKBArticleParams{ID: "KB-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"KB-001", "ECLSS Overview", "Full body content."} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in result:\n%s", want, text)
		}
	}
}

func TestReadKBArticleMissingID(t *testing.T) {
	withMock(t, &mockKBClient{})
	_, err := callText(t, ReadKBArticle(), ReadKBArticleParams{ID: ""})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestReadKBArticleNotFound(t *testing.T) {
	withMock(t, &mockKBClient{
		getFn: func(id string) (*client.Article, error) {
			return nil, fmt.Errorf("NOT_FOUND: article KB-999 not found")
		},
	})
	_, err := callText(t, ReadKBArticle(), ReadKBArticleParams{ID: "KB-999"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadKBArticleNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, ReadKBArticle(), ReadKBArticleParams{ID: "KB-001"})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- create_kb_article ---

func TestCreateKBArticleSuccess(t *testing.T) {
	withMock(t, &mockKBClient{
		createFn: func(title, body, category, createdBy string, tags []string) (*client.Article, error) {
			return &client.Article{
				ID:       "KB-031",
				Title:    title,
				Body:     body,
				Category: category,
				Tags:     tags,
			}, nil
		},
	})
	text, err := callText(t, CreateKBArticle(), CreateKBArticleParams{
		Title:    "New Procedure",
		Body:     "Step-by-step instructions.",
		Category: "operations",
		Tags:     []string{"procedure"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "KB-031") {
		t.Errorf("missing KB-031 in result:\n%s", text)
	}
	if !strings.Contains(text, "Created article") {
		t.Errorf("missing 'Created article' in result:\n%s", text)
	}
}

func TestCreateKBArticleDefaultCreatedBy(t *testing.T) {
	var capturedCreatedBy string
	withMock(t, &mockKBClient{
		createFn: func(_, _, _, createdBy string, _ []string) (*client.Article, error) {
			capturedCreatedBy = createdBy
			return &client.Article{ID: "KB-032"}, nil
		},
	})
	callText(t, CreateKBArticle(), CreateKBArticleParams{
		Title: "T", Body: "B", Category: "medical",
	})
	if capturedCreatedBy != "ground-control" {
		t.Errorf("created_by = %q, want ground-control", capturedCreatedBy)
	}
}

func TestCreateKBArticleMissingTitle(t *testing.T) {
	withMock(t, &mockKBClient{})
	_, err := callText(t, CreateKBArticle(), CreateKBArticleParams{Body: "B", Category: "medical"})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestCreateKBArticleMissingBody(t *testing.T) {
	withMock(t, &mockKBClient{})
	_, err := callText(t, CreateKBArticle(), CreateKBArticleParams{Title: "T", Category: "medical"})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
}

func TestCreateKBArticleMissingCategory(t *testing.T) {
	withMock(t, &mockKBClient{})
	_, err := callText(t, CreateKBArticle(), CreateKBArticleParams{Title: "T", Body: "B"})
	if err == nil {
		t.Fatal("expected error for missing category")
	}
}

func TestCreateKBArticleClientError(t *testing.T) {
	withMock(t, &mockKBClient{
		createFn: func(_, _, _, _ string, _ []string) (*client.Article, error) {
			return nil, fmt.Errorf("BAD_REQUEST: invalid category")
		},
	})
	_, err := callText(t, CreateKBArticle(), CreateKBArticleParams{Title: "T", Body: "B", Category: "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateKBArticleNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	_, err := callText(t, CreateKBArticle(), CreateKBArticleParams{Title: "T", Body: "B", Category: "medical"})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- update_kb_article ---

func TestUpdateKBArticleSuccess(t *testing.T) {
	withMock(t, &mockKBClient{
		updateFn: func(id string, updates map[string]interface{}) (*client.Article, error) {
			if id != "KB-001" {
				t.Errorf("id = %q, want KB-001", id)
			}
			return &client.Article{ID: "KB-001", Title: "Updated Title", Category: "life-support"}, nil
		},
	})
	title := "Updated Title"
	text, err := callText(t, UpdateKBArticle(), UpdateKBArticleParams{
		ID:    "KB-001",
		Title: &title,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(text, "Updated article KB-001") {
		t.Errorf("result:\n%s", text)
	}
}

func TestUpdateKBArticleMissingID(t *testing.T) {
	withMock(t, &mockKBClient{})
	_, err := callText(t, UpdateKBArticle(), UpdateKBArticleParams{})
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestUpdateKBArticleNoFields(t *testing.T) {
	withMock(t, &mockKBClient{})
	_, err := callText(t, UpdateKBArticle(), UpdateKBArticleParams{ID: "KB-001"})
	if err == nil {
		t.Fatal("expected error when no fields to update")
	}
}

func TestUpdateKBArticleOnlySuppliedFields(t *testing.T) {
	var capturedUpdates map[string]interface{}
	withMock(t, &mockKBClient{
		updateFn: func(_ string, updates map[string]interface{}) (*client.Article, error) {
			capturedUpdates = updates
			return &client.Article{ID: "KB-001"}, nil
		},
	})
	score := 0.9
	callText(t, UpdateKBArticle(), UpdateKBArticleParams{
		ID:              "KB-001",
		UsefulnessScore: &score,
	})
	if _, ok := capturedUpdates["usefulness_score"]; !ok {
		t.Error("usefulness_score not in updates")
	}
	if _, ok := capturedUpdates["title"]; ok {
		t.Error("title should not be in updates when not supplied")
	}
}

func TestUpdateKBArticleCuratorFields(t *testing.T) {
	var capturedUpdates map[string]interface{}
	withMock(t, &mockKBClient{
		updateFn: func(_ string, updates map[string]interface{}) (*client.Article, error) {
			capturedUpdates = updates
			return &client.Article{ID: "KB-001"}, nil
		},
	})
	dup := "KB-002"
	notes := "flagged as duplicate"
	callText(t, UpdateKBArticle(), UpdateKBArticleParams{
		ID:           "KB-001",
		DuplicateOf:  &dup,
		CuratorNotes: &notes,
	})
	if capturedUpdates["duplicate_of"] != "KB-002" {
		t.Errorf("duplicate_of = %v", capturedUpdates["duplicate_of"])
	}
	if capturedUpdates["curator_notes"] != "flagged as duplicate" {
		t.Errorf("curator_notes = %v", capturedUpdates["curator_notes"])
	}
}

func TestUpdateKBArticleClientError(t *testing.T) {
	withMock(t, &mockKBClient{
		updateFn: func(_ string, _ map[string]interface{}) (*client.Article, error) {
			return nil, fmt.Errorf("NOT_FOUND: article KB-999 not found")
		},
	})
	title := "x"
	_, err := callText(t, UpdateKBArticle(), UpdateKBArticleParams{ID: "KB-999", Title: &title})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdateKBArticleNilClient(t *testing.T) {
	SetClient(nil)
	t.Cleanup(func() { SetClient(nil) })
	title := "x"
	_, err := callText(t, UpdateKBArticle(), UpdateKBArticleParams{ID: "KB-001", Title: &title})
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

// --- list_kb_categories ---

func TestListKBCategories(t *testing.T) {
	// No client needed — static response.
	text, err := callText(t, ListKBCategories(), ListKBCategoriesParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{
		"life-support", "navigation", "comms", "power", "propulsion",
		"eva", "medical", "operations", "thermal", "structures",
	}
	for _, cat := range expected {
		if !strings.Contains(text, cat) {
			t.Errorf("missing category %q in:\n%s", cat, text)
		}
	}
}

func TestListKBCategories10Items(t *testing.T) {
	text, _ := callText(t, ListKBCategories(), ListKBCategoriesParams{})
	count := 0
	for _, cat := range validCategories {
		if strings.Contains(text, cat) {
			count++
		}
	}
	if count != 10 {
		t.Errorf("found %d of 10 categories in output:\n%s", count, text)
	}
}
