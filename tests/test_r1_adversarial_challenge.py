#!/usr/bin/env python3
"""tests/test_r1_adversarial_challenge.py

Empirical Adversarial Challenge Suite for Requirement R1:
Docker Container User Settings, Multi-Arch Directives, and Compose Robustness.
"""

import os
import re
import shutil
import subprocess
import sys
import unittest
from pathlib import Path
import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent
DOCKERFILE_PATH = REPO_ROOT / "Dockerfile"
CASAOS_COMPOSE_PATH = REPO_ROOT / "Apps" / "Orbit" / "docker-compose.yml"
PROD_COMPOSE_PATH = REPO_ROOT / "docker-compose.prod.yml"
WORKFLOW_PATH = REPO_ROOT / ".github" / "workflows" / "docker-publish.yml"
BACKEND_DIR = REPO_ROOT / "backend"


class TestContainerUserSettingsAdversarial(unittest.TestCase):
    """Adversarial stress-testing of Dockerfile user settings & runtime identity."""

    @classmethod
    def setUpClass(cls):
        cls.content = DOCKERFILE_PATH.read_text(encoding="utf-8")
        cls.lines = cls.content.splitlines()

    def test_no_active_user_directive_in_runtime_stage(self):
        """Stress-test: Scan for any active USER directive in the production runtime stage."""
        stage_3_lines = []
        in_stage_3 = False
        stage_count = 0
        for line in self.lines:
            stripped = line.strip()
            if stripped.upper().startswith("FROM "):
                stage_count += 1
                if stage_count == 3:
                    in_stage_3 = True
            if in_stage_3:
                stage_3_lines.append(stripped)

        self.assertTrue(in_stage_3, "Dockerfile must define stage 3 production runtime.")
        
        # Check every line in stage 3 for USER directives
        user_directives = [
            l for l in stage_3_lines 
            if re.match(r"^\s*USER\s+", l, re.IGNORECASE)
        ]
        self.assertEqual(
            len(user_directives),
            0,
            f"Adversarial failure: Found active USER directive in stage 3: {user_directives}",
        )

    def test_no_uncommented_user_orbit_anywhere(self):
        """Verify 'USER orbit' does not exist in any stage of the Dockerfile."""
        for idx, line in enumerate(self.lines, 1):
            stripped = line.strip()
            if stripped.startswith("#"):
                continue
            self.assertFalse(
                re.match(r"^USER\s+orbit\b", stripped, re.IGNORECASE),
                f"Line {idx} contains forbidden active 'USER orbit': '{stripped}'",
            )

    def test_default_runtime_user_is_root(self):
        """Verify runtime stage base image is debian:bookworm-slim which defaults to UID 0 (root)."""
        stage_froms = [
            l.strip() for l in self.lines if l.strip().upper().startswith("FROM ")
        ]
        self.assertGreaterEqual(len(stage_froms), 3)
        runtime_base = stage_froms[-1]
        self.assertIn("debian:bookworm-slim", runtime_base.lower())

    def test_data_dir_writable_by_root(self):
        """Verify data directory creation allows root execution and non-root fallback."""
        mkdir_lines = [
            l.strip() for l in self.lines if "mkdir" in l and "/data" in l
        ]
        self.assertTrue(len(mkdir_lines) > 0, "Expected /data directory creation in Dockerfile.")
        self.assertTrue(
            any("orbit:orbit" in l for l in mkdir_lines),
            "Expected /data to have orbit:orbit ownership for flexibility while root retains CAP_DAC_OVERRIDE.",
        )


