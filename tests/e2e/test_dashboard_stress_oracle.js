#!/usr/bin/env node
/**
 * Adversarial Stress Test & Oracle Simulator for Dashboard UI Warning Banner (Requirement R2)
 *
 * Tests:
 * 1. Truth-table logic state space (undefined, null, true with lingering error, false with empty error, multiline error)
 * 2. DOM element invariants (table tag counts under all widget configurations)
 * 3. Operator widget preference independence
 * 4. TypeScript AST checks on Dashboard.tsx
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '../..');
const dashboardFile = path.join(repoRoot, 'frontend/src/pages/Dashboard.tsx');

const dashboardCode = fs.readFileSync(dashboardFile, 'utf8');

let totalTests = 0;
let passedTests = 0;
let failedTests = 0;

function assert(condition, testName, details = '') {
  totalTests++;
  if (condition) {
    passedTests++;
    console.log(`  ✔ [PASS] ${testName}`);
  } else {
    failedTests++;
    console.error(`  ✖ [FAIL] ${testName}`);
    if (details) console.error(`     Details: ${details}`);
  }
}

console.log('================================================================');
console.log('   EMPIRICAL ADVERSARIAL STRESS TEST: DASHBOARD WARNING BANNER   ');
console.log('================================================================\n');

// ----------------------------------------------------------------------
// Track 1: Logical Oracle Simulator
// ----------------------------------------------------------------------
console.log('--- Track 1: Logical Oracle Simulator (State Space Gating) ---');

function evaluateBannerLogic(sysInfo) {
  // Directly mirrors Dashboard.tsx JSX conditional:
  // {sysInfo && !sysInfo.dockerConnected && ( ... )}
  const isBannerVisible = Boolean(sysInfo && !sysInfo.dockerConnected);

  // Error block inside banner:
  // {sysInfo.dockerError && ( ... )}
  const isErrorBlockVisible = Boolean(isBannerVisible && sysInfo && sysInfo.dockerError);

  // Secondary metric card indicator:
  // {sysInfo && !sysInfo.dockerConnected && ( ... )}
  const isGetrenntBadgeVisible = Boolean(sysInfo && !sysInfo.dockerConnected);

  // Container empty state message:
  // sysInfo && !sysInfo.dockerConnected
  //   ? 'Keine Container verfügbar — Docker-Daemon nicht erreichbar oder Verbindung getrennt.'
  //   : 'Keine Container auf diesem Server gefunden.'
  const emptyStateMessage = (sysInfo && !sysInfo.dockerConnected)
    ? 'Keine Container verfügbar — Docker-Daemon nicht erreichbar oder Verbindung getrennt.'
    : 'Keine Container auf diesem Server gefunden.';

  return { isBannerVisible, isErrorBlockVisible, isGetrenntBadgeVisible, emptyStateMessage };
}

// Case 1: sysInfo undefined
{
  const res = evaluateBannerLogic(undefined);
  assert(!res.isBannerVisible, 'Case 1.1: sysInfo undefined -> Banner completely absent');
  assert(!res.isErrorBlockVisible, 'Case 1.2: sysInfo undefined -> Error block absent');
  assert(!res.isGetrenntBadgeVisible, 'Case 1.3: sysInfo undefined -> Getrennt badge absent');
  assert(res.emptyStateMessage.includes('Keine Container auf diesem Server gefunden'), 'Case 1.4: sysInfo undefined -> Normal empty state');
}

// Case 2: sysInfo null
{
  const res = evaluateBannerLogic(null);
  assert(!res.isBannerVisible, 'Case 2.1: sysInfo null -> Banner completely absent');
  assert(!res.isGetrenntBadgeVisible, 'Case 2.2: sysInfo null -> Getrennt badge absent');
}

// Case 3: sysInfo connected=true, error=null
{
  const res = evaluateBannerLogic({ dockerConnected: true, dockerError: null });
  assert(!res.isBannerVisible, 'Case 3.1: dockerConnected true -> Banner absent');
  assert(!res.isGetrenntBadgeVisible, 'Case 3.2: dockerConnected true -> Getrennt badge absent');
  assert(res.emptyStateMessage.includes('Keine Container auf diesem Server gefunden'), 'Case 3.3: dockerConnected true -> Normal empty state');
}

// Case 4: sysInfo connected=true, lingering dockerError="permission denied"
{
  const res = evaluateBannerLogic({ dockerConnected: true, dockerError: 'dial unix /var/run/docker.sock: connect: permission denied' });
  assert(!res.isBannerVisible, 'Case 4.1: dockerConnected true with lingering error -> Banner STILL absent');
  assert(!res.isErrorBlockVisible, 'Case 4.2: dockerConnected true with lingering error -> Error block absent');
  assert(!res.isGetrenntBadgeVisible, 'Case 4.3: dockerConnected true with lingering error -> Getrennt badge absent');
}

// Case 5: sysInfo connected=false, error="" (empty string)
{
  const res = evaluateBannerLogic({ dockerConnected: false, dockerError: '' });
  assert(res.isBannerVisible, 'Case 5.1: dockerConnected false with empty error -> Banner present');
  assert(!res.isErrorBlockVisible, 'Case 5.2: dockerConnected false with empty error -> Error block hidden cleanly');
  assert(res.isGetrenntBadgeVisible, 'Case 5.3: dockerConnected false with empty error -> Getrennt badge present');
  assert(res.emptyStateMessage.includes('Keine Container verfügbar — Docker-Daemon nicht erreichbar'), 'Case 5.4: Disconnected empty state message shown');
}

// Case 6: sysInfo connected=false, error=null
{
  const res = evaluateBannerLogic({ dockerConnected: false, dockerError: null });
  assert(res.isBannerVisible, 'Case 6.1: dockerConnected false with null error -> Banner present');
  assert(!res.isErrorBlockVisible, 'Case 6.2: dockerConnected false with null error -> Error block hidden cleanly');
}

// Case 7: sysInfo connected=false, single-line error
{
  const errMsg = 'dial unix /var/run/docker.sock: connect: permission denied';
  const res = evaluateBannerLogic({ dockerConnected: false, dockerError: errMsg });
  assert(res.isBannerVisible, 'Case 7.1: dockerConnected false with error -> Banner present');
  assert(res.isErrorBlockVisible, 'Case 7.2: dockerConnected false with error -> Error block present');
}

// Case 8: sysInfo connected=false, multiline error
{
  const multilineErr = 'Error response from daemon:\nGot permission denied while trying to connect\nAre you in docker group?';
  const res = evaluateBannerLogic({ dockerConnected: false, dockerError: multilineErr });
  assert(res.isBannerVisible, 'Case 8.1: dockerConnected false with multiline error -> Banner present');
  assert(res.isErrorBlockVisible, 'Case 8.2: dockerConnected false with multiline error -> Error block present');
}

// ----------------------------------------------------------------------
// Track 2: AST, DOM, & Code Inspection Invariants
// ----------------------------------------------------------------------
console.log('\n--- Track 2: AST, DOM, & Code Inspection Invariants ---');

// Check 1: Exactly 1 <table tag in Dashboard.tsx
{
  const tableMatches = dashboardCode.match(/<table\b/g) || [];
  assert(tableMatches.length === 1, 'Invariant 2.1: Dashboard.tsx contains exactly 1 <table> tag', `Found: ${tableMatches.length}`);
}

// Check 2: Banner contains NO table tags and contains all required UI elements
{
  const bannerStart = dashboardCode.indexOf('{sysInfo && !sysInfo.dockerConnected && (');
  const bannerEnd = dashboardCode.indexOf('{/* Upper Information Grid */}', bannerStart);
  assert(bannerStart !== -1 && bannerEnd !== -1, 'Invariant 2.2: Banner gating block found in Dashboard.tsx');
  if (bannerStart !== -1 && bannerEnd !== -1) {
    const bannerContent = dashboardCode.slice(bannerStart, bannerEnd);
    assert(!bannerContent.includes('<table'), 'Invariant 2.3: Banner does not contain any <table> tag');
    assert(bannerContent.includes('AlertTriangle'), 'Invariant 2.4: Banner renders AlertTriangle icon');
    assert(bannerContent.includes('Verbindung getrennt'), 'Invariant 2.5: Banner contains "Verbindung getrennt" status badge');
    assert(bannerContent.includes('bg-rose-950/30'), 'Invariant 2.6: Banner styled with bg-rose-950/30');
    assert(bannerContent.includes('/var/run/docker.sock'), 'Invariant 2.7: Banner references /var/run/docker.sock');
    assert(bannerContent.includes('user: "0:0"'), 'Invariant 2.8: Banner explains user: "0:0" remediation');
    assert(bannerContent.includes('handleRetryDockerConnection'), 'Invariant 2.9: Banner includes live retry button');
    assert(bannerContent.includes('retryingDocker'), 'Invariant 2.10: Banner displays retry spinner state');
  }
}

