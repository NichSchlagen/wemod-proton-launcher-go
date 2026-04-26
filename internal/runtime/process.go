package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
)

func Run(ctx context.Context, logger *logging.Logger, name string, args []string, env map[string]string) error {
	logger = logger.WithComponent("runtime")
	logger.Debug("run command: %s %s", name, strings.Join(args, " "))
	if len(env) > 0 {
		logger.Debug("run command env keys: %s", strings.Join(sortedKeys(env), ","))
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	if err := cmd.Run(); err != nil {
		logger.Error("run failed: %s (%v)", name, err)
		return fmt.Errorf("run %s: %w", name, err)
	}
	logger.Debug("run finished successfully: %s", name)
	return nil
}

func Start(ctx context.Context, logger *logging.Logger, name string, args []string, env map[string]string) (*exec.Cmd, error) {
	logger = logger.WithComponent("runtime")
	logger.Debug("start command: %s %s", name, strings.Join(args, " "))
	if len(env) > 0 {
		logger.Debug("start command env keys: %s", strings.Join(sortedKeys(env), ","))
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	if err := cmd.Start(); err != nil {
		logger.Error("start failed: %s (%v)", name, err)
		return nil, fmt.Errorf("start %s: %w", name, err)
	}
	if cmd.Process != nil {
		logger.Info("process started: %s (pid=%d)", name, cmd.Process.Pid)
	}
	return cmd, nil
}

func StartDetached(ctx context.Context, logger *logging.Logger, name string, args []string, env map[string]string) (*exec.Cmd, error) {
	logger = logger.WithComponent("runtime")
	logger.Debug("start detached command: %s %s", name, strings.Join(args, " "))
	// Detached GUI process must not be tied to launcher context; otherwise it gets
	// terminated when the launcher exits and cancels its context.
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer devNull.Close()

	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull

	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
		logger.Debug("detached command env keys: %s", strings.Join(sortedKeys(env), ","))
	}

	if err := cmd.Start(); err != nil {
		logger.Error("start detached failed: %s (%v)", name, err)
		return nil, fmt.Errorf("start detached %s: %w", name, err)
	}
	if cmd.Process != nil {
		logger.Info("detached process started: %s (pid=%d)", name, cmd.Process.Pid)
	}
	return cmd, nil
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
