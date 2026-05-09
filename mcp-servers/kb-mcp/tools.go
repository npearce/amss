package main

import (
	"encoding/json"
	"fmt"
)

var client *KBClient

func getSearchTool() map[string]interface{} {
	return map[string]interface{}{
		"name":        "search_kb",
		"description": "Search knowledge base articles by title, body, category, or tags",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query (searches title and body)",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by category (life-support, navigation, comms, power, propulsion, eva, medical, operations, thermal, structures)",
				},
				"tags": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated tags to filter by",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Max results (default 50, max 200)",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"description": "Pagination offset (default 0)",
				},
			},
		},
	}
}

func getReadTool() map[string]interface{} {
	return map[string]interface{}{
		"name":        "read_kb",
		"description": "Read a specific KB article by ID",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Article ID (e.g., KB-001)",
				},
			},
			"required": []string{"id"},
		},
	}
}

func getCreateTool() map[string]interface{} {
	return map[string]interface{}{
		"name":        "create_kb",
		"description": "Create a new KB article",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Article title",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "Article body (markdown)",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Category (must be one of: life-support, navigation, comms, power, propulsion, eva, medical, operations, thermal, structures)",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Tags for the article",
				},
				"created_by": map[string]interface{}{
					"type":        "string",
					"description": "Creator ID (defaults to ground-control)",
				},
			},
			"required": []string{"title", "body", "category"},
		},
	}
}

func getUpdateTool() map[string]interface{} {
	return map[string]interface{}{
		"name":        "update_kb",
		"description": "Update a KB article (partial update)",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type":        "string",
					"description": "Article ID",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "New title",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "New body",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "New category",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "New tags",
				},
				"usefulness_score": map[string]interface{}{
					"type":        "number",
					"description": "Usefulness score (0.0-1.0, curator field)",
				},
				"duplicate_of": map[string]interface{}{
					"type":        "string",
					"description": "Mark as duplicate of this KB ID (curator field)",
				},
				"curator_notes": map[string]interface{}{
					"type":        "string",
					"description": "Curator notes (curator field)",
				},
				"curator_tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Curator tags (curator field)",
				},
			},
			"required": []string{"id"},
		},
	}
}

func handleSearchTool(args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	category, _ := args["category"].(string)
	tags, _ := args["tags"].(string)
	limit := 50
	offset := 0

	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	articles, total, err := client.Search(category, tags, query, limit, offset)
	if err != nil {
		return nil, err
	}

	result := fmt.Sprintf("Found %d articles (showing %d):\n\n", total, len(articles))
	for i, article := range articles {
		result += fmt.Sprintf("%d. **%s** - %s\n", i+1, article.ID, article.Title)
	}

	return result, nil
}

func handleReadTool(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing id")
	}

	article, err := client.GetArticle(id)
	if err != nil {
		return nil, err
	}

	return formatArticle(article), nil
}

func handleCreateTool(args map[string]interface{}) (interface{}, error) {
	title, _ := args["title"].(string)
	body, _ := args["body"].(string)
	category, _ := args["category"].(string)
	createdBy, _ := args["created_by"].(string)

	var tags []string
	if tagsRaw, ok := args["tags"].([]interface{}); ok {
		for _, t := range tagsRaw {
			if str, ok := t.(string); ok {
				tags = append(tags, str)
			}
		}
	}

	if title == "" || body == "" || category == "" {
		return nil, fmt.Errorf("title, body, and category are required")
	}

	article, err := client.CreateArticle(title, body, category, createdBy, tags)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Created article %s:\n\n%s", article.ID, formatArticle(article)), nil
}

func handleUpdateTool(args map[string]interface{}) (interface{}, error) {
	id, ok := args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing id")
	}

	updates := make(map[string]interface{})

	if v, ok := args["title"]; ok {
		updates["title"] = v
	}
	if v, ok := args["body"]; ok {
		updates["body"] = v
	}
	if v, ok := args["category"]; ok {
		updates["category"] = v
	}
	if v, ok := args["tags"]; ok {
		updates["tags"] = v
	}
	if v, ok := args["usefulness_score"]; ok {
		updates["usefulness_score"] = v
	}
	if v, ok := args["duplicate_of"]; ok {
		updates["duplicate_of"] = v
	}
	if v, ok := args["curator_notes"]; ok {
		updates["curator_notes"] = v
	}
	if v, ok := args["curator_tags"]; ok {
		updates["curator_tags"] = v
	}

	article, err := client.UpdateArticle(id, updates)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Updated article %s:\n\n%s", article.ID, formatArticle(article)), nil
}

func processToolCall(toolName string, args map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "search_kb":
		return handleSearchTool(args)
	case "read_kb":
		return handleReadTool(args)
	case "create_kb":
		return handleCreateTool(args)
	case "update_kb":
		return handleUpdateTool(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func getToolDefinitions() []map[string]interface{} {
	return []map[string]interface{}{
		getSearchTool(),
		getReadTool(),
		getCreateTool(),
		getUpdateTool(),
	}
}

func parseToolArgs(input interface{}) (map[string]interface{}, error) {
	switch v := input.(type) {
	case map[string]interface{}:
		return v, nil
	case string:
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(v), &args); err != nil {
			return nil, err
		}
		return args, nil
	default:
		return nil, fmt.Errorf("invalid arguments type")
	}
}
