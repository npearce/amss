package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ─── LoadScenarios ───────────────────────────────────────────

func TestLoadScenarios_Valid(t *testing.T) {
	f := writeTempJSON(t, `[
		{"name":"s1","start_offset_seconds":0,"steps":[{"method":"GET","path":"/tickets","description":"list"}]}
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
	if scenarios[0].StartOffsetSeconds != 0 {
		t.Errorf("want start_offset_seconds 0, got %d", scenarios[0].StartOffsetSeconds)
	}
	if len(scenarios[0].Steps) != 1 {
		t.Errorf("want 1 step, got %d", len(scenarios[0].Steps))
	}
}

func TestLoadScenarios_StartOffsetAndTimingFields(t *testing.T) {
	f := writeTempJSON(t, `[{
		"name":"timed",
		"start_offset_seconds":120,
		"steps":[{
			"method":"POST",
			"path":"/api/v1/tickets",
			"description":"create",
			"crew_id":"koch-c",
			"pre_delay_min":2,
			"pre_delay_max":5,
			"post_delay_min":3,
			"post_delay_max":10,
			"stash_as":"ticket_id"
		}]
	}]`)
	scenarios, err := LoadScenarios(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := scenarios[0]
	if s.StartOffsetSeconds != 120 {
		t.Errorf("want start_offset_seconds 120, got %d", s.StartOffsetSeconds)
	}
	step := s.Steps[0]
	if step.CrewID != "koch-c" {
		t.Errorf("want crew_id koch-c, got %s", step.CrewID)
	}
	if step.PreDelayMin != 2 || step.PreDelayMax != 5 {
		t.Errorf("want pre_delay 2–5, got %d–%d", step.PreDelayMin, step.PreDelayMax)
	}
	if step.PostDelayMin != 3 || step.PostDelayMax != 10 {
		t.Errorf("want post_delay 3–10, got %d–%d", step.PostDelayMin, step.PostDelayMax)
	}
	if step.StashAs != "ticket_id" {
		t.Errorf("want stash_as ticket_id, got %s", step.StashAs)
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

func TestResolveTemplates_SessionID(t *testing.T) {
	result := resolveTemplates("{{prev.session_id}}", map[string]interface{}{
		"session_id": "sess-123",
	})
	if result != "sess-123" {
		t.Errorf("got %q", result)
	}
}

// ─── humanDelay ──────────────────────────────────────────────

func TestHumanDelay_ZeroRange(t *testing.T) {
	start := time.Now()
	humanDelay(context.Background(), 0, 0)
	if time.Since(start) > 50*time.Millisecond {
		t.Error("zero delay should return immediately")
	}
}

func TestHumanDelay_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	humanDelay(ctx, 60, 120)
	if time.Since(start) > 50*time.Millisecond {
		t.Error("cancelled context should return immediately, not sleep 60s")
	}
}

func TestHumanDelay_NegativeMin(t *testing.T) {
	start := time.Now()
	humanDelay(context.Background(), -5, 0)
	if time.Since(start) > 50*time.Millisecond {
		t.Error("negative min clamped to 0 should return immediately")
	}
}

func TestHumanDelay_MaxLessThanMin(t *testing.T) {
	start := time.Now()
	humanDelay(context.Background(), 0, -1)
	if time.Since(start) > 50*time.Millisecond {
		t.Error("max < min should be clamped to min and return immediately when both 0")
	}
}

// ─── formatCrewLabel ────────────────────────────────────────

func TestFormatCrewLabel_AstronautID(t *testing.T) {
	if got := formatCrewLabel("koch-c"); got != "Koch" {
		t.Errorf("want Koch, got %s", got)
	}
}

func TestFormatCrewLabel_GroundControlID(t *testing.T) {
	if got := formatCrewLabel("gc-eclss"); got != "Gc" {
		t.Errorf("want Gc, got %s", got)
	}
}

func TestFormatCrewLabel_Empty(t *testing.T) {
	if got := formatCrewLabel(""); got != "system" {
		t.Errorf("want system, got %s", got)
	}
}

func TestFormatCrewLabel_NoDash(t *testing.T) {
	if got := formatCrewLabel("wiseman"); got != "Wiseman" {
		t.Errorf("want Wiseman, got %s", got)
	}
}

// ─── generateSessionID ────────────────────────────────────────

func TestGenerateSessionID_HasPrefix(t *testing.T) {
	id := generateSessionID()
	if !strings.HasPrefix(id, "sess-") {
		t.Errorf("want sess- prefix, got %s", id)
	}
}

func TestGenerateSessionID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 50; i++ {
		id := generateSessionID()
		if ids[id] {
			t.Errorf("duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}
}

// ─── mergeMaps ───────────────────────────────────────────────

func TestMergeMaps_Basic(t *testing.T) {
	base := map[string]interface{}{"a": 1, "b": 2}
	over := map[string]interface{}{"b": 99, "c": 3}
	result := mergeMaps(base, over)
	if result["a"] != 1 {
		t.Errorf("want a=1, got %v", result["a"])
	}
	if result["b"] != 99 {
		t.Errorf("want b=99 (override), got %v", result["b"])
	}
	if result["c"] != 3 {
		t.Errorf("want c=3, got %v", result["c"])
	}
}

func TestMergeMaps_NilBase(t *testing.T) {
	result := mergeMaps(nil, map[string]interface{}{"x": "y"})
	if result["x"] != "y" {
		t.Errorf("want x=y, got %v", result["x"])
	}
}

func TestMergeMaps_NilOverrides(t *testing.T) {
	base := map[string]interface{}{"k": "v"}
	result := mergeMaps(base, nil)
	if result["k"] != "v" {
		t.Errorf("want k=v, got %v", result["k"])
	}
}

func TestMergeMaps_DoesNotMutateInputs(t *testing.T) {
	base := map[string]interface{}{"a": 1}
	over := map[string]interface{}{"b": 2}
	mergeMaps(base, over)
	if _, ok := base["b"]; ok {
		t.Error("base should not be mutated")
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

func TestExecuteStep_TemplateInBody(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{}, "error": nil})
	}))
	defer srv.Close()

	step := Step{
		Method:      "POST",
		Path:        "/api/v1/chat",
		Body:        map[string]interface{}{"session_id": "{{prev.session_id}}", "message": "hello"},
		Description: "chat",
	}
	_, err := ExecuteStep(srv.Client(), srv.URL, step, map[string]interface{}{
		"session_id": "sess-abc-1234",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["session_id"] != "sess-abc-1234" {
		t.Errorf("want session_id sess-abc-1234 in body, got %v", received["session_id"])
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
	step := Step{Method: "GET", Path: "/tickets"}
	_, err := ExecuteStep(http.DefaultClient, "http://127.0.0.1:1", step, nil)
	if err == nil {
		t.Fatal("want network error, got nil")
	}
}

// ─── RunScenario ─────────────────────────────────────────────

func noDelay() (int, int) { return -1, -1 }

// stepND returns a Step with delays bypassed (PreDelayMin=-1 skips default substitution;
// humanDelay clamps negatives to 0 and returns immediately).
func stepND(method, path, desc string, body map[string]interface{}) Step {
	pre, post := noDelay()
	return Step{
		Method:       method,
		Path:         path,
		Description:  desc,
		Body:         body,
		PreDelayMin:  pre,
		PreDelayMax:  post,
		PostDelayMin: pre,
		PostDelayMax: post,
	}
}

func TestRunScenario_MultiStep_TemplateChain(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
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
			stepND("POST", "/tickets", "create", map[string]interface{}{"title": "t"}),
			stepND("PATCH", "/tickets/{{prev.id}}", "close", map[string]interface{}{"status": "closed"}),
		},
	}
	RunScenario(context.Background(), srv.Client(), srv.URL, scenario)
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("want 2 HTTP calls, got %d", callCount)
	}
}

func TestRunScenario_ContinuesOnError(t *testing.T) {
	// v2 behavior: non-2xx does NOT abort; all steps still run
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	scenario := Scenario{
		Name: "continues despite errors",
		Steps: []Step{
			stepND("POST", "/tickets", "step 1", map[string]interface{}{"title": "t"}),
			stepND("PATCH", "/tickets/fallback", "step 2 — still runs", map[string]interface{}{"status": "closed"}),
		},
	}
	RunScenario(context.Background(), srv.Client(), srv.URL, scenario)
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("v2 should continue after error: want 2 HTTP calls, got %d", callCount)
	}
}

func TestRunScenario_ContextCancelled(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{"id": "X"}, "error": nil,
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before starting
	scenario := Scenario{
		Name: "cancelled",
		Steps: []Step{
			stepND("GET", "/tickets", "should not run", nil),
		},
	}
	RunScenario(ctx, srv.Client(), srv.URL, scenario)
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("cancelled context: want 0 calls, got %d", callCount)
	}
}

func TestRunScenario_SessionID_InCarry(t *testing.T) {
	var receivedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{"response": "ok"}, "error": nil,
		})
	}))
	defer srv.Close()

	pre, post := noDelay()
	scenario := Scenario{
		Name: "session id propagation",
		Steps: []Step{
			{
				Method:       "POST",
				Path:         "/api/v1/chat",
				Body:         map[string]interface{}{"session_id": "{{prev.session_id}}", "message": "hello"},
				Description:  "chat with session",
				PreDelayMin:  pre,
				PreDelayMax:  post,
				PostDelayMin: pre,
				PostDelayMax: post,
			},
		},
	}
	RunScenario(context.Background(), srv.Client(), srv.URL, scenario)
	sid, _ := receivedBody["session_id"].(string)
	if !strings.HasPrefix(sid, "sess-") {
		t.Errorf("want session_id with sess- prefix in body, got %q", sid)
	}
}

func TestRunScenario_StashAs_PersistsAcrossSteps(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case "POST":
			if strings.HasSuffix(r.URL.Path, "/comments") {
				// comment returns comment object (different id)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{"id": "comment-999"}, "error": nil,
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]interface{}{"id": "AMSS-777"}, "error": nil,
				})
			}
		default:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{"id": "AMSS-777", "status": "closed"}, "error": nil,
			})
		}
	}))
	defer srv.Close()

	pre, post := noDelay()
	mkStep := func(method, path, desc string, body map[string]interface{}, stashAs string) Step {
		return Step{
			Method: method, Path: path, Description: desc, Body: body, StashAs: stashAs,
			PreDelayMin: pre, PreDelayMax: post, PostDelayMin: pre, PostDelayMax: post,
		}
	}
	scenario := Scenario{
		Name: "stash ticket id across comment step",
		Steps: []Step{
			mkStep("POST", "/tickets", "create", map[string]interface{}{"title": "t"}, "ticket_id"),
			mkStep("POST", "/tickets/{{prev.ticket_id}}/comments", "comment", map[string]interface{}{"text": "note"}, ""),
			mkStep("PUT", "/tickets/{{prev.ticket_id}}", "close", map[string]interface{}{"status": "closed"}, ""),
		},
	}
	RunScenario(context.Background(), srv.Client(), srv.URL, scenario)

	if len(paths) != 3 {
		t.Fatalf("want 3 HTTP calls, got %d: %v", len(paths), paths)
	}
	if paths[1] != "/tickets/AMSS-777/comments" {
		t.Errorf("step 2 path: want /tickets/AMSS-777/comments, got %s", paths[1])
	}
	if paths[2] != "/tickets/AMSS-777" {
		t.Errorf("step 3 path: want /tickets/AMSS-777 (stash survives comment), got %s", paths[2])
	}
}

func TestRunScenario_StartOffset_Respected(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{}, "error": nil,
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// start_offset_seconds of 60 means the scenario won't run within 50ms
	scenario := Scenario{
		Name:               "delayed start",
		StartOffsetSeconds: 60,
		Steps: []Step{
			{Method: "GET", Path: "/tickets", Description: "should not run within timeout"},
		},
	}
	RunScenario(ctx, srv.Client(), srv.URL, scenario)
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("start offset should delay execution: want 0 calls, got %d", callCount)
	}
}

// ─── RunCycle ────────────────────────────────────────────────

func TestRunCycle_AllScenariosRun(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{"id": "X"}, "error": nil,
		})
	}))
	defer srv.Close()

	scenarios := []Scenario{
		{Name: "s1", Steps: []Step{stepND("GET", "/t1", "d1", nil)}},
		{Name: "s2", Steps: []Step{stepND("GET", "/t2", "d2", nil)}},
		{Name: "s3", Steps: []Step{stepND("GET", "/t3", "d3", nil)}},
	}
	RunCycle(context.Background(), srv.Client(), srv.URL, scenarios)
	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("want 3 HTTP calls (one per scenario), got %d", callCount)
	}
}

func TestRunCycle_ContextCancelled(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{}, "error": nil,
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	scenarios := []Scenario{
		{Name: "s1", Steps: []Step{stepND("GET", "/t", "d", nil)}},
	}
	RunCycle(ctx, srv.Client(), srv.URL, scenarios)
	if atomic.LoadInt32(&callCount) != 0 {
		t.Errorf("cancelled context: want 0 calls, got %d", callCount)
	}
}

// ─── ResetStores ─────────────────────────────────────────────

func TestResetStores_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/reset" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := ResetStores(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResetStores_NetworkError(t *testing.T) {
	err := ResetStores(context.Background(), http.DefaultClient, "http://127.0.0.1:1")
	if err == nil {
		t.Fatal("want network error, got nil")
	}
}

func TestResetStores_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ResetStores(ctx, srv.Client(), srv.URL)
	if err == nil {
		t.Fatal("want error for cancelled context, got nil")
	}
}

// ─── Default timing constants ────────────────────────────────

func TestDefaultTimingConstants(t *testing.T) {
	if defaultPreDelayMin <= 0 {
		t.Errorf("defaultPreDelayMin should be positive, got %d", defaultPreDelayMin)
	}
	if defaultPreDelayMax < defaultPreDelayMin {
		t.Errorf("defaultPreDelayMax (%d) should be >= defaultPreDelayMin (%d)", defaultPreDelayMax, defaultPreDelayMin)
	}
	if defaultPostDelayMin <= 0 {
		t.Errorf("defaultPostDelayMin should be positive, got %d", defaultPostDelayMin)
	}
	if defaultPostDelayMax < defaultPostDelayMin {
		t.Errorf("defaultPostDelayMax (%d) should be >= defaultPostDelayMin (%d)", defaultPostDelayMax, defaultPostDelayMin)
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
