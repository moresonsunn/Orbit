#!/usr/bin/env python3
"""tests/test_docker_multiarch.py

Comprehensive Test Suite for Orbit Docker Multi-Arch Build & CI Workflow.
Covers Tiers 1-4:
  - Tier 1: Dockerfile multi-arch native build stage directives & cross-compile variables.
  - Tier 2: GitHub Actions docker-publish.yml timeout and buildx platform configurations.
  - Tier 3: Go toolchain cross-compilation functionality (ELF AArch64 binary validation).
  - Tier 4: Docker Buildx syntax validation and multi-arch capability checks.
"""

import os
import re
import shutil
import struct
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent
DOCKERFILE_PATH = REPO_ROOT / "Dockerfile"
WORKFLOW_PATH = REPO_ROOT / ".github" / "workflows" / "docker-publish.yml"
BACKEND_DIR = REPO_ROOT / "backend"


class TestTier1DockerfileMultiArch(unittest.TestCase):
    """Tier 1: Dockerfile multi-arch native build stage & cross-compilation assertions."""

    @classmethod
    def setUpClass(cls):
        cls.assertTrue(
            DOCKERFILE_PATH.is_file(),
            f"Dockerfile does not exist at expected path: {DOCKERFILE_PATH}",
        )
        cls.dockerfile_content = DOCKERFILE_PATH.read_text(encoding="utf-8")
        cls.lines = [line.strip() for line in cls.dockerfile_content.splitlines()]

    def test_frontend_builder_platform_buildplatform(self):
        """Verify frontend-builder stage pins --platform=$BUILDPLATFORM to avoid QEMU emulation."""
        pattern = re.compile(
            r"^FROM\s+--platform=\$BUILDPLATFORM\s+node:\S+\s+AS\s+frontend-builder",
            re.IGNORECASE,
        )
        match = any(pattern.search(line) for line in self.lines)
        self.assertTrue(
            match,
            "Dockerfile frontend-builder stage must declare '--platform=$BUILDPLATFORM'. "
            "Expected: 'FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-builder'",
        )

    def test_backend_builder_platform_buildplatform(self):
        """Verify backend-builder stage pins --platform=$BUILDPLATFORM to run Go compiler natively."""
        pattern = re.compile(
            r"^FROM\s+--platform=\$BUILDPLATFORM\s+golang:\S+\s+AS\s+backend-builder",
            re.IGNORECASE,
        )
        match = any(pattern.search(line) for line in self.lines)
        self.assertTrue(
            match,
            "Dockerfile backend-builder stage must declare '--platform=$BUILDPLATFORM'. "
            "Expected: 'FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS backend-builder'",
        )

    def test_target_args_declared(self):
        """Verify ARG TARGETOS and ARG TARGETARCH are declared for BuildKit injection."""
        has_targetos = any(
            re.search(r"^ARG\s+TARGETOS\b", line) for line in self.lines
        )
        has_targetarch = any(
            re.search(r"^ARG\s+TARGETARCH\b", line) for line in self.lines
        )
        self.assertTrue(
            has_targetos,
            "Dockerfile must declare 'ARG TARGETOS' in backend-builder stage.",
        )
        self.assertTrue(
            has_targetarch,
            "Dockerfile must declare 'ARG TARGETARCH' in backend-builder stage.",
        )

    def test_cgo_disabled(self):
        """Verify CGO_ENABLED=0 is explicitly configured for static cross-compilation."""
        cgo_pattern = re.compile(r"ENV\s+.*CGO_ENABLED=0|CGO_ENABLED=0\s+go\s+build")
        match = any(cgo_pattern.search(line) for line in self.lines)
        self.assertTrue(
            match,
            "Dockerfile must set CGO_ENABLED=0 to ensure pure-Go static compilation without gcc toolchain.",
        )

    def test_go_build_cross_compile_flags(self):
        """Verify go build command uses GOOS and GOARCH variables."""
        build_pattern = re.compile(
            r"GOOS=\$\{?TARGETOS\}?\s+GOARCH=\$\{?TARGETARCH\}?\s+go\s+build"
        )
        match = any(build_pattern.search(line) for line in self.lines)
        self.assertTrue(
            match,
            "Dockerfile backend build step must invoke "
            "'GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build ...'",
        )

    def test_runtime_stage_targets_target_platform(self):
        """Verify the final runtime stage does NOT pin to $BUILDPLATFORM, targeting target architecture."""
        from_lines = [
            line for line in self.lines if line.upper().startswith("FROM")
        ]
        self.assertGreaterEqual(
            len(from_lines),
            3,
            "Dockerfile must contain at least 3 stages (frontend, backend, runtime).",
        )
        runtime_stage_line = from_lines[-1]
        self.assertNotIn(
            "--platform=$BUILDPLATFORM",
            runtime_stage_line,
            "Production runtime stage must NOT pin to $BUILDPLATFORM; "
            "it must inherit or explicitly specify $TARGETPLATFORM.",
        )
        self.assertTrue(
            "debian" in runtime_stage_line.lower()
            or "alpine" in runtime_stage_line.lower(),
            f"Runtime stage expected Debian/Alpine base, found: {runtime_stage_line}",
        )

    def test_frontend_dist_embedded_in_backend(self):
        """Verify frontend build artifacts are copied from frontend-builder to backend dist folder."""
        copy_pattern = re.compile(
            r"COPY\s+--from=frontend-builder\s+/app/frontend/dist\s+\./web/dist"
        )
        match = any(copy_pattern.search(line) for line in self.lines)
        self.assertTrue(
            match,
            "Dockerfile must copy built frontend assets into ./web/dist for go:embed.",
        )

    def test_healthcheck_directive_present(self):
        """Verify production runtime stage defines HEALTHCHECK probe."""
        healthcheck_present = any(
            line.upper().startswith("HEALTHCHECK") for line in self.lines
        )
        self.assertTrue(
            healthcheck_present,
            "Dockerfile must define a container HEALTHCHECK directive.",
        )


