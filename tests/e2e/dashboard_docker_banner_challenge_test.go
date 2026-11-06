package e2e

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestEmpiricalChallenge_DashboardDockerWarningBanner executes deep adversarial verification
// of Requirement R2: Dashboard UI Docker connection status warning banner in frontend/src/pages/Dashboard.tsx.
func TestEmpiricalChallenge_DashboardDockerWarningBanner(t *testing.T) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("Prerequisite failed: cannot locate repo root: %v", err)
	}

	dashPath := filepath.Join(repoRoot, "frontend", "src", "pages", "Dashboard.tsx")
	dashContent, dashExists, err := ReadFileIfExists(dashPath)
	if err != nil || !dashExists {
		t.Fatalf("Failed to read Dashboard.tsx: %v", err)
	}

	// -------------------------------------------------------------------------
	// 1. Strict AST / DOM Table Invariant (totalTables == 1)
	// -------------------------------------------------------------------------
	t.Run("AST_TableCountInvariant", func(t *testing.T) {
		tableTagCount := strings.Count(dashContent, "<table")
		if tableTagCount != 1 {
			t.Fatalf("Violation of DOM table invariant: expected exactly 1 <table> tag in Dashboard.tsx, found %d", tableTagCount)
		}

		tableCloseCount := strings.Count(dashContent, "</table>")
		if tableCloseCount != 1 {
			t.Fatalf("Violation of DOM table invariant: expected exactly 1 </table> closing tag in Dashboard.tsx, found %d", tableCloseCount)
		}
	})

	// -------------------------------------------------------------------------
	// 2. Banner Conditional Gating & Edge Case Resilience
	// -------------------------------------------------------------------------
	t.Run("Banner_ConditionalGatingSyntax", func(t *testing.T) {
		// Must gate on sysInfo truthy AND !sysInfo.dockerConnected to guard against undefined / initial loading
		expectedGating := "{sysInfo && !sysInfo.dockerConnected && ("
		if !strings.Contains(dashContent, expectedGating) {
			t.Errorf("Dashboard.tsx does not properly guard warning banner with '%s'", expectedGating)
		}

		// Ensure no reckless unguarded direct accesses to sysInfo.dockerConnected in JSX
		recklessGatingPattern := regexp.MustCompile(`\{\s*!sysInfo\.dockerConnected\s*&&`)
		if recklessGatingPattern.MatchString(dashContent) {
			t.Errorf("Dashboard.tsx contains unguarded '!sysInfo.dockerConnected' which crashes if sysInfo is undefined")
		}
	})

	// -------------------------------------------------------------------------
	// 3. Banner DOM Structure, Styling, and German Localization
	// -------------------------------------------------------------------------
	t.Run("Banner_DOMStructureAndAccessibility", func(t *testing.T) {
		// Icon presence
		if !strings.Contains(dashContent, "<AlertTriangle") {
			t.Errorf("Dashboard.tsx banner does not render AlertTriangle icon")
		}

		// Prominent alert styling tokens
		requiredClasses := []string{
			"bg-rose-950/30",
			"border-rose-800/60",
			"text-rose-200",
		}
		for _, cls := range requiredClasses {
			if !strings.Contains(dashContent, cls) {
				t.Errorf("Dashboard.tsx warning banner missing required styling class: %s", cls)
			}
		}

		// Prominent headline
		expectedHeadline := "Docker Engine nicht erreichbar — Zugriff verweigert oder Dienst inaktiv"
		if !strings.Contains(dashContent, expectedHeadline) {
			t.Errorf("Dashboard.tsx warning banner missing expected headline: %q", expectedHeadline)
		}

		// Status pill badge
		expectedBadge := "Verbindung getrennt"
		if !strings.Contains(dashContent, expectedBadge) {
			t.Errorf("Dashboard.tsx warning banner missing status badge: %q", expectedBadge)
		}

		// Actionable configuration guidance
		expectedRemediations := []string{
			"/var/run/docker.sock",
			`user: "0:0"`,
			`/var/run/docker.sock:/var/run/docker.sock`,
		}
		for _, rem := range expectedRemediations {
			if !strings.Contains(dashContent, rem) {
				t.Errorf("Dashboard.tsx warning banner missing actionable guidance token: %q", rem)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 4. Banner Error Detail Handling (Empty, Populated, Multiline)
	// -------------------------------------------------------------------------
	t.Run("Banner_ErrorDetailMonospaceBlock", func(t *testing.T) {
		// Error container must be conditionally rendered only when dockerError is non-empty
		expectedErrorGating := "{sysInfo.dockerError && ("
		if !strings.Contains(dashContent, expectedErrorGating) {
			t.Errorf("Dashboard.tsx warning banner does not conditionally gate error details with '%s'", expectedErrorGating)
		}

		// Formatting classes for daemon error output
		requiredFormatting := []string{
			"break-all",
			"select-all",
			"font-mono",
			"Daemon-Fehler:",
		}
		for _, fmtToken := range requiredFormatting {
			if !strings.Contains(dashContent, fmtToken) {
				t.Errorf("Dashboard.tsx warning banner error details missing formatting token: %s", fmtToken)
			}
		}
	})

	// -------------------------------------------------------------------------
	// 5. Retry Live Connection Action Mechanism
	// -------------------------------------------------------------------------
	t.Run("Banner_RetryActionMechanism", func(t *testing.T) {
		// State hook for retry in-flight
		if !strings.Contains(dashContent, "const [retryingDocker, setRetryingDocker] = useState(false)") {
			t.Errorf("Dashboard.tsx missing retryingDocker state declaration")
		}

		// handleRetryDockerConnection handler
		if !strings.Contains(dashContent, "const handleRetryDockerConnection = async () =>") {
			t.Errorf("Dashboard.tsx missing handleRetryDockerConnection function")
		}

		// api.getSystemInfo call
		if !strings.Contains(dashContent, "await api.getSystemInfo()") {
			t.Errorf("handleRetryDockerConnection does not call api.getSystemInfo()")
		}

		// Retry button with spinner
		if !strings.Contains(dashContent, "onClick={handleRetryDockerConnection}") {
			t.Errorf("Dashboard.tsx banner missing retry button bound to handleRetryDockerConnection")
		}
		if !strings.Contains(dashContent, "retryingDocker && 'animate-spin'") {
			t.Errorf("Dashboard.tsx retry button missing animate-spin animation binding")
		}
	})

	// -------------------------------------------------------------------------
	// 6. Secondary Metric Card Disconnection Indicator
	// -------------------------------------------------------------------------
	t.Run("MetricCard_DisconnectionIndicator", func(t *testing.T) {
		// In Docker Engine metric card header, check for 'Getrennt' indicator
		metricCardGating := "{sysInfo && !sysInfo.dockerConnected && ("
		if !strings.Contains(dashContent, metricCardGating) {
			t.Errorf("Dashboard.tsx secondary metric card missing guarded disconnection check '%s'", metricCardGating)
		}

		if !strings.Contains(dashContent, "Getrennt") {
			t.Errorf("Dashboard.tsx secondary metric card missing 'Getrennt' indicator badge")
		}
	})

	// -------------------------------------------------------------------------
	// 7. Container Table Disconnected Empty State Adaptation
	// -------------------------------------------------------------------------
	t.Run("ContainerTable_DisconnectedEmptyState", func(t *testing.T) {
		expectedEmptyStateCondition := "sysInfo && !sysInfo.dockerConnected"
		if !strings.Contains(dashContent, expectedEmptyStateCondition) {
			t.Errorf("Dashboard.tsx container list empty state does not check '%s'", expectedEmptyStateCondition)
		}

		expectedEmptyTextDisconnected := "Keine Container verfügbar — Docker-Daemon nicht erreichbar oder Verbindung getrennt."
		if !strings.Contains(dashContent, expectedEmptyTextDisconnected) {
			t.Errorf("Dashboard.tsx container list missing disconnected empty state message: %q", expectedEmptyTextDisconnected)
		}

		expectedEmptyTextConnected := "Keine Container auf diesem Server gefunden."
		if !strings.Contains(dashContent, expectedEmptyTextConnected) {
			t.Errorf("Dashboard.tsx container list missing normal empty state message: %q", expectedEmptyTextConnected)
		}
	})

	// -------------------------------------------------------------------------
	// 8. Warning Banner Placement Independence from User Preferences
	// -------------------------------------------------------------------------
	t.Run("Banner_PlacementIndependentOfWidgetPrefs", func(t *testing.T) {
		bannerIdx := strings.Index(dashContent, "{sysInfo && !sysInfo.dockerConnected && (")
		if bannerIdx == -1 {
			t.Fatalf("Could not locate warning banner in Dashboard.tsx")
		}

		summaryIdx := strings.Index(dashContent, "{prefs.showSummary && (")
		if summaryIdx == -1 {
			t.Fatalf("Could not locate {prefs.showSummary && ( in Dashboard.tsx")
		}

		containersIdx := strings.Index(dashContent, "{prefs.showContainers && (")
		if containersIdx == -1 {
			t.Fatalf("Could not locate {prefs.showContainers && ( in Dashboard.tsx")
		}

		// The warning banner must appear BEFORE widget-specific preference toggles
		// so an operator who hid summary or charts still receives critical daemon connection warnings.
		if bannerIdx > summaryIdx {
			t.Errorf("Dashboard warning banner is nested inside or after prefs.showSummary, compromising operator visibility")
		}
		if bannerIdx > containersIdx {
			t.Errorf("Dashboard warning banner is nested after prefs.showContainers, compromising operator visibility")
		}
	})
}
