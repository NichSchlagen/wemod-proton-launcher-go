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
	if err := os.MkdirAll(cfg.Paths.PrefixDir, 0o755); err != nil {
		return fmt.Errorf("create prefix dir: %w", err)
	}

	env := map[string]string{
		"WINEPREFIX": cfg.Paths.PrefixDir,
	}
	verbs := []string{"corefonts", "dotnet48"}
	for _, verb := range verbs {
		logger.Info("installing winetricks %s", verb)
		if err := process.Run(ctx, logger, "winetricks", []string{"-q", verb}, env); err != nil {
			return err
		}
	}
	logger.Info("prefix build finished")
	return nil
}
