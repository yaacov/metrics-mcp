package presets

import (
	"fmt"
	"strings"
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
	assertContains(t, p.Query, `{namespace="myns",pod=~"`)
}

func TestGetPreset_EmptySelector_SingleMetric(t *testing.T) {
	p, ok := GetPreset("namespace_network_rx", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	want := `topk(10, sort_desc(sum by (namespace)(rate(container_network_receive_bytes_total{namespace="myns",}[5m]))))`
	if p.Query != want {
		t.Fatalf("want:\n  %s\ngot:\n  %s", want, p.Query)
	}
}

func TestGetPreset_EmptySelector_TwoMetrics(t *testing.T) {
	p, ok := GetPreset("namespace_network_errors", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	want := `topk(10, sum by (namespace)(rate(container_network_receive_errors_total{namespace="myns",}[5m])) + sum by (namespace)(rate(container_network_transmit_errors_total{namespace="myns",}[5m])))`
	if p.Query != want {
		t.Fatalf("want:\n  %s\ngot:\n  %s", want, p.Query)
	}
}

func TestGetPreset_DoesNotMutatePresetMap(t *testing.T) {
	before, _ := GetPreset("namespace_network_errors", "")
	GetPreset("namespace_network_errors", "injected-ns")
	after, _ := GetPreset("namespace_network_errors", "")

	if before.Query != after.Query {
		t.Fatalf("preset map was mutated: before=%s after=%s", before.Query, after.Query)
	}
}

func TestGetPreset_SimpleMetricWithNamespace(t *testing.T) {
	p, ok := GetPreset("mtv_net_throughput", "myns")
	if !ok {
		t.Fatal("expected preset to exist")
	}
	assertContains(t, p.Query, `{namespace="myns"}`)
}

func TestGetPreset_GeneralClusterPresets(t *testing.T) {
	names := []string{
		"cluster_cpu_utilization",
		"cluster_memory_utilization",
		"cluster_pod_status",
		"cluster_node_readiness",
		"namespace_cpu_usage",
		"namespace_memory_usage",
		"pod_restarts_top10",
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			_, ok := GetPreset(name, "")
			if !ok {
				t.Fatal("expected preset to exist")
			}
		})
	}
}

// TestGetPreset_AllPresets_NamespaceInjection verifies that every registered
// preset produces a structurally valid query when a namespace is applied.
func TestGetPreset_AllPresets_NamespaceInjection(t *testing.T) {
	const ns = "test-ns"
	nsSelector := fmt.Sprintf(`namespace="%s"`, ns)

	for _, orig := range Presets {
		t.Run(orig.Name, func(t *testing.T) {
			p, ok := GetPreset(orig.Name, ns)
			if !ok {
				t.Fatal("preset should exist")
			}

			if !strings.Contains(p.Query, nsSelector) {
				t.Fatalf("namespace selector missing from query:\n  %s", p.Query)
			}

			if open, close := strings.Count(p.Query, "{"), strings.Count(p.Query, "}"); open != close {
				t.Fatalf("unbalanced braces (%d open, %d close) in query:\n  %s", open, close, p.Query)
			}

			if open, close := strings.Count(p.Query, "("), strings.Count(p.Query, ")"); open != close {
				t.Fatalf("unbalanced parens (%d open, %d close) in query:\n  %s", open, close, p.Query)
			}

			isComplex := strings.ContainsAny(orig.Query, "()+")
			if isComplex {
				suffix := fmt.Sprintf(`{namespace="%s"}`, ns)
				if strings.HasSuffix(p.Query, suffix) {
					t.Fatalf("namespace selector appended at end of complex expression:\n  %s", p.Query)
				}
			}

			for i := 0; i < len(p.Query); i++ {
				if p.Query[i] == '{' {
					depth := 1
					j := i + 1
					for j < len(p.Query) && depth > 0 {
						if p.Query[j] == '{' {
							depth++
						} else if p.Query[j] == '}' {
							depth--
						}
						j++
					}
					block := p.Query[i:j]
					if !strings.Contains(block, nsSelector) {
						t.Fatalf("selector block without namespace: %s\nin query: %s", block, p.Query)
					}
					i = j - 1
				}
			}
		})
	}
}

// TestGetPreset_AllPresets_NoNamespace verifies queries are unchanged without namespace.
func TestGetPreset_AllPresets_NoNamespace(t *testing.T) {
	for _, orig := range Presets {
		t.Run(orig.Name, func(t *testing.T) {
			p, ok := GetPreset(orig.Name, "")
			if !ok {
				t.Fatal("preset should exist")
			}
			if p.Query != orig.Query {
				t.Fatalf("query changed without namespace:\n  want: %s\n  got:  %s", orig.Query, p.Query)
			}
		})
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected query to contain %q\ngot: %s", needle, haystack)
	}
}
