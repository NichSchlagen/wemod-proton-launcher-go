package prefix

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

const defaultPrefixRepoAPI = "https://api.github.com/repos/NichSchlagen/wemod-prefix/releases/latest"

type githubRelease struct {
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func Download(ctx context.Context, cfg *config.Config, logger *logging.Logger) error {
	logger = logger.WithComponent("prefix.download")
	logger.Info("prefix download workflow started")
	logger.Debug("download dir=%s prefix dir=%s", cfg.Paths.DownloadDir, cfg.Paths.PrefixDir)

	if err := os.MkdirAll(cfg.Paths.DownloadDir, 0o755); err != nil {
		logger.Error("failed creating download dir %s: %v", cfg.Paths.DownloadDir, err)
		return fmt.Errorf("create download dir: %w", err)
	}

	archivePath := filepath.Join(cfg.Paths.DownloadDir, "prefix.zip")

	url := strings.TrimSpace(cfg.Prefix.DownloadURL)
	if url == "" || strings.EqualFold(url, "auto") {
		logger.Debug("prefix.download_url set to auto; resolving latest release asset")
		resolved, err := resolveLatestPrefixAssetURL(ctx)
		if err != nil {
			logger.Error("failed resolving latest prefix release URL: %v", err)
			return err
		}
		url = resolved
	}

	logger.Info("downloading prefix from %s", url)
	fmt.Println("Downloading WeMod prefix ...")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("download request failed: %v", err)
		return fmt.Errorf("download prefix: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		logger.Warn("primary prefix URL returned status %d", resp.StatusCode)
		fallbackURL, fallbackErr := resolveLatestPrefixAssetURL(ctx)
		if fallbackErr != nil {
			logger.Error("fallback URL resolution failed after status %d: %v", resp.StatusCode, fallbackErr)
			return fmt.Errorf("download prefix failed with status %d", resp.StatusCode)
		}
		if fallbackURL != url {
			logger.Warn("configured prefix URL failed (status %d), retrying with latest release", resp.StatusCode)
			fmt.Println("Retrying with latest release ...")
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, fallbackURL, nil)
			if err != nil {
				return fmt.Errorf("create fallback request: %w", err)
			}
			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				logger.Error("fallback download failed: %v", err)
				return fmt.Errorf("download fallback prefix: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				logger.Error("fallback URL returned status %d", resp.StatusCode)
				return fmt.Errorf("download prefix failed with status %d", resp.StatusCode)
			}
		} else {
			logger.Error("download failed with status %d and fallback URL equals primary URL", resp.StatusCode)
			return fmt.Errorf("download prefix failed with status %d", resp.StatusCode)
		}
	}

	out, err := os.Create(archivePath)
	if err != nil {
		logger.Error("failed creating archive file %s: %v", archivePath, err)
		return fmt.Errorf("create archive file: %w", err)
	}

	total := resp.ContentLength
	written, err := io.Copy(out, &progressReader{r: resp.Body, total: total})
	if err != nil {
		out.Close()
		logger.Error("failed writing archive file %s: %v", archivePath, err)
		return fmt.Errorf("save archive: %w", err)
	}
	if err := out.Close(); err != nil {
		logger.Error("failed closing archive file %s: %v", archivePath, err)
		return fmt.Errorf("close archive file: %w", err)
	}
	logger.Debug("archive downloaded: %.2f MB", float64(written)/1024.0/1024.0)
	fmt.Printf("\rDownloaded %.1f MB                        \n", float64(written)/1024/1024)

	// Remove old prefix before extracting
	if err := os.RemoveAll(cfg.Paths.PrefixDir); err != nil {
		logger.Error("failed removing old prefix dir %s: %v", cfg.Paths.PrefixDir, err)
		return fmt.Errorf("remove old prefix: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.PrefixDir, 0o755); err != nil {
		logger.Error("failed recreating prefix dir %s: %v", cfg.Paths.PrefixDir, err)
		return fmt.Errorf("create prefix dir: %w", err)
	}

	fmt.Println("Extracting prefix archive ...")
	logger.Info("extracting prefix archive to %s", cfg.Paths.PrefixDir)
	if err := extractZip(archivePath, cfg.Paths.PrefixDir); err != nil {
		logger.Error("failed extracting prefix archive %s: %v", archivePath, err)
		return err
	}

	// Clean up downloaded archive
	_ = os.Remove(archivePath)

	// Initialize the prefix on this system so the first WeMod launch is clean.
	// Run wineboot headless (DISPLAY unset) to prevent Wine error dialogs while
	// still performing all registry and server-registration work.
	fmt.Println("Initializing Wine prefix ...")
	logger.Info("running wineboot -u to initialize prefix")
	if bootErr := runWinebootHeadless(ctx, cfg.Paths.PrefixDir); bootErr != nil {
		logger.Warn("wineboot initialization failed, first launch may show a Wine dialog: %v", bootErr)
	} else {
		logger.Debug("wineboot initialization finished successfully")
	}

	fmt.Printf("Prefix ready at %s\n", cfg.Paths.PrefixDir)
	logger.Info("prefix ready at %s", cfg.Paths.PrefixDir)
	logger.Info("prefix download workflow completed")
	return nil
}

// progressReader prints a simple download progress line.
type progressReader struct {
	r       io.Reader
	total   int64
	written int64
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.written += int64(n)
	if p.total > 0 {
		pct := float64(p.written) / float64(p.total) * 100
		fmt.Printf("\r  %.0f%% (%.1f / %.1f MB)", pct, float64(p.written)/1024/1024, float64(p.total)/1024/1024)
	} else {
		fmt.Printf("\r  %.1f MB downloaded", float64(p.written)/1024/1024)
	}
	return n, err
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

	// Resolve dest to absolute path for traversal checks.
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve dest path: %w", err)
	}

	for _, f := range r.File {
		target := filepath.Join(destAbs, f.Name)

		// Path traversal guard.
		if !strings.HasPrefix(target, destAbs+string(os.PathSeparator)) && target != destAbs {
			return fmt.Errorf("zip entry %q would escape destination, refusing to extract", f.Name)
		}

		mode := f.Mode()

		// Symlink.
		if mode&os.ModeSymlink != 0 {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("open symlink zip entry %s: %w", f.Name, err)
			}
			linkTargetBytes, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return fmt.Errorf("read symlink target %s: %w", f.Name, err)
			}
			linkTarget := string(linkTargetBytes)
			// Remove existing entry.
			_ = os.Remove(target)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent dir for symlink %s: %w", target, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				return fmt.Errorf("create symlink %s -> %s: %w", target, linkTarget, err)
			}
			continue
		}

		// Directory.
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}
			continue
		}

		// Regular file.
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

// runWinebootHeadless runs "wineboot -u" with DISPLAY unset so Wine cannot
// open error dialogs on screen. All other environment variables are kept so
// that Wine can find its libraries and the prefix.
func runWinebootHeadless(ctx context.Context, prefixDir string) error {
	overrides := map[string]string{
		"WINEPREFIX":       prefixDir,
		"DISPLAY":          "",
		"WAYLAND_DISPLAY":  "",
		"WINEDLLOVERRIDES": "mscoree,mshtml=",
	}

	// Build env: start from os.Environ, then apply overrides (replace if key exists).
	base := os.Environ()
	filtered := make([]string, 0, len(base))
	for _, kv := range base {
		key := kv
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			key = kv[:idx]
		}
		if _, overridden := overrides[key]; !overridden {
			filtered = append(filtered, kv)
		}
	}
	for k, v := range overrides {
		filtered = append(filtered, k+"="+v)
	}

	cmd := exec.CommandContext(ctx, "wineboot", "-u")
	cmd.Env = filtered
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}
