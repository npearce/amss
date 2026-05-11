package main

import "os"

type Config struct {
	Port                   string
	KBStoreURL             string
	TicketStoreURL         string
	CrewStoreURL           string
	MissionSupportAgentURL string
	KagentAgentNamespace   string
	KBCuratorAgentURL      string
	StubMode               bool
}

func LoadConfig() *Config {
	return &Config{
		Port:                   getEnv("PORT", "8080"),
		KBStoreURL:             getEnv("KB_STORE_URL", "http://kb-store:8081"),
		TicketStoreURL:         getEnv("TICKET_STORE_URL", "http://ticket-store:8082"),
		CrewStoreURL:           getEnv("CREW_STORE_URL", "http://crew-store:8083"),
		MissionSupportAgentURL: getEnv("MISSION_SUPPORT_AGENT_URL", "http://kagent-controller.kagent.svc.cluster.local:8083"),
		KagentAgentNamespace:   getEnv("KAGENT_AGENT_NAMESPACE", "kagent"),
		KBCuratorAgentURL:      getEnv("KB_CURATOR_AGENT_URL", "http://kb-curator-agent:8091"),
		StubMode:               getEnv("STUB_MODE", "true") != "false",
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
