package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRewriteDomainConst(t *testing.T) {
	dir := t.TempDir()
	constPath := filepath.Join(dir, "const.py")

	input := "from typing import Final\nDOMAIN: Final = \"knx\"\n"
	if err := os.WriteFile(constPath, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteDomainConst(constPath, "knx9"); err != nil {
		t.Fatalf("rewriteDomainConst failed: %v", err)
	}

	out, err := os.ReadFile(constPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "from typing import Final\nDOMAIN: Final = \"knx9\"\n" {
		t.Fatalf("unexpected const.py content: %s", string(out))
	}
}

func TestRewriteManifest(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	input := `{"domain":"knx","name":"KNX","version":"0.0.1"}`
	if err := os.WriteFile(manifestPath, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := rewriteManifest(manifestPath, "knx12", "2025.3.0"); err != nil {
		t.Fatalf("rewriteManifest failed: %v", err)
	}

	out, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("manifest is not valid json: %v", err)
	}

	if payload["domain"] != "knx12" {
		t.Fatalf("unexpected domain: %v", payload["domain"])
	}
	if payload["name"] != "KNX12" {
		t.Fatalf("unexpected name: %v", payload["name"])
	}
	if payload["version"] != "1.0.0" {
		t.Fatalf("unexpected version: %v", payload["version"])
	}
}
