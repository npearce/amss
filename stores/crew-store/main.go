package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	seedPath := os.Getenv("SEED_PATH")
	if seedPath == "" {
		seedPath = "seed-data/crew.json"
	}

	s, err := NewStore(seedPath)
	if err != nil {
		log.Fatalf("failed to initialize store: %v", err)
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

	log.Printf("crew-store listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
