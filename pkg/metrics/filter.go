package metrics

import (
	"github.com/yaacov/kubectl-metrics/pkg/selector"
)

// FilterData applies a label selector to a raw Prometheus API response.
// It parses the selector string, filters the "data.result" slice in place,
// and returns the modified response. An empty selector is a no-op.
func FilterData(data map[string]interface{}, selectorStr string) (map[string]interface{}, error) {
	if selectorStr == "" || data == nil {
		return data, nil
	}

	sel, err := selector.Parse(selectorStr)
	if err != nil {
		return nil, err
	}
	if len(sel) == 0 {
		return data, nil
	}

	dataField, _ := data["data"].(map[string]interface{})
	if dataField == nil {
		return data, nil
	}
	resultSlice, _ := dataField["result"].([]interface{})
	if len(resultSlice) == 0 {
		return data, nil
	}

	dataField["result"] = sel.Filter(resultSlice)
	return data, nil
}
