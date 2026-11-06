## 2026-11-06T01:07:19Z
You are worker_m1_ui, an implementation worker agent.
Your working directory is: c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_m1_ui

MANDATORY FIRST STEP:
Read c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\ORIGINAL_REQUEST.md before starting any work.

MANDATORY INTEGRITY WARNING:
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

REFERENCE FILES:
- `PROJECT.md` at repository root
- Explorer findings report: `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\explorer_survey_ui\handoff.md`

WRITE OWNERSHIP:
You exclusively own:
- `frontend/src/components/layout/Sidebar.tsx`
- `frontend/src/pages/Dashboard.tsx`
- Minor polish in: `frontend/src/pages/NetworkFlow.tsx`, `frontend/src/pages/Networks.tsx`, `frontend/src/pages/ContainerDetail.tsx`, `frontend/src/pages/GPU.tsx`, `frontend/src/pages/Settings.tsx`, `frontend/src/pages/Containers.tsx`, `frontend/src/components/auth/LoginModal.tsx`, `frontend/src/components/auth/SetupModal.tsx`, `frontend/src/components/common/ActionConfirmModal.tsx`.
You MUST NOT edit any backend Go files, `.github/` workflow files, docker-compose files, or `README.md`.

TASKS FOR MILESTONE M1:
1. Fix Collapsed Sidebar Header Collision in `frontend/src/components/layout/Sidebar.tsx`:
   - In collapsed state (`w-16`), center the brand logo at 32px (`mx-auto` or flex centering).
   - Position the expand toggle button (`>`) on the border seam as a floating edge handle (`absolute -right-3 top-1/2 -translate-y-1/2 w-6 h-6 rounded-full bg-[#1c1c1c] border border-[#2e2e2e] shadow-md flex items-center justify-center text-neutral-400 hover:text-white hover:bg-[#282828] transition-all z-40`). Zero collision with the logo!
   - In expanded state, preserve clean layout with logo on left and collapse button `<` on right.
   - Align navigation items on the 32px vertical axis in collapsed state (`collapsed ? 'justify-center px-0 py-2' : 'px-2.5 py-1.5'`).
2. Fix Dashboard "Anpassen" (Customize) Modal in `frontend/src/pages/Dashboard.tsx`:
   - Render the missing `showConfigModal` dialog JSX in the component render tree with widget toggles and preset buttons (Kompakt, Standard, Pro Sysadmin) using `saveDashboardPrefs`.
   - Wrap the Active Containers table with `{prefs.showContainers && (` so the preference toggle is properly respected.
3. Apply View-Level Polish:
   - Fix `py-0.2` typos to `py-0.5` in `NetworkFlow.tsx` (lines 1140 & 1145) and `Networks.tsx` (line 346).
   - Add `overflow-x-auto` to the tab bar container in `ContainerDetail.tsx` (line 233).
   - Standardize `backdrop-blur-xs` to `backdrop-blur-sm` in `LoginModal.tsx`, `SetupModal.tsx`, and `ActionConfirmModal.tsx`.
   - Standardize `text-slate-400` to `text-neutral-400` in `GPU.tsx` and `Settings.tsx`.
   - Polish German search placeholder in `Containers.tsx` line 144 to `"Suche nach Name, Image oder ID..."`.

VERIFICATION:
Execute `npm run build` in `frontend/` to confirm zero TypeScript errors and clean bundle compilation.
Document all commands and verification results in `c:\Users\phili\Documents\GitHub\Orbit\.agents\teamwork\worker_m1_ui\handoff.md` and send a completion message to the orchestrator.
