package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestHandler(t *testing.T) http.Handler {
	t.Helper()
	s, err := NewStore(seedPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	store = s

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /crew", handleGetCrew)
	mux.HandleFunc("GET /crew/{id}/activity", handleGetCrewActivity)
	mux.HandleFunc("GET /crew/{id}", handleGetCrewMember)
	mux.HandleFunc("GET /conversations", handleGetConversations)
	mux.HandleFunc("POST /conversations", handleCreateConversation)
	mux.HandleFunc("POST /reset", handleReset)
	return mux
}

func doRequest(t *testing.T, h http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decodeEnvelope(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var env map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&env); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return env
}

// --- Health ---

func TestHandleHealth(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/health", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data, ok := env["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data is not a map: %v", env["data"])
	}
	if data["status"] != "ok" {
		t.Errorf("status = %v, want ok", data["status"])
	}
	if data["store"] != "crew-store" {
		t.Errorf("store = %v, want crew-store", data["store"])
	}
}

func TestHandleHealthMethodNotAllowed(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/health", nil)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

// --- GET /crew ---

func TestHandleGetCrewAll(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 20 {
		t.Errorf("total = %v, want 20", data["total"])
	}
	crew := data["crew"].([]interface{})
	if len(crew) != 20 {
		t.Errorf("crew len = %d, want 20", len(crew))
	}
}

func TestHandleGetCrewFilterArtemisII(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew?mission=artemis-ii", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 12 {
		t.Errorf("total = %v, want 12", data["total"])
	}
}

func TestHandleGetCrewMissionIncludesGroundControl(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew?mission=artemis-iii", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	crew := data["crew"].([]interface{})

	gcCount := 0
	for _, item := range crew {
		m := item.(map[string]interface{})
		if m["mission"] == "all" {
			gcCount++
		}
	}
	if gcCount != 8 {
		t.Errorf("ground-control count = %d, want 8", gcCount)
	}
}

func TestHandleGetCrewFilterAll(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew?mission=all", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 8 {
		t.Errorf("total = %v, want 8", data["total"])
	}
}

func TestHandleGetCrewSearch(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew?search=wiseman", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 1 {
		t.Errorf("total = %v, want 1", data["total"])
	}
}

func TestHandleGetCrewSearchNoMatch(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew?search=xyznonexistent", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", data["total"])
	}
	crew := data["crew"].([]interface{})
	if len(crew) != 0 {
		t.Errorf("crew len = %d, want 0", len(crew))
	}
}

func TestHandleGetCrewPagination(t *testing.T) {
	h := setupTestHandler(t)
	w1 := doRequest(t, h, "GET", "/crew?limit=5&offset=0", nil)
	w2 := doRequest(t, h, "GET", "/crew?limit=5&offset=5", nil)

	env1 := decodeEnvelope(t, w1)
	env2 := decodeEnvelope(t, w2)

	data1 := env1["data"].(map[string]interface{})
	data2 := env2["data"].(map[string]interface{})

	crew1 := data1["crew"].([]interface{})
	crew2 := data2["crew"].([]interface{})
	if len(crew1) != 5 {
		t.Errorf("page1 len = %d, want 5", len(crew1))
	}
	if len(crew2) != 5 {
		t.Errorf("page2 len = %d, want 5", len(crew2))
	}
}

func TestHandleGetCrewDefaultLimit(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["limit"].(float64) != 50 {
		t.Errorf("limit = %v, want 50", data["limit"])
	}
}

func TestHandleGetCrewMaxLimit(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew?limit=999", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["limit"].(float64) != 200 {
		t.Errorf("limit = %v, want 200 (capped)", data["limit"])
	}
}

func TestHandleGetCrewEnvelope(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew", nil)
	env := decodeEnvelope(t, w)
	if _, ok := env["data"]; !ok {
		t.Error("response missing data field")
	}
	if _, ok := env["error"]; !ok {
		t.Error("response missing error field")
	}
	if env["error"] != nil {
		t.Errorf("error should be null, got %v", env["error"])
	}
}

// --- GET /crew/{id} ---

func TestHandleGetCrewMember(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/wiseman-r", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["id"] != "wiseman-r" {
		t.Errorf("id = %v, want wiseman-r", data["id"])
	}
	if data["name"] != "Reid Wiseman" {
		t.Errorf("name = %v, want Reid Wiseman", data["name"])
	}
}

func TestHandleGetCrewMemberNotFound(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/nobody", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	env := decodeEnvelope(t, w)
	if env["error"] == nil {
		t.Error("expected error in envelope")
	}
	errInfo := env["error"].(map[string]interface{})
	if errInfo["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", errInfo["code"])
	}
}

