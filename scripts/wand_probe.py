#!/usr/bin/env python3

import argparse
import os
import shutil
import subprocess
import tempfile
import time
import zipfile
from pathlib import Path


def parse_args() -> argparse.Namespace:
    home = Path.home()
    default_install = home / ".local" / "share" / "wemod-launcher" / "wemod_bin"
    default_prefix = home / ".local" / "share" / "wemod-launcher" / "wemod_prefix"
    default_cache = Path("/tmp") / "wand-probe-cache"

    parser = argparse.ArgumentParser(
        description="Probe WeMod/Wand startup behavior under Wine and classify renderer startup status."
    )
    parser.add_argument(
        "--versions",
        nargs="+",
        metavar="KIND:VERSION",
        default=[
            "wand:12.0.3",
            "wand:12.1.0",
            "wand:12.9.0",
            "wand:12.12.1",
            "wemod:11.6.0",
        ],
        help="Version entries in the form kind:version where kind is wand or wemod.",
    )
    parser.add_argument("--install-dir", type=Path, default=default_install)
    parser.add_argument("--prefix-dir", type=Path, default=default_prefix)
    parser.add_argument("--cache-dir", type=Path, default=default_cache)
    parser.add_argument("--observe-timeout", type=float, default=20.0)
    parser.add_argument("--renderer-stable-sec", type=float, default=2.0)
    parser.add_argument("--black-pattern-sec", type=float, default=2.0)
    return parser.parse_args()


def parse_version_entry(entry: str) -> tuple[str, str]:
    if ":" not in entry:
        raise ValueError(f"invalid version entry '{entry}', expected kind:version")
    kind, version = entry.split(":", 1)
    kind = kind.strip().lower()
    version = version.strip()
    if kind not in {"wand", "wemod"}:
        raise ValueError(f"invalid kind '{kind}' in '{entry}', expected wand or wemod")
    if not version:
        raise ValueError(f"missing version in '{entry}'")
    return kind, version


def url_for(kind: str, version: str) -> str:
    base = "https://storage-cdn.wemod.com/app/releases/stable/"
    if kind == "wand":
        return f"{base}Wand-{version}-full.nupkg"
    return f"{base}WeMod-{version}-full.nupkg"


def cached_nupkg_path(cache_dir: Path, kind: str, version: str) -> Path:
    name = f"{'Wand' if kind == 'wand' else 'WeMod'}-{version}-full.nupkg"
    return cache_dir / name


def download_if_needed(cache_dir: Path, kind: str, version: str) -> Path:
    cache_dir.mkdir(parents=True, exist_ok=True)
    target = cached_nupkg_path(cache_dir, kind, version)
    if target.exists() and target.stat().st_size > 0:
        return target

    subprocess.run(
        [
            "curl",
            "-fsSL",
            "-A",
            "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36",
            url_for(kind, version),
            "-o",
            str(target),
        ],
        check=True,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    return target


def install_nupkg(nupkg: Path, install_dir: Path) -> None:
    with tempfile.TemporaryDirectory(prefix="wand-probe-") as td:
        tmp = Path(td)
        with zipfile.ZipFile(nupkg) as zf:
            zf.extractall(tmp)

        net_roots = sorted((tmp / "lib").glob("net*"))
        if not net_roots:
            raise RuntimeError("no lib/net* payload in nupkg")
        net_root = net_roots[0]

        if install_dir.exists():
            shutil.rmtree(install_dir)
        install_dir.mkdir(parents=True, exist_ok=True)

        for item in net_root.iterdir():
            dst = install_dir / item.name
            if item.is_dir():
                shutil.copytree(item, dst)
            else:
                shutil.copy2(item, dst)


def kill_wemod_processes() -> None:
    for entry in Path("/proc").iterdir():
        if not entry.is_dir() or not entry.name.isdigit():
            continue
        cmdline = entry / "cmdline"
        try:
            raw = cmdline.read_bytes()
        except Exception:
            continue
        if not raw:
            continue
        text = raw.replace(b"\x00", b" ").decode("utf-8", "ignore").lower()
        if "wand.exe" in text or "wemod.exe" in text:
            try:
                os.kill(int(entry.name), 9)
            except Exception:
                pass


def inspect_state() -> dict[str, int]:
    state = {"main": 0, "renderer": 0, "gpu": 0, "utility": 0}
    for entry in Path("/proc").iterdir():
        if not entry.is_dir() or not entry.name.isdigit():
            continue
        cmdline = entry / "cmdline"
        try:
            raw = cmdline.read_bytes()
        except Exception:
            continue
        if not raw:
            continue

        text = raw.replace(b"\x00", b" ").decode("utf-8", "ignore").lower()
        if "wand.exe" not in text and "wemod.exe" not in text:
            continue

        if "--type=renderer" in text:
            state["renderer"] += 1
        elif "--type=gpu-process" in text:
            state["gpu"] += 1
        elif "--type=utility" in text:
            state["utility"] += 1
        else:
            state["main"] += 1

    return state


def quick_probe(
    install_dir: Path,
    prefix_dir: Path,
    observe_timeout_sec: float,
    renderer_stable_sec: float,
    black_pattern_sec: float,
) -> tuple[str, str]:
    exe = install_dir / "WeMod.exe"
    if not exe.exists():
        return "ERROR", "WeMod.exe missing after install"

    env = os.environ.copy()
    env["WINEPREFIX"] = str(prefix_dir)
    env["WINEDEBUG"] = "-all"

    proc = subprocess.Popen(
        ["wine", str(exe)],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        env=env,
    )

    start = time.time()
    renderer_since = None
    black_since = None

    try:
        while time.time() - start < observe_timeout_sec:
            state = inspect_state()

            if state["renderer"] > 0:
                if renderer_since is None:
                    renderer_since = time.time()
            else:
                renderer_since = None

            if (
                state["main"] > 0
                and state["gpu"] > 0
                and state["utility"] > 0
                and state["renderer"] == 0
            ):
                if black_since is None:
                    black_since = time.time()
            else:
                black_since = None

            if renderer_since and (time.time() - renderer_since) >= renderer_stable_sec:
                return "STARTED", f"renderer stable (state={state})"

            if black_since and (time.time() - black_since) >= black_pattern_sec:
                return "BLACK_PATTERN", f"main+gpu+utility without renderer (state={state})"

            time.sleep(0.25)

        state = inspect_state()
        return "TIMEOUT", f"no stable renderer within {observe_timeout_sec:.0f}s (state={state})"
    finally:
        try:
            proc.kill()
        except Exception:
            pass
        kill_wemod_processes()


def main() -> None:
    args = parse_args()

    parsed_versions = []
    for entry in args.versions:
        parsed_versions.append(parse_version_entry(entry))

    print("kind,version,status,summary")

    for kind, version in parsed_versions:
        kill_wemod_processes()
        try:
            nupkg = download_if_needed(args.cache_dir, kind, version)
            install_nupkg(nupkg, args.install_dir)
            status, summary = quick_probe(
                args.install_dir,
                args.prefix_dir,
                args.observe_timeout,
                args.renderer_stable_sec,
                args.black_pattern_sec,
            )
            print(f"{kind},{version},{status},{summary.replace(',', ';')}")
        except Exception as exc:
            print(f"{kind},{version},ERROR,{str(exc).replace(',', ';')}")

    kill_wemod_processes()


if __name__ == "__main__":
    main()
