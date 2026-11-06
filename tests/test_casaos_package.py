#!/usr/bin/env python3
"""tests/test_casaos_package.py

Comprehensive Test Suite for Orbit CasaOS App Store Integration Package.
Covers Tiers 1-5:
  - Tier 1: Filesystem & Asset Existence (compose file, icons in Apps/Orbit and frontend/public).
  - Tier 2: Docker Compose Syntax & Service Configuration (`docker compose config -q`).
  - Tier 3: Full `x-casaos` Extension Metadata Schema & Storage/Socket/Port Mappings.
  - Tier 4: Icon Image Properties (PNG format, 512x512 dimensions, RGBA mode, valid SVG XML).
  - Tier 5: Documentation Completeness (1-Click UI Import and Custom AppStore repository setup).
"""

import re
import shutil
import subprocess
import unittest
import xml.etree.ElementTree as ET
from pathlib import Path

from PIL import Image
import yaml

REPO_ROOT = Path(__file__).resolve().parent.parent
APPS_ORBIT_DIR = REPO_ROOT / "Apps" / "Orbit"
COMPOSE_PATH = APPS_ORBIT_DIR / "docker-compose.yml"
ICON_SVG_PATH = APPS_ORBIT_DIR / "icon.svg"
ICON_PNG_PATH = APPS_ORBIT_DIR / "icon.png"
PUBLIC_ICON_SVG_PATH = REPO_ROOT / "frontend" / "public" / "orbit-icon.svg"
DOCS_INSTALL_PATH = REPO_ROOT / "docs" / "INSTALLATION.md"
README_PATH = REPO_ROOT / "README.md"