class TestTier2CIWorkflowConfig(unittest.TestCase):
    """Tier 2: GitHub Actions docker-publish.yml workflow assertions."""

    @classmethod
    def setUpClass(cls):
        cls.assertTrue(
            WORKFLOW_PATH.is_file(),
            f"Workflow file does not exist at: {WORKFLOW_PATH}",
        )
        content = WORKFLOW_PATH.read_text(encoding="utf-8")
        try:
            cls.workflow = yaml.safe_load(content)
        except Exception as e:
            cls.fail(f"Failed to parse workflow YAML: {e}")

    def test_workflow_has_build_and_push_job(self):
        """Verify workflow declares jobs.build-and-push."""
        jobs = self.workflow.get("jobs", {})
        self.assertIn(
            "build-and-push",
            jobs,
            "Workflow must contain 'build-and-push' job under 'jobs'.",
        )

    def test_workflow_timeout_minutes_is_30(self):
        """Verify jobs.build-and-push sets timeout-minutes: 30 to prevent runaway builds."""
        job = self.workflow.get("jobs", {}).get("build-and-push", {})
        timeout = job.get("timeout-minutes")
        self.assertEqual(
            timeout,
            30,
            f"jobs.build-and-push timeout-minutes must be set to 30. Current value: {timeout}",
        )

    def test_workflow_multiarch_platforms(self):
        """Verify build-push-action targets both linux/amd64 and linux/arm64."""
        job = self.workflow.get("jobs", {}).get("build-and-push", {})
        steps = job.get("steps", [])
        build_step = None
        for step in steps:
            uses = step.get("uses", "")
            if "docker/build-push-action" in uses:
                build_step = step
                break

        self.assertIsNotNone(
            build_step,
            "Workflow must include a step using 'docker/build-push-action'.",
        )
        platforms = build_step.get("with", {}).get("platforms", "")
        self.assertIn(
            "linux/amd64",
            platforms,
            f"Build platforms must include 'linux/amd64', got: '{platforms}'",
        )
        self.assertIn(
            "linux/arm64",
            platforms,
            f"Build platforms must include 'linux/arm64', got: '{platforms}'",
        )

    def test_workflow_qemu_and_buildx_steps(self):
        """Verify workflow retains setup-qemu-action and setup-buildx-action."""
        job = self.workflow.get("jobs", {}).get("build-and-push", {})
        steps = job.get("steps", [])
        step_uses = [step.get("uses", "") for step in steps]
        has_qemu = any("docker/setup-qemu-action" in u for u in step_uses)
        has_buildx = any("docker/setup-buildx-action" in u for u in step_uses)
        self.assertTrue(
            has_qemu,
            "Workflow must retain 'docker/setup-qemu-action' for runtime stage operations.",
        )
        self.assertTrue(
            has_buildx,
            "Workflow must retain 'docker/setup-buildx-action'.",
        )


