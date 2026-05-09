package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func testConfig() *Config {
	return &Config{
		Port:                   "0",
		KBStoreURL:             "http://unused-kb:9999",
		TicketStoreURL:         "http://unused-ticket:9999",
		CrewStoreURL:           "http://unused-crew:9999",
		MissionSupportAgentURL: "http://unused-agent:9999",
		KBCuratorAgentURL:      "http://unused-curator:9999",
	}
}

// mockBackend starts a test server that records the last request path and method,
// responds with statusCode and body on every call.
func mockBackend(t *testing.T, statusCode int, body string) (*httptest.Server, *string, *string) {
	t.Helper()
	var mu sync.Mutex
	gotPath := ""
	gotMethod := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		gotMethod = r.Method
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &gotPath, &gotMethod
}

// TestHandleHealth verifies the /health endpoint.
func TestHandleHealth(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	NewServer(testConfig()).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	var env Envelope
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data is not an object")
	}
	if data["status"] != "ok" {
		t.Errorf("status = %q, want ok", data["status"])
	}
	if data["service"] != "bff" {
		t.Errorf("service = %q, want bff", data["service"])
	}
}

// TestCORSPreflight verifies CORS headers on OPTIONS requests.
func TestCORSPreflight(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/articles", nil)
	NewServer(testConfig()).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want 204", w.Code)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("ACAO = %q, want *", origin)
	}
}

// TestCORSHeaders verifies CORS headers are present on normal responses.
func TestCORSHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	NewServer(testConfig()).ServeHTTP(w, r)

	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("ACAO = %q, want *", origin)
	}
}

// TestProxyKBRoutes verifies all KB article routes proxy to kb-store.
func TestProxyKBRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list articles", http.MethodGet, "/articles", ""},
		{"get article", http.MethodGet, "/articles/KB-001", ""},
		{"create article", http.MethodPost, "/articles", `{"title":"test","body":"b","category":"power"}`},
		{"update article", http.MethodPut, "/articles/KB-001", `{"title":"updated"}`},
		{"delete article", http.MethodDelete, "/articles/KB-001", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock, gotPath, gotMethod := mockBackend(t, http.StatusOK, `{"data":{},"error":null}`)

			cfg := testConfig()
			cfg.KBStoreURL = mock.URL
			bff := httptest.NewServer(NewServer(cfg))
			t.Cleanup(bff.Close)

			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req, _ := http.NewRequest(tc.method, bff.URL+tc.path, body)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("got status %d, want 200", resp.StatusCode)
			}
			if *gotPath != tc.path {
				t.Errorf("backend got path %q, want %q", *gotPath, tc.path)
			}
			if *gotMethod != tc.method {
				t.Errorf("backend got method %q, want %q", *gotMethod, tc.method)
			}
		})
	}
}

// TestProxyKBRouteError verifies that backend errors are proxied back.
func TestProxyKBRouteError(t *testing.T) {
	mock, _, _ := mockBackend(t, http.StatusNotFound, `{"data":null,"error":{"code":"NOT_FOUND","message":"Article KB-999 not found"}}`)

	cfg := testConfig()
	cfg.KBStoreURL = mock.URL
	bff := httptest.NewServer(NewServer(cfg))
	t.Cleanup(bff.Close)

	resp, err := http.Get(bff.URL + "/articles/KB-999")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("got status %d, want 404", resp.StatusCode)
	}
}

