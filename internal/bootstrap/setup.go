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
	logger = logger.WithComponent("bootstrap.setup")
	logger.Info("setup workflow started")
	if err := EnsureWeMod(ctx, cfg, logger, false); err != nil {
		logger.Error("setup failed while ensuring WeMod binary: %v", err)
		return err
	}

	strategy := "build"
	if cfg.General.Interactive {
		choice, err := askSetupStrategy()
		if err != nil {
			logger.Error("failed reading setup strategy: %v", err)
			return err
		}
		strategy = choice
	}
	logger.Info("selected setup strategy: %s", strategy)

	switch strategy {
	case "skip":
		logger.Info("prefix setup skipped by user")
		return nil
	case "download":
		logger.Debug("executing prefix.Download")
		return prefix.Download(ctx, cfg, logger)
	default:
		logger.Debug("executing prefix.Build")
		return prefix.Build(ctx, cfg, logger)
	}
}

func askSetupStrategy() (string, error) {
	fmt.Println("How do you want to set up the WeMod prefix?")
	fmt.Println("  [d] download  – prebuilt prefix, ready in ~1-2 min (recommended)")
	fmt.Println("  [b] build     – install via winetricks locally, takes ~10-20 min")
	fmt.Println("  [s] skip      – do nothing")
	fmt.Print("Choice [D/b/s]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read setup choice: %w", err)
	}
	normalized := strings.ToLower(strings.TrimSpace(line))
	switch normalized {
	case "s", "skip":
		return "skip", nil
	case "b", "build":
		return "build", nil
	default:
		return "download", nil
	}
}