class TestComposeRobustnessAdversarial(unittest.TestCase):
    """Stress-test compose configurations against YAML parsing, edge cases, and CasaOS specs."""

    def test_casaos_compose_yaml_parse_and_user_root(self):
        """Verify Apps/Orbit/docker-compose.yml parses cleanly and enforces user: '0:0'."""
        content = CASAOS_COMPOSE_PATH.read_text(encoding="utf-8")
        data = yaml.safe_load(content)
        
        # Assert service orbit configuration
        orbit_svc = data.get("services", {}).get("orbit", {})
        user_val = orbit_svc.get("user")
        
        # Ensure user is strictly "0:0"
        self.assertEqual(
            str(user_val),
            "0:0",
            f"CasaOS compose user MUST be '0:0', got {user_val} (type: {type(user_val)})",
        )
        self.assertIsInstance(
            user_val,
            str,
            f"CasaOS compose user must be quoted string '0:0' to prevent YAML sexagesimal parse bug, got {type(user_val)}",
        )

    def test_prod_compose_yaml_parse_and_user_root(self):
        """Verify docker-compose.prod.yml parses cleanly and enforces user: '0:0'."""
        content = PROD_COMPOSE_PATH.read_text(encoding="utf-8")
        data = yaml.safe_load(content)
        
        orbit_svc = data.get("services", {}).get("orbit", {})
        user_val = orbit_svc.get("user")
        
        self.assertEqual(
            str(user_val),
            "0:0",
            f"Production compose user MUST be '0:0', got {user_val}",
        )
        self.assertIsInstance(
            user_val,
            str,
            f"Production compose user must be quoted string '0:0', got {type(user_val)}",
        )

    def test_docker_socket_rw_mount(self):
        """Verify /var/run/docker.sock is mounted read-write (NOT :ro) across both compose files."""
        for compose_file in [CASAOS_COMPOSE_PATH, PROD_COMPOSE_PATH]:
            content = compose_file.read_text(encoding="utf-8")
            data = yaml.safe_load(content)
            orbit_svc = data["services"]["orbit"]
            volumes = orbit_svc.get("volumes", [])
            
            sock_found = False
            for v in volumes:
                v_str = str(v)
                if "/var/run/docker.sock" in v_str:
                    sock_found = True
                    # Must NOT have :ro flag
                    self.assertFalse(
                        v_str.endswith(":ro"),
                        f"Docker socket must be read-write in {compose_file.name}, found: {v_str}",
                    )
            self.assertTrue(sock_found, f"Docker socket mount missing in {compose_file.name}")

    def test_casaos_env_variable_expansion_resilience(self):
        """Adversarial test: Test docker compose config under missing AppID vs populated AppID."""
        docker_bin = shutil.which("docker")
        if not docker_bin:
            self.skipTest("Docker CLI not available on host.")

        # Test with AppID unset
        env_empty = os.environ.copy()
        env_empty.pop("AppID", None)
        proc1 = subprocess.run(
            [docker_bin, "compose", "-f", str(CASAOS_COMPOSE_PATH), "config", "-q"],
            env=env_empty,
            capture_output=True,
            text=True,
        )
        self.assertEqual(
            proc1.returncode,
            0,
            f"docker compose config failed with empty AppID: {proc1.stderr}",
        )

        # Test with AppID set to 'dev.orbit.orbit'
        env_set = os.environ.copy()
        env_set["AppID"] = "dev.orbit.orbit"
        proc2 = subprocess.run(
            [docker_bin, "compose", "-f", str(CASAOS_COMPOSE_PATH), "config", "-q"],
            env=env_set,
            capture_output=True,
            text=True,
        )
        self.assertEqual(
            proc2.returncode,
            0,
            f"docker compose config failed with populated AppID: {proc2.stderr}",
        )

    def test_security_opt_no_new_privileges_with_root(self):
        """Verify security_opt: no-new-privileges:true is present without conflicting with root UID."""
        for compose_file in [CASAOS_COMPOSE_PATH, PROD_COMPOSE_PATH]:
            content = compose_file.read_text(encoding="utf-8")
            data = yaml.safe_load(content)
            orbit_svc = data["services"]["orbit"]
            sec_opts = orbit_svc.get("security_opt", [])
            self.assertTrue(
                any("no-new-privileges:true" in opt for opt in sec_opts),
                f"Missing security_opt no-new-privileges:true in {compose_file.name}",
            )


class TestMultiArchCrossCompileAdversarial(unittest.TestCase):
    """Adversarial verification of Multi-Arch cross-compilation pipeline."""

    def test_go_arm64_cross_compilation_integrity(self):
        """Verify backend compiles cleanly for linux/arm64 producing valid ELF binary."""
        go_bin = shutil.which("go")
        if not go_bin:
            self.skipTest("Go compiler not found on host.")

        out_bin = REPO_ROOT / "tests" / "orbit_arm64_test_bin"
        try:
            env = os.environ.copy()
            env["CGO_ENABLED"] = "0"
            env["GOOS"] = "linux"
            env["GOARCH"] = "arm64"

            proc = subprocess.run(
                [go_bin, "build", "-ldflags=-s -w", "-o", str(out_bin), "./cmd/orbit"],
                cwd=str(BACKEND_DIR),
                env=env,
                capture_output=True,
                text=True,
                timeout=60,
            )
            self.assertEqual(
                proc.returncode,
                0,
                f"Go ARM64 cross-compilation failed: {proc.stderr}",
            )
            self.assertTrue(out_bin.is_file(), "Compiled binary not found.")

            # Validate ELF magic number and ARM64 machine header
            with open(out_bin, "rb") as f:
                header = f.read(20)
                # Magic bytes: 0x7F 'E' 'L' 'F'
                self.assertEqual(header[:4], b"\x7fELF", "Binary is not a valid ELF file.")
                # 64-bit ELF (EI_CLASS = 2)
                self.assertEqual(header[4], 2, "ELF file is not 64-bit.")
                # Little-endian (EI_DATA = 1)
                self.assertEqual(header[5], 1, "ELF file is not little-endian.")
                # e_machine field at offset 18 (2 bytes little-endian): EM_AARCH64 = 183 (0x00B7)
                machine = header[18] | (header[19] << 8)
                self.assertEqual(
                    machine,
                    183,
                    f"ELF e_machine expected 183 (AArch64), got: {machine}",
                )
        finally:
            if out_bin.is_file():
                try:
                    out_bin.unlink()
                except OSError:
                    pass

    def test_workflow_has_both_amd64_arm64_and_timeout(self):
        """Verify CI workflow docker-publish.yml targets both architectures with 30m timeout."""
        content = WORKFLOW_PATH.read_text(encoding="utf-8")
        data = yaml.safe_load(content)
        job = data.get("jobs", {}).get("build-and-push", {})
        
        self.assertEqual(job.get("timeout-minutes"), 30)
        
        platforms = ""
        for step in job.get("steps", []):
            if "docker/build-push-action" in step.get("uses", ""):
                platforms = step.get("with", {}).get("platforms", "")
                break
        
        self.assertIn("linux/amd64", platforms)
        self.assertIn("linux/arm64", platforms)


if __name__ == "__main__":
    unittest.main(verbosity=2)
