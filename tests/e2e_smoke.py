#!/usr/bin/env python3
"""E2E smoke tests for kubectl-metrics.

Runs a suite of tests against every CLI command using metrics common
to all OpenShift clusters.  Build the binary first (e.g. ``make e2e``
or ``make build && python3 tests/e2e_smoke.py``).

Usage:
    make e2e
"""

import json
import os
import subprocess
import sys
from typing import List

BINARY = os.path.join(os.path.dirname(__file__), "..", "kubectl-metrics")

# When set to a non-empty value, --insecure-skip-tls-verify is appended to
# every invocation.  Useful for clusters whose Thanos/Prometheus routes use
# certificates signed by an internal CA that is not in the system trust store
# (the common case on OpenShift).
INSECURE = os.environ.get("INSECURE_SKIP_TLS_VERIFY", "").lower() in ("1", "true", "yes")

passed = 0
failed = 0
errors = []  # type: List[str]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def run(args):
    """Run the binary with the given args, return (stdout, stderr, returncode)."""
    extra = ["--insecure-skip-tls-verify"] if INSECURE else []
    try:
        result = subprocess.run(
            [BINARY] + extra + args,
            capture_output=True,
            text=True,
            timeout=60,
        )
        return result.stdout, result.stderr, result.returncode
    except subprocess.TimeoutExpired as exc:
        stdout = exc.stdout or ""
        stderr = (exc.stderr or "") + "\n[TIMEOUT] command timed out after 60s"
        return stdout, stderr, 1


def record(name: str, ok: bool, detail: str = ""):
    global passed, failed
    if ok:
        passed += 1
        print(f"  PASS  {name}")
    else:
        failed += 1
        msg = f"  FAIL  {name}"
        if detail:
            msg += f"  -- {detail}"
        print(msg)
        errors.append(name)


def assert_exit_ok(name: str, rc: int, stderr: str = "") -> bool:
    ok = rc == 0
    record(name, ok, f"exit={rc} stderr={stderr[:120]}" if not ok else "")
    return ok


def assert_exit_fail(name: str, rc: int) -> bool:
    ok = rc != 0
    record(name, ok, "expected non-zero exit code" if not ok else "")
    return ok


def assert_contains(name: str, text: str, substring: str) -> bool:
    ok = substring in text
    record(name, ok, f"output missing '{substring}'" if not ok else "")
    return ok


def parse_json(text: str):
    try:
        return json.loads(text), None
    except json.JSONDecodeError as exc:
        return None, str(exc)


def assert_valid_json(name: str, text: str):
    data, err = parse_json(text)
    record(name, err is None, f"invalid JSON: {err}" if err else "")
    return data


# ---------------------------------------------------------------------------
# Tests: version
# ---------------------------------------------------------------------------

def test_version():
    print("[version]")
    stdout, stderr, rc = run(["version"])
    if assert_exit_ok("version exits 0", rc, stderr):
        assert_contains("version output", stdout, "kubectl-metrics")


# ---------------------------------------------------------------------------
# Tests: help-promql
# ---------------------------------------------------------------------------

def test_help_promql():
    print("[help-promql]")
    stdout, stderr, rc = run(["help-promql"])
    if assert_exit_ok("help-promql exits 0", rc, stderr):
        assert_contains("help-promql mentions PromQL", stdout, "PromQL")


# ---------------------------------------------------------------------------
# Tests: discover
# ---------------------------------------------------------------------------

def test_discover():
    print("[discover]")

    # 1. Basic discover returns metric names (should include the universal "up" metric)
    stdout, stderr, rc = run(["discover"])
    if assert_exit_ok("discover exits 0", rc, stderr):
        assert_contains("discover output contains 'up'", stdout, "up")

    # 2. Keyword filter
    stdout, stderr, rc = run(["discover", "--keyword", "container_cpu"])
    if assert_exit_ok("discover --keyword container_cpu", rc, stderr):
        assert_contains("discover keyword has container_cpu",
                        stdout, "container_cpu_usage_seconds_total")

    # 3. Group-by-prefix
    stdout, stderr, rc = run(["discover", "--keyword", "kube_node", "--group-by-prefix"])
    if assert_exit_ok("discover --group-by-prefix", rc, stderr):
        assert_contains("discover grouped has kube_node", stdout, "kube_node")


# ---------------------------------------------------------------------------
# Tests: labels
# ---------------------------------------------------------------------------

