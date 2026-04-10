# WeMod Launcher (Go)

Linux CLI launcher for WeMod with Steam/Proton integration.
Starts WeMod alongside your game, keeps you logged in across all game prefixes, and handles Proton/Wine setup automatically.

## Why This Was Rewritten in Go

- Self-contained Linux binary – no Python runtime required
- Stricter error handling and safer file/process operations
- Reliable Steam/Proton command parsing, prefix detection, and recovery behavior
- Typed, modular codebase (`launch`, `doctor`, `setup`, `prefix`) for easier long-term maintenance

## How It Works

1. **Setup (once):** Build a dedicated WeMod prefix and download the WeMod binary.
2. **Login (once):** Start WeMod in standalone mode (`wemod`), log into your WeMod account, configure settings.
3. **Play:** Use `wemod %command%` as a Steam launch option. Your login and settings are synced into the game's Proton prefix automatically.

## Requirements

- Linux
- `wine`
- `wineserver`
- `winetricks`
- `go` (optional, only needed for building from source)
- optional: `notify-send`, `zenity` (desktop notifications and progress dialogs)

## Build from Source (Optional)

```bash
go build -o wemod-launcher ./cmd/wemod-launcher
chmod +x ./wemod
```

## Installation

### 1) Download from GitHub Releases

```bash
mkdir -p ~/.local/share/wemod-launcher-app
cd ~/.local/share/wemod-launcher-app

# Download the correct archive for your architecture from:
# https://github.com/NichSchlagen/wemod-proton-launcher-go/releases/latest
# Use `amd64` for x86_64, `arm64` for aarch64/ARM64.
ARCH=amd64

# Verify checksum (optional, recommended)
sha256sum -c "wemod-launcher-linux-${ARCH}.tar.gz.sha256"

# Extract (contains: wemod-launcher and wemod)
tar -xzf "wemod-launcher-linux-${ARCH}.tar.gz"
chmod +x ./wemod-launcher ./wemod
```

### 2) Install dependencies

```bash
# Debian/Ubuntu
sudo apt install -y wine winetricks zenity libnotify-bin

# Arch
sudo pacman -S --needed wine winetricks zenity libnotify
```

### 3) Run setup

Two options for setting up the WeMod prefix:

**Option A: Quick Download (Recommended)**

```bash
./wemod doctor         # check dependencies
./wemod prefix download # download pre-built prefix (~2-3 min)
./wemod                # start WeMod standalone to log in
```

**Option B: Local Build**

```bash
./wemod doctor   # check dependencies
./wemod setup    # build WeMod prefix locally (~20-30 min)
```

### 4) Log in to WeMod (once)

```bash
./wemod          # starts WeMod standalone – log in and configure settings
```

### 5) Set Steam launch option

```text
/absolute/path/to/wemod %command%
```

Your WeMod login and settings will be synced into the game's Proton prefix automatically on first launch.

### 6) (Optional) Register global `wemod` command

Keep `wemod` and `wemod-launcher` in the same directory. Only symlink `wemod`:

```bash
mkdir -p ~/.local/bin
ln -sf ~/.local/share/wemod-launcher-app/wemod ~/.local/bin/wemod
```

Ensure `~/.local/bin` is in your `PATH` (add to `~/.bashrc` or `~/.zshrc`):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then you can use:

```bash
wemod doctor
wemod setup
wemod %command%   # as Steam launch option
```

## CLI Commands

| Command | Description |
|---|---|
| `launch [--] <game command...>` | Launch WeMod with a game (default when called via `%command%`) |
| `setup` | Download WeMod binary and build the Wine prefix |
| `doctor` | Check system dependencies |
| `sync [--] <proton game command...>` | Copy WeMod login/settings from own prefix into a Proton game prefix |
| `reset` | Delete and recreate the own WeMod prefix (`paths.prefix_dir`) |
| `prefix download` | Download a ready-made own WeMod prefix |
| `prefix build` | Build own WeMod prefix locally with winetricks |
| `config init` | (Re)create the default config file |
| `help` | Show command overview |

### Global Flags

| Flag | Description |
|---|---|
| `--config <path>` | Use a custom TOML config file |
| `--log-level <debug\|info\|warn\|error>` | Override log level for this run |
| `--non-interactive` | Disable all prompts |
| `--version` | Print version |

## Steam/Proton Behavior

- `%command%` is supported directly as a Steam launch option
- Proton calls (`.../proton waitforexitandrun ...`) are detected automatically
- When Proton is detected, WeMod runs inside the game's Proton prefix
- `corefonts` and `dotnet48` are installed into the game prefix on first launch (required by WeMod)
- WeMod login data and settings are synced from the own prefix into the game prefix on every launch
- Plain `.exe` calls without a Proton/Wine wrapper are rejected with a clear error

## Configuration

Config is created automatically at `~/.config/wemod-launcher/wemod.toml`.

Important defaults:

| Key | Default |
|---|---|
| `paths.wemod_exe_path` | `~/.local/share/wemod-launcher/wemod_bin/WeMod.exe` |
| `paths.prefix_dir` | `~/.local/share/wemod-launcher/wemod_prefix` |
| `general.log_file` | `~/.local/share/wemod-launcher/wemod-launcher.log` |
| `general.log_level` | `info` |

## Troubleshooting

- Run `./wemod doctor` to check all dependencies.
- Re-run setup: `./wemod setup`
- Log in again: `./wemod` (standalone, no game)
- Force a manual sync into a game prefix:
	`./wemod sync -- /path/to/proton waitforexitandrun ...`
- Reset own prefix if it got corrupted: `./wemod reset`
- Check logs: `~/.local/share/wemod-launcher/wemod-launcher.log`
- For more verbose output: `./wemod --log-level debug %command%`

## Known Issues

- `PROTON_ENABLE_WAYLAND=1` can cause a white WeMod window. Disable it for this launch option:

```text
PROTON_ENABLE_WAYLAND=0 /absolute/path/to/wemod %command%
```

## Notes

- Linux-only
- Not officially affiliated with WeMod

## Credits

- Original WeMod launcher: [DeckCheatz/wemod-launcher](https://github.com/DeckCheatz/wemod-launcher)
- Thanks to [Marvin1099](https://github.com/Marvin1099) for the original groundwork
- This repository is a Go rewrite tailored for Linux Steam/Proton workflows
- WeMod is a third-party product and trademark of its respective owners