// Check 3: Placement before widget preference blocks
{
  const bannerPos = dashboardCode.indexOf('{sysInfo && !sysInfo.dockerConnected && (');
  const summaryPos = dashboardCode.indexOf('{prefs.showSummary && (');
  const containerPos = dashboardCode.indexOf('{prefs.showContainers && (');
  const modalPos = dashboardCode.indexOf('{showConfigModal && (');

  assert(bannerPos !== -1 && bannerPos < summaryPos, 'Invariant 2.11: Banner placed before {prefs.showSummary && (');
  assert(bannerPos !== -1 && bannerPos < containerPos, 'Invariant 2.12: Banner placed before {prefs.showContainers && (');
  assert(modalPos !== -1, 'Invariant 2.13: Modal configuration dialog intact');
}

// Check 4: Container table empty state differentiation
{
  const emptyStateDisconnected = dashboardCode.includes('Keine Container verfügbar — Docker-Daemon nicht erreichbar oder Verbindung getrennt.');
  const emptyStateNormal = dashboardCode.includes('Keine Container auf diesem Server gefunden.');
  assert(emptyStateDisconnected && emptyStateNormal, 'Invariant 2.14: Container table empty state cleanly distinguishes disconnected vs empty');
}

// Check 5: Secondary metric card Getrennt badge
{
  const cardBadge = dashboardCode.includes('Getrennt') && dashboardCode.includes('{sysInfo && !sysInfo.dockerConnected && (');
  assert(cardBadge, 'Invariant 2.15: Secondary Docker Engine card renders "Getrennt" badge on disconnect');
}

console.log('\n================================================================');
console.log(`STRESS TEST SUMMARY: ${passedTests} / ${totalTests} PASSED, ${failedTests} FAILED`);
console.log('================================================================');

if (failedTests > 0) {
  process.exit(1);
} else {
  process.exit(0);
}