def test_labels():
    print("[labels]")

    # 1. All labels (no --metric)
    stdout, stderr, rc = run(["labels"])
    if assert_exit_ok("labels exits 0", rc, stderr):
        assert_contains("labels has __name__", stdout, "__name__")

    # 2. Labels for a specific metric
    stdout, stderr, rc = run(["labels", "--metric", "up"])
    if assert_exit_ok("labels --metric up", rc, stderr):
        assert_contains("labels for up has 'job'", stdout, "job")


# ---------------------------------------------------------------------------
# Tests: query (instant)
# ---------------------------------------------------------------------------

def test_query():
    print("[query]")

    # 1. Simple "up" query — markdown (default)
    stdout, stderr, rc = run(["query", "--query", "up"])
    if assert_exit_ok("query up markdown", rc, stderr):
        record("query up non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 2. JSON output
    stdout, stderr, rc = run(["query", "--query", "up", "--output", "json"])
    if assert_exit_ok("query up json", rc, stderr):
        data = assert_valid_json("query up valid json", stdout)
        if isinstance(data, list):
            record("query up json has results", len(data) > 0,
                   "empty JSON array" if len(data) == 0 else "")
        elif data is not None:
            record("query up json has results", False,
                   f"expected JSON array but got {type(data).__name__}")

    # 3. Query with --name label
    stdout, stderr, rc = run([
        "query", "--query", "up", "--name", "availability", "--output", "json",
    ])
    if assert_exit_ok("query up with --name", rc, stderr):
        assert_valid_json("query up --name valid json", stdout)

    # 4. Query with --selector filter
    stdout, stderr, rc = run([
        "query", "--query", "up",
        "--output", "json",
        "--selector", "job=prometheus-k8s",
    ])
    if assert_exit_ok("query up --selector", rc, stderr):
        data = assert_valid_json("query up --selector json", stdout)
        if isinstance(data, list) and len(data) > 0:
            all_match = all(
                item.get("metric", {}).get("job") == "prometheus-k8s"
                for item in data
            )
            record("query up --selector all match", all_match,
                   "not all results match selector" if not all_match else "")
        elif isinstance(data, list):
            record("query up --selector has results", False, "empty JSON array")
        elif data is not None:
            record("query up --selector has results", False,
                   f"expected JSON array but got {type(data).__name__}")

    # 5. CSV output
    stdout, stderr, rc = run(["query", "--query", "up", "--output", "csv"])
    if assert_exit_ok("query up csv", rc, stderr):
        record("query up csv non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 6. Aggregation query
    stdout, stderr, rc = run([
        "query", "--query", "count(up)", "--output", "json",
    ])
    if assert_exit_ok("query count(up) json", rc, stderr):
        assert_valid_json("query count(up) valid json", stdout)


# ---------------------------------------------------------------------------
# Tests: query-range
# ---------------------------------------------------------------------------

def test_query_range():
    print("[query-range]")

    # 1. Basic range query
    stdout, stderr, rc = run([
        "query-range", "--query", "up",
        "--start", "-15m", "--step", "60s",
    ])
    if assert_exit_ok("query-range up -15m", rc, stderr):
        record("query-range up non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 2. JSON output
    stdout, stderr, rc = run([
        "query-range", "--query", "up",
        "--start", "-15m", "--step", "60s",
        "--output", "json",
    ])
    if assert_exit_ok("query-range up json", rc, stderr):
        assert_valid_json("query-range up valid json", stdout)

    # 3. Range query with --no-pivot
    stdout, stderr, rc = run([
        "query-range", "--query", "up",
        "--start", "-15m", "--step", "60s",
        "--no-pivot",
    ])
    assert_exit_ok("query-range up --no-pivot", rc, stderr)

    # 4. Aggregated range query
    stdout, stderr, rc = run([
        "query-range",
        "--query", "sum(rate(container_cpu_usage_seconds_total[5m])) by (namespace)",
        "--start", "-15m", "--step", "60s",
        "--output", "json",
    ])
    if assert_exit_ok("query-range container_cpu json", rc, stderr):
        assert_valid_json("query-range container_cpu valid json", stdout)


# ---------------------------------------------------------------------------
# Tests: preset (instant + range)
# ---------------------------------------------------------------------------

def test_preset():
    print("[preset]")

    # 1. cluster_cpu_utilization — instant (default)
    stdout, stderr, rc = run(["preset", "--name", "cluster_cpu_utilization"])
    if assert_exit_ok("preset cluster_cpu_utilization", rc, stderr):
        record("preset cpu non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 2. cluster_memory_utilization
    stdout, stderr, rc = run(["preset", "--name", "cluster_memory_utilization"])
    assert_exit_ok("preset cluster_memory_utilization", rc, stderr)

    # 3. cluster_pod_status — JSON
    stdout, stderr, rc = run([
        "preset", "--name", "cluster_pod_status", "--output", "json",
    ])
    if assert_exit_ok("preset cluster_pod_status json", rc, stderr):
        data = assert_valid_json("preset pod_status valid json", stdout)
        if isinstance(data, dict):
            items = data.get("data", [])
            record("preset pod_status has results",
                   isinstance(items, list) and len(items) > 0,
                   "missing or empty 'data' array in envelope")
        elif isinstance(data, list):
            record("preset pod_status has results", len(data) > 0,
                   "empty JSON array" if len(data) == 0 else "")
        elif data is not None:
            record("preset pod_status has results", False,
                   f"expected JSON object or array but got {type(data).__name__}")

    # 4. cluster_node_readiness
    stdout, stderr, rc = run(["preset", "--name", "cluster_node_readiness"])
    assert_exit_ok("preset cluster_node_readiness", rc, stderr)

    # 5. namespace_cpu_usage — JSON
    stdout, stderr, rc = run([
        "preset", "--name", "namespace_cpu_usage", "--output", "json",
    ])
    if assert_exit_ok("preset namespace_cpu_usage json", rc, stderr):
        assert_valid_json("preset namespace_cpu valid json", stdout)

    # 6. namespace_memory_usage
    stdout, stderr, rc = run(["preset", "--name", "namespace_memory_usage"])
    assert_exit_ok("preset namespace_memory_usage", rc, stderr)

    # 7. pod_restarts_top10
    stdout, stderr, rc = run(["preset", "--name", "pod_restarts_top10"])
    assert_exit_ok("preset pod_restarts_top10", rc, stderr)

    # 8. Range mode: cluster_cpu_utilization with --start
    stdout, stderr, rc = run([
        "preset", "--name", "cluster_cpu_utilization",
        "--start", "-15m", "--step", "60s",
    ])
    if assert_exit_ok("preset cpu range --start -15m", rc, stderr):
        record("preset cpu range non-empty", len(stdout.strip()) > 0,
               "empty output" if not stdout.strip() else "")

    # 9. Range mode with JSON: cluster_pod_status
    stdout, stderr, rc = run([
        "preset", "--name", "cluster_pod_status",
        "--start", "-15m", "--step", "60s",
        "--output", "json",
    ])
    if assert_exit_ok("preset pod_status range json", rc, stderr):
        assert_valid_json("preset pod_status range valid json", stdout)


# ---------------------------------------------------------------------------
# Tests: error handling
# ---------------------------------------------------------------------------

def test_errors():
    print("[error handling]")

    # 1. query without --query
    _, _, rc = run(["query"])
    assert_exit_fail("query missing --query", rc)

    # 2. preset without --name
    _, _, rc = run(["preset"])
    assert_exit_fail("preset missing --name", rc)

    # 3. query with invalid PromQL
    _, _, rc = run(["query", "--query", "invalid{{{"])
    assert_exit_fail("query invalid PromQL", rc)

    # 4. query-range without --query
    _, _, rc = run(["query-range"])
    assert_exit_fail("query-range missing --query", rc)

    # 5. preset with unknown preset name (exits 0 but prints help)
    stdout, _, rc = run(["preset", "--name", "nonexistent_preset_xyz"])
    if assert_exit_ok("preset unknown name exits 0", rc):
        assert_contains("preset unknown name message", stdout, "Unknown preset")


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    if not os.path.isfile(BINARY):
        print(f"Binary not found: {BINARY}")
        print("Run 'make build' first, or use 'make e2e'.")
        sys.exit(2)

    print("=" * 60)
    print("E2E Smoke Tests")
    print("=" * 60)

    test_version()
    test_help_promql()
    test_discover()
    test_labels()
    test_query()
    test_query_range()
    test_preset()
    test_errors()

    print("=" * 60)
    print(f"Results: {passed} passed, {failed} failed")
    if errors:
        print("Failed tests:")
        for name in errors:
            print(f"  - {name}")
    print("=" * 60)

    sys.exit(1 if failed > 0 else 0)


if __name__ == "__main__":
    main()
