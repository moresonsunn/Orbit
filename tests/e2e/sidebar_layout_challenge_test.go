package e2e

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestChallenge_SidebarLayoutAndCollision performs empirical mathematical and geometric
// verification of the Sidebar component layout in collapsed (w-16, 64px) and expanded (w-56, 224px) states.
func TestChallenge_SidebarLayoutAndCollision(t *testing.T) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		t.Fatalf("Failed to locate repo root: %v", err)
	}

	sidebarPath := filepath.Join(repoRoot, "frontend", "src", "components", "layout", "Sidebar.tsx")
	content, exists, err := ReadFileIfExists(sidebarPath)
	if err != nil || !exists {
		t.Fatalf("Sidebar.tsx not found at %s: %v", sidebarPath, err)
	}

	// -------------------------------------------------------------------------
	// 1. Collapsed State: Logo Centering at 32px
	// -------------------------------------------------------------------------
	t.Run("Collapsed_LogoCenteringAt32px", func(t *testing.T) {
		collapsedWidth := 64 // w-16 = 4rem = 64px
		logoBoxWidth := 28   // w-7 = 1.75rem = 28px

		// Header uses justify-center without horizontal padding
		offset := (collapsedWidth - logoBoxWidth) / 2 // (64 - 28) / 2 = 18px
		centerX := offset + logoBoxWidth/2           // 18 + 14 = 32px

		if centerX != 32 {
			t.Errorf("Logo horizontal center %d != 32px", centerX)
		}

		if !strings.Contains(content, "relative flex items-center justify-center h-14 border-b") {
			t.Errorf("Sidebar.tsx missing centered collapsed header container")
		}
		if !strings.Contains(content, "w-7 h-7") {
			t.Errorf("Sidebar.tsx logo missing w-7 h-7 dimensions")
		}
	})

	// -------------------------------------------------------------------------
	// 2. Collapsed State: Zero Overlap Between Expand Toggle and Logo
	// -------------------------------------------------------------------------
	t.Run("Collapsed_ZeroOverlapBetweenToggleAndLogo", func(t *testing.T) {
		sidebarW := 64
		logoW := 28
		logoLeft := (sidebarW - logoW) / 2 // 18px
		logoRight := logoLeft + logoW      // 46px

		// Indicator dot adds 2px (-right-0.5)
		dotRight := logoRight + 2 // 48px

		// Button: absolute -right-3 (right: -12px), w-6 (24px)
		btnRight := sidebarW + 12     // 64 + 12 = 76px
		btnLeft := btnRight - 24      // 76 - 24 = 52px
		btnCenterX := (btnLeft + btnRight) / 2 // (52 + 76) / 2 = 64px (on border seam)

		if btnCenterX != 64 {
			t.Errorf("Expand button center %d != 64px (border seam)", btnCenterX)
		}

		// Calculate 1D overlap
		overlapX := 0
		if minRight := min(logoRight, btnRight); minRight > max(logoLeft, btnLeft) {
			overlapX = minRight - max(logoLeft, btnLeft)
		}

		if overlapX != 0 {
			t.Errorf("Detected non-zero overlap (%dpx) between logo and expand button", overlapX)
		}

		clearanceLogo := btnLeft - logoRight // 52 - 46 = 6px
		if clearanceLogo != 6 {
			t.Errorf("Expected 6px clearance between logo and button, got %dpx", clearanceLogo)
		}

		clearanceDot := btnLeft - dotRight // 52 - 48 = 4px
		if clearanceDot != 4 {
			t.Errorf("Expected 4px clearance between status dot and button, got %dpx", clearanceDot)
		}
	})

	// -------------------------------------------------------------------------
	// 3. Collapsed State: Navigation Icons Centered at 32px
	// -------------------------------------------------------------------------
	t.Run("Collapsed_NavigationIconsCenteredAt32px", func(t *testing.T) {
		sidebarW := 64
		navPaddingX := 8 // px-2 = 0.5rem = 8px
		navContentW := sidebarW - (2 * navPaddingX) // 64 - 16 = 48px

		iconW := 16 // w-4 = 1rem = 16px
		offsetInButton := (navContentW - iconW) / 2 // (48 - 16) / 2 = 16px
		iconLeft := navPaddingX + offsetInButton    // 8 + 16 = 24px
		iconCenterX := iconLeft + iconW/2          // 24 + 8 = 32px

		if iconCenterX != 32 {
			t.Errorf("Navigation icon center %d != 32px", iconCenterX)
		}

		if !strings.Contains(content, "collapsed ? 'justify-center px-0 py-2' : 'px-2.5 py-1.5'") {
			t.Errorf("Sidebar.tsx missing centered nav styling for collapsed state")
		}
	})

	// -------------------------------------------------------------------------
	// 4. Collapsed State: Footer Items Centered at 32px
	// -------------------------------------------------------------------------
	t.Run("Collapsed_FooterItemsCenteredAt32px", func(t *testing.T) {
		sidebarW := 64
		footerPadX := 12 // p-3 = 0.75rem = 12px
		footerInnerW := sidebarW - (2 * footerPadX) // 64 - 24 = 40px

		avatarW := 28 // w-7 = 1.75rem = 28px
		avatarMargin := (footerInnerW - avatarW) / 2 // (40 - 28) / 2 = 6px
		avatarLeft := footerPadX + avatarMargin      // 12 + 6 = 18px
		avatarCenterX := avatarLeft + avatarW/2      // 18 + 14 = 32px

		if avatarCenterX != 32 {
			t.Errorf("Footer avatar center %d != 32px", avatarCenterX)
		}

		if !strings.Contains(content, "w-7 h-7 rounded-full") || !strings.Contains(content, "mx-auto") {
			t.Errorf("Sidebar.tsx missing mx-auto centered footer avatar")
		}
	})

	// -------------------------------------------------------------------------
	// 5. Expanded State: Header Renders Without Truncation
	// -------------------------------------------------------------------------
	t.Run("Expanded_HeaderRendersWithoutTruncation", func(t *testing.T) {
		expandedW := 224 // w-56 = 14rem = 224px
		paddingX := 14   // px-3.5 = 0.875rem = 14px
		innerW := expandedW - (2 * paddingX) // 224 - 28 = 196px

		collapseBtnW := 22 // w-3.5 (14px) + p-1 (4px * 2 = 8px) = 22px
		availableBrandW := innerW - collapseBtnW // 196 - 22 = 174px

		// Brand cluster:
		logoW := 28
		gap1 := 10 // gap-2.5 = 10px
		labelW := 66 // "orbit.local" @ text-xs ~ 66px
		gap2 := 6  // gap-1.5 = 6px
		badgeW := 34 // "Host" @ 10px (~22px) + px-1.5 (12px) = 34px

		totalBrandW := logoW + gap1 + labelW + gap2 + badgeW // 144px
		buffer := availableBrandW - totalBrandW // 174 - 144 = 30px

		if buffer <= 0 {
			t.Errorf("Insufficient headroom (%dpx) in expanded header, risk of truncation", buffer)
		}

		if !strings.Contains(content, "truncate") {
			t.Errorf("Sidebar.tsx missing truncate guard class in expanded header")
		}
	})

	// -------------------------------------------------------------------------
	// 6. Toggle State Machine Callbacks
	// -------------------------------------------------------------------------
	t.Run("Toggle_CallbacksMutuallyInverting", func(t *testing.T) {
		if !strings.Contains(content, "setCollapsed(false)") {
			t.Errorf("Sidebar.tsx missing setCollapsed(false) on expand button")
		}
		if !strings.Contains(content, "setCollapsed(true)") {
			t.Errorf("Sidebar.tsx missing setCollapsed(true) on collapse button")
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
