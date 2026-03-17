package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"knx-updater/internal/config"
)

type UpdaterService struct {
	cfg     config.Config
	ha      *HAService
	domains *DomainService
	client  *http.Client
}

func NewUpdaterService(cfg config.Config, ha *HAService, domains *DomainService) *UpdaterService {
	return &UpdaterService{
		cfg:     cfg,
		ha:      ha,
		domains: domains,
		client: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (s *UpdaterService) UpdateDomain(ctx context.Context, domain string, requestedVersion string, logf func(string)) error {
	if err := s.domains.ValidateDomain(domain); err != nil {
		return err
	}

	version := requestedVersion
	if version == "" {
		v, err := s.ha.GetVersion(ctx)
		if err != nil {
			return err
		}
		version = v
	}

	if logf != nil {
		logf(fmt.Sprintf("using Home Assistant version %s", version))
	}

	tmpDir, err := os.MkdirTemp("", "knx-manager-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "source.zip")
	if err := s.downloadArchive(ctx, version, archivePath); err != nil {
		return err
	}
	if logf != nil {
		logf("download complete")
	}

	extractedDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractedDir, 0o755); err != nil {
		return err
	}

	repoPrefix := fmt.Sprintf("%s-%s", s.cfg.GitHubRepo, version)
	if err := extractFolderFromZip(archivePath, repoPrefix, s.cfg.SourceFolder, extractedDir); err != nil {
		return err
	}
	if logf != nil {
		logf("knx integration extracted")
	}

	constPath := filepath.Join(extractedDir, "const.py")
	if err := rewriteDomainConst(constPath, domain); err != nil {
		return err
	}

	manifestPath := filepath.Join(extractedDir, "manifest.json")
	if err := rewriteManifest(manifestPath, domain, version); err != nil {
		return err
	}
	if logf != nil {
		logf("component files rewritten")
	}

	if err := s.domains.ReplaceDomainFromDir(domain, extractedDir); err != nil {
		return err
	}
	if logf != nil {
		logf("domain updated successfully")
	}

	return nil
}

func (s *UpdaterService) UpdateAll(ctx context.Context, logf func(string)) error {
	domains, err := s.domains.ListDomains()
	if err != nil {
		return err
	}
	if len(domains) == 0 {
		if logf != nil {
			logf("no installed knx domains found")
		}
		return nil
	}

	for _, domain := range domains {
		if logf != nil {
			logf("updating " + domain.Name)
		}
		if err := s.UpdateDomain(ctx, domain.Name, "", logf); err != nil {
			return fmt.Errorf("update failed for %s: %w", domain.Name, err)
		}
	}

	return nil
}

func (s *UpdaterService) downloadArchive(ctx context.Context, version string, outputPath string) error {
	url := fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.zip", s.cfg.GitHubOwner, s.cfg.GitHubRepo, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractFolderFromZip(archivePath string, repoPrefix string, sourceFolder string, outputDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	zipPrefix := strings.TrimSuffix(repoPrefix+"/"+sourceFolder, "/") + "/"
	foundAny := false

	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, zipPrefix) {
			continue
		}
		foundAny = true
		relative := strings.TrimPrefix(file.Name, zipPrefix)
		if relative == "" {
			continue
		}

		targetPath := filepath.Join(outputDir, filepath.FromSlash(relative))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		in, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			in.Close()
			return err
		}

		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}

	if !foundAny {
		return fmt.Errorf("source folder %s not found in archive", sourceFolder)
	}

	return nil
}

func rewriteDomainConst(filePath string, domain string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content := string(contentBytes)
	replaced := strings.ReplaceAll(content, `DOMAIN: Final = "knx"`, fmt.Sprintf(`DOMAIN: Final = "%s"`, domain))
	if replaced == content {
		replaced = strings.ReplaceAll(content, `DOMAIN = "knx"`, fmt.Sprintf(`DOMAIN = "%s"`, domain))
	}
	if replaced == content {
		return fmt.Errorf("could not locate knx DOMAIN definition in %s", filePath)
	}

	return os.WriteFile(filePath, []byte(replaced), 0o644)
}

func rewriteManifest(filePath string, domain string, _ string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var manifest map[string]any
	if err := json.Unmarshal(contentBytes, &manifest); err != nil {
		return err
	}

	manifest["domain"] = domain
	manifest["name"] = strings.ToUpper(domain)
	manifest["version"] = "1.0.0"

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, out, 0o644)
}
