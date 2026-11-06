# Handoff Report: Comprehensive UI & Browser Audit

**Agent:** `explorer_survey_ui`  
**Date:** 2026-11-06  
**Status:** Completed  
**Subject:** Sidebar Collapsed Header Fix & Comprehensive 13-View UI Audit for Orbit Production Server Deployment  

---

## 1. Observation

### 1.1 Sidebar Collapsed Header Layout Collision (`frontend/src/components/layout/Sidebar.tsx`)
Direct inspection of `frontend/src/components/layout/Sidebar.tsx` reveals:
- **Container Sizing (lines 56–59):**
  ```tsx
  <aside className={cn(
    'relative flex flex-col h-screen bg-[#141414] border-r border-[#222222] transition-all duration-300 z-30 select-none font-sans',
    collapsed ? 'w-16' : 'w-56'
  )}>
  ```
  In collapsed state, the sidebar width is `w-16` = 4rem = 64px.
- **Brand Header Structure (lines 61–90):**
  ```tsx
  <div className="flex items-center justify-between h-14 px-3.5 border-b border-[#222222]">
    <div className="flex items-center gap-2.5 overflow-hidden cursor-pointer" onClick={() => setCurrentTab('dashboard')}>
      <div className="relative flex items-center justify-center w-7 h-7 rounded-md bg-[#222222] text-neutral-300 font-bold text-xs shrink-0 border border-[#2e2e2e]">
        <span className="tracking-tighter">O</span>
        <span
          className={cn(
            'absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full border border-[#141414]',
            dockerConnected ? 'bg-emerald-500' : 'bg-rose-500 animate-pulse'
          )}
          title={dockerConnected ? 'Docker Daemon verbunden' : 'Docker Daemon getrennt'}
        />
      </div>
      {!collapsed && (
        <div className="flex items-center gap-1.5 truncate">
          <span className="font-semibold text-xs text-neutral-200 truncate">orbit.local</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-[#222222] text-neutral-400 font-normal">
            Host
          </span>
        </div>
      )}
    </div>

    <button
      onClick={() => setCollapsed(!collapsed)}
      className="p-1 rounded text-neutral-500 hover:text-white hover:bg-white/[0.05] transition-colors"
      title={collapsed ? 'Sidebar ausklappen' : 'Sidebar einklappen'}
    >
      {collapsed ? <ChevronRight className="w-3.5 h-3.5" /> : <ChevronLeft className="w-3.5 h-3.5" />}
    </button>
  </div>
  ```
- **Dimensional Calculation & Collision:**
  - Header padding `px-3.5` = 14px left + 14px right = 28px padding.
  - Available width inside header = `64px - 28px = 36px`.
  - Logo box: `w-7` = 28px wide.
  - Chevron toggle button: `w-3.5` (14px) + `p-1` (8px horizontal) = 22px wide.
  - Combined required width = `28px + 22px = 50px`.
  - Available width (36px) is **14px smaller** than the required 50px.
  - Consequence: Under `justify-between`, the logo is pinned against the left padding (at x = 14px) and the chevron button is pinned against the right padding (at x = 42px). The chevron physically overlaps or visually collides with the right edge of the logo box (which extends to x = 14 + 28 = 42px).
- **Vertical Alignment Inconsistency Across Sidebar:**
  - In collapsed state (64px width, horizontal center at x = 32px):
    1. Header Logo: Starts at x = 14px, center at x = 28px (**Off-center by -4px**).
    2. Nav Icons: Container has `px-2` (8px), buttons have `px-2.5` (10px). The `w-4` (16px) icon starts at x = 18px, center at x = 26px (**Off-center by -6px**).
    3. Footer Avatar (lines 141–145): Uses `w-7 h-7 mx-auto`, starts at x = 18px, center at x = 32px (**True center**).
  - Result: The header logo, nav icons, and user avatar are all staggered at different horizontal positions down the collapsed sidebar.

---

### 1.2 View-by-View Audit Findings Across All 13 Views

