#!/usr/bin/env python3
"""tests/run_all_tests.py

Master Test Runner for Orbit Multi-Arch CI & CasaOS App Store Integration Suites.
Runs all test tiers across:
  1. Docker Multi-Arch Build & CI Workflow (`tests/test_docker_multiarch.py`)
  2. CasaOS App Store Integration Package (`tests/test_casaos_package.py`)

Usage:
  python tests/run_all_tests.py
  python tests/run_all_tests.py --suite=docker
  python tests/run_all_tests.py --suite=casaos
  python tests/run_all_tests.py --tier=1
  python tests/run_all_tests.py --tier=2
  python tests/run_all_tests.py --tier=3
  python tests/run_all_tests.py --tier=4
  python tests/run_all_tests.py --tier=5
  python tests/run_all_tests.py --verbose
"""

import argparse
import sys
import time
import unittest
from pathlib import Path

# Add REPO_ROOT and REPO_ROOT/tests to sys.path so modules can import cleanly
REPO_ROOT = Path(__file__).resolve().parent.parent
TESTS_DIR = REPO_ROOT / "tests"

if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))
if str(TESTS_DIR) not in sys.path:
    sys.path.insert(0, str(TESTS_DIR))

import test_casaos_package
import test_docker_multiarch


def build_test_suite(suite_filter: str = None, tier_filter: int = None) -> unittest.TestSuite:
    suite = unittest.TestSuite()
    loader = unittest.TestLoader()

    # Define test classes per suite and tier
    # Suite 1: Docker Multi-Arch
    docker_classes = [
        (1, test_docker_multiarch.TestTier1DockerfileMultiArch),
        (2, test_docker_multiarch.TestTier2CIWorkflowConfig),
        (3, test_docker_multiarch.TestTier3GoCrossCompilation),
        (4, test_docker_multiarch.TestTier4DockerBuildxDryRun),
    ]

    # Suite 2: CasaOS Package
    casaos_classes = [
        (1, test_casaos_package.TestTier1CasaOSFilesystem),
        (2, test_casaos_package.TestTier2DockerComposeSyntax),
        (3, test_casaos_package.TestTier3CasaOSMetadataSchema),
        (4, test_casaos_package.TestTier4CasaOSIconProperties),
        (5, test_casaos_package.TestTier5CasaOSDocumentation),
    ]

    # Filter suites
    selected_docker = suite_filter is None or suite_filter in ("docker", "multiarch", "ci")
    selected_casaos = suite_filter is None or suite_filter in ("casaos", "appstore", "package")

    if selected_docker:
        for tier, cls in docker_classes:
            if tier_filter is None or tier == tier_filter:
                suite.addTests(loader.loadTestsFromTestCase(cls))

    if selected_casaos:
        for tier, cls in casaos_classes:
            if tier_filter is None or tier == tier_filter:
                suite.addTests(loader.loadTestsFromTestCase(cls))

    return suite


class SummaryTestResult(unittest.TextTestResult):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.tier_stats = {}

    def startTest(self, test):
        super().startTest(test)
        cls_name = test.__class__.__name__
        if "Tier1" in cls_name:
            tier = 1
        elif "Tier2" in cls_name:
            tier = 2
        elif "Tier3" in cls_name:
            tier = 3
        elif "Tier4" in cls_name:
            tier = 4
        elif "Tier5" in cls_name:
            tier = 5
        else:
            tier = 0

        suite_name = "CasaOS Package" if "casaos" in test.__class__.__module__ else "Docker Multi-Arch"
        key = (suite_name, tier)
        if key not in self.tier_stats:
            self.tier_stats[key] = {"run": 0, "pass": 0, "fail": 0, "error": 0, "skip": 0}
        self.tier_stats[key]["run"] += 1

    def _record(self, test, stat_key: str):
        test_str = f"{test.__class__.__name__} {getattr(test, 'description', '')} {str(test)}"
        tier = self._get_tier(test_str)
        suite_name = self._get_suite(test)
        key = (suite_name, tier if tier in (1, 2, 3, 4, 5) else 1)
        if key in self.tier_stats:
            self.tier_stats[key][stat_key] += 1

    def addSuccess(self, test):
        super().addSuccess(test)
        self._record(test, "pass")

    def addFailure(self, test, err):
        super().addFailure(test, err)
        self._record(test, "fail")

    def addError(self, test, err):
        super().addError(test, err)
        self._record(test, "error")

    def addSkip(self, test, reason):
        super().addSkip(test, reason)
        self._record(test, "skip")

    @staticmethod
    def _get_tier(cls_name: str) -> int:
        for t in range(1, 6):
            if f"Tier{t}" in cls_name:
                return t
        return 0

    @staticmethod
    def _get_suite(test_obj) -> str:
        mod = getattr(test_obj, "__module__", "") or test_obj.__class__.__module__
        if "casaos" in mod:
            return "CasaOS Package"
        return "Docker Multi-Arch"


