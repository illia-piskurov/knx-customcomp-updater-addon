package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDomain(t *testing.T) {
	svc := NewDomainService(t.TempDir())

	if err := svc.ValidateDomain("knx2"); err != nil {
		t.Fatalf("expected valid domain, got error: %v", err)
	}
	if err := svc.ValidateDomain("knx"); err == nil {
		t.Fatal("expected invalid domain error")
	}
	if err := svc.ValidateDomain("../knx2"); err == nil {
		t.Fatal("expected traversal-like invalid domain error")
	}
}

func TestDeleteDomainHardDelete(t *testing.T) {
	root := t.TempDir()
	svc := NewDomainService(root)

	domainPath := filepath.Join(root, "knx2")
	if err := os.MkdirAll(domainPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(domainPath, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteDomain("knx2"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := os.Stat(domainPath); !os.IsNotExist(err) {
		t.Fatalf("expected domain folder to be removed, stat err=%v", err)
	}
}

func TestReplaceDomainFromDirReplacesContent(t *testing.T) {
	root := t.TempDir()
	svc := NewDomainService(root)

	target := filepath.Join(root, "knx7")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := svc.ReplaceDomainFromDir("knx7", source); err != nil {
		t.Fatalf("replace failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old file should not exist after replace, stat err=%v", err)
	}

	content, err := os.ReadFile(filepath.Join(target, "new.txt"))
	if err != nil {
		t.Fatalf("new file missing: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("unexpected new file content: %s", string(content))
	}
}
