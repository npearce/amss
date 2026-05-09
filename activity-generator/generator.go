package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Scenario is a named sequence of steps to execute against the BFF.
type Scenario struct {
	Name  string `json:"name"`
	Steps []Step `json:"steps"`
}

// Step is a single HTTP request within a scenario.
type Step struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Body        map[string]interface{} `json:"body,omitempty"`
	Description string                 `json:"description"`
}

// StepResult holds the outcome of executing a single step.
type StepResult struct {
	StatusCode   int
	ResolvedPath string
	Data         map[string]interface{}
	RawBody      string
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

// resolveTemplates replaces {{prev.<key>}} placeholders in s using prevData.
func resolveTemplates(s string, prevData map[string]interface{}) string {
	for k, v := range prevData {
		placeholder := "{{prev." + k + "}}"
		s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", v))
	}
	return s
}

// ExecuteStep sends one HTTP request and returns the result.
// prevData contains the data map from the previous step's response envelope.
func ExecuteStep(client *http.Client, bffURL string, step Step, prevData map[string]interface{}) (StepResult, error) {
	resolvedPath := resolveTemplates(step.Path, prevData)
	url := bffURL + resolvedPath

	var bodyReader io.Reader
	if step.Body != nil {
		raw, err := json.Marshal(step.Body)
		if err != nil {
			return StepResult{}, fmt.Errorf("marshal body: %w", err)
		}
		resolved := resolveTemplates(string(raw), prevData)
		bodyReader = bytes.NewBufferString(resolved)
	}

	req, err := http.NewRequest(step.Method, url, bodyReader)
	if err != nil {
		return StepResult{}, fmt.Errorf("build request: %w", err)
	}
	if step.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return StepResult{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return StepResult{}, fmt.Errorf("read body: %w", err)
	}
	raw := string(rawBytes)

	result := StepResult{
		StatusCode:   resp.StatusCode,
		ResolvedPath: resolvedPath,
		RawBody:      raw,
	}

	// Unwrap the standard envelope { "data": {...}, "error": null }
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

// RunScenario executes all steps in a scenario sequentially.
// On a non-2xx response, it logs the error and stops the current scenario.
func RunScenario(client *http.Client, bffURL string, scenario Scenario) {
	fmt.Printf("[scenario] %s\n", scenario.Name)
	var prevData map[string]interface{}

	for i, step := range scenario.Steps {
		fmt.Printf("  [step %d] %s %s — %s\n", i+1, step.Method, step.Path, step.Description)
		result, err := ExecuteStep(client, bffURL, step, prevData)
		if err != nil {
			fmt.Printf("  [error] step %d failed: %v\n", i+1, err)
			return
		}
		if result.StatusCode < 200 || result.StatusCode >= 300 {
			fmt.Printf("  [error] step %d returned %d: %s\n", i+1, result.StatusCode, result.RawBody)
			return
		}
		fmt.Printf("  [ok] %d %s\n", result.StatusCode, result.ResolvedPath)
		prevData = result.Data
	}
}
