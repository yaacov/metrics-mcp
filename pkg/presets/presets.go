// Package presets provides pre-configured PromQL queries for MTV / Forklift monitoring.
package presets

import (
	"fmt"
	"strings"
)

const migrationPodRegex = `.*virt-v2v.*|.*populator.*|.*importer.*|.*cdi-upload.*`

// Preset holds a named pre-configured PromQL query.
type Preset struct {
	Name        string
	Description string
	Query       string
	Type        string // "instant" (default/empty) or "range"
	Start       string // default start offset for range presets, e.g. "-1h"
	Step        string // default step for range presets, e.g. "60s"
}

// IsRange returns true if this preset is a range query.
func (p Preset) IsRange() bool {
	return p.Type == "range"
}

// DisplayType returns "[instant]" or "[range]" for listing output.
func (p Preset) DisplayType() string {
	if p.IsRange() {
		return "[range]"
	}
	return "[instant]"
}

// Presets is the list of all available preset queries.
var Presets = []Preset{
	// MTV migration status
	{
		Name:        "mtv_migration_status",
		Description: "Migration counts by status (succeeded / failed / running)",
		Query:       "mtv_migrations_status_total",
	},
	{
		Name:        "mtv_plan_status",
		Description: "Plan-level status counts",
		Query:       "mtv_plans_status",
	},

	// MTV data transfer & throughput
	{
		Name:        "mtv_data_transferred",
		Description: "Total bytes migrated per plan",
		Query:       "mtv_migration_data_transferred_bytes",
	},
	{
		Name:        "mtv_net_throughput",
		Description: "Migration network throughput",
		Query:       "mtv_migration_net_throughput",
	},
	{
		Name:        "mtv_storage_throughput",
		Description: "Migration storage throughput",
		Query:       "mtv_migration_storage_throughput",
	},
	{
		Name:        "mtv_migration_duration",
		Description: "Migration duration per plan (seconds)",
		Query:       "mtv_migration_duration_seconds",
	},

	// Migration pod network traffic
	{
		Name:        "mtv_migration_pod_rx",
		Description: "Migration pod receive rate (bytes/sec, top 20)",
		Query:       fmt.Sprintf(`topk(20, sort_desc(sum by (namespace,pod)(rate(container_network_receive_bytes_total{pod=~"%s"}[5m]))))`, migrationPodRegex),
	},
	{
		Name:        "mtv_migration_pod_tx",
		Description: "Migration pod transmit rate (bytes/sec, top 20)",
		Query:       fmt.Sprintf(`topk(20, sort_desc(sum by (namespace,pod)(rate(container_network_transmit_bytes_total{pod=~"%s"}[5m]))))`, migrationPodRegex),
	},
	{
		Name:        "mtv_forklift_traffic",
		Description: "Forklift operator pod network traffic (bytes/sec)",
		Query:       `sum by (pod)(rate(container_network_receive_bytes_total{pod=~"forklift.*"}[5m]))`,
	},

	// General namespace network
	{
		Name:        "mtv_namespace_network_rx",
		Description: "Top 10 namespaces by network receive rate",
		Query:       "topk(10, sort_desc(sum by (namespace)(rate(container_network_receive_bytes_total{}[5m]))))",
	},
	{
		Name:        "mtv_namespace_network_tx",
		Description: "Top 10 namespaces by network transmit rate",
		Query:       "topk(10, sort_desc(sum by (namespace)(rate(container_network_transmit_bytes_total{}[5m]))))",
	},
	{
		Name:        "mtv_network_errors",
		Description: "Network errors + drops by namespace (top 10)",
		Query:       "topk(10, sum by (namespace)(rate(container_network_receive_errors_total{}[5m])) + sum by (namespace)(rate(container_network_transmit_errors_total{}[5m])))",
	},

	// KubeVirt VMI live-migration
	{
		Name:        "mtv_vmi_migrations_pending",
		Description: "KubeVirt VMI migrations in pending phase",
		Query:       "kubevirt_vmi_migrations_in_pending_phase",
	},
	{
		Name:        "mtv_vmi_migrations_running",
		Description: "KubeVirt VMI migrations in running phase",
		Query:       "kubevirt_vmi_migrations_in_running_phase",
	},

	// ---- Range presets (time-series trends) ----

	{
		Name:        "mtv_net_throughput_over_time",
		Description: "Migration network throughput trend",
		Query:       "mtv_migration_net_throughput",
		Type:        "range",
		Start:       "-1h",
		Step:        "60s",
	},
	{
		Name:        "mtv_storage_throughput_over_time",
		Description: "Migration storage throughput trend",
		Query:       "mtv_migration_storage_throughput",
		Type:        "range",
		Start:       "-1h",
		Step:        "60s",
	},
	{
		Name:        "mtv_data_transferred_over_time",
		Description: "Data transfer progress over time",
		Query:       "mtv_migration_data_transferred_bytes",
		Type:        "range",
		Start:       "-6h",
		Step:        "5m",
	},
	{
		Name:        "mtv_migration_status_over_time",
		Description: "Migration status counts over time",
		Query:       "mtv_migrations_status_total",
		Type:        "range",
		Start:       "-6h",
		Step:        "5m",
	},
	{
		Name:        "mtv_migration_pod_rx_over_time",
		Description: "Migration pod receive rate trend (top 20)",
		Query:       fmt.Sprintf(`topk(20, sort_desc(sum by (namespace,pod)(rate(container_network_receive_bytes_total{pod=~"%s"}[5m]))))`, migrationPodRegex),
		Type:        "range",
		Start:       "-1h",
		Step:        "60s",
	},
	{
		Name:        "mtv_namespace_network_rx_over_time",
		Description: "Top 10 namespaces by RX rate trend",
		Query:       "topk(10, sort_desc(sum by (namespace)(rate(container_network_receive_bytes_total{}[5m]))))",
		Type:        "range",
		Start:       "-1h",
		Step:        "60s",
	},
}

// presetMap provides O(1) lookup by name.
var presetMap map[string]Preset

func init() {
	presetMap = make(map[string]Preset, len(Presets))
	for _, p := range Presets {
		presetMap[p.Name] = p
	}
}

// ListPresets returns name+description for every preset.
func ListPresets() []Preset {
	return Presets
}

// GetPreset resolves a preset by name and applies an optional namespace filter.
// Returns the preset with the query adjusted for the namespace, and true.
// Returns a zero Preset and false if not found.
func GetPreset(name, namespace string) (Preset, bool) {
	p, ok := presetMap[name]
	if !ok {
		return Preset{}, false
	}

	if namespace != "" {
		if strings.Contains(p.Query, "{namespace}") {
			p.Query = strings.Replace(p.Query, "{namespace}", namespace, 1)
		} else if strings.Contains(p.Query, "{") {
			p.Query = strings.ReplaceAll(p.Query, "{", fmt.Sprintf(`{namespace="%s",`, namespace))
		} else {
			p.Query = fmt.Sprintf(`%s{namespace="%s"}`, p.Query, namespace)
		}
	}

	return p, true
}
