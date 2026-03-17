package config

import "os"

type Config struct {
	ListenAddr          string
	CustomComponentsDir string
	StaticDir           string
	SupervisorURL       string
	SupervisorToken     string
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
		SupervisorToken:     os.Getenv("SUPERVISOR_TOKEN"),
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
