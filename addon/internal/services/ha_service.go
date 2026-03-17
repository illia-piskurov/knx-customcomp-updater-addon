package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"knx-updater/internal/config"
)

type HAService struct {
	cfg    config.Config
	client *http.Client
}

type supervisorCoreInfoResponse struct {
	Data struct {
		Version string `json:"version"`
	} `json:"data"`
}

type homeAssistantConfigResponse struct {
	Version string `json:"version"`
}

func NewHAService(cfg config.Config) *HAService {
	return &HAService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *HAService) GetVersion(ctx context.Context) (string, error) {
	token, tokenErr := s.resolveSupervisorToken()
	if token != "" {
		version, err := s.getVersionFromSupervisor(ctx, token)
		if err == nil {
			return version, nil
		}
		tokenErr = err
	}

	haToken := strings.TrimSpace(s.cfg.HomeAssistantToken)
	if haToken != "" {
		version, err := s.getVersionFromHomeAssistant(ctx, haToken)
		if err == nil {
			return version, nil
		}
		if tokenErr != nil {
			return "", fmt.Errorf("supervisor and homeassistant version lookup failed: %v; %v", tokenErr, err)
		}
		return "", err
	}

	if tokenErr != nil {
		return "", fmt.Errorf("version lookup failed: %v; HOMEASSISTANT_TOKEN/HASSIO_TOKEN is not set", tokenErr)
	}

	return "", errors.New("SUPERVISOR_TOKEN and HOMEASSISTANT_TOKEN/HASSIO_TOKEN are not set")
}

func (s *HAService) resolveSupervisorToken() (string, error) {
	token := strings.TrimSpace(s.cfg.SupervisorToken)
	if token == "" {
		return "", errors.New("SUPERVISOR_TOKEN is not set")
	}
	return token, nil
}

func (s *HAService) getVersionFromSupervisor(ctx context.Context, token string) (string, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.SupervisorURL+"/core/info", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("supervisor core info request failed with status %d", resp.StatusCode)
	}

	var payload supervisorCoreInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Data.Version == "" {
		return "", errors.New("supervisor response does not contain home assistant version")
	}

	return payload.Data.Version, nil
}

func (s *HAService) getVersionFromHomeAssistant(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.HomeAssistantURL+"/api/config", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("home assistant config request failed with status %d", resp.StatusCode)
	}

	var payload homeAssistantConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Version == "" {
		return "", errors.New("home assistant response does not contain version")
	}

	return payload.Version, nil
}