| View # | View Name | Route / Component Path | Status | Key Observations & Defects |
|---|---|---|---|---|
| 1 | **Dashboard** | `/` &bull; `frontend/src/pages/Dashboard.tsx` | **DEFECT** | **1. Unrendered Customize Modal:** Line 18 imports `SlidersHorizontal, X, Check, Eye, EyeOff`, line 59 declares `showConfigModal`, and line 166 has the "Anpassen" button `onClick={() => setShowConfigModal(true)}`, but **no modal dialog JSX is returned** in the component tree. Clicking "Anpassen" does nothing.<br>**2. Unused `prefs.showContainers`:** Line 512 renders Active Containers table unconditionally, ignoring `prefs.showContainers`. |
| 2 | **Containers** | `/containers` &bull; `frontend/src/pages/Containers.tsx` | Clean (Minor Polish) | Clean flat list vs. Compose Stacks toggle, bulk actions, and search. Minor localization inconsistency in line 144 (`placeholder="Search by name, image, or ID..."` English in German UI) and line 628 (`${activeModal?.action} werden?`). |
| 3 | **Container Detail** | `/containers/:id` &bull; `frontend/src/pages/ContainerDetail.tsx` | Clean (Minor Polish) | 6 tabs (Overview, Performance, Network & Ports, Logs, Compose Export, Inspect). Clean responsive charts and pre-formatted code. Line 233 tab container lacks `overflow-x-auto`, which could cause wrapping on narrow laptop screens. |
| 4 | **Network Flow** | `/network-flow` &bull; `frontend/src/pages/NetworkFlow.tsx` | Clean (Minor Polish) | Interactive `@xyflow/react` graph with custom Tailscale dark styling in `index.css`. Donut chart, KPI cards, paginated socket table. Minor CSS typos in lines 1140 & 1145: `py-0.2` on LAN/WAN badges (non-standard Tailwind token). |
| 5 | **Networks** | `/networks` &bull; `frontend/src/pages/Networks.tsx` | Clean (Minor Polish) | UniFi-style IPAM subnet inspector, allocation pool progress bar, connected container endpoint table. Minor CSS typo in line 346: `py-0.2` on filter count badge (same non-standard token). |
| 6 | **Images** | `/images` &bull; `frontend/src/pages/Images.tsx` | Clean (Minor Polish) | Image repository list, tags, size, age, container counts, prune and delete modals. Long repository tags in line 103 could benefit from `truncate max-w-[280px]` with `title` tooltip to avoid wide table blowout. |
| 7 | **Volumes** | `/volumes` &bull; `frontend/src/pages/Volumes.tsx` | **Clean** | Persistent volume table with name, driver, mountpoint, size, creation date, delete and prune modals. Proper `truncate max-w-[200px]` and `max-w-[260px]` applied. |
| 8 | **GPU Monitoring** | `/gpu` &bull; `frontend/src/pages/GPU.tsx` | Clean (Minor Polish) | NVML / DRM telemetry cards, VRAM, temp, power draw, process attribution table. Line 36 uses `text-slate-400` instead of `text-neutral-400`. |
| 9 | **Events** | `/events` &bull; `frontend/src/pages/Events.tsx` | **Clean** | Real-time event stream with severity filter (All, Warning, Critical), timestamp formatting, truncated actor IDs. Fully responsive. |
| 10 | **Logs** | `/logs` &bull; `frontend/src/pages/Logs.tsx` | Clean (Minor Polish) | Centralized container log stream with line tail selection, search filter, and text download. Container selector `<select>` in line 72 would benefit from `max-w-[180px] sm:max-w-xs truncate` to prevent overflow on mobile. |
| 11 | **Settings** | `/settings` &bull; `frontend/src/pages/Settings.tsx` | Clean (Minor Polish) | Feature flag overview, sampling interval configuration, alert thresholds, SQLite retention. Line 53 uses `text-slate-400` instead of `text-neutral-400`. |
| 12 | **Login Modal** | `/login` &bull; `frontend/src/components/auth/LoginModal.tsx` | Clean (Minor Polish) | Centered auth modal with backdrop. Line 35 uses `backdrop-blur-xs` (standard is `backdrop-blur-sm`). |
| 13 | **Setup Modal** | `/setup` &bull; `frontend/src/components/auth/SetupModal.tsx` | Clean (Minor Polish) | First-run setup modal with token, password confirmation, and length validation. Line 48 uses `backdrop-blur-xs` (standard is `backdrop-blur-sm`). |

---

## 2. Logic Chain

