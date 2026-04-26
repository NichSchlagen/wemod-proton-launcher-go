package doctor

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"runtime"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

type Options struct {
	FailOnMissing bool
}

func Run(_ context.Context, cfg *config.Config, logger *logging.Logger, options Options) error {
	logger = logger.WithComponent("doctor")
	logger.Debug("doctor run started (fail_on_missing=%t)", options.FailOnMissing)
	logger.Debug("work_dir=%s prefix_dir=%s download_dir=%s", cfg.Paths.WorkDir, cfg.Paths.PrefixDir, cfg.Paths.DownloadDir)

	if runtime.GOOS != "linux" {
		logger.Error("unsupported OS detected: %s", runtime.GOOS)
		return fmt.Errorf("unsupported OS %s: Linux only", runtime.GOOS)
	}

	required := []string{"wine", "wineserver", "winetricks"}
	var missing []string
	for _, bin := range required {
		if _, err := osexec.LookPath(bin); err != nil {
			missing = append(missing, bin)
			logger.Warn("missing dependency: %s", bin)
			continue
		}
		logger.Info("dependency OK: %s", bin)
	}

	if err := os.MkdirAll(cfg.Paths.WorkDir, 0o755); err != nil {
		logger.Error("failed creating work dir %s: %v", cfg.Paths.WorkDir, err)
		return fmt.Errorf("create work dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.PrefixDir, 0o755); err != nil {
		logger.Error("failed creating prefix dir %s: %v", cfg.Paths.PrefixDir, err)
		return fmt.Errorf("create prefix dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.DownloadDir, 0o755); err != nil {
		logger.Error("failed creating download dir %s: %v", cfg.Paths.DownloadDir, err)
		return fmt.Errorf("create download dir: %w", err)
	}

	if len(missing) > 0 && options.FailOnMissing {
		logger.Error("doctor failed due to missing dependencies: %v", missing)
		return fmt.Errorf("missing required dependencies: %v", missing)
	}
	if len(missing) > 0 {
		logger.Warn("doctor completed with missing optional dependencies: %v", missing)
	}
	if len(missing) == 0 {
		logger.Info("doctor finished successfully")
	}
	logger.Debug("doctor run completed")
	return nil
}
