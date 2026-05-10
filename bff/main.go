package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := LoadConfig()
	server := NewServer(cfg)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting BFF on %s", addr)
	if err := http.ListenAndServe(addr, server); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
