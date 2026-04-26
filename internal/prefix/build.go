package prefix

import (
	"context"
	"fmt"
	"os"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
	process "github.com/NichSchlagen/wemod-proton-launcher-go/internal/runtime"
)

func Build(ctx context.Context, cfg *config.Config, logger *logging.Logger) error {
	logger = logger.WithComponent("prefix.build")
	logger.Info("prefix build started")
	logger.Debug("target prefix dir: %s", cfg.Paths.PrefixDir)

	if err := os.MkdirAll(cfg.Paths.PrefixDir, 0o755); err != nil {
		logger.Error("failed creating prefix dir %s: %v", cfg.Paths.PrefixDir, err)
		return fmt.Errorf("create prefix dir: %w", err)
	}

	env := map[string]string{
		"WINEPREFIX": cfg.Paths.PrefixDir,
	}
	verbs := []string{"corefonts", "dotnet48"}
	for _, verb := range verbs {
		logger.Info("installing winetricks %s", verb)
		if err := process.Run(ctx, logger, "winetricks", []string{"-q", verb}, env); err != nil {
			logger.Error("winetricks failed for %s: %v", verb, err)
			return err
		}
		logger.Debug("winetricks %s installed", verb)
	}
	logger.Info("prefix build finished")
	return nil
}