func TestHandleGetCrewMemberEnvelope(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/gc-flight", nil)
	env := decodeEnvelope(t, w)
	if env["data"] == nil {
		t.Error("data should not be nil")
	}
	if env["error"] != nil {
		t.Errorf("error should be null, got %v", env["error"])
	}
}

// --- GET /crew/{id}/activity ---

func TestHandleGetCrewActivityEmpty(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/wiseman-r/activity", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["crew_id"] != "wiseman-r" {
		t.Errorf("crew_id = %v, want wiseman-r", data["crew_id"])
	}
	if data["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", data["total"])
	}
	convs := data["conversations"].([]interface{})
	if len(convs) != 0 {
		t.Errorf("conversations len = %d, want 0", len(convs))
	}
}

func TestHandleGetCrewActivityReturnsData(t *testing.T) {
	h := setupTestHandler(t)
	// create conversations first
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q1", "response": "r1",
	})
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s2", "query": "q2", "response": "r2",
	})

	w := doRequest(t, h, "GET", "/crew/wiseman-r/activity", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
}

func TestHandleGetCrewActivityDefaultLimit20(t *testing.T) {
	h := setupTestHandler(t)
	for i := 0; i < 25; i++ {
		doRequest(t, h, "POST", "/conversations", map[string]interface{}{
			"crew_id": "wiseman-r", "mission": "artemis-ii",
			"session_id": fmt.Sprintf("s%d", i), "query": "q", "response": "r",
		})
	}
	w := doRequest(t, h, "GET", "/crew/wiseman-r/activity", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	convs := data["conversations"].([]interface{})
	if len(convs) != 20 {
		t.Errorf("conversations len = %d, want 20 (default limit)", len(convs))
	}
	if data["total"].(float64) != 25 {
		t.Errorf("total = %v, want 25", data["total"])
	}
}

func TestHandleGetCrewActivityMaxLimit100(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/wiseman-r/activity?limit=999", nil)
	// No conversations but limit header should be capped — just verify 200
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestHandleGetCrewActivityNewestFirst(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "first", "response": "r",
	})
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s2", "query": "second", "response": "r",
	})

	// Force timestamps so order is deterministic
	store.conversations["conv-0001"].Timestamp = "2024-01-01T10:00:00Z"
	store.conversations["conv-0002"].Timestamp = "2024-01-01T12:00:00Z"

	w := doRequest(t, h, "GET", "/crew/wiseman-r/activity", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	convs := data["conversations"].([]interface{})
	if len(convs) < 2 {
		t.Fatal("expected at least 2 conversations")
	}
	first := convs[0].(map[string]interface{})
	if first["id"] != "conv-0002" {
		t.Errorf("first conv = %v, want conv-0002 (newest)", first["id"])
	}
}

func TestHandleGetCrewActivityNotFound(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/nobody/activity", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	env := decodeEnvelope(t, w)
	errInfo := env["error"].(map[string]interface{})
	if errInfo["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", errInfo["code"])
	}
}

// --- GET /conversations ---

func TestHandleGetConversationsEmpty(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/conversations", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 0 {
		t.Errorf("total = %v, want 0", data["total"])
	}
}

func TestHandleGetConversationsAll(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q1", "response": "r1",
	})
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "glover-v", "mission": "artemis-ii", "session_id": "s2", "query": "q2", "response": "r2",
	})
	w := doRequest(t, h, "GET", "/conversations", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
}

func TestHandleGetConversationsFilterCrewID(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "glover-v", "mission": "artemis-ii", "session_id": "s2", "query": "q", "response": "r",
	})
	w := doRequest(t, h, "GET", "/conversations?crew_id=wiseman-r", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 1 {
		t.Errorf("total = %v, want 1", data["total"])
	}
}

func TestHandleGetConversationsFilterMission(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "patel-a", "mission": "artemis-iii", "session_id": "s2", "query": "q", "response": "r",
	})
	w := doRequest(t, h, "GET", "/conversations?mission=artemis-iii", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 1 {
		t.Errorf("total = %v, want 1", data["total"])
	}
}

func TestHandleGetConversationsFilterSessionID(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "session-A", "query": "q", "response": "r",
	})
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "session-B", "query": "q", "response": "r",
	})
	w := doRequest(t, h, "GET", "/conversations?session_id=session-A", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 1 {
		t.Errorf("total = %v, want 1", data["total"])
	}
}

