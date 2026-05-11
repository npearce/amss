package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// helpers

func makeA2AResponse(text string, history []a2aMessage) *A2AResponse {
	return &A2AResponse{
		JSONRPC: "2.0",
		ID:      "test-1",
		Result: &a2aResult{
			Artifacts: []a2aArtifact{
				{Parts: []a2aPart{{Kind: "text", Text: text}}},
			},
			History: history,
		},
	}
}

func serveA2AResponse(t *testing.T, resp *A2AResponse, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(resp)
	}))
}

// callAgent tests

func TestCallAgent_Success(t *testing.T) {
	want := "Per KB-001, nominal WCS pressure is 14.7 psia."
	srv := serveA2AResponse(t, makeA2AResponse(want, nil), http.StatusOK)
	defer srv.Close()

	resp, err := callAgent(context.Background(), http.DefaultClient, srv.URL+"/", "sess-1", "WCS pressure?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := extractAgentText(resp)
	if got != want {
		t.Errorf("text = %q; want %q", got, want)
	}
}

func TestCallAgent_SetsJSONRPCFields(t *testing.T) {
	var captured a2aRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&captured)
		json.NewEncoder(w).Encode(makeA2AResponse("ok", nil))
	}))
	defer srv.Close()

	callAgent(context.Background(), http.DefaultClient, srv.URL+"/", "msg-42", "hello")

	if captured.JSONRPC != "2.0" {
		t.Errorf("jsonrpc = %q; want 2.0", captured.JSONRPC)
	}
	if captured.ID != "msg-42" {
		t.Errorf("id = %q; want msg-42", captured.ID)
	}
	if captured.Method != "message/send" {
		t.Errorf("method = %q; want message/send", captured.Method)
	}
	if len(captured.Params.Message.Parts) == 0 || captured.Params.Message.Parts[0].Text != "hello" {
		t.Errorf("message text not forwarded correctly: %+v", captured.Params.Message)
	}
	if captured.Params.Message.Parts[0].Kind != "text" {
		t.Errorf("part kind = %q; want text", captured.Params.Message.Parts[0].Kind)
	}
}

func TestCallAgent_AgentRPCError(t *testing.T) {
	errResp := &A2AResponse{
		JSONRPC: "2.0",
		ID:      "e1",
		Error:   &a2aErrorBlock{Code: -32600, Message: "Invalid Request"},
	}
	srv := serveA2AResponse(t, errResp, http.StatusOK)
	defer srv.Close()

	_, err := callAgent(context.Background(), http.DefaultClient, srv.URL+"/", "e1", "test")
	if err == nil {
		t.Fatal("expected error for RPC error response, got nil")
	}
}

func TestCallAgent_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json at all"))
	}))
	defer srv.Close()

	_, err := callAgent(context.Background(), http.DefaultClient, srv.URL+"/", "x", "msg")
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
}

func TestCallAgent_ConnectionRefused(t *testing.T) {
	_, err := callAgent(context.Background(), http.DefaultClient, "http://127.0.0.1:1/", "x", "msg")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestCallAgent_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		json.NewEncoder(w).Encode(makeA2AResponse("late", nil))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := callAgent(ctx, http.DefaultClient, srv.URL+"/", "t1", "msg")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestCallAgent_ContentTypeHeader(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		json.NewEncoder(w).Encode(makeA2AResponse("ok", nil))
	}))
	defer srv.Close()

	callAgent(context.Background(), http.DefaultClient, srv.URL+"/", "h1", "msg")
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", gotContentType)
	}
}

// extractAgentText tests

func TestExtractAgentText_ValidResponse(t *testing.T) {
	resp := makeA2AResponse("Per KB-001, use the flush valve.", nil)
	got := extractAgentText(resp)
	want := "Per KB-001, use the flush valve."
	if got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestExtractAgentText_NilResponse(t *testing.T) {
	if got := extractAgentText(nil); got != "" {
		t.Errorf("got %q; want empty string", got)
	}
}

func TestExtractAgentText_NilResult(t *testing.T) {
	resp := &A2AResponse{JSONRPC: "2.0", ID: "x"}
	if got := extractAgentText(resp); got != "" {
		t.Errorf("got %q; want empty string", got)
	}
}

func TestExtractAgentText_EmptyArtifacts(t *testing.T) {
	resp := &A2AResponse{
		JSONRPC: "2.0",
		ID:      "x",
		Result:  &a2aResult{Artifacts: []a2aArtifact{}},
	}
	if got := extractAgentText(resp); got != "" {
		t.Errorf("got %q; want empty string", got)
	}
}

func TestExtractAgentText_EmptyParts(t *testing.T) {
	resp := &A2AResponse{
		JSONRPC: "2.0",
		ID:      "x",
		Result: &a2aResult{
			Artifacts: []a2aArtifact{{Parts: []a2aPart{}}},
		},
	}
	if got := extractAgentText(resp); got != "" {
		t.Errorf("got %q; want empty string", got)
	}
}

// extractKBReferences tests

func TestExtractKBReferences_MultipleRefs(t *testing.T) {
	history := []a2aMessage{
		{Role: "tool", Parts: []a2aPart{{Kind: "text", Text: "Searched KB-001 and KB-003 for WCS procedures."}}},
		{Role: "tool", Parts: []a2aPart{{Kind: "text", Text: "Read KB-002 for CO2 scrubber details."}}},
	}
	resp := makeA2AResponse("answer", history)
	refs := extractKBReferences(resp)

	want := map[string]bool{"KB-001": true, "KB-002": true, "KB-003": true}
	if len(refs) != 3 {
		t.Fatalf("got %d refs: %v; want 3", len(refs), refs)
	}
	for _, r := range refs {
		if !want[r] {
			t.Errorf("unexpected ref %q", r)
		}
	}
}

func TestExtractKBReferences_Deduplicated(t *testing.T) {
	history := []a2aMessage{
		{Role: "tool", Parts: []a2aPart{{Kind: "text", Text: "KB-001 was found."}}},
		{Role: "tool", Parts: []a2aPart{{Kind: "text", Text: "Also KB-001 confirmed."}}},
	}
	resp := makeA2AResponse("answer", history)
	refs := extractKBReferences(resp)
	if len(refs) != 1 || refs[0] != "KB-001" {
		t.Errorf("expected [KB-001], got %v", refs)
	}
}

func TestExtractKBReferences_NoRefs(t *testing.T) {
	history := []a2aMessage{
		{Role: "assistant", Parts: []a2aPart{{Kind: "text", Text: "No articles matched."}}},
	}
	resp := makeA2AResponse("answer", history)
	refs := extractKBReferences(resp)
	if len(refs) != 0 {
		t.Errorf("expected empty slice, got %v", refs)
	}
}

func TestExtractKBReferences_NilResponse(t *testing.T) {
	refs := extractKBReferences(nil)
	if refs == nil || len(refs) != 0 {
		t.Errorf("expected empty slice for nil response, got %v", refs)
	}
}

func TestExtractKBReferences_EmptyHistory(t *testing.T) {
	resp := makeA2AResponse("answer", nil)
	refs := extractKBReferences(resp)
	if len(refs) != 0 {
		t.Errorf("expected empty slice, got %v", refs)
	}
}
