package bootstrap

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/prefix"
)

func RunSetup(ctx context.Context, cfg *config.Config, logger *logging.Logger) error {
	if err := EnsureWeMod(ctx, cfg, logger, false); err != nil {
		return err
	}

	strategy := "download"
	if cfg.General.Interactive {
		choice, err := askSetupStrategy()
		if err != nil {
			return err
		}
		strategy = choice
	}

	switch strategy {
	case "build":
		return prefix.Build(ctx, cfg, logger)
	case "skip":
		logger.Info("prefix setup skipped by user")
		return nil
	default:
		return prefix.Download(ctx, cfg, logger)
	}
}

func askSetupStrategy() (string, error) {
	fmt.Print("Prefix einrichten: [d]ownload, [b]uild oder [s]kip? [d/B/s]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read setup choice: %w", err)
	}
	normalized := strings.ToLower(strings.TrimSpace(line))
	switch normalized {
	case "", "d", "download":
		return "download", nil
	case "b", "build":
		return "build", nil
	case "s", "skip":
		return "skip", nil
	default:
		return "download", nil
	}
}