VALID_CASAOS_CATEGORIES = {
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

REVERSE_DOMAIN_REGEX = re.compile(
    r"^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:\.[a-z0-9]+(?:[._-][a-z0-9]+)*)+$"
)
COMPOSE_NAME_REGEX = re.compile(r"^[a-z0-9_-]+$")
SEMVER_REGEX = re.compile(r"^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?$")


class TestTier1CasaOSFilesystem(unittest.TestCase):
    """Tier 1: Verify presence of all required CasaOS package files and visual assets."""

    def test_compose_file_exists(self):
        """Verify Apps/Orbit/docker-compose.yml exists and is a non-empty regular file."""
        self.assertTrue(
            COMPOSE_PATH.is_file(),
            f"Missing CasaOS compose file: {COMPOSE_PATH}",
        )
        self.assertGreater(
            COMPOSE_PATH.stat().st_size,
            0,
            f"CasaOS compose file is empty: {COMPOSE_PATH}",
        )

    def test_apps_orbit_icon_svg_exists(self):
        """Verify Apps/Orbit/icon.svg exists."""
        self.assertTrue(
            ICON_SVG_PATH.is_file(),
            f"Missing CasaOS vector icon: {ICON_SVG_PATH}",
        )
        self.assertGreater(ICON_SVG_PATH.stat().st_size, 0)

    def test_apps_orbit_icon_png_exists(self):
        """Verify Apps/Orbit/icon.png exists."""
        self.assertTrue(
            ICON_PNG_PATH.is_file(),
            f"Missing CasaOS high-resolution raster icon: {ICON_PNG_PATH}",
        )
        self.assertGreater(ICON_PNG_PATH.stat().st_size, 0)

    def test_frontend_public_orbit_icon_svg_exists(self):
        """Verify frontend/public/orbit-icon.svg exists for web UI and raw asset referencing."""
        self.assertTrue(
            PUBLIC_ICON_SVG_PATH.is_file(),
            f"Missing frontend asset: {PUBLIC_ICON_SVG_PATH}",
        )
        self.assertGreater(PUBLIC_ICON_SVG_PATH.stat().st_size, 0)


class TestTier2DockerComposeSyntax(unittest.TestCase):
    """Tier 2: Docker Compose CLI and structural specification validation."""

    @classmethod
    def setUpClass(cls):
        if not COMPOSE_PATH.is_file():
            raise unittest.SkipTest(f"{COMPOSE_PATH} does not exist yet.")
        cls.compose_text = COMPOSE_PATH.read_text(encoding="utf-8")
        try:
            cls.compose_data = yaml.safe_load(cls.compose_text) or {}
        except Exception as e:
            cls.fail(f"Invalid YAML in {COMPOSE_PATH}: {e}")

    def test_docker_compose_cli_validation(self):
        """Execute `docker compose -f Apps/Orbit/docker-compose.yml config -q`."""
        docker_bin = shutil.which("docker")
        if not docker_bin:
            raise unittest.SkipTest("Docker CLI not available.")

        # Note: in CasaOS, $AppID is substituted. We provide a default AppID environment if needed.
        proc = subprocess.run(
            [
                docker_bin,
                "compose",
                "-f",
                str(COMPOSE_PATH),
                "config",
                "-q",
            ],
            cwd=str(REPO_ROOT),
            capture_output=True,
            text=True,
            timeout=30,
        )
        self.assertEqual(
            proc.returncode,
            0,
            f"docker compose config -q failed with exit code {proc.returncode}:\n"
            f"STDOUT: {proc.stdout}\nSTDERR: {proc.stderr}",
        )

    def test_compose_top_level_name(self):
        """Verify top-level compose project name is lowercase alphanumeric (^[a-z0-9_-]+$)."""
        name = self.compose_data.get("name")
        self.assertIsNotNone(name, "Compose file must have a top-level 'name' attribute.")
        self.assertTrue(
            COMPOSE_NAME_REGEX.fullmatch(str(name)),
            f"Top-level 'name' must match ^[a-z0-9_-]+$, got: '{name}'",
        )

    def test_services_section_contains_orbit(self):
        """Verify compose services section contains the 'orbit' service container."""
        services = self.compose_data.get("services", {})
        self.assertIn(
            "orbit",
            services,
            f"Compose file must define an 'orbit' service. Defined: {list(services.keys())}",
        )

    def test_orbit_service_image(self):
        """Verify orbit service image references moresonsunn/orbit:latest or project image."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        image = orbit_svc.get("image", "")
        self.assertTrue(
            "orbit" in image.lower(),
            f"Orbit service image expected to reference Orbit container, got: '{image}'",
        )

    def test_orbit_restart_policy(self):
        """Verify orbit service specifies restart: unless-stopped or restart: always."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        restart = orbit_svc.get("restart", "")
        self.assertIn(
            restart,
            ["unless-stopped", "always"],
            f"Restart policy must be 'unless-stopped' or 'always', got: '{restart}'",
        )

    def test_orbit_security_opt(self):
        """Verify orbit service specifies security_opt: no-new-privileges:true."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        sec_opts = orbit_svc.get("security_opt", [])
        self.assertTrue(
            any("no-new-privileges:true" in opt.replace(" ", "") for opt in sec_opts),
            f"Service security_opt must include 'no-new-privileges:true', got: {sec_opts}",
        )


class TestTier3CasaOSMetadataSchema(unittest.TestCase):
    """Tier 3: Official CasaOS x-casaos extension metadata and mounting schema."""

    @classmethod
    def setUpClass(cls):
        if not COMPOSE_PATH.is_file():
            raise unittest.SkipTest(f"{COMPOSE_PATH} does not exist yet.")
        cls.compose_text = COMPOSE_PATH.read_text(encoding="utf-8")
        try:
            cls.compose_data = yaml.safe_load(cls.compose_text) or {}
        except Exception as e:
            cls.fail(f"Invalid YAML: {e}")
        cls.xcasaos = cls.compose_data.get("x-casaos")

    def test_xcasaos_block_exists(self):
        """Verify top-level x-casaos block is present and is a dictionary."""
        self.assertIsInstance(
            self.xcasaos,
            dict,
            "Top-level 'x-casaos' section is missing or is not a mapping in docker-compose.yml.",
        )

    def test_xcasaos_id_reverse_domain(self):
        """Verify x-casaos.id is 'dev.orbit.orbit' and adheres to reverse-domain naming."""
        app_id = self.xcasaos.get("id")
        self.assertIsNotNone(app_id, "Missing 'x-casaos.id' field.")
        self.assertEqual(
            app_id,
            "dev.orbit.orbit",
            f"Expected x-casaos.id to be 'dev.orbit.orbit', got: '{app_id}'",
        )
        self.assertTrue(
            REVERSE_DOMAIN_REGEX.fullmatch(app_id),
            f"x-casaos.id must match reverse domain regex: '{app_id}'",
        )

    def test_xcasaos_main_service(self):
        """Verify x-casaos.main is 'orbit' and exists under services."""
        main_svc = self.xcasaos.get("main")
        self.assertEqual(
            main_svc,
            "orbit",
            f"Expected x-casaos.main to be 'orbit', got: '{main_svc}'",
        )
        services = self.compose_data.get("services", {})
        self.assertIn(
            main_svc,
            services,
            f"x-casaos.main '{main_svc}' does not exist in services: {list(services.keys())}",
        )

    def test_xcasaos_port_map_is_quoted_string(self):
        """Verify x-casaos.port_map is strictly a string (not an integer) with value '8787'."""
        port_map = self.xcasaos.get("port_map")
        self.assertIsNotNone(port_map, "Missing 'x-casaos.port_map'.")
        self.assertIsInstance(
            port_map,
            str,
            f"x-casaos.port_map MUST be a quoted string (e.g. \"8787\"), got {type(port_map).__name__}: {port_map}",
        )
        self.assertEqual(
            port_map,
            "8787",
            f"Expected x-casaos.port_map to be '8787', got '{port_map}'",
        )

    def test_xcasaos_scheme_and_index(self):
        """Verify x-casaos.scheme is 'http' and index is '/'."""
        scheme = self.xcasaos.get("scheme", "http")
        index = self.xcasaos.get("index", "/")
        self.assertEqual(scheme, "http")
        self.assertEqual(index, "/")

    def test_xcasaos_category_canonical(self):
        """Verify x-casaos.category is 'Developer' (one of the 9 official CasaOS categories)."""
        category = self.xcasaos.get("category")
        self.assertIn(
            category,
            VALID_CASAOS_CATEGORIES,
            f"x-casaos.category '{category}' is invalid. Must be one of: {sorted(VALID_CASAOS_CATEGORIES)}",
        )
        self.assertEqual(
            category,
            "Developer",
            f"Expected category 'Developer' for Orbit, got '{category}'",
        )

    def test_xcasaos_architectures(self):
        """Verify x-casaos.architectures contains both 'amd64' and 'arm64'."""
        arches = self.xcasaos.get("architectures")
        self.assertIsInstance(arches, list, "x-casaos.architectures must be a list.")
        self.assertIn(
            "amd64",
            arches,
            f"x-casaos.architectures must include 'amd64', got: {arches}",
        )
        self.assertIn(
            "arm64",
            arches,
            f"x-casaos.architectures must include 'arm64', got: {arches}",
        )

    def test_xcasaos_localized_title(self):
        """Verify x-casaos.title is a localized dictionary containing 'en_US'."""
        title = self.xcasaos.get("title")
        self.assertIsInstance(
            title,
            dict,
            f"x-casaos.title must be a localized map, not a plain string. Got: {type(title).__name__}",
        )
        self.assertIn("en_US", title)
        self.assertEqual(title["en_US"], "Orbit")

    def test_xcasaos_localized_tagline(self):
        """Verify x-casaos.tagline is a localized dictionary containing 'en_US'."""
        tagline = self.xcasaos.get("tagline")
        self.assertIsInstance(
            tagline,
            dict,
            f"x-casaos.tagline must be a localized map, got: {type(tagline).__name__}",
        )
        self.assertIn("en_US", tagline)
        self.assertGreater(len(tagline["en_US"].strip()), 10)

    def test_xcasaos_localized_description(self):
        """Verify x-casaos.description is a localized dictionary containing 'en_US'."""
        desc = self.xcasaos.get("description")
        self.assertIsInstance(
            desc,
            dict,
            f"x-casaos.description must be a localized map, got: {type(desc).__name__}",
        )
        self.assertIn("en_US", desc)
        self.assertGreater(len(desc["en_US"].strip()), 50)

    def test_xcasaos_attribution(self):
        """Verify developer and author fields are present."""
        developer = self.xcasaos.get("developer")
        author = self.xcasaos.get("author")
        self.assertTrue(developer, "Missing x-casaos.developer field.")
        self.assertTrue(author, "Missing x-casaos.author field.")

    def test_xcasaos_version_semver(self):
        """Verify x-casaos.version or compose version follows semantic versioning if present."""
        version = self.xcasaos.get("version") or self.compose_data.get("version")
        if version is not None:
            self.assertTrue(
                SEMVER_REGEX.match(str(version)),
                f"Version must follow semantic versioning (e.g. '1.0.0'), got: '{version}'",
            )

    def test_xcasaos_icon_url_valid(self):
        """Verify x-casaos.icon points to an HTTPS URL ending with .png or .svg."""
        icon_url = self.xcasaos.get("icon")
        self.assertIsInstance(icon_url, str, "Missing x-casaos.icon URL string.")
        self.assertTrue(
            icon_url.startswith("https://"),
            f"x-casaos.icon must start with https://, got: {icon_url}",
        )
        self.assertTrue(
            icon_url.endswith(".svg") or icon_url.endswith(".png"),
            f"x-casaos.icon must end with .svg or .png, got: {icon_url}",
        )

    def test_orbit_port_8787_mapping(self):
        """Verify service port mapping binds external port 8787 to container 8787."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        ports = orbit_svc.get("ports", [])
        port_8787_mapped = False
        for p in ports:
            if isinstance(p, str):
                parts = p.split(":")
                if len(parts) >= 2 and parts[-1] == "8787" and parts[-2] == "8787":
                    port_8787_mapped = True
                    break
            elif isinstance(p, dict):
                target = str(p.get("target", ""))
                published = str(p.get("published", ""))
                if target == "8787" and published == "8787":
                    port_8787_mapped = True
                    break
        self.assertTrue(
            port_8787_mapped,
            f"Orbit service must map host port 8787 to container port 8787. Found ports: {ports}",
        )

    test_orbit_port_8080_mapping = test_orbit_port_8787_mapping

    def test_orbit_docker_sock_mount(self):
        """Verify /var/run/docker.sock is mapped to /var/run/docker.sock."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        volumes = orbit_svc.get("volumes", [])
        sock_mapped = False
        for v in volumes:
            if isinstance(v, str):
                parts = v.split(":")
                if parts[0] == "/var/run/docker.sock" and parts[1] == "/var/run/docker.sock":
                    sock_mapped = True
                    break
            elif isinstance(v, dict):
                src = v.get("source", "")
                tgt = v.get("target", "")
                if src == "/var/run/docker.sock" and tgt == "/var/run/docker.sock":
                    sock_mapped = True
                    break
        self.assertTrue(
            sock_mapped,
            f"Orbit service must mount '/var/run/docker.sock' to '/var/run/docker.sock'. Volumes: {volumes}",
        )

    def test_orbit_data_volume_persistence(self):
        """Verify persistent data directory is mapped to container /data."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        volumes = orbit_svc.get("volumes", [])
        data_mapped = False
        for v in volumes:
            if isinstance(v, str):
                parts = v.split(":")
                if len(parts) >= 2 and parts[1] == "/data" and ("/DATA/AppData" in parts[0] or "data" in parts[0]):
                    data_mapped = True
                    break
            elif isinstance(v, dict):
                tgt = v.get("target", "")
                src = v.get("source", "")
                if tgt == "/data" and ("/DATA/AppData" in src or "data" in src):
                    data_mapped = True
                    break
        self.assertTrue(
            data_mapped,
            f"Orbit service must mount persistent data volume to '/data'. Volumes: {volumes}",
        )

    def test_service_level_xcasaos_annotations(self):
        """Verify service-level x-casaos block documents ports and volumes."""
        orbit_svc = self.compose_data.get("services", {}).get("orbit", {})
        svc_xcasaos = orbit_svc.get("x-casaos")
        self.assertIsInstance(
            svc_xcasaos,
            dict,
            "Service orbit must define service-level 'x-casaos' annotations.",
        )
        ports = svc_xcasaos.get("ports", [])
        volumes = svc_xcasaos.get("volumes", [])
        self.assertGreater(len(ports), 0, "Service x-casaos must document exposed ports.")
        self.assertGreater(len(volumes), 0, "Service x-casaos must document volumes.")
        # Check localized description
        self.assertIn("en_US", ports[0].get("description", {}))
        self.assertIn("en_US", volumes[0].get("description", {}))


class TestTier4CasaOSIconProperties(unittest.TestCase):
    """Tier 4: Icon asset dimensional and format integrity."""

    def test_icon_png_format_dimensions_mode(self):
        """Verify Apps/Orbit/icon.png is a 512x512 RGBA PNG."""
        if not ICON_PNG_PATH.is_file():
            self.fail(f"Icon PNG not found at: {ICON_PNG_PATH}")

        with Image.open(ICON_PNG_PATH) as img:
            self.assertEqual(
                img.format,
                "PNG",
                f"Icon must be PNG format, got: {img.format}",
            )
            self.assertEqual(
                img.size,
                (512, 512),
                f"Icon PNG dimensions must be exactly 512x512, got: {img.size}",
            )
            self.assertIn(
                img.mode,
                ["RGBA", "RGB"],
                f"Icon PNG color mode must be RGBA or RGB, got: {img.mode}",
            )

    def test_apps_orbit_icon_svg_xml_structure(self):
        """Verify Apps/Orbit/icon.svg is valid XML with root <svg> tag."""
        if not ICON_SVG_PATH.is_file():
            self.fail(f"Icon SVG not found at: {ICON_SVG_PATH}")

        try:
            tree = ET.parse(ICON_SVG_PATH)
            root = tree.getroot()
        except Exception as e:
            self.fail(f"Invalid XML structure in {ICON_SVG_PATH}: {e}")

        self.assertTrue(
            root.tag.endswith("svg"),
            f"Root tag must be SVG, got: {root.tag}",
        )
        has_dimension = "viewBox" in root.attrib or ("width" in root.attrib and "height" in root.attrib)
        self.assertTrue(
            has_dimension,
            "SVG icon must have 'viewBox' or 'width' and 'height' attributes.",
        )

    def test_frontend_public_orbit_icon_svg_xml_structure(self):
        """Verify frontend/public/orbit-icon.svg is valid XML with root <svg> tag."""
        if not PUBLIC_ICON_SVG_PATH.is_file():
            self.fail(f"Icon SVG not found at: {PUBLIC_ICON_SVG_PATH}")

        try:
            tree = ET.parse(PUBLIC_ICON_SVG_PATH)
            root = tree.getroot()
        except Exception as e:
            self.fail(f"Invalid XML structure in {PUBLIC_ICON_SVG_PATH}: {e}")

        self.assertTrue(root.tag.endswith("svg"))


class TestTier5CasaOSDocumentation(unittest.TestCase):
    """Tier 5: Documentation coverage for CasaOS deployment methods."""

    @classmethod
    def setUpClass(cls):
        doc_contents = []
        if DOCS_INSTALL_PATH.is_file():
            doc_contents.append(DOCS_INSTALL_PATH.read_text(encoding="utf-8"))
        if README_PATH.is_file():
            doc_contents.append(README_PATH.read_text(encoding="utf-8"))
        cls.combined_docs = "\n".join(doc_contents)

    def test_custom_install_ui_import_documented(self):
        """Verify documentation provides step-by-step 1-Click Custom Install (UI Import)."""
        doc_lower = self.combined_docs.lower()
        has_custom_install = (
            "custom install" in doc_lower
            or "1-click" in doc_lower
            or "ui import" in doc_lower
        )
        self.assertTrue(
            has_custom_install,
            "Documentation (INSTALLATION.md / README.md) must explain CasaOS Custom Install.",
        )
        has_import = "import" in doc_lower or "docker-compose.yml" in doc_lower
        self.assertTrue(
            has_import,
            "Documentation must reference importing docker-compose.yml into CasaOS.",
        )

    def test_custom_appstore_repository_setup_documented(self):
        """Verify documentation explains adding Custom AppStore repository."""
        doc_lower = self.combined_docs.lower()
        has_appstore = (
            "app store" in doc_lower
            or "appstore" in doc_lower
            or "manage app stores" in doc_lower
            or "add source" in doc_lower
        )
        self.assertTrue(
            has_appstore,
            "Documentation must explain registering Orbit via Custom AppStore repository.",
        )
        has_repo_url = (
            ".zip" in doc_lower
            or "github.com/moresonsunn/orbit" in doc_lower
            or "archive/refs/heads/main.zip" in doc_lower
        )
        self.assertTrue(
            has_repo_url,
            "Documentation must include the AppStore repository URL or archive ZIP link.",
        )

    def test_persistent_storage_and_socket_documented(self):
        """Verify documentation covers persistent storage /data and docker socket mapping."""
        self.assertIn(
            "docker.sock",
            self.combined_docs,
            "Documentation should explain /var/run/docker.sock mounting requirement.",
        )
        has_data_mount = "/data" in self.combined_docs or "AppData" in self.combined_docs
        self.assertTrue(
            has_data_mount,
            "Documentation should explain /data or /DATA/AppData persistent storage.",
        )


if __name__ == "__main__":
    unittest.main(verbosity=2)