// TestProxyTicketRoutes verifies all ticket routes proxy to ticket-store.
func TestProxyTicketRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list tickets", http.MethodGet, "/tickets", ""},
		{"get ticket", http.MethodGet, "/tickets/AMSS-001", ""},
		{"create ticket", http.MethodPost, "/tickets", `{"title":"t","description":"d","severity":"P3","category":"power","reported_by":"wiseman-r","mission":"artemis-ii"}`},
		{"update ticket", http.MethodPut, "/tickets/AMSS-001", `{"status":"resolved"}`},
		{"add comment", http.MethodPost, "/tickets/AMSS-001/comments", `{"author":"wiseman-r","text":"Fixed"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock, gotPath, gotMethod := mockBackend(t, http.StatusOK, `{"data":{},"error":null}`)

			cfg := testConfig()
			cfg.TicketStoreURL = mock.URL
			bff := httptest.NewServer(NewServer(cfg))
			t.Cleanup(bff.Close)

			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req, _ := http.NewRequest(tc.method, bff.URL+tc.path, body)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("got status %d, want 200", resp.StatusCode)
			}
			if *gotPath != tc.path {
				t.Errorf("backend got path %q, want %q", *gotPath, tc.path)
			}
			if *gotMethod != tc.method {
				t.Errorf("backend got method %q, want %q", *gotMethod, tc.method)
			}
		})
	}
}

// TestProxyCrewRoutes verifies crew and conversation routes proxy to crew-store.
func TestProxyCrewRoutes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list crew", http.MethodGet, "/crew", ""},
		{"get crew member", http.MethodGet, "/crew/wiseman-r", ""},
		{"crew activity", http.MethodGet, "/crew/wiseman-r/activity", ""},
		{"list conversations", http.MethodGet, "/conversations", ""},
		{"post conversation", http.MethodPost, "/conversations", `{"crew_id":"wiseman-r","mission":"artemis-ii","session_id":"s1","query":"q","response":"r"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock, gotPath, gotMethod := mockBackend(t, http.StatusOK, `{"data":{},"error":null}`)

			cfg := testConfig()
			cfg.CrewStoreURL = mock.URL
			bff := httptest.NewServer(NewServer(cfg))
			t.Cleanup(bff.Close)

			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			req, _ := http.NewRequest(tc.method, bff.URL+tc.path, body)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("got status %d, want 200", resp.StatusCode)
			}
			if *gotPath != tc.path {
				t.Errorf("backend got path %q, want %q", *gotPath, tc.path)
			}
			if *gotMethod != tc.method {
				t.Errorf("backend got method %q, want %q", *gotMethod, tc.method)
			}
		})
	}
}

