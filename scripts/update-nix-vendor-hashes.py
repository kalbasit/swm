#!/usr/bin/env python3
"""
Update vendorHash in nix/packages/*/default.nix for all Go packages.

Runs all packages in parallel. For each package:
  1. Try `nix build .#PKG.goModules` — fails immediately if hash is wrong.
  2. If it succeeds (possibly from cache), try `--rebuild` to force
     recomputation against the current source.
  3. If either attempt fails, extract the correct hash from the error output
     and patch the vendorHash line in the corresponding nix file.
"""

import re
import subprocess
import sys
import tempfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent

# Plugins first — they share proto/sdk/go deps so Nix can reuse fetched
# store paths across them. swm last since it has a larger fileset.
PACKAGES = [
    "swm-plugin-forge-github",
    "swm-plugin-picker-fzf",
    "swm-plugin-session-tmux",
    "swm-plugin-vcs-git",
    "swm",
]


def _nix_build(pkg: str, extra_args: list[str]) -> tuple[int, str]:
    """Run `nix build .#PKG.goModules [extra_args]`; return (exit_code, stderr)."""
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=f".{pkg}.log", delete=False
    ) as f:
        logpath = Path(f.name)
    try:
        with logpath.open("w") as logfile:
            result = subprocess.run(
                ["nix", "build", "--print-build-logs", f".#{pkg}.goModules"]
                + extra_args,
                cwd=REPO_ROOT,
                stdout=subprocess.DEVNULL,
                stderr=logfile,
            )
        return result.returncode, logpath.read_text()
    finally:
        logpath.unlink(missing_ok=True)


def _extract_hash(log: str) -> str | None:
    m = re.search(r"got:\s+(\S+)", log)
    return m.group(1) if m else None


def update_package(pkg: str) -> tuple[str, bool, str]:
    """
    Ensure the vendorHash for pkg is current.

    Returns (pkg, was_updated, detail_message).
    Raises RuntimeError on unrecoverable failure.
    """
    nix_file = REPO_ROOT / "nix" / "packages" / pkg / "default.nix"

    ret, log = _nix_build(pkg, [])

    if ret == 0:
        # Build succeeded but may have hit the Nix store cache.  Force a
        # rebuild so the hash is validated against the current source tree.
        ret, log = _nix_build(pkg, ["--rebuild"])

    if ret == 0:
        return pkg, False, "up to date"

    new_hash = _extract_hash(log)
    if not new_hash:
        raise RuntimeError(
            f"nix build failed for {pkg} but no 'got:' hash found in output.\n"
            f"--- stderr ---\n{log}"
        )

    content = nix_file.read_text()
    patched = re.sub(
        r'vendorHash\s*=\s*"[^"]*";',
        f'vendorHash = "{new_hash}";',
        content,
    )
    if patched == content:
        raise RuntimeError(
            f"vendorHash pattern not found (or already matches) in {nix_file} "
            f"despite build failure — manual inspection required."
        )

    nix_file.write_text(patched)
    return pkg, True, new_hash


def main() -> int:
    print(f"Updating vendorHashes for {len(PACKAGES)} packages in parallel…", flush=True)

    failed: list[str] = []
    with ThreadPoolExecutor(max_workers=len(PACKAGES)) as pool:
        futures = {pool.submit(update_package, pkg): pkg for pkg in PACKAGES}
        for fut in as_completed(futures):
            pkg = futures[fut]
            try:
                _, changed, info = fut.result()
                tag = "↻" if changed else "✓"
                print(f"  {tag}  {pkg}: {info}", flush=True)
            except Exception as exc:  # noqa: BLE001
                print(f"  ✗  {pkg}: FAILED — {exc}", file=sys.stderr, flush=True)
                failed.append(pkg)

    if failed:
        print(f"\nFailed packages: {', '.join(sorted(failed))}", file=sys.stderr)
        return 1

    print("\nDone.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
