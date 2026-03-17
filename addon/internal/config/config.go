package config

import "os"

type Config struct {
	ListenAddr          string
	CustomComponentsDir string
	StaticDir           string
	SupervisorURL       string
	SupervisorToken     string
	HomeAssistantURL    string
	HomeAssistantToken  string
	GitHubOwner         string
	GitHubRepo          string
	SourceFolder        string
}

func Load() Config {
	return Config{
		ListenAddr:          getEnv("KNX_MANAGER_LISTEN", ":8080"),
		CustomComponentsDir: getEnv("CUSTOM_COMPONENTS_DIR", "/config/custom_components"),
		StaticDir:           getEnv("KNX_MANAGER_STATIC_DIR", "web/static"),
		SupervisorURL:       getEnv("SUPERVISOR_URL", "http://supervisor"),
		SupervisorToken:     firstNonEmpty(os.Getenv("SUPERVISOR_TOKEN"), os.Getenv("HASSIO_TOKEN")),
		HomeAssistantURL:    getEnv("HOMEASSISTANT_URL", "http://homeassistant:8123"),
		HomeAssistantToken:  firstNonEmpty(os.Getenv("HOMEASSISTANT_TOKEN"), os.Getenv("HASSIO_TOKEN")),
		GitHubOwner:         getEnv("GITHUB_OWNER", "home-assistant"),
		GitHubRepo:          getEnv("GITHUB_REPO", "core"),
		SourceFolder:        getEnv("SOURCE_FOLDER", "homeassistant/components/knx"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
