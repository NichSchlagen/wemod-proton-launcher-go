package launch

import (
	"testing"

	"github.com/NichSchlagen/wemod-proton-launcher-go/internal/config"
)

func TestParseGameCommandArgs_ProtonLaunch(t *testing.T) {
	args := []string{"launch", "--", "/tmp/Proton/proton", "waitforexitandrun", "/tmp/game.sh"}
	gameCmd, info, err := parseGameCommandArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gameCmd) != 3 {
		t.Fatalf("unexpected game command length: %d", len(gameCmd))
	}
	if gameCmd[0] != "/tmp/Proton/proton" {
		t.Fatalf("unexpected first arg: %s", gameCmd[0])
	}
	if info != "detected Proton launch command" {
		t.Fatalf("unexpected parse info: %s", info)
	}
}

func TestParseGameCommandArgs_RejectWindowsExe(t *testing.T) {
	_, _, err := parseGameCommandArgs([]string{`C:\\Games\\foo.exe`})
	if err == nil {
		t.Fatal("expected error for windows executable without wrapper")
	}
}

func TestResolveWeModPrefix_ProtonFromWineprefix(t *testing.T) {
	t.Setenv("WINEPREFIX", "/tmp/custom-pfx")
	t.Setenv("STEAM_COMPAT_DATA_PATH", "")

	cfg := &config.Config{}
	cfg.Paths.PrefixDir = "/tmp/own-prefix"

	prefix, protonMode := resolveWeModPrefix(cfg, []string{"/tmp/proton", "run", "/tmp/game.sh"})
	if !protonMode {
		t.Fatal("expected proton mode")
	}
	if prefix != "/tmp/custom-pfx" {
		t.Fatalf("unexpected prefix: %s", prefix)
	}
}

func TestResolveWeModPrefix_ProtonFromCompatData(t *testing.T) {
	t.Setenv("WINEPREFIX", "")
	t.Setenv("STEAM_COMPAT_DATA_PATH", "/tmp/compat/12345")

	cfg := &config.Config{}
	cfg.Paths.PrefixDir = "/tmp/own-prefix"

	prefix, protonMode := resolveWeModPrefix(cfg, []string{"/tmp/proton", "run", "/tmp/game.sh"})
	if !protonMode {
		t.Fatal("expected proton mode")
	}
	if prefix != "/tmp/compat/12345/pfx" {
		t.Fatalf("unexpected prefix: %s", prefix)
	}
}

func TestResolveWeModPrefix_NonProtonFallsBackToOwnPrefix(t *testing.T) {
	t.Setenv("WINEPREFIX", "/tmp/custom-pfx")
	t.Setenv("STEAM_COMPAT_DATA_PATH", "/tmp/compat/12345")

	cfg := &config.Config{}
	cfg.Paths.PrefixDir = "/tmp/own-prefix"

	prefix, protonMode := resolveWeModPrefix(cfg, []string{"bash", "-lc", "echo hi"})
	if protonMode {
		t.Fatal("did not expect proton mode")
	}
	if prefix != "/tmp/own-prefix" {
		t.Fatalf("unexpected prefix: %s", prefix)
	}
}
