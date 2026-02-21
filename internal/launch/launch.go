package launch

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/logging"
	process "github.com/NichSchlagen/wemod-proton-launcher-go/internal/runtime"
)

const runtimeReadyMarker = ".wemod_launcher_runtime_ready"

type wemodRuntime struct {
	cmd *exec.Cmd
}

func Run(ctx context.Context, cfg *config.Config, logger *logging.Logger, args []string) error {
	gameCmd, parseInfo, err := parseGameCommandArgs(args)
	if err != nil {
		return err
	}
	if parseInfo != "" {
		logger.Info(parseInfo)
	}

	if _, err := os.Stat(cfg.Paths.WeModExePath); err != nil {
		if cfg.General.Interactive {
			ok, askErr := askYesNo("WeMod.exe not found. Run setup now? [Y/n]: ")
			if askErr != nil {
				return askErr
			}
			if ok {
				return fmt.Errorf("WeMod executable missing at %s; run: wemod-launcher setup", cfg.Paths.WeModExePath)
			}
		}
		return fmt.Errorf("WeMod executable missing at %s", cfg.Paths.WeModExePath)
	}

	wemodPrefix, protonMode := resolveWeModPrefix(cfg, gameCmd)
	env := map[string]string{"WINEPREFIX": wemodPrefix}
	logger.Info("using WeMod prefix: %s", wemodPrefix)

	if protonMode {
		userNotice("Checking game prefix dependencies (corefonts/dotnet48) ...")
		if err := ensurePrefixRuntime(ctx, logger, wemodPrefix, gameCmd, protonMode); err != nil {
			logger.Warn("game prefix runtime prep failed, continuing anyway: %v", err)
			userNotice("Prefix preparation failed, starting WeMod anyway ...")
		} else {
			userNotice("Game prefix dependencies are ready.")
		}
	}

	if len(gameCmd) == 0 {
		wemodProc, err := startWeModProcess(ctx, cfg, logger, gameCmd, env, protonMode)
		if err != nil {
			return fmt.Errorf("start wemod: %w", err)
		}
		logger.Info("WeMod started (pid=%d)", wemodProc.cmd.Process.Pid)
		logger.Info("no game command provided, leaving WeMod running")
		return nil
	}

	if err := validateCommand(gameCmd[0]); err != nil {
		return err
	}

	logger.Info("starting game command: %s", strings.Join(gameCmd, " "))
	gameProc, err := process.Start(ctx, logger, gameCmd[0], gameCmd[1:], nil)
	if err != nil {
		return fmt.Errorf("start game: %w", err)
	}

	var wemodProc *wemodRuntime
	if protonMode {
		logger.Info("proton mode: delaying WeMod start to avoid blocking game launch")
		time.Sleep(2 * time.Second)
		wemodProc, err = startWeModProcessWithRecovery(ctx, cfg, logger, gameCmd, env, protonMode)
		if err != nil {
			logger.Warn("failed to start WeMod after game launch: %v", err)
		} else {
			logger.Info("WeMod started (pid=%d)", wemodProc.cmd.Process.Pid)
		}
	} else {
		wemodProc, err = startWeModProcessWithRecovery(ctx, cfg, logger, gameCmd, env, protonMode)
		if err != nil {
			_ = gameProc.Process.Kill()
			return fmt.Errorf("start wemod: %w", err)
		}
		logger.Info("WeMod started (pid=%d)", wemodProc.cmd.Process.Pid)
	}

	if err := gameProc.Wait(); err != nil {
		logger.Warn("game process exited with error: %v", err)
	} else {
		logger.Info("game process finished")
	}

	return nil
}

func parseGameCommandArgs(args []string) ([]string, string, error) {
	if len(args) == 0 {
		return nil, "", nil
	}

	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "launch" && len(normalized) == 0 {
			continue
		}
		normalized = append(normalized, arg)
	}
	if len(normalized) > 0 && normalized[0] == "--" {
		normalized = normalized[1:]
	}
	if len(normalized) == 0 {
		return nil, "", nil
	}

	verbIndex := protonVerbIndex(normalized)
	if verbIndex > 0 && strings.Contains(strings.ToLower(filepath.Base(normalized[verbIndex-1])), "proton") {
		return normalized[verbIndex-1:], "detected Proton launch command", nil
	}

	if looksLikeWindowsExecutable(normalized[0]) {
		return nil, "", errors.New("invalid game command: got a Windows executable without Proton/Wine wrapper; use Steam %command% unchanged or pass a Linux/Proton command")
	}

	return normalized, "using direct game command", nil
}

func validateCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return errors.New("game command is empty")
	}

	if strings.ContainsRune(command, os.PathSeparator) {
		st, err := os.Stat(command)
		if err != nil {
			return fmt.Errorf("game command path not found: %s", command)
		}
		if st.Mode()&0o111 == 0 {
			return fmt.Errorf("game command is not executable: %s", command)
		}
		return nil
	}

	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf("game command not found in PATH: %s", command)
	}
	return nil
}

func startWeModProcess(ctx context.Context, cfg *config.Config, logger *logging.Logger, gameCmd []string, env map[string]string, protonMode bool) (*wemodRuntime, error) {
	if protonMode {
		if protonWine := resolveProtonWineBinary(gameCmd[0]); protonWine != "" {
			logger.Info("starting WeMod with Proton wine binary")
			cmd, err := process.Start(ctx, logger, protonWine, []string{cfg.Paths.WeModExePath}, env)
			if err != nil {
				return nil, err
			}
			return &wemodRuntime{cmd: cmd}, nil
		}
		logger.Info("starting WeMod with system wine in Proton prefix")
		cmd, err := process.Start(ctx, logger, "wine", []string{cfg.Paths.WeModExePath}, env)
		if err != nil {
			return nil, err
		}
		return &wemodRuntime{cmd: cmd}, nil
	}

	logger.Info("starting WeMod directly with wine")
	cmd, err := process.Start(ctx, logger, "wine", []string{cfg.Paths.WeModExePath}, env)
	if err != nil {
		return nil, err
	}
	return &wemodRuntime{cmd: cmd}, nil
}

func startWeModProcessWithRecovery(ctx context.Context, cfg *config.Config, logger *logging.Logger, gameCmd []string, env map[string]string, protonMode bool) (*wemodRuntime, error) {
	wemodProc, err := startWeModProcess(ctx, cfg, logger, gameCmd, env, protonMode)
	if err == nil {
		return wemodProc, nil
	}

	logger.Warn("initial WeMod start failed: %v", err)
	userNotice("WeMod start failed, initializing Wine and retrying ...")

	winebootBinary := resolveWineBootBinary(gameCmd, protonMode)
	if bootErr := process.Run(ctx, logger, winebootBinary, []string{"-u"}, env); bootErr != nil {
		logger.Warn("wineboot recovery failed (%s): %v", winebootBinary, bootErr)
	}

	time.Sleep(2 * time.Second)

	wemodProc, retryErr := startWeModProcess(ctx, cfg, logger, gameCmd, env, protonMode)
	if retryErr == nil {
		userNotice("WeMod started successfully on retry.")
		return wemodProc, nil
	}

	return nil, fmt.Errorf("initial error: %w; retry error: %w", err, retryErr)
}

