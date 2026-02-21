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
	return &Runner{logger: logger}
}

func (r *Runner) Run(ctx context.Context, cfg *config.Config, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided")
	}

	r.logger.Info("command started: %s", args[0])

	var err error
	switch args[0] {
	case "launch":
		err = launch.Run(ctx, cfg, r.logger, args[1:])
	case "setup":
		if err = doctor.Run(ctx, cfg, r.logger, doctor.Options{FailOnMissing: true}); err != nil {
			break
		}
		err = bootstrap.RunSetup(ctx, cfg, r.logger)
	case "doctor":
		err = doctor.Run(ctx, cfg, r.logger, doctor.Options{FailOnMissing: false})
	case "prefix":
		if len(args) < 2 {
			printPrefixUsage()
			return ErrUsage
		}
		switch args[1] {
		case "download":
			err = prefix.Download(ctx, cfg, r.logger)
		case "build":
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
		err = nil
	case "help":
		printMainUsage()
		err = nil
	default:
		err = launch.Run(ctx, cfg, r.logger, args)
	}

	if err != nil {
		r.logger.Warn("command failed: %s (%v)", args[0], err)
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
	fmt.Println("  prefix download")
	fmt.Println("  prefix build")
	fmt.Println("  config init")
}

func printPrefixUsage() {
	fmt.Println("usage: wemod-launcher prefix <download|build>")
}

func printConfigUsage() {
	fmt.Println("usage: wemod-launcher config init")
}
