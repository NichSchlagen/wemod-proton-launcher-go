package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/cli"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

func Run(ctx context.Context, args []string) error {
	global := flag.NewFlagSet("wemod-launcher", flag.ContinueOnError)
	configPath := global.String("config", "", "Path to wemod launcher TOML config")
	nonInteractive := global.Bool("non-interactive", false, "Disable prompts and require explicit flags")
	logLevel := global.String("log-level", "", "Override log level (debug|info|warn|error)")
	showVersion := global.Bool("version", false, "Print version")
	global.SetOutput(os.Stderr)

	globalArgs, commandArgs, err := splitGlobalAndCommandArgs(args)
	if err != nil {
		return err
	}

	if err := global.Parse(globalArgs); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println(config.AppVersion)
		return nil
	}

	rest := commandArgs
	if len(rest) == 0 {
		rest = []string{"launch"}
	}

	cfg, cfgPath, err := config.LoadOrCreate(*configPath)
	if err != nil {
		return err
	}

	configLevel, err := logging.NormalizeLevel(cfg.General.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid config general.log_level in %s: %w", cfgPath, err)
	}
	cfg.General.LogLevel = configLevel

	if *nonInteractive {
		cfg.General.Interactive = false
	}

	levelSource := "config"
	if strings.TrimSpace(*logLevel) != "" {
		normalizedLevel, err := logging.NormalizeLevel(*logLevel)
		if err != nil {
			return err
		}
		cfg.General.LogLevel = normalizedLevel
		levelSource = "cli"
	}

	logger, err := logging.New(cfg)
	if err != nil {
		return err
	}
	defer logger.Close()
	logger = logger.WithComponent("app")

	logger.StartupBanner(fmt.Sprintf("wemod-launcher %s session start", config.AppVersion))
	logger.Info("wemod-launcher %s started", config.AppVersion)
	logger.Info("using config %s", cfgPath)
	logger.Info("effective log level: %s (source=%s)", logger.LevelName(), levelSource)
	logger.Debug("raw args: %q", args)
	logger.Debug("global args: %q", globalArgs)
	logger.Debug("command args: %q", rest)
	if *nonInteractive {
		logger.Info("interactive prompts disabled by CLI")
	}

	runner := cli.NewRunner(logger)
	if err := runner.Run(ctx, cfg, rest); err != nil {
		if errors.Is(err, cli.ErrUsage) {
			return nil
		}
		if strings.TrimSpace(err.Error()) == "" {
			return errors.New("command failed")
		}
		return err
	}

	return nil
}

func splitGlobalAndCommandArgs(args []string) ([]string, []string, error) {
	globalArgs := make([]string, 0)
	commandArgs := make([]string, 0)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			commandArgs = append(commandArgs, args[i+1:]...)
			break
		}

		switch {
		case arg == "--non-interactive" || arg == "--version":
			globalArgs = append(globalArgs, arg)
		case arg == "--config" || arg == "--log-level":
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("missing value for %s", arg)
			}
			globalArgs = append(globalArgs, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "--log-level="):
			globalArgs = append(globalArgs, arg)
		default:
			commandArgs = append(commandArgs, arg)
		}
	}

	return globalArgs, commandArgs, nil
}