// TestHandleChat_Success verifies a valid chat request calls the agent and logs the conversation.
func TestHandleChat_Success(t *testing.T) {
	agentPayload := `{"response":"Pressure nominal","kb_articles_referenced":["KB-001"],"ticket_created":null}`
	mockAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat" || r.Method != http.MethodPost {
			t.Errorf("agent got %s %s, want POST /chat", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, agentPayload)
	}))
	defer mockAgent.Close()

	var convBody []byte
	mockCrew := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/conversations" && r.Method == http.MethodPost {
			convBody, _ = io.ReadAll(r.Body)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"data":{},"error":null}`)
	}))
	defer mockCrew.Close()

	cfg := testConfig()
	cfg.MissionSupportAgentURL = mockAgent.URL
	cfg.CrewStoreURL = mockCrew.URL
	bff := httptest.NewServer(NewServer(cfg))
	defer bff.Close()

	reqBody := `{"crew_id":"wiseman-r","session_id":"sess-001","mission":"artemis-ii","message":"What is WCS pressure?"}`
	resp, err := http.Post(bff.URL+"/chat", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want 200", resp.StatusCode)
	}

	var env Envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if env.Error != nil {
		t.Fatalf("unexpected error: %v", env.Error)
	}

	// Verify conversation was logged to crew-store
	if len(convBody) == 0 {
		t.Error("conversation was not logged to crew-store")
	}
	var conv map[string]interface{}
	if err := json.Unmarshal(convBody, &conv); err != nil {
		t.Fatalf("conv body decode error: %v", err)
	}
	if conv["crew_id"] != "wiseman-r" {
		t.Errorf("conv crew_id = %v, want wiseman-r", conv["crew_id"])
	}
	if conv["query"] != "What is WCS pressure?" {
		t.Errorf("conv query = %v, want the original message", conv["query"])
	}
}

// TestHandleChat_MissingFields verifies that missing required fields return 400.
func TestHandleChat_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing crew_id", `{"message":"hello"}`},
		{"missing message", `{"crew_id":"wiseman-r"}`},
		{"empty body", `{}`},
	}

	server := NewServer(testConfig())

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(tc.body))
			server.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("got status %d, want 400", w.Code)
			}
			var env Envelope
			json.NewDecoder(w.Body).Decode(&env)
			if env.Error == nil || env.Error.Code != "BAD_REQUEST" {
				t.Errorf("expected BAD_REQUEST error, got %v", env.Error)
			}
		})
	}
}

// TestHandleChat_InvalidBody verifies malformed JSON returns 400.
func TestHandleChat_InvalidBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader("not json"))
	NewServer(testConfig()).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", w.Code)
	}
}

// TestHandleChat_AgentError verifies that when the agent is unreachable (stub mode off),
// the BFF falls back to the stub response and includes an unavailability note.
func TestHandleChat_AgentError(t *testing.T) {
	cfg := testConfig()
	cfg.MissionSupportAgentURL = "http://127.0.0.1:1"
	// StubMode defaults to false in testConfig; agent unreachable → graceful stub fallback.

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(`{"crew_id":"wiseman-r","message":"wcs pressure"}`))
	NewServer(cfg).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want 200 (stub fallback)", w.Code)
	}
	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)
	if env.Error != nil {
		t.Errorf("expected no error in fallback mode, got %v", env.Error)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data is not an object")
	}
	resp, _ := data["response"].(string)
	if !strings.Contains(strings.ToLower(resp), "unavailable") {
		t.Errorf("fallback response should mention agent unavailability, got %q", resp)
	}
}

// TestHandleCurate_Success verifies the curate endpoint proxies to the curator agent.
func TestHandleCurate_Success(t *testing.T) {
	var gotPath string
	var gotBody []byte
	mockCurator := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"curated":3},"error":null}`)
	}))
	defer mockCurator.Close()

	cfg := testConfig()
	cfg.KBCuratorAgentURL = mockCurator.URL
	bff := httptest.NewServer(NewServer(cfg))
	defer bff.Close()

	reqBody := `{"article_ids":["KB-001","KB-018"]}`
	resp, err := http.Post(bff.URL+"/curate", "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}
	if gotPath != "/curate" {
		t.Errorf("curator got path %q, want /curate", gotPath)
	}
	if !bytes.Equal(gotBody, []byte(reqBody)) {
		t.Errorf("curator got body %q, want %q", gotBody, reqBody)
	}
}

// TestHandleCurate_AgentError verifies that when the curator agent is unreachable (stub mode off),
// the BFF falls back to the stub curator report.
func TestHandleCurate_AgentError(t *testing.T) {
	cfg := testConfig()
	cfg.KBCuratorAgentURL = "http://127.0.0.1:1"
	// StubMode defaults to false in testConfig; agent unreachable → graceful stub fallback.

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/curate", strings.NewReader(`{}`))
	NewServer(cfg).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want 200 (stub fallback)", w.Code)
	}
	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)
	if env.Error != nil {
		t.Errorf("expected no error in fallback mode, got %v", env.Error)
	}
	if env.Data == nil {
		t.Error("expected stub curator report in data, got nil")
	}
}

