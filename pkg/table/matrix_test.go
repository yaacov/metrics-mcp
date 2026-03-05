package table

import (
	"strings"
	"testing"
)

// makeMatrixResult builds a synthetic Prometheus matrix result entry.
func makeMatrixResult(labels map[string]interface{}, values ...interface{}) interface{} {
	return map[string]interface{}{
		"metric": labels,
		"values": values,
	}
}

func tsVal(ts float64, val string) interface{} {
	return []interface{}{ts, val}
}

func TestRenderMatrix_PivotIncludesName(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu", "pod": "pod-a"},
			tsVal(1000, "0.5"),
		),
		makeMatrixResult(
			map[string]interface{}{"__name__": "mem", "pod": "pod-a"},
			tsVal(1000, "1024"),
		),
	}

	out := RenderMatrix(results, Options{Markdown: true})

	if !strings.Contains(out, "cpu/pod-a") {
		t.Errorf("expected pivot column 'cpu/pod-a' in output:\n%s", out)
	}
	if !strings.Contains(out, "mem/pod-a") {
		t.Errorf("expected pivot column 'mem/pod-a' in output:\n%s", out)
	}
	if !strings.Contains(out, "TIMESTAMP") {
		t.Errorf("expected TIMESTAMP header in output:\n%s", out)
	}
}

func TestRenderMatrix_PivotSingleQuery(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "q1", "pod": "pod-a"},
			tsVal(1000, "0.5"),
		),
		makeMatrixResult(
			map[string]interface{}{"__name__": "q1", "pod": "pod-b"},
			tsVal(1000, "0.3"),
		),
	}

	out := RenderMatrix(results, Options{Markdown: true})

	if !strings.Contains(out, "q1/pod-a") {
		t.Errorf("expected pivot column 'q1/pod-a' in output:\n%s", out)
	}
	if !strings.Contains(out, "q1/pod-b") {
		t.Errorf("expected pivot column 'q1/pod-b' in output:\n%s", out)
	}
}

func TestRenderMatrix_NoPivotShowsMetricColumn(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu", "pod": "pod-a"},
			tsVal(1000, "0.5"),
		),
		makeMatrixResult(
			map[string]interface{}{"__name__": "mem", "pod": "pod-a"},
			tsVal(1000, "1024"),
		),
	}

	out := RenderMatrix(results, Options{NoPivot: true, Markdown: true})

	if !strings.Contains(out, "METRIC") {
		t.Errorf("expected METRIC header in no-pivot output:\n%s", out)
	}
	if !strings.Contains(out, "cpu") {
		t.Errorf("expected 'cpu' metric name in output:\n%s", out)
	}
	if !strings.Contains(out, "mem") {
		t.Errorf("expected 'mem' metric name in output:\n%s", out)
	}
	if !strings.Contains(out, "POD") {
		t.Errorf("expected POD label column in output:\n%s", out)
	}
}

func TestRenderMatrix_PivotNoLabels(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu"},
			tsVal(1000, "42"),
		),
		makeMatrixResult(
			map[string]interface{}{"__name__": "mem"},
			tsVal(1000, "99"),
		),
	}

	out := RenderMatrix(results, Options{Markdown: true})

	if !strings.Contains(out, "cpu") {
		t.Errorf("expected 'cpu' column in output:\n%s", out)
	}
	if !strings.Contains(out, "mem") {
		t.Errorf("expected 'mem' column in output:\n%s", out)
	}
}

func TestRenderMatrix_GroupByName(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu", "pod": "pod-a"},
			tsVal(1000, "0.5"),
		),
		makeMatrixResult(
			map[string]interface{}{"__name__": "mem", "pod": "pod-a"},
			tsVal(1000, "1024"),
		),
	}

	out := RenderMatrix(results, Options{GroupBy: "__name__", Markdown: true})

	if !strings.Contains(out, "--- __name__: cpu ---") {
		t.Errorf("expected group header for cpu in output:\n%s", out)
	}
	if !strings.Contains(out, "--- __name__: mem ---") {
		t.Errorf("expected group header for mem in output:\n%s", out)
	}
}
