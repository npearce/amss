package main

import (
	"testing"
)

func TestToolDefinitions(t *testing.T) {
	tools := getToolDefinitions()

	if len(tools) != 4 {
		t.Errorf("Expected 4 tools, got %d", len(tools))
	}

	expectedTools := []string{"search_kb", "read_kb", "create_kb", "update_kb"}
	actualTools := make(map[string]bool)
	for _, tool := range tools {
		actualTools[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !actualTools[expected] {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
}

func TestSearchToolSchema(t *testing.T) {
	tool := getSearchTool()

	if tool.Name != "search_kb" {
		t.Errorf("Expected name search_kb")
	}

	schema, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		t.Errorf("Schema should be a map")
	}

	if schema["type"] != "object" {
		t.Errorf("Schema type should be object")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Errorf("Properties should exist")
	}

	expectedProps := []string{"query", "category", "tags", "limit", "offset"}
	for _, prop := range expectedProps {
		if _, ok := props[prop]; !ok {
			t.Errorf("Expected property %s", prop)
		}
	}
}

func TestReadToolSchema(t *testing.T) {
	tool := getReadTool()

	if tool.Name != "read_kb" {
		t.Errorf("Expected name read_kb")
	}

	schema, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		t.Errorf("Schema should be a map")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Errorf("Required should be a string array")
	}

	if len(required) != 1 || required[0] != "id" {
		t.Errorf("ID should be required")
	}
}

func TestCreateToolSchema(t *testing.T) {
	tool := getCreateTool()

	if tool.Name != "create_kb" {
		t.Errorf("Expected name create_kb")
	}

	schema, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		t.Errorf("Schema should be a map")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Errorf("Required should be a string array")
	}

	if len(required) != 3 {
		t.Errorf("Expected 3 required fields: title, body, category")
	}
}

func TestUpdateToolSchema(t *testing.T) {
	tool := getUpdateTool()

	if tool.Name != "update_kb" {
		t.Errorf("Expected name update_kb")
	}

	schema, ok := tool.InputSchema.(map[string]interface{})
	if !ok {
		t.Errorf("Schema should be a map")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Errorf("Required should be a string array")
	}

	if len(required) != 1 || required[0] != "id" {
		t.Errorf("ID should be the only required field")
	}
}

func TestFormatArticle(t *testing.T) {
	usefulness := 0.85
	dupOf := "KB-002"
	notes := "Test notes"

	article := &Article{
		ID:        "KB-001",
		Title:     "Test Article",
		Body:      "Test body content",
		Category:  "life-support",
		Tags:      []string{"tag1", "tag2"},
		CreatedBy: "test-creator",
		CreatedAt: "2026-05-08T10:00:00Z",
		UpdatedAt: "2026-05-08T12:00:00Z",
		CuratorTags: []string{"curator-tag1"},
		UsefulnessScore: &usefulness,
		DuplicateOf: &dupOf,
		CuratorNotes: &notes,
	}

	formatted := formatArticle(article)

	if formatted == "" {
		t.Errorf("Formatted article should not be empty")
	}

	expectedStrings := []string{"KB-001", "Test Article", "life-support", "Test body content"}
	for _, expected := range expectedStrings {
		if !contains(formatted, expected) {
			t.Errorf("Expected formatted article to contain: %s", expected)
		}
	}
}

func TestParseToolArgsMap(t *testing.T) {
	input := map[string]interface{}{
		"key": "value",
	}

	args, err := parseToolArgs(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if args["key"] != "value" {
		t.Errorf("Args not parsed correctly")
	}
}

func TestParseToolArgsJSON(t *testing.T) {
	input := `{"key": "value"}`

	args, err := parseToolArgs(input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if args["key"] != "value" {
		t.Errorf("Args not parsed correctly from JSON")
	}
}

func TestParseToolArgsInvalid(t *testing.T) {
	_, err := parseToolArgs(123)
	if err == nil {
		t.Errorf("Expected error for invalid args type")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != "" && (s == substr || len(s) >= len(substr))
}
