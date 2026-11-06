package e2e

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEmpiricalChallenge_DashboardCustomizeModalAndGating runs deep empirical validation
// of Dashboard.tsx and dashboardConfig.ts for milestone M1 review.
func TestEmpiricalChallenge_DashboardCustomizeModalAndGating(t *testing.T) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("Prerequisite failed: cannot locate repo root: %v", err)
	}

	dashPath := filepath.Join(repoRoot, "frontend", "src", "pages", "Dashboard.tsx")
	dashContent, dashExists, err := ReadFileIfExists(dashPath)
	if err != nil || !dashExists {
		t.Fatalf("Failed to read Dashboard.tsx: %v", err)
	}

	configPath := filepath.Join(repoRoot, "frontend", "src", "lib", "dashboardConfig.ts")
	configContent, configExists, err := ReadFileIfExists(configPath)
	if err != nil || !configExists {
		t.Fatalf("Failed to read dashboardConfig.ts: %v", err)
	}

	// -------------------------------------------------------------------------
	// 1. Preset Configuration Integrity
	// -------------------------------------------------------------------------
	t.Run("PresetConfigurationIntegrity", func(t *testing.T) {
		// Verify minimal preset
		if !strings.Contains(configContent, "minimal: {") {
			t.Errorf("dashboardConfig.ts missing 'minimal' preset definition")
		}
		// Verify balanced preset
		if !strings.Contains(configContent, "balanced: {") {
			t.Errorf("dashboardConfig.ts missing 'balanced' preset definition")
		}
		// Verify pro preset
		if !strings.Contains(configContent, "pro: {") {
			t.Errorf("dashboardConfig.ts missing 'pro' preset definition")
		}

		// Verify expected fields for each preset
		requiredFields := []string{
			"preset",
			"showSummary",
			"showDockerOverview",
			"showThermals",
			"showLoadAvg",
			"showGPU",
			"showCharts",
			"showContainers",
			"compactMode",
		}

		for _, f := range requiredFields {
			if !strings.Contains(configContent, f+":") {
				t.Errorf("dashboardConfig.ts missing field: %s", f)
			}
		}

		// Verify minimal configuration specifics
		minBlockRegex := regexp.MustCompile(`minimal:\s*\{([^}]+)\}`)
		minMatch := minBlockRegex.FindStringSubmatch(configContent)
		if len(minMatch) < 2 {
			t.Fatalf("Could not extract minimal config block")
		}
		minBody := minMatch[1]
		if !strings.Contains(minBody, "showDockerOverview: false") {
			t.Errorf("minimal preset expected showDockerOverview: false")
		}
		if !strings.Contains(minBody, "showCharts: false") {
			t.Errorf("minimal preset expected showCharts: false")
		}
		if !strings.Contains(minBody, "compactMode: true") {
			t.Errorf("minimal preset expected compactMode: true")
		}
		if !strings.Contains(minBody, "showContainers: true") {
			t.Errorf("minimal preset expected showContainers: true")
		}

		// Verify pro configuration specifics
		proBlockRegex := regexp.MustCompile(`pro:\s*\{([^}]+)\}`)
		proMatch := proBlockRegex.FindStringSubmatch(configContent)
		if len(proMatch) < 2 {
			t.Fatalf("Could not extract pro config block")
		}
		proBody := proMatch[1]
		if !strings.Contains(proBody, "showGPU: true") {
			t.Errorf("pro preset expected showGPU: true")
		}
		if !strings.Contains(proBody, "showCharts: true") {
			t.Errorf("pro preset expected showCharts: true")
		}
		if !strings.Contains(proBody, "compactMode: false") {
			t.Errorf("pro preset expected compactMode: false")
		}
	})

	// -------------------------------------------------------------------------
	// 2. Customize Modal Dialog Rendering & Controls
	// -------------------------------------------------------------------------
	t.Run("CustomizeModalRenderingAndControls", func(t *testing.T) {
		// State hook declaration
		if !strings.Contains(dashContent, "const [showConfigModal, setShowConfigModal] = useState(false)") {
			t.Errorf("Dashboard.tsx does not initialize showConfigModal to false")
		}

		// Open trigger button
		if !strings.Contains(dashContent, "setShowConfigModal(true)") {
			t.Errorf("Dashboard.tsx missing trigger calling setShowConfigModal(true)")
		}

		// Modal conditional gating in JSX
		if !strings.Contains(dashContent, "{showConfigModal && (") {
			t.Errorf("Dashboard.tsx modal not conditionally gated by '{showConfigModal && ('")
		}

		// Close handlers (X button and Fertig button)
		closeCount := strings.Count(dashContent, "setShowConfigModal(false)")
		if closeCount < 2 {
			t.Errorf("Dashboard.tsx expected at least 2 close triggers calling setShowConfigModal(false), got %d", closeCount)
		}

		// Backdrop blur and styling
		if !strings.Contains(dashContent, "fixed inset-0 z-50 flex items-center justify-center") {
			t.Errorf("Modal missing standard overlay position classes")
		}
		if !strings.Contains(dashContent, "backdrop-blur-sm") {
			t.Errorf("Modal missing backdrop-blur-sm class")
		}
		if !strings.Contains(dashContent, "Dashboard anpassen") {
			t.Errorf("Modal missing German title 'Dashboard anpassen'")
		}
	})

	// -------------------------------------------------------------------------
	// 3. Widget Toggle Reactivity & LocalStorage
	// -------------------------------------------------------------------------
	t.Run("WidgetToggleReactivityAndLocalStorage", func(t *testing.T) {
		// Verify updatePref function implementation
		if !strings.Contains(dashContent, "const updatePref =") {
			t.Errorf("Dashboard.tsx missing updatePref implementation")
		}
		if !strings.Contains(dashContent, "saveDashboardPrefs(next)") {
			t.Errorf("Dashboard.tsx updatePref or setPreset does not call saveDashboardPrefs")
		}

		// Verify 8 widget toggle items in modal
		widgetKeys := []string{
			"showSummary",
			"showDockerOverview",
			"showLoadAvg",
			"showThermals",
			"showGPU",
			"showCharts",
			"showContainers",
			"compactMode",
		}
		for _, wk := range widgetKeys {
			expectedKeyStr := "key: '" + wk + "'"
			if !strings.Contains(dashContent, expectedKeyStr) {
				t.Errorf("Dashboard.tsx modal missing widget toggle entry for '%s'", wk)
			}
		}

		// Verify localStorage key in dashboardConfig.ts
		if !strings.Contains(configContent, "'orbit_dashboard_prefs'") {
			t.Errorf("dashboardConfig.ts does not use 'orbit_dashboard_prefs' storage key")
		}
		// Verify try-catch error handling around localStorage
		if !strings.Contains(configContent, "try {") || !strings.Contains(configContent, "catch (err)") {
			t.Errorf("dashboardConfig.ts missing try-catch error boundary for localStorage")
		}
	})

	// -------------------------------------------------------------------------
	// 4. Active Containers DOM Gating
	// -------------------------------------------------------------------------
	t.Run("ActiveContainersDOMGating", func(t *testing.T) {
		// Check that prefs.showContainers gates the container table
		if !strings.Contains(dashContent, "{prefs.showContainers && (") {
			t.Errorf("Dashboard.tsx does not gate active containers table with '{prefs.showContainers && ('")
		}

		// Verify container table tag is inside this block
		blockIdx := strings.Index(dashContent, "{prefs.showContainers && (")
		if blockIdx == -1 {
			t.Fatalf("Could not locate {prefs.showContainers && ( block")
		}
		subAfter := dashContent[blockIdx:]
		tableIdx := strings.Index(subAfter, "<table")
		if tableIdx == -1 || tableIdx > 600 {
			t.Errorf("Container table not found directly inside prefs.showContainers gating block")
		}

		// Ensure there are no other tables outside this conditional block
		totalTables := strings.Count(dashContent, "<table")
		if totalTables != 1 {
			t.Errorf("Expected exactly 1 <table> in Dashboard.tsx, found %d", totalTables)
		}
	})
}