func TestHandleGetConversationsPagination(t *testing.T) {
	h := setupTestHandler(t)
	for i := 0; i < 6; i++ {
		doRequest(t, h, "POST", "/conversations", map[string]interface{}{
			"crew_id": "wiseman-r", "mission": "artemis-ii",
			"session_id": fmt.Sprintf("s%d", i), "query": "q", "response": "r",
		})
	}
	w1 := doRequest(t, h, "GET", "/conversations?limit=3&offset=0", nil)
	w2 := doRequest(t, h, "GET", "/conversations?limit=3&offset=3", nil)

	env1 := decodeEnvelope(t, w1)
	env2 := decodeEnvelope(t, w2)
	data1 := env1["data"].(map[string]interface{})
	data2 := env2["data"].(map[string]interface{})

	c1 := data1["conversations"].([]interface{})
	c2 := data2["conversations"].([]interface{})
	if len(c1) != 3 {
		t.Errorf("page1 len = %d, want 3", len(c1))
	}
	if len(c2) != 3 {
		t.Errorf("page2 len = %d, want 3", len(c2))
	}
}

func TestHandleGetConversationsDefaultLimit50(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/conversations", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["limit"].(float64) != 50 {
		t.Errorf("limit = %v, want 50", data["limit"])
	}
}

func TestHandleGetConversationsMaxLimit200(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/conversations?limit=999", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["limit"].(float64) != 200 {
		t.Errorf("limit = %v, want 200 (capped)", data["limit"])
	}
}

// --- POST /conversations ---

func TestHandleCreateConversation(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id":    "wiseman-r",
		"mission":    "artemis-ii",
		"session_id": "sess-1",
		"query":      "How is the oxygen level?",
		"response":   "It is nominal.",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["crew_id"] != "wiseman-r" {
		t.Errorf("crew_id = %v, want wiseman-r", data["crew_id"])
	}
	if data["mission"] != "artemis-ii" {
		t.Errorf("mission = %v, want artemis-ii", data["mission"])
	}
}

func TestHandleCreateConversationIDFormat(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["id"] != "conv-0001" {
		t.Errorf("id = %v, want conv-0001", data["id"])
	}
}

func TestHandleCreateConversationTimestamp(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	ts, ok := data["timestamp"].(string)
	if !ok || ts == "" {
		t.Fatalf("timestamp missing or empty: %v", data["timestamp"])
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp %q is not valid RFC3339: %v", ts, err)
	}
}

func TestHandleCreateConversationWithTicket(t *testing.T) {
	h := setupTestHandler(t)
	ticket := "AMSS-001"
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1",
		"query": "q", "response": "r", "ticket_created": ticket,
	})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["ticket_created"] != ticket {
		t.Errorf("ticket_created = %v, want %s", data["ticket_created"], ticket)
	}
}

func TestHandleCreateConversationMissingCrewID(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCreateConversationMissingMission(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "session_id": "s1", "query": "q", "response": "r",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCreateConversationMissingSessionID(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "query": "q", "response": "r",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCreateConversationMissingQuery(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "response": "r",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCreateConversationMissingResponse(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleCreateConversationInvalidJSON(t *testing.T) {
	h := setupTestHandler(t)
	req := httptest.NewRequest("POST", "/conversations", strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// TestHandleCreateConversationCrewNotFound must return 400, NOT 404.
func TestHandleCreateConversationCrewNotFound(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "ghost", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (not 404)", w.Code)
	}
	env := decodeEnvelope(t, w)
	errInfo := env["error"].(map[string]interface{})
	if errInfo["code"] != "BAD_REQUEST" {
		t.Errorf("code = %v, want BAD_REQUEST", errInfo["code"])
	}
}

// --- POST /reset ---

func TestHandleReset(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	w := doRequest(t, h, "POST", "/reset", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["crew_count"].(float64) != 20 {
		t.Errorf("crew_count = %v, want 20", data["crew_count"])
	}
	if data["message"] != "Reset to seed data" {
		t.Errorf("message = %v, want 'Reset to seed data'", data["message"])
	}
}

func TestHandleResetMethodNotAllowed(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/reset", nil)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleResetClearsConversations(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/conversations", map[string]interface{}{
		"crew_id": "wiseman-r", "mission": "artemis-ii", "session_id": "s1", "query": "q", "response": "r",
	})
	doRequest(t, h, "POST", "/reset", nil)

	w := doRequest(t, h, "GET", "/conversations", nil)
	env := decodeEnvelope(t, w)
	data := env["data"].(map[string]interface{})
	if data["total"].(float64) != 0 {
		t.Errorf("total after reset = %v, want 0", data["total"])
	}
}

func TestHandleResetCrewIntact(t *testing.T) {
	h := setupTestHandler(t)
	doRequest(t, h, "POST", "/reset", nil)
	w := doRequest(t, h, "GET", "/crew/wiseman-r", nil)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (crew intact after reset)", w.Code)
	}
}

func TestHandleErrorEnvelope(t *testing.T) {
	h := setupTestHandler(t)
	w := doRequest(t, h, "GET", "/crew/nobody", nil)
	env := decodeEnvelope(t, w)
	if env["data"] != nil {
		t.Errorf("data should be null on error, got %v", env["data"])
	}
	if env["error"] == nil {
		t.Error("error should not be null on error response")
	}
}
