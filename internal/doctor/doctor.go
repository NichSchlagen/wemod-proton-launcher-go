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
	if runtime.GOOS != "linux" {
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
		return fmt.Errorf("create work dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.PrefixDir, 0o755); err != nil {
		return fmt.Errorf("create prefix dir: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.DownloadDir, 0o755); err != nil {
		return fmt.Errorf("create download dir: %w", err)
	}

	if len(missing) > 0 && options.FailOnMissing {
		return fmt.Errorf("missing required dependencies: %v", missing)
	}
	if len(missing) == 0 {
		logger.Info("doctor finished successfully")
	}
	return nil
}
