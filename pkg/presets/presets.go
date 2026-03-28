// Package presets provides pre-configured PromQL queries for cluster monitoring
// and MTV / Forklift migration tracking.
package presets

import (
	"fmt"
	"strings"
)

// Preset holds a named pre-configured PromQL query.
// Every preset works as both an instant query (default) and a range query
// (when --start is provided).
type Preset struct {
	Name        string
	Description string
	Query       string
}

// Presets is the list of all available preset queries.
var Presets = []Preset{
	// ========== General cluster health ==========

	{
		Name:        "cluster_cpu_utilization",
		Description: "Cluster CPU utilization percentage",
		Query:       `100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])))`,
	},
	{
		Name:        "cluster_memory_utilization",
		Description: "Cluster memory utilization percentage",
		Query:       "100 * (1 - sum(node_memory_MemAvailable_bytes{}) / sum(node_memory_MemTotal_bytes{}))",
	},
	{
		Name:        "cluster_pod_status",
		Description: "Pod counts by phase (Running, Pending, Failed, Succeeded, Unknown)",
		Query:       "sum(kube_pod_status_phase{}) by (phase)",
	},
	{
		Name:        "cluster_node_readiness",
		Description: "Node readiness status counts",
		Query:       `sum(kube_node_status_condition{condition="Ready"}) by (status)`,
	},

	// ========== Namespace-level resource usage ==========

	{
		Name:        "namespace_cpu_usage",
		Description: "Top 10 namespaces by CPU usage (cores)",
		Query:       "topk(10, sort_desc(sum by (namespace)(rate(container_cpu_usage_seconds_total{}[5m]))))",
	},
	{
		Name:        "namespace_memory_usage",
		Description: "Top 10 namespaces by memory usage (bytes)",
		Query:       "topk(10, sort_desc(sum by (namespace)(container_memory_working_set_bytes{})))",
	},
	{
		Name:        "namespace_network_rx",
		Description: "Top 10 namespaces by network receive rate",
		Query:       "topk(10, sort_desc(sum by (namespace)(rate(container_network_receive_bytes_total{}[5m]))))",
	},
	{
		Name:        "namespace_network_tx",
		Description: "Top 10 namespaces by network transmit rate",
		Query:       "topk(10, sort_desc(sum by (namespace)(rate(container_network_transmit_bytes_total{}[5m]))))",
	},
	{
		Name:        "namespace_network_errors",
		Description: "Network errors + drops by namespace (top 10)",
		Query:       "topk(10, sum by (namespace)(rate(container_network_receive_errors_total{}[5m])) + sum by (namespace)(rate(container_network_transmit_errors_total{}[5m])))",
	},
	{
		Name:        "pod_restarts_top10",
		Description: "Top 10 pods by container restart count",
		Query:       "topk(10, sort_desc(sum by (namespace,pod)(kube_pod_container_status_restarts_total{})))",
	},

	// ========== MTV / Forklift migration status ==========

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
	{
		Name:        "mtv_migration_duration",
		Description: "Migration duration per plan (seconds)",
		Query:       "mtv_migration_duration_seconds",
	},
	{
		Name:        "mtv_avg_migration_duration",
		Description: "Average migration duration (seconds)",
		Query:       "avg(mtv_migrations_duration_seconds_sum{} / mtv_migrations_duration_seconds_count{})",
	},

	// ========== MTV data transfer & throughput ==========

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

	// ========== MTV migration pod network traffic ==========

	{
		Name:        "mtv_migration_pod_rx",
		Description: "Migration pod receive rate (bytes/sec, top 20, filter by namespace)",
		Query:       `topk(20, sort_desc(sum by (namespace,pod)(rate(container_network_receive_bytes_total{}[5m]))))`,
	},
	{
		Name:        "mtv_migration_pod_tx",
		Description: "Migration pod transmit rate (bytes/sec, top 20, filter by namespace)",
		Query:       `topk(20, sort_desc(sum by (namespace,pod)(rate(container_network_transmit_bytes_total{}[5m]))))`,
	},
	{
		Name:        "mtv_populator_cpu",
		Description: "Populator pod CPU usage rate (oVirt/OpenStack)",
		Query:       `topk(20, sort_desc(sum by (namespace,pod)(rate(container_cpu_usage_seconds_total{pod=~"populat.*"}[1m]))))`,
	},
	{
		Name:        "mtv_forklift_traffic",
		Description: "Forklift operator pod network traffic (bytes/sec)",
		Query:       `sum by (pod)(rate(container_network_receive_bytes_total{pod=~"forklift.*"}[5m]))`,
	},

	// ========== KubeVirt VMI live-migration ==========

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

	// ========== KubeVirt VM resource usage ==========

	{
		Name:        "vm_cpu_usage",
		Description: "Top 10 VMs by CPU usage (cores)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(rate(kubevirt_vmi_cpu_usage_seconds_total{}[5m]))))",
	},
	{
		Name:        "vm_memory_usage",
		Description: "Top 10 VMs by resident memory (bytes)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(kubevirt_vmi_memory_resident_bytes{})))",
	},
	{
		Name:        "vm_network_rx",
		Description: "Top 10 VMs by network receive rate (bytes/sec)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(rate(kubevirt_vmi_network_receive_bytes_total{}[5m]))))",
	},
	{
		Name:        "vm_network_tx",
		Description: "Top 10 VMs by network transmit rate (bytes/sec)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(rate(kubevirt_vmi_network_transmit_bytes_total{}[5m]))))",
	},
	{
		Name:        "vm_storage_read",
		Description: "Top 10 VMs by disk read throughput (bytes/sec)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(rate(kubevirt_vmi_storage_read_traffic_bytes_total{}[5m]))))",
	},
	{
		Name:        "vm_storage_write",
		Description: "Top 10 VMs by disk write throughput (bytes/sec)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(rate(kubevirt_vmi_storage_write_traffic_bytes_total{}[5m]))))",
	},
	{
		Name:        "vm_storage_iops",
		Description: "Top 10 VMs by total IOPS (read + write)",
		Query:       "topk(10, sort_desc(sum by (namespace,name)(rate(kubevirt_vmi_storage_iops_read_total{}[5m]) + rate(kubevirt_vmi_storage_iops_write_total{}[5m]))))",
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