func resolveProtonWineBinary(protonPath string) string {
	protonDir := filepath.Dir(protonPath)
	candidates := []string{
		filepath.Join(protonDir, "files", "bin", "wine"),
		filepath.Join(protonDir, "files", "bin", "wine64"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && st.Mode()&0o111 != 0 {
			return candidate
		}
	}
	return ""
}

func resolveProtonWineServerBinary(protonWineBinary string) string {
	candidate := filepath.Join(filepath.Dir(protonWineBinary), "wineserver")
	if st, err := os.Stat(candidate); err == nil && st.Mode()&0o111 != 0 {
		return candidate
	}
	return "wineserver"
}

func resolveWineBootBinary(gameCmd []string, protonMode bool) string {
	if protonMode && len(gameCmd) > 0 {
		if protonWine := resolveProtonWineBinary(gameCmd[0]); protonWine != "" {
			candidate := filepath.Join(filepath.Dir(protonWine), "wineboot")
			if st, err := os.Stat(candidate); err == nil && st.Mode()&0o111 != 0 {
				return candidate
			}
		}
	}
	return "wineboot"
}

func resolveWeModPrefix(cfg *config.Config, gameCmd []string) (string, bool) {
	if isProtonCommand(gameCmd) {
		if w := strings.TrimSpace(os.Getenv("WINEPREFIX")); w != "" {
			return w, true
		}
		if compat := strings.TrimSpace(os.Getenv("STEAM_COMPAT_DATA_PATH")); compat != "" {
			return filepath.Join(compat, "pfx"), true
		}
	}
	return cfg.Paths.PrefixDir, false
}

func isProtonCommand(gameCmd []string) bool {
	verbIndex := protonVerbIndex(gameCmd)
	if verbIndex <= 0 {
		return false
	}
	base := strings.ToLower(filepath.Base(gameCmd[verbIndex-1]))
	return strings.Contains(base, "proton")
}

func ensurePrefixRuntime(ctx context.Context, logger *logging.Logger, prefixPath string, gameCmd []string, protonMode bool) error {
	markerPath := filepath.Join(prefixPath, runtimeReadyMarker)
	if _, err := os.Stat(markerPath); err == nil {
		logger.Info("runtime marker found in game prefix, skipping winetricks checks")
		userNotice("Game prefix already prepared, skipping installation.")
		return nil
	}

	env := buildPrefixRuntimeEnv(prefixPath, gameCmd, protonMode)
	userNotice("Checking installed components in game prefix ...")
	var installed map[string]bool
	err := withProgressDialog(
		ctx,
		"WeMod Launcher",
		"Checking installed components in game prefix ...",
		func() error {
			var listErr error
			installed, listErr = listInstalledVerbs(ctx, env)
			return listErr
		},
	)
	if err != nil {
		logger.Warn("could not inspect winetricks verbs in game prefix, skipping auto-install for this run: %v", err)
		return nil
	}
	userNotice("Dependency check completed.")

	required := []string{"corefonts", "dotnet48"}
	for _, verb := range required {
		if installed[verb] {
			userNotice("Already installed: %s", verb)
			continue
		}
		logger.Info("installing %s into game prefix for WeMod/game integration", verb)
		userNotice("Installing into game prefix: %s", verb)
		if err := withProgressDialog(
			ctx,
			"WeMod Launcher",
			fmt.Sprintf("Installing %s into game prefix ...", verb),
			func() error {
				return process.Run(ctx, logger, "winetricks", []string{"-q", verb}, env)
			},
		); err != nil {
			return fmt.Errorf("install %s: %w", verb, err)
		}
		userNotice("Installed: %s", verb)
	}

	if err := os.WriteFile(markerPath, []byte("ok\n"), 0o644); err != nil {
		logger.Warn("failed to write runtime marker: %v", err)
	}

	return nil
}

func userNotice(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(os.Stderr, "[wemod-launcher] %s\n", message)
	if _, err := exec.LookPath("notify-send"); err != nil {
		return
	}
	cmd := exec.Command("notify-send", "-a", "wemod-launcher", "WeMod Launcher", message)
	_ = cmd.Run()
}

func withProgressDialog(ctx context.Context, title string, text string, run func() error) error {
	if _, err := exec.LookPath("zenity"); err != nil {
		return run()
	}

	cmd := exec.CommandContext(ctx, "zenity", "--progress", "--title", title, "--text", text, "--pulsate", "--no-cancel", "--auto-close", "--percentage=0")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return run()
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return run()
	}

	_, _ = fmt.Fprintln(stdin, "0")
	_, _ = fmt.Fprintf(stdin, "# %s\n", text)

	runErr := run()

	_, _ = fmt.Fprintln(stdin, "100")
	_ = stdin.Close()

	waitDone := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
	}

	return runErr
}

func buildPrefixRuntimeEnv(prefixPath string, gameCmd []string, protonMode bool) map[string]string {
	env := map[string]string{"WINEPREFIX": prefixPath}
	if protonMode && len(gameCmd) > 0 {
		if protonWine := resolveProtonWineBinary(gameCmd[0]); protonWine != "" {
			env["WINE"] = protonWine
			env["WINESERVER"] = resolveProtonWineServerBinary(protonWine)
		}
	}
	return env
}

func listInstalledVerbs(ctx context.Context, env map[string]string) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "winetricks", "list-installed")
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	output, err := cmd.CombinedOutput()
	installed := map[string]bool{}
	for _, line := range strings.Split(string(output), "\n") {
		verb := strings.ToLower(strings.TrimSpace(line))
		if verb == "" || strings.Contains(verb, "warning") {
			continue
		}
		installed[verb] = true
	}
	if err != nil {
		return installed, fmt.Errorf("winetricks list-installed: %w", err)
	}
	return installed, nil
}

func looksLikeWindowsExecutable(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasSuffix(lower, ".exe") {
		return true
	}
	if strings.Contains(lower, `:\\`) || strings.Contains(lower, `:/`) {
		return true
	}
	return false
}

func indexOf(items []string, needle string) int {
	for i, item := range items {
		if item == needle {
			return i
		}
	}
	return -1
}

func protonVerbIndex(args []string) int {
	waitIdx := indexOf(args, "waitforexitandrun")
	if waitIdx >= 0 {
		return waitIdx
	}
	return indexOf(args, "run")
}

func askYesNo(prompt string) (bool, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read prompt input: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" || line == "y" || line == "yes" {
		return true, nil
	}
	return false, nil
}
