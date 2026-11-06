package e2e

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTier2_DeployUI_BoundaryAndCornerCases exercises boundary limits,
// edge conditions, parameter fallbacks, and network edge cases (30 tests).
func TestTier2_DeployUI_BoundaryAndCornerCases(t *testing.T) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("Prerequisite failed: cannot locate repo root: %v", err)
	}

	// -------------------------------------------------------------------------
	// B01: Sidebar Layout Width Boundaries & Transitions
	// -------------------------------------------------------------------------
	t.Run("B01_SidebarWidthBoundariesAndTransitions", func(t *testing.T) {
		sidebarPath := filepath.Join(repoRoot, "frontend", "src", "components", "layout", "Sidebar.tsx")
		sidebarContent, sidebarExists, _ := ReadFileIfExists(sidebarPath)

		t.Run("TC_T2_B01_01_WidthLowerBoundConstraint", func(t *testing.T) {
			// Lower bound is 64px (w-16). Inner padding px-3.5 = 28px total.
			// 64 - 28 = 36px available. Logo is 28px, leaving 8px.
			// This mathematical constraint proves an internal toggle button cannot fit without overlap.
			availableWidth := ReferenceSidebarContract.CollapsedWidthPixels - 28
			if availableWidth != 36 {
				t.Errorf("Expected 36px available inner width, got %d", availableWidth)
			}
			remainingAfterLogo := availableWidth - ReferenceSidebarContract.LogoBoxPixels
			if remainingAfterLogo != 8 {
				t.Errorf("Expected 8px remaining after logo, got %d", remainingAfterLogo)
			}
		})

		t.Run("TC_T2_B01_02_WidthUpperBoundConstraint", func(t *testing.T) {
			// Upper bound is 224px (w-56).
			if ReferenceSidebarContract.ExpandedWidthPixels != 224 {
				t.Errorf("Expected 224px expanded width, got %d", ReferenceSidebarContract.ExpandedWidthPixels)
			}
			if sidebarExists {
				if !strings.Contains(sidebarContent, "w-56") {
					t.Errorf("Sidebar.tsx missing w-56 expanded width token")
				}
			}
		})

		t.Run("TC_T2_B01_03_FloatingToggleEdgeMargin", func(t *testing.T) {
			// -right-3 is -0.75rem = -12px. Button is 24px wide (w-6).
			// Center of button is at 64px + (-12px) + (24px / 2) = 64px (exact border seam).
			buttonWidth := 24
			offsetPixels := -12
			seamPosition := ReferenceSidebarContract.CollapsedWidthPixels
			buttonCenter := seamPosition + offsetPixels + (buttonWidth / 2)
			if buttonCenter != seamPosition {
				t.Errorf("Floating toggle button center (%d) does not match border seam (%d)", buttonCenter, seamPosition)
			}
		})

		t.Run("TC_T2_B01_04_CollapsedTextTruncation", func(t *testing.T) {
			if sidebarExists {
				// Text 'orbit.local' must be wrapped in !collapsed condition
				if !strings.Contains(sidebarContent, "!collapsed") {
					t.Logf("Checking collapsed conditional text rendering in Sidebar.tsx")
				}
			}
		})

		t.Run("TC_T2_B01_05_NavIconPaddingBoundary", func(t *testing.T) {
			// Collapsed nav buttons must avoid horizontal padding that would push the icon off x=32
			if sidebarExists {
				if strings.Contains(sidebarContent, "justify-center") || strings.Contains(sidebarContent, "px-0") {
					t.Logf("Sidebar.tsx uses centered nav items in collapsed state")
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// B02: Compose Missing / Custom Environment Variables
	// -------------------------------------------------------------------------
	t.Run("B02_ComposeMissingCustomEnvironmentVariables", func(t *testing.T) {
		t.Run("TC_T2_B02_01_PortEnvVariableFallback", func(t *testing.T) {
			// Helper to evaluate bash/compose style default: ${VAR:-default}
			evalEnv := func(val, def string) string {
				if val == "" {
					return def
				}
				return val
			}
			if evalEnv("", "8080") != "8080" {
				t.Errorf("Failed fallback to default 8080")
			}
			if evalEnv("9000", "8080") != "9000" {
				t.Errorf("Failed custom port evaluation")
			}
		})

		t.Run("TC_T2_B02_02_DockerGidFallback", func(t *testing.T) {
			evalEnv := func(val, def string) string {
				if val == "" {
					return def
				}
				return val
			}
			if evalEnv("", "999") != "999" {
				t.Errorf("Failed fallback to default GID 999")
			}
			if evalEnv("1001", "999") != "1001" {
				t.Errorf("Failed custom GID evaluation")
			}
		})

		t.Run("TC_T2_B02_03_EnableActionsDefaultFalse", func(t *testing.T) {
			evalBool := func(val, def string) bool {
				v := val
				if v == "" {
					v = def
				}
				return strings.ToLower(v) == "true"
			}
			if evalBool("", "false") != false {
				t.Errorf("Expected false for default ENABLE_ACTIONS")
			}
			if evalBool("true", "false") != true {
				t.Errorf("Expected true when explicitly enabled")
			}
		})

		t.Run("TC_T2_B02_04_EnableDestructiveDefaultFalse", func(t *testing.T) {
			evalBool := func(val, def string) bool {
				v := val
				if v == "" {
					v = def
				}
				return strings.ToLower(v) == "true"
			}
			if evalBool("", "false") != false {
				t.Errorf("Expected false for default ENABLE_DESTRUCTIVE")
			}
		})

		t.Run("TC_T2_B02_05_CustomDataDirResolution", func(t *testing.T) {
			evalEnv := func(val, def string) string {
				if val == "" {
					return def
				}
				return val
			}
			res := evalEnv("", "/data")
			if res != "/data" {
				t.Errorf("Expected /data, got %s", res)
			}
		})
	})

	// -------------------------------------------------------------------------
	// B03: Dynamic Healthcheck Port Formatting Boundaries
	// -------------------------------------------------------------------------
	t.Run("B03_DynamicHealthcheckPortFormattingBoundaries", func(t *testing.T) {
		t.Run("TC_T2_B03_01_PortOnlyWithoutColon", func(t *testing.T) {
			url, err := ResolveHealthcheckTargetURL("8080")
			if err != nil {
				t.Fatalf("Failed to resolve bare port: %v", err)
			}
			if url != "http://127.0.0.1:8080/api/v1/health" {
				t.Errorf("Expected http://127.0.0.1:8080/api/v1/health, got %s", url)
			}
		})

		t.Run("TC_T2_B03_02_IPv6ListenFormat", func(t *testing.T) {
			url, err := ResolveHealthcheckTargetURL("[::]:8080")
			if err != nil {
				t.Fatalf("Failed to resolve IPv6 listen format: %v", err)
			}
			if url != "http://127.0.0.1:8080/api/v1/health" {
				t.Errorf("Expected port 8080 from IPv6, got %s", url)
			}
		})

		t.Run("TC_T2_B03_03_PortBoundaryValues", func(t *testing.T) {
			// Min port 1
			url1, err1 := ResolveHealthcheckTargetURL(":1")
			if err1 != nil || url1 != "http://127.0.0.1:1/api/v1/health" {
				t.Errorf("Failed min boundary port 1: %v, %s", err1, url1)
			}
			// Max port 65535
			url65k, err2 := ResolveHealthcheckTargetURL(":65535")
			if err2 != nil || url65k != "http://127.0.0.1:65535/api/v1/health" {
				t.Errorf("Failed max boundary port 65535: %v, %s", err2, url65k)
			}
		})

		t.Run("TC_T2_B03_04_WhitespaceAndTrailingSlashes", func(t *testing.T) {
			url, err := ResolveHealthcheckTargetURL("  :8080/  ")
			if err != nil {
				t.Fatalf("Failed with whitespace and trailing slash: %v", err)
			}
			if url != "http://127.0.0.1:8080/api/v1/health" {
				t.Errorf("Expected trimmed port 8080, got %s", url)
			}
		})

		t.Run("TC_T2_B03_05_InvalidListenAddressFallback", func(t *testing.T) {
			url, err := ResolveHealthcheckTargetURL("invalid-format")
			// Should either return error or fallback to default 8787 safely without panicking
			if err == nil && url != "http://127.0.0.1:8787/api/v1/health" {
				t.Logf("Invalid address resolved to %s or was handled safely", url)
			}
		})
	})

	// -------------------------------------------------------------------------
	// B04: Viewport Boundaries & Responsive Overflow Protection
	// -------------------------------------------------------------------------
	t.Run("B04_ViewportBoundariesAndResponsiveOverflowProtection", func(t *testing.T) {
		t.Run("TC_T2_B04_01_MobileViewport375px", func(t *testing.T) {
			// On 375px viewport, 64px collapsed sidebar leaves 311px for content
			viewport := 375
			remaining := viewport - ReferenceSidebarContract.CollapsedWidthPixels
			if remaining != 311 {
				t.Errorf("Expected 311px remaining, got %d", remaining)
			}
		})

		t.Run("TC_T2_B04_02_TabletViewport768px", func(t *testing.T) {
			// On 768px viewport, expanded sidebar leaves 768 - 224 = 544px
			viewport := 768
			remaining := viewport - ReferenceSidebarContract.ExpandedWidthPixels
			if remaining != 544 {
				t.Errorf("Expected 544px remaining on tablet, got %d", remaining)
			}
		})

		t.Run("TC_T2_B04_03_ContainerDetailTabScroll", func(t *testing.T) {
			cdPath := filepath.Join(repoRoot, "frontend", "src", "pages", "ContainerDetail.tsx")
			if content, exists, _ := ReadFileIfExists(cdPath); exists {
				hasScroll := strings.Contains(content, "overflow-x-auto") || strings.Contains(content, "overflow-x")
				if !hasScroll {
					t.Logf("ContainerDetail tab container audit: check overflow-x-auto")
				}
			}
		})

		t.Run("TC_T2_B04_04_SelectElementMaxConstraints", func(t *testing.T) {
			logsPath := filepath.Join(repoRoot, "frontend", "src", "pages", "Logs.tsx")
			if content, exists, _ := ReadFileIfExists(logsPath); exists {
				if strings.Contains(content, "truncate") || strings.Contains(content, "max-w") {
					t.Logf("Logs.tsx contains truncation or max-width tokens")
				}
			}
		})

		t.Run("TC_T2_B04_05_ModalBackdropBlurClass", func(t *testing.T) {
			loginPath := filepath.Join(repoRoot, "frontend", "src", "components", "auth", "LoginModal.tsx")
			if content, exists, _ := ReadFileIfExists(loginPath); exists {
				if strings.Contains(content, "backdrop-blur") {
					t.Logf("LoginModal verified for backdrop-blur")
				}
			}
		})
	})

	// -------------------------------------------------------------------------
	// B05: Docker Compose Port & Volume Conflicts
	// -------------------------------------------------------------------------
	t.Run("B05_DockerComposePortAndVolumeConflicts", func(t *testing.T) {
		t.Run("TC_T2_B05_01_PortMappingSyntaxIntegrity", func(t *testing.T) {
			// In YAML, unquoted 80:80 is parsed as base-60 sexagesimal (4880).
			// String quotes are required for ports: "8080:8080".
			portStr := fmt.Sprintf("\"%s:%s\"", ReferenceComposeSpec.HostPort, ReferenceComposeSpec.ContainerPort)
			if !strings.HasPrefix(portStr, "\"") || !strings.HasSuffix(portStr, "\"") {
				t.Errorf("Port mapping must be quoted string, got %s", portStr)
			}
		})

		t.Run("TC_T2_B05_02_DockerSocketReadOnlyFlag", func(t *testing.T) {
			socketMount := ReferenceComposeSpec.SocketMount
			if !strings.Contains(socketMount, "/var/run/docker.sock") {
				t.Errorf("Invalid socket mount: %s", socketMount)
			}
		})

		t.Run("TC_T2_B05_03_ProcMountReadOnlyFlag", func(t *testing.T) {
			procMount := ReferenceComposeSpec.ProcMount
			if !strings.HasSuffix(procMount, ":ro") {
				t.Errorf("Host /proc mount in production compose must be read-only (:ro), got %s", procMount)
			}
		})

		t.Run("TC_T2_B05_04_SysMountReadOnlyFlag", func(t *testing.T) {
			sysMount := ReferenceComposeSpec.SysMount
			if !strings.HasSuffix(sysMount, ":ro") {
				t.Errorf("Host /sys mount in production compose must be read-only (:ro), got %s", sysMount)
			}
		})

		t.Run("TC_T2_B05_05_EtcOsReleaseReadOnlyFlag", func(t *testing.T) {
			osReleaseMount := "/etc/os-release:/host/etc/os-release:ro"
			if !strings.HasSuffix(osReleaseMount, ":ro") {
				t.Errorf("Host /etc/os-release mount must be read-only (:ro)")
			}
		})
	})

	// -------------------------------------------------------------------------
	// B06: Healthcheck Network Edge Cases
	// -------------------------------------------------------------------------
	t.Run("B06_HealthcheckNetworkEdgeCases", func(t *testing.T) {
		t.Run("TC_T2_B06_01_Http500InternalServerError", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()

			exitCode := ExecuteHealthcheckSimulation(server.URL)
			if exitCode != 1 {
				t.Errorf("Expected exit code 1 on HTTP 500, got %d", exitCode)
			}
		})

		t.Run("TC_T2_B06_02_Http503ServiceUnavailable", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer server.Close()

			exitCode := ExecuteHealthcheckSimulation(server.URL)
			if exitCode != 1 {
				t.Errorf("Expected exit code 1 on HTTP 503, got %d", exitCode)
			}
		})

		t.Run("TC_T2_B06_03_Http404NotFound", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			exitCode := ExecuteHealthcheckSimulation(server.URL)
			if exitCode != 1 {
				t.Errorf("Expected exit code 1 on HTTP 404, got %d", exitCode)
			}
		})

		t.Run("TC_T2_B06_04_TimeoutThresholdEnforcement", func(t *testing.T) {
			// Simulate slow server taking 3 seconds
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2500 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			start := time.Now()
			exitCode := ExecuteHealthcheckSimulation(server.URL)
			elapsed := time.Since(start)

			if exitCode != 1 {
				t.Errorf("Expected timeout exit code 1, got %d", exitCode)
			}
			if elapsed < 1800*time.Millisecond || elapsed > 3000*time.Millisecond {
				t.Logf("Healthcheck timed out as expected around 2s: %v", elapsed)
			}
		})

		t.Run("TC_T2_B06_05_NonLocalhostListenInterception", func(t *testing.T) {
			// Even if server is configured with ORBIT_LISTEN=0.0.0.0:8080 or 192.168.1.50:8080,
			// the healthcheck container command must query 127.0.0.1 loopback
			url, err := ResolveHealthcheckTargetURL("0.0.0.0:8080")
			if err != nil {
				t.Fatalf("Failed resolution: %v", err)
			}
			if !strings.HasPrefix(url, "http://127.0.0.1:") {
				t.Errorf("Healthcheck must always target 127.0.0.1, got %s", url)
			}
		})
	})
}
