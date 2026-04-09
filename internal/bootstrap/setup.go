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

	strategy := "build"
	if cfg.General.Interactive {
		choice, err := askSetupStrategy()
		if err != nil {
			return err
		}
		strategy = choice
	}

	switch strategy {
	case "skip":
		logger.Info("prefix setup skipped by user")
		return nil
	default:
		return prefix.Build(ctx, cfg, logger)
	}
}

func askSetupStrategy() (string, error) {
	fmt.Print("WeMod-Prefix einrichten: [b]uild oder [s]kip? [B/s]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read setup choice: %w", err)
	}
	normalized := strings.ToLower(strings.TrimSpace(line))
	switch normalized {
	case "s", "skip":
		return "skip", nil
	default:
		return "build", nil
	}
}
