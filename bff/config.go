package main

import "os"

type Config struct {
	Port                   string
	KBStoreURL             string
	TicketStoreURL         string
	CrewStoreURL           string
	MissionSupportAgentURL string
	KBCuratorAgentURL      string
}

func LoadConfig() *Config {
	return &Config{
		Port:                   getEnv("PORT", "8080"),
		KBStoreURL:             getEnv("KB_STORE_URL", "http://kb-store:8081"),
		TicketStoreURL:         getEnv("TICKET_STORE_URL", "http://ticket-store:8082"),
		CrewStoreURL:           getEnv("CREW_STORE_URL", "http://crew-store:8083"),
		MissionSupportAgentURL: getEnv("MISSION_SUPPORT_AGENT_URL", "http://mission-support-agent:8090"),
		KBCuratorAgentURL:      getEnv("KB_CURATOR_AGENT_URL", "http://kb-curator-agent:8091"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