### 2.1 Why the Sidebar Header Collides & How to Correct It
1. **Premise:** In collapsed state, the sidebar is 64px wide.
2. **Observation:** The header contains both a 28px logo box and a 22px chevron toggle button placed side-by-side inside a container with 28px horizontal padding.
3. **Deduction:** `28px + 22px = 50px`, which exceeds `64px - 28px = 36px` by 14px. Physical collision and squishing is inevitable in a horizontal flex layout.
4. **Premise 2:** In collapsed state, the brand logo must be horizontally centered on the sidebar's centerline (x = 32px).
5. **Deduction 2:** If the logo is centered at x = 32px, its left edge is at x = 18px and its right edge is at x = 46px. The remaining space to the right border of the sidebar is `64px - 46px = 18px`.
6. **Inference:** A 22px button physically cannot fit inside the 18px right margin without overlapping the logo.
7. **Resolution:** The expand toggle button in collapsed state must be positioned on the border seam (floating edge handle: `absolute -right-3 top-1/2 -translate-y-1/2 w-6 h-6 rounded-full bg-[#1c1c1c] border border-[#2e2e2e] shadow-md flex items-center justify-center hover:bg-[#282828] text-neutral-400 hover:text-white z-40`). This eliminates all collision, leaves the logo centered at 32px, and matches modern IDE/dashboard interaction patterns (Linear, VS Code, Supabase).
8. **Premise 3:** In collapsed state, all interactive elements should align on the 32px vertical axis.
9. **Resolution:** Setting `collapsed ? 'justify-center px-0' : 'px-2.5'` on navigation buttons centers the 16px nav icons in the 48px button (`(48 - 16)/2 = 16px` offset, `8 + 16 = 24px` to `40px`, center = 32px). This creates vertical alignment between Header Logo (center 32px), Nav Icons (center 32px), and Footer Avatar (center 32px).

### 2.2 Why the Dashboard "Anpassen" (Customize) Button is Broken
1. **Observation:** In `Dashboard.tsx`, clicking "Anpassen" executes `setShowConfigModal(true)`.
2. **Observation:** An inspection of all occurrences of `showConfigModal` in `Dashboard.tsx` shows only two lines:
   - Line 59: `const [showConfigModal, setShowConfigModal] = useState(false);`
   - Line 166: `onClick={() => setShowConfigModal(true)}`
3. **Deduction:** There is zero JSX rendering `showConfigModal`. The state updates, but nothing renders to screen.
4. **Resolution:** Add a customize modal dialog rendering `prefs` toggles (`showSummary`, `showDockerOverview`, `showThermals`, `showLoadAvg`, `showGPU`, `showCharts`, `showContainers`, `compactMode`) and preset buttons (`minimal`, `balanced`, `pro`) with `saveDashboardPrefs`.

---

## 3. Caveats

- **Mode Constraint:** This investigation was conducted in read-only mode; source files were not modified during this survey.
- **Backend API Telemetry:** The UI components have been audited via source code analysis and Vite production compilation (`npm --prefix frontend run build`). Live WebSocket real-time feeds depend on a running Go backend and Docker daemon socket.
- **Tailwind CSS v4 Engine:** The project uses `@tailwindcss/vite` (Tailwind CSS v4.3.4) with `@theme` block in `src/index.css`. Arbitrary CSS properties (e.g., `lg:grid-cols-16`) compile dynamically in v4, but token typos like `py-0.2` should be standardized to `py-0.5`.

---

## 4. Conclusion & Concrete Recommendations

### 4.1 Recommended Fix: `frontend/src/components/layout/Sidebar.tsx`

Replace the header rendering and nav item styling with:

