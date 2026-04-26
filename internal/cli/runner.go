package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/bootstrap"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/doctor"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/launch"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/prefix"
)

var ErrUsage = errors.New("usage")

type Runner struct {
	logger *logging.Logger
}

func NewRunner(logger *logging.Logger) *Runner {
	return &Runner{logger: logger.WithComponent("cli")}
}

func (r *Runner) Run(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}

	r.logger.Info("command started: %s", args[0])
	r.logger.Debug("full command args: %q", args)

	var err error
	switch args[0] {
	case "launch":
		r.logger.Debug("dispatch to launch.Run")
		err = launch.Run(ctx, cfg, r.logger, args[1:])
	case "setup":
		r.logger.Debug("dispatch to setup workflow")
		if err = doctor.Run(ctx, cfg, r.logger, doctor.Options{FailOnMissing: true}); err != nil {
			break
		}
		err = bootstrap.RunSetup(ctx, cfg, r.logger)
	case "doctor":
		r.logger.Debug("dispatch to doctor.Run")
		err = doctor.Run(ctx, cfg, r.logger, doctor.Options{FailOnMissing: false})
	case "sync":
		r.logger.Debug("dispatch to launch.Sync")
		err = launch.Sync(ctx, cfg, r.logger, args[1:])
	case "reset":
		r.logger.Debug("dispatch to launch.ResetOwnPrefix")
		err = launch.ResetOwnPrefix(cfg, r.logger)
	case "prefix":
		if len(args) < 2 {
			printPrefixUsage()
			return ErrUsage
		}
		switch args[1] {
		case "download":
			r.logger.Debug("dispatch to prefix.Download")
			err = prefix.Download(ctx, cfg, r.logger)
		case "build":
			r.logger.Debug("dispatch to prefix.Build")
			err = prefix.Build(ctx, cfg, r.logger)
		default:
			printPrefixUsage()
			return ErrUsage
		}
	case "config":
		if len(args) < 2 || args[1] != "init" {
			printConfigUsage()
			return ErrUsage
		}
		r.logger.Info("config init requested; config is auto-created/updated on startup")
		err = nil
	case "help":
		printMainUsage()
		err = nil
	default:
		r.logger.Debug("unknown top-level command, fallback to launch.Run")
		err = launch.Run(ctx, cfg, r.logger, args)
	}

	if err != nil {
		r.logger.Error("command failed: %s (%v)", args[0], err)
		return err
	}
	r.logger.Info("command completed: %s", args[0])
	return nil
}

func printMainUsage() {
	fmt.Println("wemod-launcher commands:")
	fmt.Println("  launch [--] <game command...>")
	fmt.Println("  setup")
	fmt.Println("  doctor")
	fmt.Println("  sync [--] <proton game command...>")
	fmt.Println("  reset")
	fmt.Println("  prefix <download|build>")
	fmt.Println("  config init")
	fmt.Println("")
	fmt.Println("global options:")
	fmt.Println("  --config <path>")
	fmt.Println("  --log-level <debug|info|warn|error>")
	fmt.Println("  --non-interactive")
	fmt.Println("  --version")
}

func printPrefixUsage() {
	fmt.Println("usage: wemod-launcher prefix <download|build>")
}

func printConfigUsage() {
	fmt.Println("usage: wemod-launcher config init")
}
