package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knx-updater/internal/config"
)

func TestGetVersionFromSupervisorToken(t *testing.T) {
	tokenValue := "abc-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tokenValue {
			t.Fatalf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":{"version":"2026.3.1"}}`))
	}))
	defer server.Close()

	svc := NewHAService(config.Config{
		SupervisorURL:   server.URL,
		SupervisorToken: tokenValue,
	})

	version, err := svc.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion returned error: %v", err)
	}
	if version != "2026.3.1" {
		t.Fatalf("unexpected version: %s", version)
	}
}

func TestGetVersionFallbackToHomeAssistantAPI(t *testing.T) {
	haToken := "ha-token"
	haServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+haToken {
			t.Fatalf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/config" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"version":"2026.4.0"}`))
	}))
	defer haServer.Close()

	unreachableSupervisor := "http://127.0.0.1:9"

	svc := NewHAService(config.Config{
		SupervisorURL:      unreachableSupervisor,
		SupervisorToken:    "supervisor-token",
		HomeAssistantURL:   haServer.URL,
		HomeAssistantToken: haToken,
	})

	version, err := svc.GetVersion(context.Background())
	if err != nil {
		t.Fatalf("GetVersion returned error: %v", err)
	}
	if version != "2026.4.0" {
		t.Fatalf("unexpected version: %s", version)
	}
}

func TestGetVersionFailsWhenNoTokens(t *testing.T) {
	svc := NewHAService(config.Config{})

	_, err := svc.GetVersion(context.Background())
	if err == nil {
		t.Fatal("expected an error when no tokens are configured")
	}

	if !strings.Contains(err.Error(), "not set") {
		t.Fatalf("unexpected error: %v", err)
	}
}
