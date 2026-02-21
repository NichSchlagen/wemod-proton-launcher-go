# Wand Findings (Wine/Proton)

Status: 2026-02-21

## Investigation Goal

This document captures the root-cause-focused investigation into why older WeMod versions start under Linux/Wine/Proton while current Wand versions end up in a black-screen state.

## Executive Summary

- In the tested environment, **WeMod 11.x** starts reliably.
- **Wand 12.x** consistently falls into a startup pattern without a renderer process.
- The stable boundary is therefore at the transition **WeMod 11.x -> Wand 12.x**.

## Test Methodology

The tests were repeated using the same baseline process:

1. Download the target `full.nupkg` and extract `lib/net*` into `wemod_bin`.
2. Launch `WeMod.exe` with a fixed `WINEPREFIX`.
3. Observe process state via `/proc/<pid>/cmdline` and classify roles:
	- `main`
	- `renderer` (`--type=renderer`)
	- `gpu` (`--type=gpu-process`)
	- `utility` (`--type=utility`)
4. Classify outcome:
	- `STARTED`: renderer appears and remains stable.
	- `BLACK_PATTERN`: `main + gpu + utility` present, but **no** `renderer` during the observation window.
	- `TIMEOUT`: no stable renderer within the observation window.

Reusable script in this repository:

- `scripts/wand_probe.py`
- Example:

```bash
python3 scripts/wand_probe.py \
  --versions wand:12.0.3 wand:12.12.1 wemod:11.6.0
```

## Verified Core Results

| Product | Version | Result | Observation |
|---|---:|---|---|
| WeMod | 11.0.0 | STARTED | Stable renderer present |
| WeMod | 11.5.0 | STARTED | Stable renderer present |
| WeMod | 11.6.0 | STARTED | Stable renderer present |
| Wand | 12.0.3 | BLACK_PATTERN | `main+gpu+utility`, no renderer |
| Wand | 12.1.0 | BLACK_PATTERN | `main+gpu+utility`, no renderer |
| Wand | 12.9.0 | BLACK_PATTERN | `main+gpu+utility`, no renderer |
| Wand | 12.12.1 | BLACK_PATTERN | `main+gpu+utility`, no renderer |

This confirms the transition break starts **as early as Wand 12.0.3**.

## Additional Series Checks

Separate runs and release probes also covered additional Wand 12.x builds (including 12.2.x, 12.4.x, 12.10.x, 12.11.x) and showed the same no-renderer failure pattern.

## Mitigations That Were Tried

Beyond plain direct launch, multiple launcher-side recovery strategies were tested:

- Safe flag launch (`--disable-gpu`, `--use-angle=swiftshader`, etc.)
- Aggressive flags (adding `--in-process-gpu`, `--no-sandbox`)
- Rescue flags + software GL environment overrides (`llvmpipe`)
- Proton wine binary vs. system `wine` inside Proton prefix
- Chromium/Electron verbose environment logging enabled

Outcome: In the tested environment, none of these approaches produced a consistently stable renderer for Wand 12.x.

## Technical Interpretation

- The failure pattern matches a **renderer startup compatibility issue** in the Wand 12.x stack under Wine/Proton.
- Based on current evidence, this is **not** isolated to a single late 12.x patch.
- Stable operation is currently tied to the 11.x line.

## Current Operational Workaround

- Keep production use on **WeMod 11.6.0**.
- If Wand is retested, record results in a reproducible matrix with:
  - exact Wand version
  - Wine/Proton version
  - prefix type
  - flags/environment used

## Update Rule for This File

For each new retest, append:

1. Date
2. Version(s)
3. Result (`STARTED`/`BLACK_PATTERN`/`TIMEOUT`)
4. Relevant environment details
