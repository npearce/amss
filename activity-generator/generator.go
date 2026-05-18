package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultPreDelayMin  = 3
	defaultPreDelayMax  = 8
	defaultPostDelayMin = 5
	defaultPostDelayMax = 15
)

// Scenario is a narrative thread run by one or more crew members.
type Scenario struct {
	Name               string `json:"name"`
	StartOffsetSeconds int    `json:"start_offset_seconds"`
	Steps              []Step `json:"steps"`
}

// Step is a single HTTP action within a scenario.
type Step struct {
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Body         map[string]interface{} `json:"body,omitempty"`
	Description  string                 `json:"description"`
	CrewID       string                 `json:"crew_id,omitempty"`
	PreDelayMin  int                    `json:"pre_delay_min,omitempty"`
	PreDelayMax  int                    `json:"pre_delay_max,omitempty"`
	PostDelayMin int                    `json:"post_delay_min,omitempty"`
	PostDelayMax int                    `json:"post_delay_max,omitempty"`
	// StashAs saves Data["id"] from this step's response into the scenario's
	// carry map under this key, making it available as {{prev.<key>}} in all
	// subsequent steps regardless of intermediate step responses.
	StashAs string `json:"stash_as,omitempty"`
}

// StepResult holds the outcome of executing one HTTP step.
type StepResult struct {
	StatusCode   int
	ResolvedPath string
	Data         map[string]interface{}
	RawBody      string
}

// AuthConfig holds optional JWT auth configuration for BFF requests.
//
// Three modes:
//   - refresh: all Keycloak fields set — token is fetched and refreshed automatically
//   - static:  only StaticToken set — used as-is for every request, no refresh
//   - none:    neither set — requests are made without an Authorization header
type AuthConfig struct {
	StaticToken          string
	KeycloakURL          string
	KeycloakClientID     string
	KeycloakClientSecret string
	KeycloakUsername     string
	KeycloakPassword     string

	mu           sync.Mutex
	currentToken string
}

// mode returns "refresh", "static", or "none".
func (a *AuthConfig) mode() string {
	if a.KeycloakURL != "" && a.KeycloakClientID != "" && a.KeycloakClientSecret != "" &&
		a.KeycloakUsername != "" && a.KeycloakPassword != "" {
		return "refresh"
	}
	if a.StaticToken != "" {
		return "static"
	}
	return "none"
}

// Token returns the current bearer token, or empty string if none is configured.
func (a *AuthConfig) Token() string {
	switch a.mode() {
	case "static":
		return a.StaticToken
	case "refresh":
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.currentToken
	default:
		return ""
	}
}

// Refresh fetches a fresh token from Keycloak using the password grant.
// It is a no-op when not in refresh mode.
func (a *AuthConfig) Refresh(client *http.Client) error {
	if a.mode() != "refresh" {
		return nil
	}
	vals := url.Values{
		"client_id":     {a.KeycloakClientID},
		"client_secret": {a.KeycloakClientSecret},
		"username":      {a.KeycloakUsername},
		"password":      {a.KeycloakPassword},
		"grant_type":    {"password"},
	}
	resp, err := client.PostForm(
		a.KeycloakURL+"/realms/master/protocol/openid-connect/token",
		vals,
	)
	if err != nil {
		return fmt.Errorf("keycloak token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("keycloak token: HTTP %d: %s", resp.StatusCode, body)
	}
	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}
	if result.AccessToken == "" {
		return fmt.Errorf("keycloak returned empty access_token")
	}
	a.mu.Lock()
	a.currentToken = result.AccessToken
	a.mu.Unlock()
	return nil
}