class TestTier3GoCrossCompilation(unittest.TestCase):
    """Tier 3: Go toolchain cross-compilation functionality tests (ELF verification)."""

    @classmethod
    def setUpClass(cls):
        # Verify Go toolchain is installed
        go_bin = shutil.which("go")
        if not go_bin:
            raise unittest.SkipTest("Go toolchain ('go') not found in PATH.")
        cls.go_bin = go_bin
        cls.temp_dir = tempfile.mkdtemp(prefix="orbit_cross_test_")

    @classmethod
    def tearDownClass(cls):
        if os.path.exists(cls.temp_dir):
            shutil.rmtree(cls.temp_dir, ignore_errors=True)

    def test_go_cross_compile_arm64_elf_validation(self):
        """Cross-compile Orbit to Linux ARM64 and verify ELF AArch64 executable headers."""
        output_bin = Path(self.temp_dir) / "orbit_arm64_test"
        env = os.environ.copy()
        env["CGO_ENABLED"] = "0"
        env["GOOS"] = "linux"
        env["GOARCH"] = "arm64"

        cmd = [
            self.go_bin,
            "build",
            "-ldflags=-s -w",
            "-o",
            str(output_bin),
            "./cmd/orbit",
        ]

        proc = subprocess.run(
            cmd,
            cwd=str(BACKEND_DIR),
            env=env,
            capture_output=True,
            text=True,
            timeout=120,
        )

        self.assertEqual(
            proc.returncode,
            0,
            f"Go cross-compilation to linux/arm64 failed with exit code {proc.returncode}:\n"
            f"STDOUT: {proc.stdout}\nSTDERR: {proc.stderr}",
        )
        self.assertTrue(
            output_bin.is_file(),
            f"Expected compiled binary at {output_bin}",
        )
        self.assertGreater(
            output_bin.stat().st_size,
            1_000_000,
            "Compiled ARM64 binary is unusually small (<1MB).",
        )

        # Validate ELF Header (20 bytes minimum)
        with open(output_bin, "rb") as f:
            header = f.read(20)

        self.assertEqual(
            len(header),
            20,
            "Could not read complete 20-byte ELF identification header.",
        )
        # Magic bytes: 0x7F, 'E', 'L', 'F'
        self.assertEqual(
            header[:4],
            b"\x7fELF",
            f"Invalid ELF magic bytes: {list(header[:4])}",
        )
        # EI_CLASS: 2 = 64-bit architecture
        self.assertEqual(
            header[4],
            2,
            f"Expected ELFCLASS64 (2), got: {header[4]}",
        )
        # EI_DATA: 1 = 2's complement, little endian
        self.assertEqual(
            header[5],
            1,
            f"Expected ELFDATA2LSB (1), got: {header[5]}",
        )
        # e_machine: byte offset 18..20 (little-endian uint16)
        e_machine = struct.unpack("<H", header[18:20])[0]
        # EM_AARCH64 = 183 (0xB7)
        self.assertEqual(
            e_machine,
            183,
            f"Expected e_machine=183 (0xB7 EM_AARCH64), got: {e_machine} ({hex(e_machine)})",
        )

    def test_go_cross_compile_amd64_elf_validation(self):
        """Cross-compile Orbit to Linux AMD64 and verify ELF x86-64 executable headers."""
        output_bin = Path(self.temp_dir) / "orbit_amd64_test"
        env = os.environ.copy()
        env["CGO_ENABLED"] = "0"
        env["GOOS"] = "linux"
        env["GOARCH"] = "amd64"

        cmd = [
            self.go_bin,
            "build",
            "-ldflags=-s -w",
            "-o",
            str(output_bin),
            "./cmd/orbit",
        ]

        proc = subprocess.run(
            cmd,
            cwd=str(BACKEND_DIR),
            env=env,
            capture_output=True,
            text=True,
            timeout=120,
        )

        self.assertEqual(
            proc.returncode,
            0,
            f"Go cross-compilation to linux/amd64 failed:\nSTDERR: {proc.stderr}",
        )

        with open(output_bin, "rb") as f:
            header = f.read(20)

        self.assertEqual(header[:4], b"\x7fELF")
        self.assertEqual(header[4], 2)  # 64-bit
        # EM_X86_64 = 62 (0x3E)
        e_machine = struct.unpack("<H", header[18:20])[0]
        self.assertEqual(
            e_machine,
            62,
            f"Expected e_machine=62 (0x3E EM_X86_64), got: {e_machine}",
        )


class TestTier4DockerBuildxDryRun(unittest.TestCase):
    """Tier 4: Docker Buildx syntax and multi-architecture builder validation."""

    @classmethod
    def setUpClass(cls):
        docker_bin = shutil.which("docker")
        if not docker_bin:
            raise unittest.SkipTest("Docker CLI not found in PATH.")
        # Check if Docker daemon is running and reachable
        try:
            check = subprocess.run([docker_bin, "info"], capture_output=True, timeout=5)
            if check.returncode != 0:
                raise unittest.SkipTest(f"Docker daemon is not running: {check.stderr.decode('utf-8', errors='ignore')[:100]}")
        except Exception as e:
            raise unittest.SkipTest(f"Cannot connect to Docker daemon: {e}")
        cls.docker_bin = docker_bin

    def test_docker_buildx_check_syntax(self):
        """Verify Dockerfile passes buildx build --check without syntax warnings or errors."""
        cmd = [self.docker_bin, "buildx", "build", "--check", "."]
        try:
            proc = subprocess.run(
                cmd,
                cwd=str(REPO_ROOT),
                capture_output=True,
                text=True,
                timeout=60,
            )
        except Exception as e:
            self.fail(f"Failed to execute docker buildx build --check: {e}")

        self.assertEqual(
            proc.returncode,
            0,
            f"docker buildx build --check failed with code {proc.returncode}:\n"
            f"STDOUT: {proc.stdout}\nSTDERR: {proc.stderr}",
        )

    def test_docker_buildx_multiarch_builder_support(self):
        """Verify active Docker Buildx builder supports linux/amd64 and linux/arm64 platforms."""
        cmd = [self.docker_bin, "buildx", "inspect"]
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30,
        )
        self.assertEqual(
            proc.returncode,
            0,
            f"docker buildx inspect failed:\n{proc.stderr}",
        )
        output = proc.stdout.lower()
        self.assertIn("linux/amd64", output)
        self.assertIn("linux/arm64", output)


if __name__ == "__main__":
    unittest.main(verbosity=2)