def main():
    parser = argparse.ArgumentParser(description="Orbit Master Test Runner")
    parser.add_argument(
        "--suite",
        choices=["docker", "multiarch", "ci", "casaos", "appstore", "package"],
        help="Run only specific test suite",
    )
    parser.add_argument(
        "--tier",
        type=int,
        choices=[1, 2, 3, 4, 5],
        help="Run only specific tier across selected suites",
    )
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Enable verbose test output",
    )
    args = parser.parse_args()

    suite_name_map = {
        "docker": "Docker Multi-Arch & CI",
        "multiarch": "Docker Multi-Arch & CI",
        "ci": "Docker Multi-Arch & CI",
        "casaos": "CasaOS App Store Integration",
        "appstore": "CasaOS App Store Integration",
        "package": "CasaOS App Store Integration",
    }

    print("=" * 76)
    print("      ORBIT MASTER TEST RUNNER: MULTI-ARCH CI & CASAOS INTEGRATION")
    print("=" * 76)
    print(f"Directory:    {REPO_ROOT}")
    print(f"Suite Filter: {suite_name_map.get(args.suite, 'All Suites')}")
    print(f"Tier Filter:  {f'Tier {args.tier}' if args.tier else 'All Tiers (1-5)'}")
    print(f"Verbosity:    {'High' if args.verbose else 'Standard'}")
    print("-" * 76)

    suite = build_test_suite(args.suite, args.tier)
    total_tests = suite.countTestCases()
    if total_tests == 0:
        print("[!] No test cases matched the selected filter.")
        sys.exit(1)

    runner = unittest.TextTestRunner(
        verbosity=2 if args.verbose else 1,
        resultclass=SummaryTestResult,
    )

    start_time = time.time()
    result = runner.run(suite)
    elapsed = time.time() - start_time

    print()
    print("=" * 76)
    print(f"                       TEST SUITE EXECUTION BREAKDOWN")
    print("=" * 76)

    # Print Tier Breakdown
    sorted_keys = sorted(result.tier_stats.keys(), key=lambda x: (x[0], x[1]))
    current_suite = None
    for key in sorted_keys:
        s_name, tier = key
        stats = result.tier_stats[key]
        if s_name != current_suite:
            current_suite = s_name
            print(f"\n  --- {current_suite} ---")
        run_count = stats["run"]
        passed = stats["pass"]
        status = "PASS" if (stats["fail"] == 0 and stats["error"] == 0) else "FAIL"
        color_indicator = "✔" if status == "PASS" else "✖"
        print(
            f"  {color_indicator} Tier {tier}: {passed:2d} / {run_count:2d} passed "
            f"({stats['fail']} failed, {stats['error']} errors, {stats['skip']} skipped) [{status}]"
        )

    print()
    print("-" * 76)
    all_passed = len(result.failures) == 0 and len(result.errors) == 0
    passed_total = result.testsRun - len(result.failures) - len(result.errors) - len(result.skipped)
    pass_rate = (passed_total / result.testsRun * 100) if result.testsRun > 0 else 0.0

    print(f"  TOTAL TESTS RUN: {result.testsRun}")
    print(f"  TOTAL PASSED:    {passed_total}")
    print(f"  TOTAL FAILED:    {len(result.failures)}")
    print(f"  TOTAL ERRORS:    {len(result.errors)}")
    print(f"  TOTAL SKIPPED:   {len(result.skipped)}")
    print(f"  PASS RATE:       {pass_rate:.1f}%")
    print(f"  TIME ELAPSED:    {elapsed:.2f}s")
    print("=" * 76)

    if all_passed:
        print(f"✔ ALL {result.testsRun} TESTS PASSED SUCCESSFULLY! [Status: READY]")
        sys.exit(0)
    else:
        print(f"✖ TEST SUITE FAILED ({len(result.failures)} failures, {len(result.errors)} errors)")
        sys.exit(1)


if __name__ == "__main__":
    main()
