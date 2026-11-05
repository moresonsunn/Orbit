export type DashboardPreset = 'minimal' | 'balanced' | 'pro';

export interface DashboardPreferences {
  preset: DashboardPreset;
  showSummary: boolean;         // 4 Kernmetriken: CPU, RAM, Netzwerk, Festplatte
  showDockerOverview: boolean;  // Docker Engine Status-Leiste
  showThermals: boolean;        // Hardware-Temperaturen (CPU, NVMe, Mainboard)
  showLoadAvg: boolean;         // 1 min / 5 min / 15 min Load Average
  showGPU: boolean;             // GPU-Telemetrie
  showCharts: boolean;          // Echtzeit-Verlaufsdiagramme
  showContainers: boolean;      // Aktive Container-Tabelle
  compactMode: boolean;         // Kompakte Ansicht
}

const STORAGE_KEY = 'orbit_dashboard_prefs';

export const PRESET_CONFIGS: Record<DashboardPreset, DashboardPreferences> = {
  minimal: {
    preset: 'minimal',
    showSummary: true,
    showDockerOverview: false,
    showThermals: false,
    showLoadAvg: false,
    showGPU: false,
    showCharts: false,
    showContainers: true,
    compactMode: true,
  },
  balanced: {
    preset: 'balanced',
    showSummary: true,
    showDockerOverview: true,
    showThermals: true,
    showLoadAvg: true,
    showGPU: false,
    showCharts: false,
    showContainers: true,
    compactMode: false,
  },
  pro: {
    preset: 'pro',
    showSummary: true,
    showDockerOverview: true,
    showThermals: true,
    showLoadAvg: true,
    showGPU: true,
    showCharts: true,
    showContainers: true,
    compactMode: false,
  },
};

export function loadDashboardPrefs(): DashboardPreferences {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) {
      const parsed = JSON.parse(raw);
      return { ...PRESET_CONFIGS.balanced, ...parsed };
    }
  } catch (err) {
    console.error('Failed to parse dashboard preferences:', err);
  }
  return PRESET_CONFIGS.balanced;
}

export function saveDashboardPrefs(prefs: DashboardPreferences): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch (err) {
    console.error('Failed to save dashboard preferences:', err);
  }
}

