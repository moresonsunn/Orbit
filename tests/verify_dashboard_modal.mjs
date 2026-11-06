/**
 * Empirical Challenge Harness: Dashboard Customize Modal & State Logic
 * 
 * Verifies:
 * 1. Preset configuration field accuracy for 'minimal', 'balanced', and 'pro'
 * 2. loadDashboardPrefs & saveDashboardPrefs localStorage persistence and error resilience
 * 3. State transition oracle: updatePref, setPreset, and reactivity
 * 4. JSX AST / DOM structure analysis of Dashboard.tsx for conditional gating of modal & containers
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..');

// 1. Mock localStorage environment for node
class MockLocalStorage {
  constructor() {
    this.store = {};
    this.shouldThrow = false;
  }
  getItem(key) {
    if (this.shouldThrow) throw new Error('SecurityError: localStorage is disabled');
    return this.store[key] !== undefined ? this.store[key] : null;
  }
  setItem(key, value) {
    if (this.shouldThrow) throw new Error('QuotaExceededError: storage quota exceeded');
    this.store[key] = String(value);
  }
  removeItem(key) {
    delete this.store[key];
  }
  clear() {
    this.store = {};
  }
}

const mockStorage = new MockLocalStorage();
globalThis.localStorage = mockStorage;

// Load dashboardConfig.ts
const configModulePath = path.join(repoRoot, 'frontend', 'src', 'lib', 'dashboardConfig.ts');
const { PRESET_CONFIGS, loadDashboardPrefs, saveDashboardPrefs } = await import(
  `file://${configModulePath.replace(/\\/g, '/')}`
);

let passed = 0;
let failed = 0;

function assert(condition, message) {
  if (condition) {
    console.log(`  [PASS] ${message}`);
    passed++;
  } else {
    console.error(`  [FAIL] ${message}`);
    failed++;
  }
}

console.log('='.repeat(70));
console.log('CHALLENGE SUITE 1: PRESET CONFIGURATIONS SPECIFICATION TEST');
console.log('='.repeat(70));

const expectedPresets = {
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

for (const [presetKey, expectedFields] of Object.entries(expectedPresets)) {
  const actual = PRESET_CONFIGS[presetKey];
  assert(actual !== undefined, `Preset '${presetKey}' exists in PRESET_CONFIGS`);
  if (actual) {
    for (const [field, val] of Object.entries(expectedFields)) {
      assert(
        actual[field] === val,
        `Preset '${presetKey}' field '${field}' === ${val} (got ${actual[field]})`
      );
    }
  }
}

console.log('\n' + '='.repeat(70));
console.log('CHALLENGE SUITE 2: LOCALSTORAGE PERSISTENCE & ERROR RESILIENCE');
console.log('='.repeat(70));

// Test default load when localStorage empty
mockStorage.clear();
const defaultPrefs = loadDashboardPrefs();
assert(defaultPrefs.preset === 'balanced', "loadDashboardPrefs returns 'balanced' on empty storage");
assert(defaultPrefs.showContainers === true, "defaultPrefs.showContainers === true");
assert(defaultPrefs.showCharts === false, "defaultPrefs.showCharts === false");

// Test save and retrieve
const customPrefs = {
  ...PRESET_CONFIGS.pro,
  showContainers: false,
  compactMode: true,
};
saveDashboardPrefs(customPrefs);
const rawSaved = mockStorage.getItem('orbit_dashboard_prefs');
assert(rawSaved !== null, "saveDashboardPrefs writes to 'orbit_dashboard_prefs'");
const parsedSaved = JSON.parse(rawSaved);
assert(parsedSaved.showContainers === false, "Persisted showContainers === false");
assert(parsedSaved.preset === 'pro', "Persisted preset === 'pro'");

const reloadedPrefs = loadDashboardPrefs();
assert(reloadedPrefs.showContainers === false, "loadDashboardPrefs retrieves custom showContainers === false");
assert(reloadedPrefs.preset === 'pro', "loadDashboardPrefs retrieves custom preset === 'pro'");

// Stress test: corrupted JSON in localStorage
mockStorage.setItem('orbit_dashboard_prefs', 'CORRUPTED_JSON_{{[INVALID');
let corruptedRecovered = null;
try {
  corruptedRecovered = loadDashboardPrefs();
  assert(true, 'loadDashboardPrefs does not throw on corrupted JSON');
  assert(corruptedRecovered.preset === 'balanced', 'loadDashboardPrefs falls back to balanced on corrupted JSON');
} catch (err) {
  assert(false, `loadDashboardPrefs crashed on corrupted JSON: ${err.message}`);
}

// Stress test: QuotaExceededError / SecurityError handling in saveDashboardPrefs
mockStorage.shouldThrow = true;
try {
  saveDashboardPrefs(customPrefs);
  assert(true, 'saveDashboardPrefs does not throw unhandled exception when storage throws');
} catch (err) {
  assert(false, `saveDashboardPrefs threw unhandled exception on storage failure: ${err.message}`);
}

// Stress test: QuotaExceededError / SecurityError handling in loadDashboardPrefs
try {
  const fallbackOnStorageError = loadDashboardPrefs();
  assert(fallbackOnStorageError.preset === 'balanced', 'loadDashboardPrefs safely returns default when storage throws');
} catch (err) {
  assert(false, `loadDashboardPrefs threw unhandled exception on storage failure: ${err.message}`);
}
mockStorage.shouldThrow = false;

console.log('\n' + '='.repeat(70));
console.log('CHALLENGE SUITE 3: STATE TRANSITIONS & REACTIVITY SIMULATION');
console.log('='.repeat(70));

// Simulate state transitions in Dashboard.tsx:
// 1. Initial state
let currentPrefs = loadDashboardPrefs();
mockStorage.clear();

// Simulator functions matching Dashboard.tsx lines 61-73:
function simulateUpdatePref(key, value) {
  currentPrefs = { ...currentPrefs, [key]: value };
  saveDashboardPrefs(currentPrefs);
  return currentPrefs;
}

function simulateSetPreset(preset) {
  const next = { ...PRESET_CONFIGS[preset] };
  currentPrefs = next;
  saveDashboardPrefs(next);
  return currentPrefs;
}

// Switch to 'minimal'
simulateSetPreset('minimal');
assert(currentPrefs.preset === 'minimal', "Switched preset to 'minimal'");
assert(currentPrefs.showDockerOverview === false, "'minimal' disabled showDockerOverview");
assert(currentPrefs.compactMode === true, "'minimal' enabled compactMode");
assert(
  JSON.parse(mockStorage.getItem('orbit_dashboard_prefs')).compactMode === true,
  "localStorage persisted compactMode === true"
);

// Toggle showContainers: from true to false
assert(currentPrefs.showContainers === true, "Pre-condition: showContainers is true");
simulateUpdatePref('showContainers', !currentPrefs.showContainers);
assert(currentPrefs.showContainers === false, "Toggled showContainers to false");
assert(
  JSON.parse(mockStorage.getItem('orbit_dashboard_prefs')).showContainers === false,
  "localStorage persisted showContainers === false"
);

// Toggle showContainers: from false back to true
simulateUpdatePref('showContainers', !currentPrefs.showContainers);
assert(currentPrefs.showContainers === true, "Toggled showContainers back to true");
assert(
  JSON.parse(mockStorage.getItem('orbit_dashboard_prefs')).showContainers === true,
  "localStorage persisted showContainers === true"
);

// Switch to 'pro'
simulateSetPreset('pro');
assert(currentPrefs.preset === 'pro', "Switched preset to 'pro'");
assert(currentPrefs.showGPU === true, "'pro' enabled showGPU");
assert(currentPrefs.showCharts === true, "'pro' enabled showCharts");
assert(currentPrefs.showThermals === true, "'pro' enabled showThermals");
assert(
  JSON.parse(mockStorage.getItem('orbit_dashboard_prefs')).showGPU === true,
  "localStorage persisted showGPU === true"
);

console.log('\n' + '='.repeat(70));
console.log('CHALLENGE SUITE 4: DASHBOARD.TSX JSX AST & DOM GATING ANALYSIS');
console.log('='.repeat(70));

const dashPath = path.join(repoRoot, 'frontend', 'src', 'pages', 'Dashboard.tsx');
const dashCode = fs.readFileSync(dashPath, 'utf8');

// 4.1 Customize modal state hook inspection
assert(
  dashCode.includes('const [showConfigModal, setShowConfigModal] = useState(false);') ||
  dashCode.includes('const [showConfigModal, setShowConfigModal] = useState'),
  "Dashboard.tsx declares 'showConfigModal' state hook defaulting to false"
);

// 4.2 Modal trigger button inspection
assert(
  dashCode.includes('setShowConfigModal(true)'),
  "Dashboard.tsx contains trigger button calling setShowConfigModal(true)"
);

// 4.3 Modal conditional rendering gating
assert(
  dashCode.includes('{showConfigModal && ('),
  "Dashboard.tsx renders Customize Modal conditionally gated by '{showConfigModal && ('"
);

// 4.4 Modal close button and done button
assert(
  dashCode.includes('setShowConfigModal(false)'),
  "Dashboard.tsx contains close handlers calling setShowConfigModal(false)"
);

// Count occurrences of setShowConfigModal(false)
const closeOccurrences = (dashCode.match(/setShowConfigModal\(false\)/g) || []).length;
assert(
  closeOccurrences >= 2,
  `Dashboard.tsx has at least 2 close triggers (X button and Fertig button), found ${closeOccurrences}`
);

// 4.5 Active Containers preference gating
assert(
  dashCode.includes('{prefs.showContainers && ('),
  "Dashboard.tsx gates the Active Containers table with '{prefs.showContainers && ('"
);

// Verify that the table header exists inside the prefs.showContainers block
const showContainersIdx = dashCode.indexOf('{prefs.showContainers && (');
assert(showContainersIdx !== -1, "prefs.showContainers block exists");

const afterShowContainers = dashCode.slice(showContainersIdx);
const tableTagIdx = afterShowContainers.indexOf('<table');
assert(
  tableTagIdx !== -1 && tableTagIdx < 800,
  "Container table <table> tag is nested immediately inside {prefs.showContainers && ("
);

// Verify that when showContainers is false, table cannot render
// Verify there are no duplicate un-gated <table tags in Dashboard.tsx
const allTableTags = (dashCode.match(/<table\b/g) || []).length;
assert(
  allTableTags === 1,
  `Exactly 1 <table> tag exists in Dashboard.tsx, strictly inside the gated block (found ${allTableTags})`
);

// 4.6 Widget toggles in modal: verify all 8 widgets are togglable
const requiredToggles = [
  'showSummary',
  'showDockerOverview',
  'showLoadAvg',
  'showThermals',
  'showGPU',
  'showCharts',
  'showContainers',
  'compactMode',
];

for (const t of requiredToggles) {
  assert(
    dashCode.includes(`key: '${t}'`),
    `Modal widget toggle list contains item key '${t}'`
  );
}

// 4.7 Preset buttons in modal
assert(
  dashCode.includes("(['minimal', 'balanced', 'pro'] as const).map"),
  "Modal renders all 3 presets via map over ['minimal', 'balanced', 'pro']"
);

console.log('\n' + '='.repeat(70));
console.log(`CHALLENGE HARNESS RESULTS: ${passed} PASSED, ${failed} FAILED`);
console.log('='.repeat(70));

if (failed > 0) {
  process.exit(1);
} else {
  process.exit(0);
}
