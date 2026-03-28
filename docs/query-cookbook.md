# Query Cookbook

Ready-to-use queries for common monitoring tasks on OpenShift clusters with ODF, OVN-Kubernetes, KubeVirt, and Forklift/MTV. All examples use CLI syntax.

For migration-specific and VM monitoring recipes see [VM & Migration Cookbook](vm-migration-cookbook.md). For command and flag details see [CLI Usage](cli-usage.md). For PromQL syntax see [PromQL Reference](promql-reference.md). For metric names and labels see [Metrics Reference](metrics-reference.md).

---

## Quick Health Dashboard

Run these five commands for a fast cluster overview:

```bash
kubectl metrics preset --name cluster_cpu_utilization
kubectl metrics preset --name cluster_memory_utilization
kubectl metrics query  --query 'ceph_health_status'
kubectl metrics preset --name namespace_network_rx
kubectl metrics preset --name mtv_migration_status
```

---

## Storage (Ceph / ODF)

### Cluster health

```bash
kubectl metrics query --query 'ceph_health_status'
```

Result: **0** = OK, **1** = WARN, **2** = ERR.

### Capacity

```bash
kubectl metrics query --query 'ceph_cluster_total_bytes'
kubectl metrics query --query 'ceph_cluster_total_used_bytes'
```

### Pool usage percentage

```bash
kubectl metrics query --query 'ceph_pool_percent_used * 100'
```

### Pool I/O rates

```bash
kubectl metrics query --query 'rate(ceph_pool_rd[5m])'
kubectl metrics query --query 'rate(ceph_pool_wr[5m])'
```

### OSD operation latency

```bash
kubectl metrics query --query 'rate(ceph_osd_op_latency_sum[5m]) / rate(ceph_osd_op_latency_count[5m])'
```

### Placement group health

```bash
kubectl metrics query --query 'ceph_pg_total'
kubectl metrics query --query 'ceph_pg_degraded'
```

---

## Network Traffic

### By namespace (presets)

```bash
kubectl metrics preset --name namespace_network_rx
kubectl metrics preset --name namespace_network_tx
kubectl metrics preset --name namespace_network_errors
```

### By pod in a specific namespace

Replace `TARGET_NAMESPACE` with the actual namespace:

```bash
kubectl metrics query \
  --query 'topk(10, sort_desc(sum by (pod)(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE"}[5m]))))'

kubectl metrics query \
  --query 'topk(10, sort_desc(sum by (pod)(rate(container_network_transmit_bytes_total{namespace="TARGET_NAMESPACE"}[5m]))))'
```

### Node-level throughput

```bash
kubectl metrics query \
  --query 'instance:node_network_receive_bytes_excluding_lo:rate1m + instance:node_network_transmit_bytes_excluding_lo:rate1m'
```

### RX and TX as a range query (combined)

```bash
kubectl metrics query-range \
  --query 'sum by (namespace)(rate(container_network_receive_bytes_total[5m]))' \
  --query 'sum by (namespace)(rate(container_network_transmit_bytes_total[5m]))' \
  --name rx_bytes_per_sec --name tx_bytes_per_sec \
  --start "-1h"
```

---

## Pod and Container Statistics

### Pod count by namespace

```bash
kubectl metrics query --query 'topk(15, count by (namespace)(kube_pod_info))'
```

### Pod phase summary

```bash
kubectl metrics preset --name cluster_pod_status
```

### CPU and memory by namespace

```bash
kubectl metrics preset --name namespace_cpu_usage
kubectl metrics preset --name namespace_memory_usage
```

### Container restarts

```bash
kubectl metrics preset --name pod_restarts_top10
```

---

## MTV / Forklift Migrations

For per-plan monitoring, per-disk I/O, and KubeVirt VM metrics see the [VM & Migration Cookbook](vm-migration-cookbook.md).

### Migration and plan status

```bash
kubectl metrics preset --name mtv_migration_status
kubectl metrics preset --name mtv_plan_status
```

### Duration

```bash
kubectl metrics preset --name mtv_migration_duration
kubectl metrics preset --name mtv_avg_migration_duration
```

### Data transfer and throughput

```bash
kubectl metrics preset --name mtv_data_transferred
kubectl metrics preset --name mtv_net_throughput
kubectl metrics preset --name mtv_storage_throughput
```

### Migration pod network traffic

```bash
kubectl metrics preset --name mtv_migration_pod_rx --namespace openshift-mtv
kubectl metrics preset --name mtv_migration_pod_tx --namespace openshift-mtv
```

### Populator pod CPU (oVirt / OpenStack)

```bash
kubectl metrics preset --name mtv_populator_cpu
```

### Forklift operator traffic

```bash
kubectl metrics preset --name mtv_forklift_traffic
```

### KubeVirt VMI live-migration

```bash
kubectl metrics preset --name mtv_vmi_migrations_pending
kubectl metrics preset --name mtv_vmi_migrations_running
```

### Migration alerts

```bash
kubectl metrics query --query 'mtv_plan_alert_status'
```

### Filtering by labels

Use PromQL label matchers or the `--selector` flag to narrow results:

```bash
# vSphere migrations only
kubectl metrics query --query 'mtv_migration_data_transferred_bytes' \
  --selector 'provider=vsphere'

# Cold migrations on oVirt
kubectl metrics query \
  --query 'mtv_migration_data_transferred_bytes{provider="ovirt", mode="Cold"}'

# Failed migrations
kubectl metrics query \
  --query 'mtv_migrations_status_total{status="Failed"}'
```

### Grouping

```bash
# Total data transferred by provider
kubectl metrics query \
  --query 'sum by (provider)(mtv_migration_data_transferred_bytes)'

# Migration counts by status and provider
kubectl metrics query \
  --query 'sum by (status, provider)(mtv_migrations_status_total)'

# Average duration by provider
kubectl metrics query \
  --query 'avg by (provider)(mtv_migration_duration_seconds)'

# Workload status by plan
kubectl metrics query \
  --query 'sum by (plan, status)(mtv_workload_migrations_status_total)'
```

### Range trends

Promote any preset to a time-series by adding `--start`:

```bash
kubectl metrics preset --name mtv_migration_status --start "-2h" --step "30s"
kubectl metrics preset --name mtv_net_throughput --start "-1h"
```

---

See [VM & Migration Cookbook](vm-migration-cookbook.md) for migration pod discovery, per-disk I/O queries, KubeVirt VM monitoring, and advanced techniques for short-lived pods and historical data.
