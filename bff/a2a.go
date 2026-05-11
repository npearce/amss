package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
)

type a2aPart struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type a2aMessage struct {
	Role  string    `json:"role"`
	Parts []a2aPart `json:"parts"`
}

type a2aArtifact struct {
	Parts []a2aPart `json:"parts"`
}

type a2aResult struct {
	Artifacts []a2aArtifact `json:"artifacts"`
	History   []a2aMessage  `json:"history"`
}

type a2aRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  a2aParamsBlock `json:"params"`
}

type a2aParamsBlock struct {
	Message a2aMessage `json:"message"`
}

type a2aErrorBlock struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// A2AResponse is the JSON-RPC 2.0 response envelope from a kagent agent.
type A2AResponse struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Result  *a2aResult     `json:"result,omitempty"`
	Error   *a2aErrorBlock `json:"error,omitempty"`
}

var kbRefPattern = regexp.MustCompile(`KB-\d+`)

// callAgent sends a message/send JSON-RPC 2.0 request to a kagent A2A endpoint.
// agentURL must end with a trailing slash (kagent requirement).
func callAgent(ctx context.Context, client *http.Client, agentURL, messageID, text string) (*A2AResponse, error) {
	payload := a2aRequest{
		JSONRPC: "2.0",
		ID:      messageID,
		Method:  "message/send",
		Params: a2aParamsBlock{
			Message: a2aMessage{
				Role:  "user",
				Parts: []a2aPart{{Kind: "text", Text: text}},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("a2a: marshal error: %v", err)
		return nil, fmt.Errorf("a2a: marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, agentURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("a2a: build request error: %v", err)
		return nil, fmt.Errorf("a2a: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("a2a: http error: %v", err)
		return nil, fmt.Errorf("a2a: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("a2a: read body error: %v", err)
		return nil, fmt.Errorf("a2a: read body: %w", err)
	}

	var a2aResp A2AResponse
	if err := json.Unmarshal(respBody, &a2aResp); err != nil {
		log.Printf("a2a: unmarshal error: %v", err)
		return nil, fmt.Errorf("a2a: unmarshal: %w", err)
	}
	if a2aResp.Error != nil {
		log.Printf("a2a: agent returned RPC error %d: %s", a2aResp.Error.Code, a2aResp.Error.Message)
		return nil, fmt.Errorf("a2a: agent error %d: %s", a2aResp.Error.Code, a2aResp.Error.Message)
	}

	return &a2aResp, nil
}

// extractAgentText returns the agent's response text from result.artifacts[0].parts[0].
func extractAgentText(resp *A2AResponse) string {
	if resp == nil || resp.Result == nil || len(resp.Result.Artifacts) == 0 {
		return ""
	}
	parts := resp.Result.Artifacts[0].Parts
	if len(parts) == 0 {
		return ""
	}
	return parts[0].Text
}

// extractKBReferences scans all history parts for KB-NNN references, deduplicated.
func extractKBReferences(resp *A2AResponse) []string {
	if resp == nil || resp.Result == nil {
		return []string{}
	}
	seen := make(map[string]struct{})
	var refs []string
	for _, msg := range resp.Result.History {
		for _, part := range msg.Parts {
			for _, match := range kbRefPattern.FindAllString(part.Text, -1) {
				if _, ok := seen[match]; !ok {
					seen[match] = struct{}{}
					refs = append(refs, match)
				}
			}
		}
	}
	if refs == nil {
		return []string{}
	}
	return refs
}