```tsx
// 1. In aside element: allow smooth transition and edge button overflow
<aside
  className={cn(
    'relative flex flex-col h-screen bg-[#141414] border-r border-[#222222] transition-all duration-300 z-30 select-none font-sans',
    collapsed ? 'w-16' : 'w-56'
  )}
>
  {/* Brand Header */}
  {collapsed ? (
    <div className="relative flex items-center justify-center h-14 border-b border-[#222222]">
      {/* Centered Logo */}
      <div
        onClick={() => setCurrentTab('dashboard')}
        className="relative flex items-center justify-center w-7 h-7 rounded-md bg-[#222222] text-neutral-300 font-bold text-xs cursor-pointer border border-[#2e2e2e] hover:border-neutral-500 transition-colors"
        title="orbit.local (Dashboard)"
      >
        <span className="tracking-tighter">O</span>
        <span
          className={cn(
            'absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full border border-[#141414]',
            dockerConnected ? 'bg-emerald-500' : 'bg-rose-500 animate-pulse'
          )}
          title={dockerConnected ? 'Docker Daemon verbunden' : 'Docker Daemon getrennt'}
        />
      </div>

      {/* Floating Edge Expand Handle (Zero collision, perfectly positioned on border) */}
      <button
        onClick={() => setCollapsed(false)}
        className="absolute -right-3 top-1/2 -translate-y-1/2 w-6 h-6 rounded-full bg-[#1c1c1c] border border-[#2e2e2e] shadow-md flex items-center justify-center text-neutral-400 hover:text-white hover:bg-[#282828] transition-all z-40"
        title="Sidebar ausklappen"
      >
        <ChevronRight className="w-3.5 h-3.5" />
      </button>
    </div>
  ) : (
    <div className="flex items-center justify-between h-14 px-3.5 border-b border-[#222222]">
      <div
        className="flex items-center gap-2.5 overflow-hidden cursor-pointer"
        onClick={() => setCurrentTab('dashboard')}
      >
        <div className="relative flex items-center justify-center w-7 h-7 rounded-md bg-[#222222] text-neutral-300 font-bold text-xs shrink-0 border border-[#2e2e2e]">
          <span className="tracking-tighter">O</span>
          <span
            className={cn(
              'absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full border border-[#141414]',
              dockerConnected ? 'bg-emerald-500' : 'bg-rose-500 animate-pulse'
            )}
            title={dockerConnected ? 'Docker Daemon verbunden' : 'Docker Daemon getrennt'}
          />
        </div>
        <div className="flex items-center gap-1.5 truncate">
          <span className="font-semibold text-xs text-neutral-200 truncate">orbit.local</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-[#222222] text-neutral-400 font-normal">
            Host
          </span>
        </div>
      </div>

      <button
        onClick={() => setCollapsed(true)}
        className="p-1 rounded text-neutral-500 hover:text-white hover:bg-white/[0.05] transition-colors"
        title="Sidebar einklappen"
      >
        <ChevronLeft className="w-3.5 h-3.5" />
      </button>
    </div>
  )}

  {/* Navigation Links - Perfect vertical centering in collapsed state */}
  <nav className="flex-1 py-3 px-2 space-y-0.5 overflow-y-auto">
    {navItems.map((item) => {
      const Icon = item.icon;
      const isActive = currentTab === item.id;
      return (
        <button
          key={item.id}
          onClick={() => setCurrentTab(item.id)}
          className={cn(
            'flex items-center w-full gap-2.5 rounded-md text-[13px] transition-colors group',
            collapsed ? 'justify-center px-0 py-2' : 'px-2.5 py-1.5',
            isActive
              ? 'bg-white/[0.08] text-white font-medium'
              : 'text-neutral-400 hover:text-neutral-200 hover:bg-white/[0.03]'
          )}
          title={collapsed ? item.label : undefined}
        >
          <Icon
            className={cn(
              'w-4 h-4 shrink-0 transition-colors',
              isActive ? 'text-white' : 'text-neutral-500 group-hover:text-neutral-300'
            )}
          />
          {!collapsed && <span className="truncate">{item.label}</span>}
        </button>
      );
    })}
  </nav>
```

---

### 4.2 Recommended Fix: `frontend/src/pages/Dashboard.tsx` (Customize Modal)

1. Add the missing Customize Modal JSX right before the closing `</div>` of `Dashboard.tsx`:
```tsx
      {/* Dashboard Configuration Modal */}
      {showConfigModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm font-sans animate-in fade-in duration-150">
          <div className="relative w-full max-w-md bg-[#181818] border border-[#282828] rounded-xl shadow-2xl p-6 space-y-5">
            <div className="flex items-center justify-between pb-3 border-b border-[#242424]">
              <div className="flex items-center gap-2">
                <SlidersHorizontal className="w-4 h-4 text-blue-400" />
                <h3 className="text-sm font-semibold text-white">Dashboard anpassen</h3>
              </div>
              <button
                onClick={() => setShowConfigModal(false)}
                className="p-1 rounded text-neutral-500 hover:text-white hover:bg-white/[0.05] transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Presets */}
            <div>
              <label className="text-xs font-medium text-neutral-400 block mb-2">Vorkonfigurierte Profile</label>
              <div className="grid grid-cols-3 gap-2">
                {(['minimal', 'balanced', 'pro'] as const).map((p) => (
                  <button
                    key={p}
                    onClick={() => setPreset(p)}
                    className={cn(
                      'px-2.5 py-1.5 rounded-md text-xs font-medium border text-center transition-colors',
                      prefs.preset === p
                        ? 'bg-blue-600 border-blue-500 text-white'
                        : 'bg-[#121212] border-[#282828] text-neutral-400 hover:text-white hover:border-[#383838]'
                    )}
                  >
                    {p === 'minimal' ? 'Kompakt' : p === 'balanced' ? 'Standard' : 'Pro Sysadmin'}
                  </button>
                ))}
              </div>
            </div>

            {/* Widget Toggles */}
            <div className="space-y-2.5 pt-2 border-t border-[#242424]">
              <label className="text-xs font-medium text-neutral-400 block mb-1">Angezeigte Widgets</label>
              {[
                { key: 'showSummary', label: 'Host-Kernmetriken (CPU, RAM, Netz, Disk)' },
                { key: 'showDockerOverview', label: 'Docker Engine Statusleiste' },
                { key: 'showLoadAvg', label: 'Load Average (1m, 5m, 15m)' },
                { key: 'showThermals', label: 'Hardware-Temperaturen (Sensoren)' },
                { key: 'showGPU', label: 'GPU-Telemetrie Vorschau' },
                { key: 'showCharts', label: 'Echtzeit-Verlaufsdiagramme' },
                { key: 'showContainers', label: 'Aktive Container-Tabelle' },
                { key: 'compactMode', label: 'Kompakte Abstände' },
              ].map((item) => {
                const isEnabled = prefs[item.key as keyof DashboardPreferences];
                return (
                  <div
                    key={item.key}
                    onClick={() => updatePref(item.key as any, !isEnabled)}
                    className="flex items-center justify-between p-2 rounded-md bg-[#121212] border border-[#242424] hover:border-[#333333] cursor-pointer transition-colors text-xs"
                  >
                    <span className="text-neutral-300">{item.label}</span>
                    <button className={cn('p-1 rounded text-xs', isEnabled ? 'text-blue-400' : 'text-neutral-600')}>
                      {isEnabled ? <Check className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                    </button>
                  </div>
                );
              })}
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => setShowConfigModal(false)}
                className="px-4 py-1.5 rounded-md text-xs font-medium bg-blue-600 hover:bg-blue-500 text-white transition-colors"
              >
                Fertig
              </button>
            </div>
          </div>
        </div>
      )}
```

