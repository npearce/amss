package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	seedPath := os.Getenv("SEED_PATH")
	if seedPath == "" {
		seedPath = "seed-data/tickets.json"
	}

	var err error
	store, err = NewStore(seedPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("GET /tickets", handleTickets)
	mux.HandleFunc("POST /tickets", handleTickets)
	mux.HandleFunc("GET /tickets/{id}", handleTicketByID)
	mux.HandleFunc("PUT /tickets/{id}", handleTicketByID)
	mux.HandleFunc("POST /tickets/{id}/comments", handleAddComment)
	mux.HandleFunc("POST /reset", handleReset)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting Ticket store on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
