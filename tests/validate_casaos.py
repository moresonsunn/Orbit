#!/usr/bin/env python3
"""tests/validate_casaos.py

Automated validator for CasaOS docker-compose files and x-casaos metadata.
Replicates the official IceWhaleTech/CasaOS-AppStore validation engine.

Usage:
    python tests/validate_casaos.py
    python tests/validate_casaos.py Apps/Orbit/docker-compose.yml
"""

import re
import shutil
import subprocess
import sys
from pathlib import Path

import yaml

NAME_PATTERN = re.compile(r"^[a-z0-9_-]+$")
REVERSE_DOMAIN_PATTERN = re.compile(
    r"^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\.[a-z0-9]+(?:[._-][a-z0-9]+)*)+$"
)
VALID_CATEGORIES = {
    "Media",
    "Productivity",
    "Home",
    "Networking",
    "AI",
    "Finance",
    "Social",
    "Developer",
    "Others",
}


def validate_compose(file_path: Path):
    errors = []
    print(f"[*] Validating CasaOS Compose package: {file_path}")

    if not file_path.is_file():
        return [f"File does not exist: {file_path}"]

    # 1. Check YAML syntax
    try:
        content = file_path.read_text(encoding="utf-8")
        data = yaml.safe_load(content) or {}
    except Exception as e:
        return [f"YAML Parse Error: {e}"]

    # 2. Check top-level name
    name = data.get("name")
    if not name or not NAME_PATTERN.fullmatch(str(name).strip()):
        errors.append(f"Invalid top-level name: '{name}'. Must match ^[a-z0-9_-]+$")

    # 3. Check services
    services = data.get("services", {})
    if not isinstance(services, dict) or not services:
        errors.append("Missing or empty 'services' mapping.")

    # 4. Check x-casaos block
    xcasaos = data.get("x-casaos")
    if not isinstance(xcasaos, dict):
        errors.append("Missing or invalid top-level 'x-casaos' dictionary.")
        return errors

    # 4a. Check ID
    app_id = str(xcasaos.get("id", "")).strip().lower()
    if not REVERSE_DOMAIN_PATTERN.fullmatch(app_id):
        errors.append(
            f"Invalid x-casaos.id: '{app_id}'. Must be reverse-domain format (e.g. dev.orbit.orbit)."
        )

    # 4b. Check main
    main_service = xcasaos.get("main")
    if not main_service or main_service not in services:
        errors.append(f"x-casaos.main '{main_service}' does not exist in services.")

    # 4c. Check port_map
    port_map = xcasaos.get("port_map")
    if not isinstance(port_map, str):
        errors.append(
            f"x-casaos.port_map must be a quoted string, got: {type(port_map).__name__} ({port_map})"
        )

    # 4d. Check category
    category = xcasaos.get("category")
    if category not in VALID_CATEGORIES:
        errors.append(
            f"Invalid category '{category}'. Must be one of: {sorted(VALID_CATEGORIES)}"
        )

    # 4e. Check localized strings
    for field in ["title", "tagline", "description"]:
        val = xcasaos.get(field)
        if not isinstance(val, dict) or "en_US" not in val:
            errors.append(f"x-casaos.{field} must be a localized dict containing 'en_US' key.")

    # 4f. Check architectures
    arches = xcasaos.get("architectures", [])
    if not isinstance(arches, list) or "amd64" not in arches or "arm64" not in arches:
        errors.append("x-casaos.architectures must be a list containing at least 'amd64' and 'arm64'.")

    # 5. Run docker compose config -q
    docker_bin = shutil.which("docker")
    if docker_bin:
        try:
            proc = subprocess.run(
                [docker_bin, "compose", "-f", str(file_path), "config", "-q"],
                capture_output=True,
                text=True,
                timeout=30,
            )
            if proc.returncode != 0:
                errors.append(f"docker compose config failed:\n{proc.stderr.strip()}")
        except Exception as e:
            errors.append(f"Error running docker compose config: {e}")
    else:
        print("[!] docker command not found, skipping docker compose CLI check.")

    return errors


if __name__ == "__main__":
    repo_root = Path(__file__).resolve().parent.parent
    target_compose = repo_root / "Apps" / "Orbit" / "docker-compose.yml"
    if len(sys.argv) > 1:
        target_compose = Path(sys.argv[1])

    errs = validate_compose(target_compose)
    if errs:
        print(f"[FAIL] {len(errs)} validation error(s) found:")
        for err in errs:
            print(f"  - {err}")
        sys.exit(1)
    else:
        print("[SUCCESS] Compose file and CasaOS metadata are completely valid!")
        sys.exit(0)
