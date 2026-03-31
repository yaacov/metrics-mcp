package table

import (
	"strings"
	"testing"
)

func makeVectorResult(labels map[string]interface{}, ts float64, val string) interface{} {
	return map[string]interface{}{
		"metric": labels,
		"value":  []interface{}{ts, val},
	}
}

func TestRenderVector_CSV(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "up", "job": "prometheus"},
			1000, "1",
		),
	}

	out := RenderVector(results, Options{Format: "csv"})

	if !strings.Contains(out, "METRIC,JOB,TIMESTAMP,VALUE") {
		t.Errorf("expected CSV header row in output:\n%s", out)
	}
	if !strings.Contains(out, "up,prometheus,") {
		t.Errorf("expected CSV data row in output:\n%s", out)
	}
}

func TestRenderVector_TSV(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "up", "job": "prometheus"},
			1000, "1",
		),
	}

	out := RenderVector(results, Options{Format: "tsv"})

	if !strings.Contains(out, "METRIC\tJOB\tTIMESTAMP\tVALUE") {
		t.Errorf("expected TSV header row in output:\n%s", out)
	}
	if !strings.Contains(out, "up\tprometheus\t") {
		t.Errorf("expected TSV data row in output:\n%s", out)
	}
}

func TestRenderVector_NoHeaders(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "up"},
			1000, "1",
		),
	}

	out := RenderVector(results, Options{Format: "csv", NoHeaders: true})

	if strings.Contains(out, "METRIC") {
		t.Errorf("expected no header row when NoHeaders is set, got:\n%s", out)
	}
	if !strings.Contains(out, "up,") {
		t.Errorf("expected data row in output:\n%s", out)
	}
}

func TestRenderMatrix_CSV_Pivot(t *testing.T) {
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

	out := RenderMatrix(results, Options{Format: "csv"})

	if !strings.Contains(out, "TIMESTAMP,") {
		t.Errorf("expected CSV TIMESTAMP header in output:\n%s", out)
	}
	if !strings.Contains(out, "cpu/pod-a") {
		t.Errorf("expected pivot column 'cpu/pod-a' in CSV output:\n%s", out)
	}
}

func TestRenderMatrix_TSV_NoPivot(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu", "pod": "pod-a"},
			tsVal(1000, "0.5"),
		),
	}

	out := RenderMatrix(results, Options{Format: "tsv", NoPivot: true})

	if !strings.Contains(out, "METRIC\tPOD\tTIMESTAMP\tVALUE") {
		t.Errorf("expected TSV header in no-pivot output:\n%s", out)
	}
}

func TestRenderMatrix_NoHeaders_Pivot(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu"},
			tsVal(1000, "42"),
		),
	}

	out := RenderMatrix(results, Options{Format: "tsv", NoHeaders: true})

	if strings.Contains(out, "TIMESTAMP") {
		t.Errorf("expected no header row when NoHeaders is set, got:\n%s", out)
	}
}

func TestRenderVector_TSV_RawValues(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "bytes_total", "job": "node"},
			1710000000.123, "1500000",
		),
		makeVectorResult(
			map[string]interface{}{"__name__": "tiny_rate", "job": "node"},
			1710000000.123, "0.00042",
		),
	}

	out := RenderVector(results, Options{Format: "tsv"})

	if strings.Contains(out, " M") || strings.Contains(out, " k") {
		t.Errorf("TSV must not contain SI suffixes, got:\n%s", out)
	}
	if !strings.Contains(out, "1500000") {
		t.Errorf("expected raw value 1500000 in TSV output:\n%s", out)
	}
	if !strings.Contains(out, "0.00042") {
		t.Errorf("expected raw value 0.00042 in TSV output:\n%s", out)
	}
	if !strings.Contains(out, "1710000000.123") {
		t.Errorf("expected raw Unix timestamp in TSV output:\n%s", out)
	}
}

func TestRenderVector_CSV_RawValues(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "bytes_total"},
			1710000000, "2500000",
		),
	}

	out := RenderVector(results, Options{Format: "csv"})

	if strings.Contains(out, " M") || strings.Contains(out, " k") {
		t.Errorf("CSV must not contain SI suffixes, got:\n%s", out)
	}
	if !strings.Contains(out, "2500000") {
		t.Errorf("expected raw value 2500000 in CSV output:\n%s", out)
	}
}

func TestRenderVector_Markdown_HumanValues(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "bytes_total"},
			1000, "1500000.5",
		),
	}

	out := RenderVector(results, Options{Format: "markdown"})

	if !strings.Contains(out, "1.5 M") {
		t.Errorf("markdown should use human-readable SI values, got:\n%s", out)
	}
	if !strings.Contains(out, "1970-01-01") {
		t.Errorf("markdown should use human-readable timestamps, got:\n%s", out)
	}
}

func TestRenderMatrix_TSV_Pivot_RawValues(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "cpu", "pod": "pod-a"},
			tsVal(1710000000, "50000"),
		),
	}

	out := RenderMatrix(results, Options{Format: "tsv"})

	if strings.Contains(out, " k") {
		t.Errorf("TSV pivot must not contain SI suffixes, got:\n%s", out)
	}
	if !strings.Contains(out, "50000") {
		t.Errorf("expected raw value 50000 in TSV pivot output:\n%s", out)
	}
	if !strings.Contains(out, "1710000000") {
		t.Errorf("expected raw Unix timestamp in TSV pivot output:\n%s", out)
	}
}

func TestRenderMatrix_TSV_NoPivot_RawValues(t *testing.T) {
	results := []interface{}{
		makeMatrixResult(
			map[string]interface{}{"__name__": "mem", "pod": "pod-a"},
			tsVal(1710000000, "0.00005"),
		),
	}

	out := RenderMatrix(results, Options{Format: "tsv", NoPivot: true})

	if strings.Contains(out, " u") || strings.Contains(out, " µ") {
		t.Errorf("TSV no-pivot must not contain SI suffixes, got:\n%s", out)
	}
	if !strings.Contains(out, "0.00005") {
		t.Errorf("expected raw value 0.00005 in TSV no-pivot output:\n%s", out)
	}
	if !strings.Contains(out, "1710000000") {
		t.Errorf("expected raw Unix timestamp in TSV no-pivot output:\n%s", out)
	}
}

func TestRenderVector_NoHeaders_IgnoredForMarkdown(t *testing.T) {
	results := []interface{}{
		makeVectorResult(
			map[string]interface{}{"__name__": "up"},
			1000, "1",
		),
	}

	out := RenderVector(results, Options{Format: "markdown", NoHeaders: true})

	if !strings.Contains(out, "METRIC") {
		t.Errorf("markdown must retain headers even when NoHeaders is set, got:\n%s", out)
	}
}
