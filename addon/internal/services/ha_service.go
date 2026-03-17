package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

func NewHAService(cfg config.Config) *HAService {
	return &HAService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (s *HAService) GetVersion(ctx context.Context) (string, error) {
	if s.cfg.SupervisorToken == "" {
		return "", errors.New("SUPERVISOR_TOKEN is not set")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.SupervisorURL+"/core/info", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.SupervisorToken)

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
