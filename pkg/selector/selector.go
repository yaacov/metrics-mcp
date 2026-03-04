// Package selector implements Kubernetes-style label selectors for filtering
// Prometheus query results. It supports equality (=, ==), inequality (!=),
// regex (=~), and negative regex (!~) operators.
package selector

import (
	"fmt"
	"regexp"
	"strings"
)

// Operator represents a label matching operator.
type Operator int

const (
	OpEqual    Operator = iota // = or ==
	OpNotEqual                 // !=
	OpRegex                    // =~
	OpNotRegex                 // !~
)

// Requirement is a single label matcher: key <op> value.
type Requirement struct {
	Key   string
	Op    Operator
	Value string
	re    *regexp.Regexp // compiled regex for OpRegex / OpNotRegex
}

// Selector is a list of requirements with AND semantics.
type Selector []Requirement

// Parse parses a comma-separated label selector string into a Selector.
// Supported operators: =, ==, !=, =~, !~
//
// Examples:
//
//	"namespace=mtv-test"
//	"namespace=mtv-test,pod=~virt-v2v.*"
//	"status!=failed,pod!~test.*"
func Parse(s string) (Selector, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	sel := make(Selector, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		req, err := parseRequirement(part)
		if err != nil {
			return nil, err
		}
		sel = append(sel, req)
	}
	return sel, nil
}

// parseRequirement parses a single "key<op>value" expression.
// Operator detection order matters: =~ and !~ before != and ==.
func parseRequirement(expr string) (Requirement, error) {
	for _, op := range []struct {
		token string
		op    Operator
	}{
		{"=~", OpRegex},
		{"!~", OpNotRegex},
		{"!=", OpNotEqual},
		{"==", OpEqual},
		{"=", OpEqual},
	} {
		idx := strings.Index(expr, op.token)
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(expr[:idx])
		value := strings.TrimSpace(expr[idx+len(op.token):])
		if key == "" {
			return Requirement{}, fmt.Errorf("selector: empty key in %q", expr)
		}
		req := Requirement{Key: key, Op: op.op, Value: value}
		if op.op == OpRegex || op.op == OpNotRegex {
			re, err := regexp.Compile("^(?:" + value + ")$")
			if err != nil {
				return Requirement{}, fmt.Errorf("selector: invalid regex in %q: %w", expr, err)
			}
			req.re = re
		}
		return req, nil
	}
	return Requirement{}, fmt.Errorf("selector: no operator found in %q (supported: =, ==, !=, =~, !~)", expr)
}

// Matches reports whether a single label set satisfies all requirements.
func (s Selector) Matches(labels map[string]interface{}) bool {
	for _, req := range s {
		val, _ := labels[req.Key].(string)
		switch req.Op {
		case OpEqual:
			if val != req.Value {
				return false
			}
		case OpNotEqual:
			if val == req.Value {
				return false
			}
		case OpRegex:
			if !req.re.MatchString(val) {
				return false
			}
		case OpNotRegex:
			if req.re.MatchString(val) {
				return false
			}
		}
	}
	return true
}

// Filter returns only the result entries whose metric labels match all requirements.
// Each entry is expected to be a map with a "metric" key containing label pairs.
func (s Selector) Filter(results []interface{}) []interface{} {
	if len(s) == 0 {
		return results
	}
	filtered := make([]interface{}, 0, len(results))
	for _, r := range results {
		m, _ := r.(map[string]interface{})
		if m == nil {
			continue
		}
		labels, _ := m["metric"].(map[string]interface{})
		if s.Matches(labels) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
