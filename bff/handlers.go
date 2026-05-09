package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Envelope struct {
	Data  interface{} `json:"data"`
	Error *ErrorInfo  `json:"error"`
}

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type ChatRequest struct {
	CrewID    string `json:"crew_id"`
	SessionID string `json:"session_id"`
	Mission   string `json:"mission"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	Response             string   `json:"response"`
	KBArticlesReferenced []string `json:"kb_articles_referenced"`
	TicketCreated        *string  `json:"ticket_created"`
}

type ResetResponse struct {
	Message string            `json:"message"`
	Results map[string]string `json:"results"`
}

type Server struct {
	cfg *Config
	mux *http.ServeMux
}

func NewServer(cfg *Config) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)

	// KB Articles — proxy to kb-store
	s.mux.HandleFunc("/articles", s.proxyTo(s.cfg.KBStoreURL))
	s.mux.HandleFunc("/articles/{id}", s.proxyTo(s.cfg.KBStoreURL))

	// Tickets — proxy to ticket-store
	s.mux.HandleFunc("/tickets", s.proxyTo(s.cfg.TicketStoreURL))
	s.mux.HandleFunc("/tickets/{id}", s.proxyTo(s.cfg.TicketStoreURL))
	s.mux.HandleFunc("POST /tickets/{id}/comments", s.proxyTo(s.cfg.TicketStoreURL))

	// Crew and conversations — proxy to crew-store
	s.mux.HandleFunc("/crew", s.proxyTo(s.cfg.CrewStoreURL))
	s.mux.HandleFunc("/crew/{id}", s.proxyTo(s.cfg.CrewStoreURL))
	s.mux.HandleFunc("/crew/{id}/activity", s.proxyTo(s.cfg.CrewStoreURL))
	s.mux.HandleFunc("/conversations", s.proxyTo(s.cfg.CrewStoreURL))

	// Agent routes
	s.mux.HandleFunc("POST /chat", s.handleChat)
	s.mux.HandleFunc("POST /curate", s.handleCurate)

	// Reset all stores
	s.mux.HandleFunc("POST /reset", s.handleReset)
}

func (s *Server) proxyTo(targetBase string) http.HandlerFunc {
	target, _ := url.Parse(targetBase)
	proxy := httputil.NewSingleHostReverseProxy(target)
	return proxy.ServeHTTP
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, &HealthResponse{Status: "ok", Service: "bff"})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}
	if req.CrewID == "" || req.Message == "" {
		respondError(w, http.StatusBadRequest, "BAD_REQUEST", "crew_id and message are required")
		return
	}
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("sess-%s", req.CrewID)
	}

	var agentResp *ChatResponse
	if s.cfg.StubMode {
		agentResp = stubChatResponse(req.Message)
	} else {
		var err error
		agentResp, err = s.callMissionAgent(req)
		if err != nil {
			agentResp = stubChatResponse(req.Message)
			agentResp.Response = "[Agent unavailable — using knowledge base stub]\n\n" + agentResp.Response
		}
	}

	s.logConversation(req, agentResp)
	respondJSON(w, http.StatusOK, agentResp)
}

func (s *Server) handleCurate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.StubMode {
		respondJSON(w, http.StatusOK, stubCuratorReport())
		return
	}

	req, err := http.NewRequest(http.MethodPost, s.cfg.KBCuratorAgentURL+"/curate", r.Body)
	if err != nil {
		report := stubCuratorReport()
		report.Summary = "[Curator agent unavailable] " + report.Summary
		respondJSON(w, http.StatusOK, report)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		report := stubCuratorReport()
		report.Summary = "[Curator agent unavailable] " + report.Summary
		respondJSON(w, http.StatusOK, report)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	stores := map[string]string{
		"kb-store":     s.cfg.KBStoreURL + "/reset",
		"ticket-store": s.cfg.TicketStoreURL + "/reset",
		"crew-store":   s.cfg.CrewStoreURL + "/reset",
	}

	results := make(map[string]string, len(stores))
	for name, resetURL := range stores {
		resp, err := http.Post(resetURL, "application/json", nil)
		if err != nil {
			results[name] = fmt.Sprintf("error: %v", err)
			continue
		}
		resp.Body.Close()
		results[name] = fmt.Sprintf("status %d", resp.StatusCode)
	}

	respondJSON(w, http.StatusOK, &ResetResponse{
		Message: "Reset all stores",
		Results: results,
	})
}

func (s *Server) callMissionAgent(req ChatRequest) (*ChatResponse, error) {
	payload, _ := json.Marshal(req)
	resp, err := http.Post(s.cfg.MissionSupportAgentURL+"/chat", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var agentResp ChatResponse
	if err := json.Unmarshal(body, &agentResp); err != nil {
		return nil, fmt.Errorf("invalid agent response: %v", err)
	}
	if agentResp.KBArticlesReferenced == nil {
		agentResp.KBArticlesReferenced = []string{}
	}
	return &agentResp, nil
}

func (s *Server) logConversation(req ChatRequest, agentResp *ChatResponse) {
	payload := map[string]interface{}{
		"crew_id":                req.CrewID,
		"mission":                req.Mission,
		"session_id":             req.SessionID,
		"query":                  req.Message,
		"response":               agentResp.Response,
		"kb_articles_referenced": agentResp.KBArticlesReferenced,
		"ticket_created":         agentResp.TicketCreated,
	}
	body, _ := json.Marshal(payload)
	httpResp, err := http.Post(s.cfg.CrewStoreURL+"/conversations", "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	httpResp.Body.Close()
}

func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&Envelope{Data: data, Error: nil})
}

func respondError(w http.ResponseWriter, statusCode int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&Envelope{
		Data:  nil,
		Error: &ErrorInfo{Code: code, Message: message},
	})
}
