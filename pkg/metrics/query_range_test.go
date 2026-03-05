package metrics

import (
	"testing"

	ptable "github.com/yaacov/kubectl-metrics/pkg/table"
)

func TestBuildNamedQueries_WithNames(t *testing.T) {
	queries := []string{"rate(cpu[5m])", "memory_bytes"}
	names := []string{"cpu", "mem"}
	got := BuildNamedQueries(queries, names)

	if len(got) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(got))
	}
	if got[0].Name != "cpu" || got[0].Query != "rate(cpu[5m])" {
		t.Errorf("query 0: got %+v", got[0])
	}
	if got[1].Name != "mem" || got[1].Query != "memory_bytes" {
		t.Errorf("query 1: got %+v", got[1])
	}
}

func TestBuildNamedQueries_AutoNames(t *testing.T) {
	queries := []string{"up", "down", "sideways"}
	got := BuildNamedQueries(queries, nil)

	if len(got) != 3 {
		t.Fatalf("expected 3 queries, got %d", len(got))
	}
	for i, want := range []string{"q1", "q2", "q3"} {
		if got[i].Name != want {
			t.Errorf("query %d: name = %q, want %q", i, got[i].Name, want)
		}
	}
}

func TestBuildNamedQueries_PartialNames(t *testing.T) {
	queries := []string{"a", "b", "c"}
	names := []string{"alpha"}
	got := BuildNamedQueries(queries, names)

	if got[0].Name != "alpha" {
		t.Errorf("query 0: name = %q, want %q", got[0].Name, "alpha")
	}
	if got[1].Name != "q2" {
		t.Errorf("query 1: name = %q, want %q", got[1].Name, "q2")
	}
	if got[2].Name != "q3" {
		t.Errorf("query 2: name = %q, want %q", got[2].Name, "q3")
	}
}

func TestFlagStrSlice_String(t *testing.T) {
	flags := map[string]any{"query": "up"}
	got := FlagStrSlice(flags, "query")
	if len(got) != 1 || got[0] != "up" {
		t.Errorf("got %v, want [up]", got)
	}
}

func TestFlagStrSlice_Array(t *testing.T) {
	flags := map[string]any{
		"query": []interface{}{"rate(cpu[5m])", "memory_bytes"},
	}
	got := FlagStrSlice(flags, "query")
	if len(got) != 2 || got[0] != "rate(cpu[5m])" || got[1] != "memory_bytes" {
		t.Errorf("got %v", got)
	}
}

func TestFlagStrSlice_Missing(t *testing.T) {
	flags := map[string]any{}
	got := FlagStrSlice(flags, "query")
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestFlagStrSlice_GoStringSlice(t *testing.T) {
	input := []string{"rate(cpu[5m])", "memory_bytes"}
	flags := map[string]any{"query": input}
	got := FlagStrSlice(flags, "query")
	if len(got) != 2 || got[0] != "rate(cpu[5m])" || got[1] != "memory_bytes" {
		t.Errorf("got %v, want %v", got, input)
	}
}

func TestFormatResult_EmptyResult_Markdown(t *testing.T) {
	data := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     []interface{}{},
		},
	}
	got := FormatResult(data, "markdown", ptable.Options{})
	if got != "(no results)" {
		t.Errorf("markdown empty: got %q, want %q", got, "(no results)")
	}
}

func TestFormatResult_EmptyResult_JSON(t *testing.T) {
	data := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     []interface{}{},
		},
	}
	got := FormatResult(data, "json", ptable.Options{})
	if got != "[]" {
		t.Errorf("json empty: got %q, want %q", got, "[]")
	}
}

func TestFormatResult_EmptyResult_Raw(t *testing.T) {
	data := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     []interface{}{},
		},
	}
	got := FormatResult(data, "raw", ptable.Options{})
	if got == "" || got == "(no results)" {
		t.Errorf("raw empty: got %q, want valid JSON", got)
	}
	if !contains(got, `"status"`) || !contains(got, `"success"`) {
		t.Errorf("raw empty: expected full Prometheus response, got %q", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
