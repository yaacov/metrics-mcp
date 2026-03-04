package presets

import (
	"testing"
)

func TestGetPreset_NotFound(t *testing.T) {
	_, ok := GetPreset("no_such_preset", "")
	if ok {
		t.Fatal("expected ok=false for unknown preset")
	}
}

func TestGetPreset_NoNamespace(t *testing.T) {
	p, ok := GetPreset("mtv_migration_status", "")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	if p.Query != "mtv_migrations_status_total" {
		t.Fatalf("query should be unchanged without namespace, got %s", p.Query)
	}
}

func TestGetPreset_SimpleMetric_AppendsSelector(t *testing.T) {
	p, ok := GetPreset("mtv_migration_status", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	want := `mtv_migrations_status_total{namespace="myns"}`
	if p.Query != want {
		t.Fatalf("want:\n  %s\ngot:\n  %s", want, p.Query)
	}
}

func TestGetPreset_SingleExistingSelector(t *testing.T) {
	p, ok := GetPreset("mtv_forklift_traffic", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	want := `sum by (pod)(rate(container_network_receive_bytes_total{namespace="myns",pod=~"forklift.*"}[5m]))`
	if p.Query != want {
		t.Fatalf("want:\n  %s\ngot:\n  %s", want, p.Query)
	}
}

func TestGetPreset_MultiMetricExistingSelector(t *testing.T) {
	p, ok := GetPreset("mtv_migration_pod_rx", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	if got := p.Query; got == "" {
		t.Fatal("query should not be empty")
	}
	// The query has one {pod=~"..."} selector; namespace should be injected.
	assertContains(t, p.Query, `{namespace="myns",pod=~"`)
}

func TestGetPreset_EmptySelector_SingleMetric(t *testing.T) {
	p, ok := GetPreset("mtv_namespace_network_rx", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	want := `topk(10, sort_desc(sum by (namespace)(rate(container_network_receive_bytes_total{namespace="myns",}[5m]))))`
	if p.Query != want {
		t.Fatalf("want:\n  %s\ngot:\n  %s", want, p.Query)
	}
}

func TestGetPreset_EmptySelector_TwoMetrics(t *testing.T) {
	p, ok := GetPreset("mtv_network_errors", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	want := `topk(10, sum by (namespace)(rate(container_network_receive_errors_total{namespace="myns",}[5m])) + sum by (namespace)(rate(container_network_transmit_errors_total{namespace="myns",}[5m])))`
	if p.Query != want {
		t.Fatalf("want:\n  %s\ngot:\n  %s", want, p.Query)
	}
}

func TestGetPreset_DoesNotMutatePresetMap(t *testing.T) {
	before, _ := GetPreset("mtv_network_errors", "")
	GetPreset("mtv_network_errors", "injected-ns")
	after, _ := GetPreset("mtv_network_errors", "")

	if before.Query != after.Query {
		t.Fatalf("preset map was mutated: before=%s after=%s", before.Query, after.Query)
	}
}

func TestGetPreset_RangePresetWithNamespace(t *testing.T) {
	p, ok := GetPreset("mtv_namespace_network_rx_over_time", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	assertContains(t, p.Query, `{namespace="myns",}`)
	if !p.IsRange() {
		t.Fatal("expected range preset")
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if idx := indexOf(haystack, needle); idx < 0 {
		t.Fatalf("expected query to contain %q\ngot: %s", needle, haystack)
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
