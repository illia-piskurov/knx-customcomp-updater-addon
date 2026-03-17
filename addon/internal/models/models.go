package models

type DomainInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type SystemInfo struct {
	HAVersion      string `json:"haVersion"`
	SupervisorURL  string `json:"supervisorUrl"`
	ComponentsPath string `json:"componentsPath"`
}
