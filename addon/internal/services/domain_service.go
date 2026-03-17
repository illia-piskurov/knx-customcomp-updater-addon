package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"knx-updater/internal/models"
)

var domainPattern = regexp.MustCompile(`^knx[0-9]+$`)

type DomainService struct {
	root string

	locksMu sync.Mutex
	locks   map[string]*sync.Mutex
}

func NewDomainService(root string) *DomainService {
	return &DomainService{
		root:  root,
		locks: map[string]*sync.Mutex{},
	}
}

func (s *DomainService) Root() string {
	return s.root
}

func (s *DomainService) ValidateDomain(domain string) error {
	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("domain must match knxN format")
	}
	return nil
}

func (s *DomainService) ListDomains() ([]models.DomainInfo, error) {
	if err := os.MkdirAll(s.root, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, err
	}

	result := make([]models.DomainInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !domainPattern.MatchString(name) {
			continue
		}
		target, err := s.safeDomainPath(name)
		if err != nil {
			continue
		}
		result = append(result, models.DomainInfo{Name: name, Path: target})
	}

	return result, nil
}

func (s *DomainService) DeleteDomain(domain string) error {
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	lock := s.domainLock(domain)
	lock.Lock()
	defer lock.Unlock()

	target, err := s.safeDomainPath(domain)
	if err != nil {
		return err
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("domain %s does not exist", domain)
	}

	return os.RemoveAll(target)
}

func (s *DomainService) ReplaceDomainFromDir(domain string, sourceDir string) error {
	if err := s.ValidateDomain(domain); err != nil {
		return err
	}
	lock := s.domainLock(domain)
	lock.Lock()
	defer lock.Unlock()

	target, err := s.safeDomainPath(domain)
	if err != nil {
		return err
	}

	staging, err := os.MkdirTemp(s.root, "staging-"+domain+"-")
	if err != nil {
		return err
	}
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := copyDir(sourceDir, staging); err != nil {
		return err
	}

	if _, err := os.Stat(target); err == nil {
		if err := os.RemoveAll(target); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
	}

	if err := os.Rename(staging, target); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}

	return nil
}

func (s *DomainService) safeDomainPath(domain string) (string, error) {
	cleanedRoot, err := filepath.Abs(s.root)
	if err != nil {
		return "", err
	}

	target := filepath.Join(cleanedRoot, domain)
	cleanedTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}

	rootLower := strings.ToLower(cleanedRoot)
	targetLower := strings.ToLower(cleanedTarget)
	prefix := rootLower + string(os.PathSeparator)
	if targetLower != rootLower && !strings.HasPrefix(targetLower, prefix) {
		return "", fmt.Errorf("path traversal rejected")
	}

	return cleanedTarget, nil
}

func (s *DomainService) domainLock(domain string) *sync.Mutex {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()

	lock, ok := s.locks[domain]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[domain] = lock
	}
	return lock
}

func copyDir(src string, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		targetPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		inFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer inFile.Close()

		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, inFile)
		return err
	})
}
