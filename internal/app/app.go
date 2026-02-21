package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/cli"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

func Run(ctx context.Context, args []string) error {
	global := flag.NewFlagSet("wemod-launcher", flag.ContinueOnError)
	configPath := global.String("config", "", "Path to wemod launcher TOML config")
	nonInteractive := global.Bool("non-interactive", false, "Disable prompts and require explicit flags")
	showVersion := global.Bool("version", false, "Print version")
	global.SetOutput(flag.CommandLine.Output())

	if err := global.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println(config.AppVersion)
		return nil
	}

	rest := global.Args()
	if len(rest) == 0 {
		rest = []string{"launch"}
	}

	cfg, cfgPath, err := config.LoadOrCreate(*configPath)
	if err != nil {
		return err
	}

	if *nonInteractive {
		cfg.General.Interactive = false
	}

	logger, err := logging.New(cfg)
	if err != nil {
		return err
	}
	defer logger.Close()

	logger.Info("wemod-launcher %s started", config.AppVersion)
	logger.Info("using config %s", cfgPath)

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