// addAuthHeader adds Authorization: Bearer <token> to req when auth has a non-empty token.
func addAuthHeader(req *http.Request, auth *AuthConfig) {
	if auth == nil {
		return
	}
	if token := auth.Token(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// LoadScenarios reads and parses scenarios from a JSON file.
func LoadScenarios(path string) ([]Scenario, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var scenarios []Scenario
	if err := json.NewDecoder(f).Decode(&scenarios); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return scenarios, nil
}

// resolveTemplates replaces {{prev.<key>}} placeholders in s using data.
func resolveTemplates(s string, data map[string]interface{}) string {
	for k, v := range data {
		placeholder := "{{prev." + k + "}}"
		s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", v))
	}
	return s
}

// humanDelay sleeps for a random duration uniformly distributed in [minSec, maxSec].
// Returns early if ctx is cancelled.
func humanDelay(ctx context.Context, minSec, maxSec int) {
	if minSec < 0 {
		minSec = 0
	}
	if maxSec < minSec {
		maxSec = minSec
	}
	var n int
	if maxSec > minSec {
		n = minSec + rand.Intn(maxSec-minSec+1)
	} else {
		n = minSec
	}
	if n <= 0 {
		return
	}
	select {
	case <-time.After(time.Duration(n) * time.Second):
	case <-ctx.Done():
	}
}

// formatCrewLabel converts a crew_id like "koch-c" into "Koch" for log output.
func formatCrewLabel(crewID string) string {
	if crewID == "" {
		return "system"
	}
	parts := strings.SplitN(crewID, "-", 2)
	name := parts[0]
	if len(name) == 0 {
		return crewID
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// generateSessionID returns a unique session identifier for one scenario cycle.
func generateSessionID() string {
	return fmt.Sprintf("sess-%d-%04x", time.Now().UnixNano(), rand.Intn(0xffff))
}

// logf writes a timestamped human-readable log line.
func logf(format string, args ...interface{}) {
	ts := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] %s\n", ts, msg)
}

// ExecuteStep sends one HTTP request and returns the result.
// prevData is used for {{prev.<key>}} template resolution in path and body.
// auth adds an Authorization header when configured.
func ExecuteStep(client *http.Client, bffURL string, step Step, prevData map[string]interface{}, auth *AuthConfig) (StepResult, error) {
	resolvedPath := resolveTemplates(step.Path, prevData)
	u := bffURL + resolvedPath

	var bodyReader io.Reader
	if step.Body != nil {
		raw, err := json.Marshal(step.Body)
		if err != nil {
			return StepResult{}, fmt.Errorf("marshal body: %w", err)
		}
		resolved := resolveTemplates(string(raw), prevData)
		bodyReader = bytes.NewBufferString(resolved)
	}

	req, err := http.NewRequest(step.Method, u, bodyReader)
	if err != nil {
		return StepResult{}, fmt.Errorf("build request: %w", err)
	}
	if step.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	addAuthHeader(req, auth)

	resp, err := client.Do(req)
	if err != nil {
		return StepResult{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return StepResult{}, fmt.Errorf("read body: %w", err)
	}

	result := StepResult{
		StatusCode:   resp.StatusCode,
		ResolvedPath: resolvedPath,
		RawBody:      string(rawBytes),
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rawBytes, &envelope); err == nil && envelope.Data != nil {
		var dataMap map[string]interface{}
		if err := json.Unmarshal(envelope.Data, &dataMap); err == nil {
			result.Data = dataMap
		}
	}
	return result, nil
}

// mergeMaps returns a new map with all base entries, then all overrides applied on top.
func mergeMaps(base, overrides map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+len(overrides))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// RunScenario executes one scenario with humanized timing.
func RunScenario(ctx context.Context, client *http.Client, bffURL string, scenario Scenario, auth *AuthConfig) {
	sessionID := generateSessionID()

	if scenario.StartOffsetSeconds > 0 {
		logf("%s: waiting %ds before starting", scenario.Name, scenario.StartOffsetSeconds)
		select {
		case <-time.After(time.Duration(scenario.StartOffsetSeconds) * time.Second):
		case <-ctx.Done():
			return
		}
	}

	carry := map[string]interface{}{
		"session_id": sessionID,
	}
	prevData := mergeMaps(nil, carry)

	for i, step := range scenario.Steps {
		if ctx.Err() != nil {
			return
		}

		preMin, preMax := step.PreDelayMin, step.PreDelayMax
		if preMin == 0 && preMax == 0 {
			preMin, preMax = defaultPreDelayMin, defaultPreDelayMax
		}
		humanDelay(ctx, preMin, preMax)
		if ctx.Err() != nil {
			return
		}

		label := formatCrewLabel(step.CrewID)
		logf("%s: %s", label, step.Description)

		result, err := ExecuteStep(client, bffURL, step, prevData, auth)
		if err != nil {
			logf("ERROR in %q step %d: %v (continuing)", scenario.Name, i+1, err)
		} else if result.StatusCode < 200 || result.StatusCode >= 300 {
			logf("ERROR in %q step %d: HTTP %d %s (continuing)", scenario.Name, i+1, result.StatusCode, result.RawBody)
		} else {
			stepData := result.Data
			if stepData == nil {
				stepData = map[string]interface{}{}
			}
			if step.StashAs != "" {
				if id, ok := stepData["id"]; ok {
					carry[step.StashAs] = id
				}
			}
			prevData = mergeMaps(stepData, carry)
		}

		postMin, postMax := step.PostDelayMin, step.PostDelayMax
		if postMin == 0 && postMax == 0 {
			postMin, postMax = defaultPostDelayMin, defaultPostDelayMax
		}
		humanDelay(ctx, postMin, postMax)
	}
}

// RunCycle launches all scenarios as concurrent goroutines and waits for all
// to finish or ctx to be cancelled.
func RunCycle(ctx context.Context, client *http.Client, bffURL string, scenarios []Scenario, auth *AuthConfig) {
	var wg sync.WaitGroup
	for _, s := range scenarios {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			RunScenario(ctx, client, bffURL, s, auth)
		}()
	}
	wg.Wait()
}

// ResetStores calls POST /api/v1/reset on the BFF to reload all seed data.
func ResetStores(ctx context.Context, client *http.Client, bffURL string, auth *AuthConfig) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, bffURL+"/api/v1/reset", nil)
	if err != nil {
		return fmt.Errorf("build reset request: %w", err)
	}
	addAuthHeader(req, auth)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("reset request failed: %w", err)
	}
	resp.Body.Close()
	return nil
}
