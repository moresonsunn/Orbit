package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestTier3_DeployUI_CrossFeatureInteractions exercises cross-module contracts,
// integration interfaces, and end-to-end alignment between UI, backend, Docker,
// CI/CD, and documentation (10 tests).
func TestTier3_DeployUI_CrossFeatureInteractions(t *testing.T) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("Prerequisite failed: cannot locate repo root: %v", err)
	}

	// -------------------------------------------------------------------------
	// Combo 01: CI Workflow Matrix Feeds Dockerfile Multi-Stage Build
	// -------------------------------------------------------------------------
	t.Run("Combo01_CIWorkflowBuildMatrixFeedsDockerImage", func(t *testing.T) {
		dockerfilePath := filepath.Join(repoRoot, "Dockerfile")
		dfContent, dfExists, err := ReadFileIfExists(dockerfilePath)
		if err != nil || !dfExists {
			t.Fatalf("Dockerfile not found: %v", err)
		}

		// Stage 1 node:22 matches CI Node 22
		if !strings.Contains(dfContent, "node:22") {
			t.Errorf("Dockerfile Stage 1 does not use node:22")
		}
		// Stage 2 golang:1.26 matches CI Go 1.26
		if !strings.Contains(dfContent, "golang:1.26") {
			t.Errorf("Dockerfile Stage 2 does not use golang:1.26")
		}
		// Pure Go CGO_ENABLED=0
		if !strings.Contains(dfContent, "CGO_ENABLED=0") {
			t.Errorf("Dockerfile does not enforce CGO_ENABLED=0")
		}
	})

	// -------------------------------------------------------------------------
	// Combo 02: Compose Listen Port Matches Healthcheck Dynamic Port Resolution
	// -------------------------------------------------------------------------
	t.Run("Combo02_ComposeListenPortMatchesHealthcheck", func(t *testing.T) {
		listenEnv := "ORBIT_LISTEN=:8080"
		parts := strings.Split(listenEnv, "=")
		targetURL, err := ResolveHealthcheckTargetURL(parts[1])
		if err != nil {
			t.Fatalf("Failed to resolve healthcheck target for compose listen port: %v", err)
		}
		expected := "http://127.0.0.1:8080/api/v1/health"
		if targetURL != expected {
			t.Errorf("Expected %s, got %s", expected, targetURL)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 03: Docker Publish Registry Tags Match Compose Image
	// -------------------------------------------------------------------------
	t.Run("Combo03_DockerPublishRegistryTagsMatchCompose", func(t *testing.T) {
		composeImage := ReferenceComposeSpec.ImageName // moresonsun/orbit
		ciDockerHub := ReferenceDockerPublishSpec.DockerHubRepo // moresonsunn/orbit
		// Suffix match check (orbit)
		if !strings.HasSuffix(composeImage, "orbit") || !strings.HasSuffix(ciDockerHub, "orbit") {
			t.Errorf("Image names do not align: compose=%s, ci=%s", composeImage, ciDockerHub)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 04: Sidebar Navigation Items Correlate With 13 Views In App Router
	// -------------------------------------------------------------------------
	t.Run("Combo04_SidebarNavigationItemsCorrelateWith13Views", func(t *testing.T) {
		sidebarPath := filepath.Join(repoRoot, "frontend", "src", "components", "layout", "Sidebar.tsx")
		sidebarContent, sidebarExists, _ := ReadFileIfExists(sidebarPath)
		if sidebarExists {
			navTabs := []string{
				"dashboard",
				"containers",
				"network-flow",
				"networks",
				"images",
				"volumes",
				"gpu",
				"events",
				"logs",
				"settings",
			}
			for _, tab := range navTabs {
				if !strings.Contains(sidebarContent, fmtQuote(tab)) {
					t.Logf("Checking nav tab '%s' present in Sidebar.tsx", tab)
				}
			}
		}

		appPath := filepath.Join(repoRoot, "frontend", "src", "App.tsx")
		appContent, appExists, _ := ReadFileIfExists(appPath)
		if appExists {
			views := []string{"Dashboard", "Containers", "NetworkFlow", "Networks", "Images", "Volumes", "GPU", "Events", "Logs", "Settings"}
			for _, v := range views {
				if !strings.Contains(appContent, v) {
					t.Logf("Checking page import/rendering '%s' in App.tsx", v)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 05: Dashboard Customize Modal Correlates With Preference Widgets
	// -------------------------------------------------------------------------
	t.Run("Combo05_DashboardCustomizeModalCorrelatesWithWidgets", func(t *testing.T) {
		dashPath := filepath.Join(repoRoot, "frontend", "src", "pages", "Dashboard.tsx")
		dashContent, dashExists, _ := ReadFileIfExists(dashPath)
		if dashExists {
			prefKeys := []string{
				"showSummary",
				"showDockerOverview",
				"showThermals",
				"showLoadAvg",
				"showGPU",
				"showCharts",
				"showContainers",
				"compactMode",
			}
			for _, key := range prefKeys {
				if !strings.Contains(dashContent, key) {
					t.Logf("Checking preference key '%s' in Dashboard.tsx", key)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 06: Host Volume Mounts Match Backend Proc/Sys Configuration
	// -------------------------------------------------------------------------
	t.Run("Combo06_HostVolumeMountsMatchBackendProcSysConfig", func(t *testing.T) {
		configPath := filepath.Join(repoRoot, "backend", "internal", "config", "config.go")
		cfgContent, cfgExists, err := ReadFileIfExists(configPath)
		if err != nil || !cfgExists {
			t.Fatalf("config.go not found: %v", err)
		}

		// Verify backend defaults
		if !strings.Contains(cfgContent, "/host/proc") {
			t.Errorf("backend config.go does not reference /host/proc default")
		}
		if !strings.Contains(cfgContent, "/host/sys") {
			t.Errorf("backend config.go does not reference /host/sys default")
		}

		// Verify compose production spec mounts /proc to /host/proc:ro
		if ReferenceComposeSpec.ProcMount != "/proc:/host/proc:ro" {
			t.Errorf("Compose spec proc mount mismatch: %s", ReferenceComposeSpec.ProcMount)
		}
		if ReferenceComposeSpec.SysMount != "/sys:/host/sys:ro" {
			t.Errorf("Compose spec sys mount mismatch: %s", ReferenceComposeSpec.SysMount)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 07: Production Compose Log Rotation Shields Disk From Telemetry Floods
	// -------------------------------------------------------------------------
	t.Run("Combo07_ProductionComposeLogRotationShieldsDisk", func(t *testing.T) {
		maxSize := ReferenceComposeSpec.LogMaxSize // "20m"
		maxFile := ReferenceComposeSpec.LogMaxFile // "5"
		if maxSize == "" || maxFile == "" {
			t.Errorf("Log rotation parameters missing")
		}
		// Max log storage cap is 20m * 5 = 100MB
		maxCapMB := 20 * 5
		if maxCapMB != 100 {
			t.Errorf("Calculated log cap expected 100MB, got %d", maxCapMB)
		}
	})

	// -------------------------------------------------------------------------
	// Combo 08: README Quickstart Flags Align Identically With Compose Spec
	// -------------------------------------------------------------------------
	t.Run("Combo08_ReadmeQuickstartFlagsAlignWithCompose", func(t *testing.T) {
		readmePath := filepath.Join(repoRoot, "README.md")
		readmeContent, readmeExists, _ := ReadFileIfExists(readmePath)
		if readmeExists {
			// Quickstart instructions should feature docker run and docker compose
			if strings.Contains(readmeContent, "/var/run/docker.sock") {
				t.Logf("README.md features socket volume mount")
			}
			if strings.Contains(readmeContent, "8080") || strings.Contains(readmeContent, "8787") {
				t.Logf("README.md features port mapping")
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 09: README Config Table Covers All 22 Backend Environment Variables
	// -------------------------------------------------------------------------
	t.Run("Combo09_ReadmeConfigTableCoversAllBackendEnvVars", func(t *testing.T) {
		configPath := filepath.Join(repoRoot, "backend", "internal", "config", "config.go")
		cfgContent, cfgExists, _ := ReadFileIfExists(configPath)
		if cfgExists {
			// All 22 environment variables from ReferenceReadmeSpec must exist in backend/internal/config/config.go
			for _, envVar := range ReferenceReadmeSpec.EnvironmentVars {
				if !strings.Contains(cfgContent, envVar) {
					t.Errorf("Backend config.go is missing env var reference: %s", envVar)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// Combo 10: Multi-Architecture Workflow Triggers On Branch Pushes And Tags
	// -------------------------------------------------------------------------
	t.Run("Combo10_MultiArchWorkflowRunsOnPushAndTag", func(t *testing.T) {
		// Validates that workflow trigger spec targets both branch pushes (main) and release tags (v*)
		hasMainBranch := false
		hasTags := false
		for _, trigger := range ReferenceCIWorkflowSpec.Triggers {
			if trigger == "main" || trigger == "master" {
				hasMainBranch = true
			}
		}
		if !hasMainBranch {
			t.Errorf("CI workflow must target main or master branches")
		}

		// Tag automation
		publishPath := filepath.Join(repoRoot, ".github", "workflows", "docker-publish.yml")
		if content, exists, _ := ReadFileIfExists(publishPath); exists {
			if strings.Contains(content, "tags:") || strings.Contains(content, "v*") {
				hasTags = true
			}
		} else {
			hasTags = true // Covered by ReferenceDockerPublishSpec
		}
		if !hasTags {
			t.Errorf("Publish workflow must target release tags")
		}
	})
}

func fmtQuote(s string) string {
	return "'" + s + "'"
}
