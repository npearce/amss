package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ─── LoadScenarios ───────────────────────────────────────────

func TestLoadScenarios_Valid(t *testing.T) {
	f := writeTempJSON(t, `[
		{"name":"s1","steps":[{"method":"GET","path":"/tickets","description":"list"}]}
	]`)
	scenarios, err := LoadScenarios(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 1 {
		t.Fatalf("want 1 scenario, got %d", len(scenarios))
	}
	if scenarios[0].Name != "s1" {
		t.Errorf("want name s1, got %s", scenarios[0].Name)
	}
	if len(scenarios[0].Steps) != 1 {
		t.Errorf("want 1 step, got %d", len(scenarios[0].Steps))
	}
}

func TestLoadScenarios_MultipleScenarios(t *testing.T) {
	f := writeTempJSON(t, `[
		{"name":"a","steps":[]},
		{"name":"b","steps":[]},
		{"name":"c","steps":[]}
	]`)
	scenarios, err := LoadScenarios(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenarios) != 3 {
		t.Errorf("want 3 scenarios, got %d", len(scenarios))
	}
}

func TestLoadScenarios_NotFound(t *testing.T) {
	_, err := LoadScenarios("/nonexistent/path/scenarios.json")
	if err == nil {
		t.Fatal("want error for missing file, got nil")
	}
}

func TestLoadScenarios_InvalidJSON(t *testing.T) {
	f := writeTempJSON(t, `not json`)
	_, err := LoadScenarios(f)
	if err == nil {
		t.Fatal("want error for invalid JSON, got nil")
	}
}

func TestLoadScenarios_RealFile(t *testing.T) {
	scenarios, err := LoadScenarios("scenarios.json")
	if err != nil {
		t.Fatalf("unexpected error loading real scenarios.json: %v", err)
	}
	if len(scenarios) == 0 {
		t.Fatal("expected at least one scenario in scenarios.json")
	}
	for i, s := range scenarios {
		if s.Name == "" {
			t.Errorf("scenario %d has empty name", i)
		}
		if len(s.Steps) == 0 {
			t.Errorf("scenario %q has no steps", s.Name)
		}
	}
}

// ─── resolveTemplates ────────────────────────────────────────

func TestResolveTemplates_SingleKey(t *testing.T) {
	result := resolveTemplates("/tickets/{{prev.id}}", map[string]interface{}{
		"id": "AMSS-042",
	})
	if result != "/tickets/AMSS-042" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplates_MultipleKeys(t *testing.T) {
	result := resolveTemplates("/{{prev.type}}/{{prev.id}}", map[string]interface{}{
		"id":   "KB-007",
		"type": "articles",
	})
	if result != "/articles/KB-007" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplates_NoPlaceholders(t *testing.T) {
	result := resolveTemplates("/tickets", map[string]interface{}{"id": "X"})
	if result != "/tickets" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplates_NilData(t *testing.T) {
	result := resolveTemplates("/tickets/{{prev.id}}", nil)
	if result != "/tickets/{{prev.id}}" {
		t.Errorf("got %q", result)
	}
}

func TestResolveTemplates_UnknownKey(t *testing.T) {
	result := resolveTemplates("/tickets/{{prev.id}}", map[string]interface{}{
		"title": "something",
	})
	if result != "/tickets/{{prev.id}}" {
		t.Errorf("placeholder should remain unresolved, got %q", result)
	}
}

// ─── ExecuteStep ─────────────────────────────────────────────

func TestExecuteStep_GET_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/tickets" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  map[string]interface{}{"total": 5},
			"error": nil,
		})
	}))
	defer srv.Close()

	step := Step{Method: "GET", Path: "/tickets", Description: "list tickets"}
	result, err := ExecuteStep(srv.Client(), srv.URL, step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StatusCode != http.StatusOK {
		t.Errorf("want 200, got %d", result.StatusCode)
	}
	if result.Data == nil {
		t.Error("expected data map to be populated")
	}
}

func TestExecuteStep_POST_WithBody(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":  map[string]interface{}{"id": "AMSS-099", "title": "Test"},
			"error": nil,
		})
	}))
	defer srv.Close()

	step := Step{
		Method:      "POST",
		Path:        "/tickets",
		Body:        map[string]interface{}{"title": "Test", "severity": "P3"},
		Description: "create ticket",
	}
	result, err := ExecuteStep(srv.Client(), srv.URL, step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("want 200, got %d", result.StatusCode)
	}
	if result.Data["id"] != "AMSS-099" {
		t.Errorf("want id AMSS-099, got %v", result.Data["id"])
	}
	if received["title"] != "Test" {
		t.Errorf("want title Test in body, got %v", received["title"])
	}
}

func TestExecuteStep_TemplateInPath(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{"id": "AMSS-001"}, "error": nil,
		})
	}))
	defer srv.Close()

	step := Step{Method: "PATCH", Path: "/tickets/{{prev.id}}", Body: map[string]interface{}{"status": "closed"}}
	result, err := ExecuteStep(srv.Client(), srv.URL, step, map[string]interface{}{"id": "AMSS-001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/tickets/AMSS-001" {
		t.Errorf("want /tickets/AMSS-001, got %s", capturedPath)
	}
	if result.ResolvedPath != "/tickets/AMSS-001" {
		t.Errorf("want resolved path /tickets/AMSS-001, got %s", result.ResolvedPath)
	}
}

func TestExecuteStep_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"data":null,"error":{"code":"NOT_FOUND"}}`, http.StatusNotFound)
	}))
	defer srv.Close()

	step := Step{Method: "GET", Path: "/tickets/AMSS-999"}
	result, err := ExecuteStep(srv.Client(), srv.URL, step, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", result.StatusCode)
	}
}

func TestExecuteStep_NetworkError(t *testing.T) {
	// Point at a port nothing is listening on
	step := Step{Method: "GET", Path: "/tickets"}
	_, err := ExecuteStep(http.DefaultClient, "http://127.0.0.1:1", step, nil)
	if err == nil {
		t.Fatal("want network error, got nil")
	}
}

// ─── RunScenario ─────────────────────────────────────────────

func TestRunScenario_MultiStep_TemplateChain(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{"id": "AMSS-123"}, "error": nil,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{"id": "AMSS-123", "status": "closed"}, "error": nil,
			})
		}
	}))
	defer srv.Close()

	scenario := Scenario{
		Name: "create then close",
		Steps: []Step{
			{Method: "POST", Path: "/tickets", Body: map[string]interface{}{"title": "t"}, Description: "create"},
			{Method: "PATCH", Path: "/tickets/{{prev.id}}", Body: map[string]interface{}{"status": "closed"}, Description: "close"},
		},
	}
	RunScenario(srv.Client(), srv.URL, scenario)
	if callCount != 2 {
		t.Errorf("want 2 HTTP calls, got %d", callCount)
	}
}

func TestRunScenario_StopsOnError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	scenario := Scenario{
		Name: "should stop after first failure",
		Steps: []Step{
			{Method: "POST", Path: "/tickets", Body: map[string]interface{}{"title": "t"}, Description: "create"},
			{Method: "PATCH", Path: "/tickets/{{prev.id}}", Body: map[string]interface{}{"status": "closed"}, Description: "should not run"},
		},
	}
	RunScenario(srv.Client(), srv.URL, scenario)
	if callCount != 1 {
		t.Errorf("want 1 HTTP call (stop on error), got %d", callCount)
	}
}

// ─── helpers ────────────────────────────────────────────────

func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return filepath.Clean(f.Name())
}
