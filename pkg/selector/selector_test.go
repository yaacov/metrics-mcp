package selector

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	sel, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sel) != 0 {
		t.Fatalf("expected empty selector, got %d requirements", len(sel))
	}
}

func TestParseOperators(t *testing.T) {
	tests := []struct {
		input string
		key   string
		op    Operator
		value string
	}{
		{"namespace=mtv-test", "namespace", OpEqual, "mtv-test"},
		{"namespace==mtv-test", "namespace", OpEqual, "mtv-test"},
		{"status!=failed", "status", OpNotEqual, "failed"},
		{"pod=~virt-v2v.*", "pod", OpRegex, "virt-v2v.*"},
		{"pod!~test.*", "pod", OpNotRegex, "test.*"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			sel, err := Parse(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sel) != 1 {
				t.Fatalf("expected 1 requirement, got %d", len(sel))
			}
			req := sel[0]
			if req.Key != tc.key {
				t.Errorf("key: got %q, want %q", req.Key, tc.key)
			}
			if req.Op != tc.op {
				t.Errorf("op: got %v, want %v", req.Op, tc.op)
			}
			if req.Value != tc.value {
				t.Errorf("value: got %q, want %q", req.Value, tc.value)
			}
		})
	}
}

func TestParseMultiple(t *testing.T) {
	sel, err := Parse("namespace=prod,pod=~nginx.*,status!=failed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sel) != 3 {
		t.Fatalf("expected 3 requirements, got %d", len(sel))
	}
	if sel[0].Key != "namespace" || sel[0].Op != OpEqual {
		t.Errorf("req[0]: got %q %v, want namespace =", sel[0].Key, sel[0].Op)
	}
	if sel[1].Key != "pod" || sel[1].Op != OpRegex {
		t.Errorf("req[1]: got %q %v, want pod =~", sel[1].Key, sel[1].Op)
	}
	if sel[2].Key != "status" || sel[2].Op != OpNotEqual {
		t.Errorf("req[2]: got %q %v, want status !=", sel[2].Key, sel[2].Op)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []string{
		"noop",      // no operator
		"=value",    // empty key
		"pod=~[bad", // invalid regex
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Fatalf("expected error for %q", input)
			}
		})
	}
}

func TestMatchesEqual(t *testing.T) {
	sel, _ := Parse("namespace=prod")
	if !sel.Matches(map[string]interface{}{"namespace": "prod"}) {
		t.Error("should match namespace=prod")
	}
	if sel.Matches(map[string]interface{}{"namespace": "dev"}) {
		t.Error("should not match namespace=dev")
	}
	if sel.Matches(map[string]interface{}{}) {
		t.Error("should not match missing label")
	}
}

func TestMatchesNotEqual(t *testing.T) {
	sel, _ := Parse("status!=failed")
	if !sel.Matches(map[string]interface{}{"status": "running"}) {
		t.Error("should match status=running")
	}
	if sel.Matches(map[string]interface{}{"status": "failed"}) {
		t.Error("should not match status=failed")
	}
	if !sel.Matches(map[string]interface{}{}) {
		t.Error("should match missing label (empty != failed)")
	}
}

func TestMatchesRegex(t *testing.T) {
	sel, _ := Parse("pod=~virt-v2v.*")
	if !sel.Matches(map[string]interface{}{"pod": "virt-v2v-abc"}) {
		t.Error("should match virt-v2v-abc")
	}
	if sel.Matches(map[string]interface{}{"pod": "nginx-123"}) {
		t.Error("should not match nginx-123")
	}
}

func TestMatchesNotRegex(t *testing.T) {
	sel, _ := Parse("pod!~test.*")
	if !sel.Matches(map[string]interface{}{"pod": "nginx-123"}) {
		t.Error("should match nginx-123")
	}
	if sel.Matches(map[string]interface{}{"pod": "test-pod"}) {
		t.Error("should not match test-pod")
	}
}

func TestMatchesMultipleAND(t *testing.T) {
	sel, _ := Parse("namespace=prod,status!=failed")
	if !sel.Matches(map[string]interface{}{"namespace": "prod", "status": "running"}) {
		t.Error("should match both conditions")
	}
	if sel.Matches(map[string]interface{}{"namespace": "prod", "status": "failed"}) {
		t.Error("should not match when one condition fails")
	}
	if sel.Matches(map[string]interface{}{"namespace": "dev", "status": "running"}) {
		t.Error("should not match when other condition fails")
	}
}

func TestMatchesEmptySelector(t *testing.T) {
	sel, _ := Parse("")
	if !sel.Matches(map[string]interface{}{"anything": "value"}) {
		t.Error("empty selector should match everything")
	}
}

func TestFilter(t *testing.T) {
	results := []interface{}{
		map[string]interface{}{
			"metric": map[string]interface{}{"namespace": "prod", "pod": "nginx-1"},
			"value":  []interface{}{1234567890.0, "42"},
		},
		map[string]interface{}{
			"metric": map[string]interface{}{"namespace": "dev", "pod": "nginx-2"},
			"value":  []interface{}{1234567890.0, "10"},
		},
		map[string]interface{}{
			"metric": map[string]interface{}{"namespace": "prod", "pod": "test-pod"},
			"value":  []interface{}{1234567890.0, "5"},
		},
	}

	sel, _ := Parse("namespace=prod")
	filtered := sel.Filter(results)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 results, got %d", len(filtered))
	}

	sel2, _ := Parse("namespace=prod,pod!~test.*")
	filtered2 := sel2.Filter(results)
	if len(filtered2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered2))
	}
	m := filtered2[0].(map[string]interface{})
	metric := m["metric"].(map[string]interface{})
	if metric["pod"] != "nginx-1" {
		t.Errorf("expected pod=nginx-1, got %v", metric["pod"])
	}
}

func TestFilterEmptySelector(t *testing.T) {
	results := []interface{}{
		map[string]interface{}{"metric": map[string]interface{}{"a": "b"}},
	}
	sel, _ := Parse("")
	filtered := sel.Filter(results)
	if len(filtered) != len(results) {
		t.Errorf("empty selector should return all results")
	}
}

func TestFilterNilEntries(t *testing.T) {
	results := []interface{}{nil, "not-a-map"}
	sel, _ := Parse("key=value")
	filtered := sel.Filter(results)
	if len(filtered) != 0 {
		t.Errorf("expected 0 results for malformed entries, got %d", len(filtered))
	}
}
