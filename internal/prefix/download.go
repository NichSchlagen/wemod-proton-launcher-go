package prefix

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

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

const defaultPrefixRepoAPI = "https://api.github.com/repos/DeckCheatz/BuiltPrefixes-dev/releases/latest"

type githubRelease struct {
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func Download(ctx context.Context, cfg *config.Config, logger *logging.Logger) error {
	if err := os.MkdirAll(cfg.Paths.DownloadDir, 0o755); err != nil {
		return fmt.Errorf("create download dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.PrefixDir, 0o755); err != nil {
		return fmt.Errorf("create prefix dir: %w", err)
	}

	archivePath := filepath.Join(cfg.Paths.DownloadDir, "prefix.zip")

	url := strings.TrimSpace(cfg.Prefix.DownloadURL)
	if url == "" || strings.EqualFold(url, "auto") {
		resolved, err := resolveLatestPrefixAssetURL(ctx)
		if err != nil {
			return err
		}
		url = resolved
	}

	logger.Info("downloading prefix from %s", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download prefix: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fallbackURL, fallbackErr := resolveLatestPrefixAssetURL(ctx)
		if fallbackErr != nil {
			return fmt.Errorf("download prefix failed with status %d", resp.StatusCode)
		}
		if fallbackURL != url {
			logger.Warn("configured prefix URL failed with status %d, retrying with %s", resp.StatusCode, fallbackURL)
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, fallbackURL, nil)
			if err != nil {
				return fmt.Errorf("create fallback request: %w", err)
			}
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("download fallback prefix: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return fmt.Errorf("download prefix failed with status %d", resp.StatusCode)
			}
		} else {
			return fmt.Errorf("download prefix failed with status %d", resp.StatusCode)
		}
	}

	out, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive file: %w", err)
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		return fmt.Errorf("save archive: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close archive file: %w", err)
	}

	logger.Info("extracting prefix archive")
	if err := extractZip(archivePath, cfg.Paths.PrefixDir); err != nil {
		return err
	}

	logger.Info("prefix ready at %s", cfg.Paths.PrefixDir)
	return nil
}

func resolveLatestPrefixAssetURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultPrefixRepoAPI, nil)
	if err != nil {
		return "", fmt.Errorf("create prefix release request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest prefix release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("latest prefix release lookup failed with status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode latest prefix release: %w", err)
	}

	for _, asset := range release.Assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		if strings.HasSuffix(name, ".zip") && strings.TrimSpace(asset.URL) != "" {
			return strings.TrimSpace(asset.URL), nil
		}
	}

	return "", fmt.Errorf("latest prefix release does not contain a .zip asset")
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create parent dir %s: %w", target, err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("create extracted file %s: %w", target, err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return fmt.Errorf("extract %s: %w", target, err)
		}
		if err := out.Close(); err != nil {
			rc.Close()
			return fmt.Errorf("close extracted file %s: %w", target, err)
		}
		if err := rc.Close(); err != nil {
			return fmt.Errorf("close zip entry %s: %w", target, err)
		}
	}

	return nil
}