2. Wrap the Active Containers table in `Dashboard.tsx` (line 512) with `{prefs.showContainers && (` so the user's preference to hide/show the table is honored.

---

### 4.3 Recommended Minor Polish Fixes
1. **Typo Fix (`py-0.2`):**
   - In `frontend/src/pages/NetworkFlow.tsx` lines 1140 & 1145, change `py-0.2` to `py-0.5`.
   - In `frontend/src/pages/Networks.tsx` line 346, change `py-0.2` to `py-0.5`.
2. **Backdrop Blur Standardization:**
   - In `frontend/src/components/auth/LoginModal.tsx` line 35, `SetupModal.tsx` line 48, and `ActionConfirmModal.tsx` line 31, replace `backdrop-blur-xs` with `backdrop-blur-sm`.
3. **ContainerDetail Tab Bar Overflow:**
   - In `frontend/src/pages/ContainerDetail.tsx` line 233, add `overflow-x-auto` to the tab container:
     `<div className="flex items-center gap-1 border-b border-[#242424] pb-2 overflow-x-auto">`
4. **Slate to Neutral Gray Consistency:**
   - In `frontend/src/pages/GPU.tsx` line 36 and `frontend/src/pages/Settings.tsx` line 53, change `text-slate-400` to `text-neutral-400`.
5. **Localization Polish:**
   - In `frontend/src/pages/Containers.tsx` line 144, update placeholder to `"Suche nach Name, Image oder ID..."`.

---

## 5. Verification Method

To independently verify these findings and any applied fixes:
1. **Frontend Typecheck & Build:**
   ```powershell
   npm --prefix frontend run build
   ```
   Must succeed with zero TypeScript or bundle errors.
2. **Visual Inspection of Collapsed Sidebar Header:**
   - Open Orbit in a web browser.
   - Click the sidebar collapse button `<` in the header.
   - Verify that the sidebar collapses to `w-16` (64px).
   - Verify that the 'O' logo box with Docker indicator is centered at 32px.
   - Verify that the expand toggle `>` floats on the right border with zero overlap or collision with the logo.
   - Verify that all navigation icons and the footer avatar are aligned on the vertical centerline.
   - Click the expand toggle `>` and confirm smooth transition to expanded width (`w-56`).
3. **Visual Inspection of Dashboard Customize Modal:**
   - Navigate to Dashboard (`/`).
   - Click the "Anpassen" button in the upper right.
   - Verify that the "Dashboard anpassen" modal appears cleanly over the backdrop.
   - Toggle widgets (e.g. Charts, Containers, Thermals) and confirm live reactivity and persistence to `localStorage`.
4. **View-by-View Walkthrough:**
   - Navigate through all 13 views: Dashboard, Containers, Container Detail, Network Flow, Networks, Images, Volumes, GPU Monitoring, Events, Logs, Settings, Login, Setup.
   - Confirm dark theme consistency, clean table padding, and absence of horizontal scrollbar on standard laptop (1280x800) and desktop (1920x1080) viewports.
