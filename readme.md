# WeMod Launcher (Go)

Linux CLI launcher for WeMod with Steam/Proton integration.
This project starts WeMod alongside your game and handles common Proton/Wine scenarios automatically.

## Why This Was Rewritten in Go

- To distribute a self-contained Linux binary (no Python runtime required for users)
- To get stricter error handling and safer file/process operations for setup and launch flows
- To improve reliability around Steam/Proton command parsing, prefix detection, and recovery behavior
- To keep long-term maintenance simpler with a typed, modular codebase (`launch`, `doctor`, `setup`, `prefix`)

## What This Tool Does

- Starts WeMod with Wine (or Proton Wine, when available)
- Detects Steam `%command%` / Proton launch commands automatically
- Uses the same prefix as the game in Proton sessions
- Creates and validates working directories and base configuration automatically
- Installs missing prefix components (`corefonts`, `dotnet48`) when required
- Supports optional desktop feedback via `notify-send` and `zenity`

## Requirements

- Linux
- `wine`
- `wineserver`
- `winetricks`
- `go` (optional, only needed for building from source)
- optional: `notify-send`, `zenity`

## Build (Optional, from Source)

```bash
go build -o wemod-launcher ./cmd/wemod-launcher
chmod +x ./wemod
```

## Installation & Usage

### 1) Download from GitHub Releases

```bash
# Create a local install directory
mkdir -p ~/.local/share/wemod-launcher-app
cd ~/.local/share/wemod-launcher-app

# Download the correct archive for your architecture from:
# https://github.com/NichSchlagen/wemod-proton-launcher-go/releases/latest

# Use `amd64` for x86_64 systems and `arm64` for aarch64/ARM64 systems.
ARCH=amd64

# Verify checksum (optional, recommended)
sha256sum -c "wemod-launcher-linux-${ARCH}.tar.gz.sha256"

# Extract (contains: wemod-launcher and wemod)
tar -xzf "wemod-launcher-linux-${ARCH}.tar.gz"
```

### 2) Install dependencies

Install at least:

- `wine`
- `wineserver`
- `winetricks`

Optional (desktop notifications / progress dialogs):

- `notify-send`
- `zenity`

Examples:

```bash
# Debian/Ubuntu
sudo apt update
sudo apt install -y wine winetricks zenity libnotify-bin

# Arch
sudo pacman -S --needed wine winetricks zenity libnotify
```

### 3) Make release files executable

```bash
chmod +x ./wemod-launcher ./wemod
```

### 4) Run diagnostics and setup

```bash
./wemod doctor
./wemod setup
```

### 5) (Optional) Register global `wemod` command

Option A (user-local, recommended):

```bash
mkdir -p ~/.local/bin
ln -sf ~/.local/share/wemod-launcher-app/wemod ~/.local/bin/wemod
```

Ensure `~/.local/bin` is in your `PATH`:

```bash
# bash/zsh
export PATH="$HOME/.local/bin:$PATH"

# fish
fish_add_path ~/.local/bin
```

Option B (system-wide):

```bash
sudo ln -sf ~/.local/share/wemod-launcher-app/wemod /usr/local/bin/wemod
```

Then you can run:

```bash
wemod doctor
wemod setup
```

### 6) Use it in Steam

Set the game's launch option to:

```text
/absolute/path/to/wemod %command%
```

If you registered the global command and Steam inherits your shell `PATH`, this may also work:

```text
wemod %command%
```

## Quick Start

```bash
./wemod doctor
./wemod setup
```

Steam Launch Option:

```text
/path/to/wemod-launcher/wemod %command%
```

## CLI Commands

- `launch [--] <game command...>`
- `setup`
- `doctor`
- `prefix download`
- `prefix build`
- `config init`
- `help`

### Global Flags

- `--config <path>`: use a custom TOML config file
- `--non-interactive`: disable prompts
- `--version`: print version

## Setup Strategies

`setup` downloads WeMod first and then offers three (interactive) prefix options:

- `download`: download a ready-made prefix from GitHub releases
- `build`: build the prefix locally with `winetricks`
- `skip`: skip prefix setup

## Steam/Proton Behavior

- `%command%` is supported directly
- Proton calls like `.../proton waitforexitandrun ...` are detected
- When Proton is detected, WeMod uses the game prefix (`WINEPREFIX` / `STEAM_COMPAT_DATA_PATH`)
- If `corefonts` or `dotnet48` are missing in Proton sessions, they are installed automatically
- Plain `.exe` calls without a Proton/Wine wrapper are intentionally rejected

## Configuration

The configuration file is created automatically at:

- `~/.config/wemod-launcher/wemod.toml`

Important defaults:

- `paths.wemod_exe_path`: `~/.local/share/wemod-launcher/wemod_bin/WeMod.exe`
- `paths.prefix_dir`: `~/.local/share/wemod-launcher/wemod_prefix`
- `general.log_file`: `~/.local/share/wemod-launcher/wemod-launcher.log`
- `prefix.download_url`: `auto`

## Troubleshooting

- First diagnostic check: `./wemod doctor`
- Re-run setup: `./wemod setup`
- Manual prefix handling:
	- `./wemod prefix download`
	- `./wemod prefix build`

## Known Issues

- `PROTON_ENABLE_WAYLAND=1` can cause a white WeMod window.
- Recommended workaround: do not set `PROTON_ENABLE_WAYLAND=1` for launch options that use `wemod %command%`.
- Example launch option:

```text
MANGOHUD=1 gamemoderun /absolute/path/to/wemod %command%
```

## Notes

- Linux-only Scope
- Not officially affiliated with WeMod

## Credits

- Original WeMod launcher: [DeckCheatz/wemod-launcher](https://github.com/DeckCheatz/wemod-launcher)
- Big thanks to Marvin1099 [Marvin1099](https://github.com/Marvin1099) for the original groundwork and for making the project publicly available
- This repository is a Go rewrite/continuation tailored for Linux Steam/Proton workflows
- Prefix download source: [DeckCheatz/BuiltPrefixes-dev](https://github.com/DeckCheatz/BuiltPrefixes-dev)
- Thanks to everyone sharing fixes, prefixes, and Linux/Proton findings with the community
- WeMod is a third-party product and trademark of its respective owners