// TestHandleReset verifies /reset calls all three stores.
func TestHandleReset(t *testing.T) {
	called := map[string]bool{}
	var mu sync.Mutex

	makeStore := func(name string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/reset" && r.Method == http.MethodPost {
				mu.Lock()
				called[name] = true
				mu.Unlock()
			}
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"data":{"message":"Reset to seed data"},"error":null}`)
		}))
	}

	kb := makeStore("kb")
	defer kb.Close()
	ticket := makeStore("ticket")
	defer ticket.Close()
	crew := makeStore("crew")
	defer crew.Close()

	cfg := testConfig()
	cfg.KBStoreURL = kb.URL
	cfg.TicketStoreURL = ticket.URL
	cfg.CrewStoreURL = crew.URL
	bff := httptest.NewServer(NewServer(cfg))
	defer bff.Close()

	resp, err := http.Post(bff.URL+"/reset", "application/json", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want 200", resp.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()
	if !called["kb"] {
		t.Error("kb-store /reset was not called")
	}
	if !called["ticket"] {
		t.Error("ticket-store /reset was not called")
	}
	if !called["crew"] {
		t.Error("crew-store /reset was not called")
	}
}

// TestHandleChat_StubMode verifies that STUB_MODE=true returns a stub response without
// calling the agent.
func TestHandleChat_StubMode(t *testing.T) {
	cfg := testConfig()
	cfg.StubMode = true
	// Agent URL points nowhere; stub mode must not call it.
	cfg.MissionSupportAgentURL = "http://127.0.0.1:1"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(
		`{"crew_id":"wiseman-r","message":"wcs flush procedure"}`))
	NewServer(cfg).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)
	if env.Error != nil {
		t.Fatalf("unexpected error: %v", env.Error)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data is not an object")
	}
	resp, _ := data["response"].(string)
	if resp == "" {
		t.Error("stub response should not be empty")
	}
	// WCS keyword → KB-001 reference expected.
	refs, _ := data["kb_articles_referenced"].([]interface{})
	found := false
	for _, ref := range refs {
		if ref == "KB-001" {
			found = true
		}
	}
	if !found {
		t.Errorf("stub response for WCS query should reference KB-001, got %v", refs)
	}
}

// TestHandleChat_StubModeDefaultQuery verifies the fallback stub response for unknown queries.
func TestHandleChat_StubModeDefaultQuery(t *testing.T) {
	cfg := testConfig()
	cfg.StubMode = true

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(
		`{"crew_id":"wiseman-r","message":"what is for dinner tonight"}`))
	NewServer(cfg).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)
	if env.Error != nil {
		t.Fatalf("unexpected error: %v", env.Error)
	}
	data, _ := env.Data.(map[string]interface{})
	refs, _ := data["kb_articles_referenced"].([]interface{})
	if len(refs) != 0 {
		t.Errorf("default stub response should have no KB refs, got %v", refs)
	}
}

// TestHandleCurate_StubMode verifies that STUB_MODE=true returns the stub curator report.
func TestHandleCurate_StubMode(t *testing.T) {
	cfg := testConfig()
	cfg.StubMode = true
	cfg.KBCuratorAgentURL = "http://127.0.0.1:1"

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/curate", strings.NewReader(`{}`))
	NewServer(cfg).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", w.Code)
	}
	var env Envelope
	json.NewDecoder(w.Body).Decode(&env)
	if env.Error != nil {
		t.Fatalf("unexpected error: %v", env.Error)
	}
	if env.Data == nil {
		t.Fatal("expected curator report in data, got nil")
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data is not an object")
	}
	// Verify key fields of the stub report are present.
	if data["articles_reviewed"] == nil {
		t.Error("stub report missing articles_reviewed")
	}
	dups, _ := data["duplicates_flagged"].([]interface{})
	if len(dups) != 3 {
		t.Errorf("stub report should have 3 duplicates flagged, got %d", len(dups))
	}
	tagged, _ := data["tagged_articles"].([]interface{})
	if len(tagged) != 4 {
		t.Errorf("stub report should have 4 tagged articles, got %d", len(tagged))
	}
	scored, _ := data["scored_articles"].([]interface{})
	if len(scored) != 5 {
		t.Errorf("stub report should have 5 scored articles, got %d", len(scored))
	}
}
