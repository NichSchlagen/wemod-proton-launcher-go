package bootstrap

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

const scoopMetadataURL = "https://raw.githubusercontent.com/Calinou/scoop-games/refs/heads/master/bucket/wemod.json"

type scoopMetadata struct {
	Architecture struct {
		Bit64 struct {
			URL string `json:"url"`
		} `json:"64bit"`
	} `json:"architecture"`
}

func EnsureWeMod(ctx context.Context, cfg *config.Config, logger *logging.Logger, force bool) error {
	if !force {
		if _, err := os.Stat(cfg.Paths.WeModExePath); err == nil {
			logger.Info("WeMod executable already present at %s", cfg.Paths.WeModExePath)
			return nil
		}
	}

	if err := os.MkdirAll(cfg.Paths.DownloadDir, 0o755); err != nil {
		return fmt.Errorf("create download dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.Paths.WeModExePath), 0o755); err != nil {
		return fmt.Errorf("create wemod bin dir: %w", err)
	}

	url, err := fetchWeModDownloadURL(ctx)
	if err != nil {
		return err
	}
	logger.Info("resolved WeMod installer URL")

	installerPath := filepath.Join(cfg.Paths.DownloadDir, "wemod-setup.exe")
	if err := downloadFile(ctx, url, installerPath); err != nil {
		return err
	}
	logger.Info("downloaded WeMod installer")

	installRoot := filepath.Dir(cfg.Paths.WeModExePath)
	if err := os.RemoveAll(installRoot); err != nil {
		return fmt.Errorf("cleanup old wemod install: %w", err)
	}
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		return fmt.Errorf("create install root: %w", err)
	}

	if err := extractNetPayload(installerPath, installRoot); err != nil {
		return err
	}
	if _, err := os.Stat(cfg.Paths.WeModExePath); err != nil {
		return fmt.Errorf("wemod extracted but executable missing at %s", cfg.Paths.WeModExePath)
	}

	logger.Info("WeMod installed to %s", installRoot)
	return nil
}

func fetchWeModDownloadURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scoopMetadataURL, nil)
	if err != nil {
		return "", fmt.Errorf("create scoop metadata request: %w", err)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch scoop metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("scoop metadata request failed with status %d", resp.StatusCode)
	}

	var metadata scoopMetadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", fmt.Errorf("decode scoop metadata: %w", err)
	}
	url := strings.TrimSpace(metadata.Architecture.Bit64.URL)
	if url == "" {
		return "", errors.New("scoop metadata does not contain a 64bit WeMod URL")
	}
	return url, nil
}

func downloadFile(ctx context.Context, url, destination string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write downloaded file: %w", err)
	}
	return nil
}

func extractNetPayload(installerPath, destination string) error {
	r, err := zip.OpenReader(installerPath)
	if err != nil {
		return fmt.Errorf("open installer archive: %w", err)
	}
	defer r.Close()

	extracted := false
	for _, file := range r.File {
		archiveName := strings.ReplaceAll(file.Name, "\\", "/")
		if !strings.HasPrefix(archiveName, "lib/net") {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(archiveName))
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("unsafe archive path: %s", archiveName)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, file.Mode()); err != nil {
				return fmt.Errorf("create extracted directory: %w", err)
			}
			extracted = true
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			return fmt.Errorf("create extracted parent dir: %w", err)
		}
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("open archive entry: %w", err)
		}
		out, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create extracted file: %w", err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("extract archive entry: %w", err)
		}
		if err := out.Close(); err != nil {
			rc.Close()
			return fmt.Errorf("close extracted file: %w", err)
		}
		if err := rc.Close(); err != nil {
			return fmt.Errorf("close archive entry: %w", err)
		}
		extracted = true
	}

	if !extracted {
		return errors.New("installer archive did not contain lib/net payload")
	}

	payloadRoot, err := findNetPayloadRoot(filepath.Join(destination, "lib"))
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(payloadRoot)
	if err != nil {
		return fmt.Errorf("read net payload directory: %w", err)
	}
	for _, entry := range entries {
		src := filepath.Join(payloadRoot, entry.Name())
		dst := filepath.Join(destination, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move extracted payload: %w", err)
		}
	}

	return os.RemoveAll(filepath.Join(destination, "lib"))
}

func findNetPayloadRoot(libDir string) (string, error) {
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return "", fmt.Errorf("read lib directory: %w", err)
	}

	candidates := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if strings.HasPrefix(strings.ToLower(entry.Name()), "net") {
			candidates = append(candidates, filepath.Join(libDir, entry.Name()))
		}
	}

	if len(candidates) == 0 {
		return "", errors.New("read net payload directory: no lib/net* payload found in installer")
	}

	sort.Strings(candidates)
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "WeMod.exe")); err == nil {
			return candidate, nil
		}
	}

	return candidates[0], nil
}
