# Metrics and Labels Reference

Metric names, descriptions, and available labels for OpenShift clusters with ODF, OVN-Kubernetes, KubeVirt, and Forklift/MTV.

Use `kubectl metrics discover` and `kubectl metrics labels` to explore what is available on your cluster:

```bash
kubectl metrics discover --keyword ceph
kubectl metrics discover --keyword network --group-by-prefix
kubectl metrics labels --metric container_network_receive_bytes_total
```

For query recipes using these metrics see [Query Cookbook](query-cookbook.md) and [VM & Migration Cookbook](vm-migration-cookbook.md).

---

## Storage Metrics (Ceph / ODF)

### Metrics

| Metric | Description |
|--------|-------------|
| `ceph_health_status` | Overall cluster health (0 = OK, 1 = WARN, 2 = ERR) |
| `ceph_cluster_total_bytes` | Total cluster capacity |
| `ceph_cluster_total_used_bytes` | Used cluster capacity |
| `ceph_pool_percent_used` | Per-pool usage percentage |
| `ceph_pool_stored` | Bytes stored per pool |
| `ceph_pool_max_avail` | Available bytes per pool |
| `ceph_pool_rd` | Read IOPS per pool (counter) |
| `ceph_pool_wr` | Write IOPS per pool (counter) |
| `ceph_pool_rd_bytes` | Read bytes per pool (counter) |
| `ceph_pool_wr_bytes` | Write bytes per pool (counter) |
| `ceph_osd_op_latency_sum` / `ceph_osd_op_latency_count` | OSD operation latency (use as rate ratio) |
| `ceph_pg_total` | Total placement groups |
| `ceph_pg_active` | Active placement groups |
| `ceph_pg_degraded` | Degraded placement groups |
| `node_filesystem_avail_bytes` | Available filesystem bytes per node |
| `node_filesystem_size_bytes` | Total filesystem bytes per node |

### Labels

| Label | Description | Example values |
|-------|-------------|----------------|
| `pool_id` | Ceph pool identifier (pool-level metrics) | `1`, `2`, `3` |
| `ceph_daemon` | OSD daemon name (OSD-level metrics) | `osd.0`, `osd.1` |
| `namespace` | Storage operator namespace | `openshift-storage` |
| `managedBy` | Managing resource | `ocs-storagecluster` |
| `job` | Scrape job | `rook-ceph-mgr`, `rook-ceph-exporter` |

---

## Network Metrics

### Metrics

| Metric | Description |
|--------|-------------|
| `container_network_receive_bytes_total` | Bytes received per pod/namespace (counter) |
| `container_network_transmit_bytes_total` | Bytes transmitted per pod/namespace (counter) |
| `container_network_receive_errors_total` | Receive errors per pod/namespace (counter) |
| `container_network_transmit_errors_total` | Transmit errors per pod/namespace (counter) |
| `container_network_receive_packets_dropped_total` | Dropped receive packets (counter) |
| `container_network_transmit_packets_dropped_total` | Dropped transmit packets (counter) |
| `node_network_receive_bytes_total` | Bytes received per node/interface (counter) |
| `node_network_transmit_bytes_total` | Bytes transmitted per node/interface (counter) |
| `instance:node_network_receive_bytes_excluding_lo:rate1m` | Pre-computed node receive rate (excludes loopback) |
| `instance:node_network_transmit_bytes_excluding_lo:rate1m` | Pre-computed node transmit rate (excludes loopback) |

### Labels

| Label | Description | Example values |
|-------|-------------|----------------|
| `namespace` | Pod namespace | `openshift-storage`, `konveyor-forklift` |
| `pod` | Pod name | `forklift-controller-6df77f6bf5-jtt7q` |
| `interface` | Network interface (per-pod metrics) | `eth0` |
| `instance` | Node instance (node-level metrics) | `10.0.0.5:9100` |
| `node` | Node name (node-level metrics) | `worker-0` |

---

## Pod and Container Metrics

### Metrics

| Metric | Description |
|--------|-------------|
| `kube_pod_info` | Pod metadata (node, namespace, IPs, owner) |
| `kube_pod_status_phase` | Pod phase (Running, Pending, Failed, Succeeded) |
| `kube_pod_container_status_restarts_total` | Container restart count |
| `kube_pod_container_status_waiting_reason` | Waiting reason (CrashLoopBackOff, ImagePullBackOff, etc.) |
| `container_cpu_usage_seconds_total` | Container CPU usage (counter) |
| `container_memory_working_set_bytes` | Container memory usage (gauge) |
| `namespace:container_cpu_usage:sum` | Pre-aggregated CPU by namespace |
| `namespace:container_memory_usage_bytes:sum` | Pre-aggregated memory by namespace |

### Labels

| Label | Description | Example values |
|-------|-------------|----------------|
| `namespace` | Pod namespace | `konveyor-forklift`, `openshift-cnv` |
| `pod` | Pod name | `forklift-controller-6df77f6bf5-jtt7q` |
| `container` | Container name | `main`, `inventory` |
| `node` | Node the pod runs on | `worker-0`, `worker-1` |
| `phase` | Pod phase (on status metrics) | `Running`, `Pending`, `Failed`, `Succeeded` |
| `uid` | Pod UID | `793fb1cb-3e58-4eef-b95a-733f237365a3` |
| `created_by_kind` | Owner resource kind (on `kube_pod_info`) | `ReplicaSet`, `DaemonSet`, `StatefulSet` |
| `created_by_name` | Owner resource name (on `kube_pod_info`) | `forklift-controller-6df77f6bf5` |
| `host_ip` | Node IP (on `kube_pod_info`) | `192.168.0.77` |
| `pod_ip` | Pod IP (on `kube_pod_info`) | `10.129.3.3` |

---

## KubeVirt VM Metrics

Metrics exposed by KubeVirt for each running Virtual Machine Instance (VMI). Use `name` and `namespace` labels to target a specific VM.

### Metrics

#### CPU

| Metric | Description |
|--------|-------------|
| `kubevirt_vmi_cpu_usage_seconds_total` | Total CPU time consumed (counter) |
| `kubevirt_vmi_cpu_system_usage_seconds_total` | System (kernel) CPU time (counter) |
| `kubevirt_vmi_cpu_user_usage_seconds_total` | User-space CPU time (counter) |
| `kubevirt_vmi_vcpu_seconds_total` | Per-vCPU time by state (counter; labels: `id`, `state`) |
| `kubevirt_vmi_vcpu_count` | Number of vCPUs allocated |
| `kubevirt_vmi_vcpu_delay_seconds_total` | vCPU scheduling delay (counter) |
| `kubevirt_vmi_vcpu_wait_seconds_total` | vCPU wait time (counter) |

#### Memory

| Metric | Description |
|--------|-------------|
| `kubevirt_vmi_memory_resident_bytes` | Resident (RSS) memory (gauge) |
| `kubevirt_vmi_memory_available_bytes` | Memory available to the guest (gauge) |
| `kubevirt_vmi_memory_used_bytes` | Memory used by the guest (gauge) |
| `kubevirt_vmi_memory_usable_bytes` | Memory usable by the guest (gauge) |
| `kubevirt_vmi_memory_domain_bytes` | Total memory allocated to the domain (gauge) |
| `kubevirt_vmi_memory_cached_bytes` | Cached memory inside the guest (gauge) |
| `kubevirt_vmi_memory_unused_bytes` | Unused memory inside the guest (gauge) |
| `kubevirt_vmi_memory_swap_in_traffic_bytes` | Swap-in traffic (counter) |
| `kubevirt_vmi_memory_swap_out_traffic_bytes` | Swap-out traffic (counter) |
| `kubevirt_vmi_memory_pgmajfault_total` | Major page faults (counter) |
| `kubevirt_vmi_memory_pgminfault_total` | Minor page faults (counter) |
| `kubevirt_vmi_memory_actual_balloon_bytes` | Balloon device actual size (gauge) |

#### Network

| Metric | Description |
|--------|-------------|
| `kubevirt_vmi_network_receive_bytes_total` | Bytes received per interface (counter) |
| `kubevirt_vmi_network_transmit_bytes_total` | Bytes transmitted per interface (counter) |
| `kubevirt_vmi_network_receive_packets_total` | Packets received per interface (counter) |
| `kubevirt_vmi_network_transmit_packets_total` | Packets transmitted per interface (counter) |
| `kubevirt_vmi_network_receive_errors_total` | Receive errors per interface (counter) |
| `kubevirt_vmi_network_transmit_errors_total` | Transmit errors per interface (counter) |
| `kubevirt_vmi_network_receive_packets_dropped_total` | Dropped receive packets (counter) |
| `kubevirt_vmi_network_transmit_packets_dropped_total` | Dropped transmit packets (counter) |
| `kubevirt_vmi_network_traffic_bytes_total` | Total network traffic (counter) |

#### Storage

| Metric | Description |
|--------|-------------|
| `kubevirt_vmi_storage_read_traffic_bytes_total` | Bytes read per drive (counter) |
| `kubevirt_vmi_storage_write_traffic_bytes_total` | Bytes written per drive (counter) |
| `kubevirt_vmi_storage_iops_read_total` | Read operations per drive (counter) |
| `kubevirt_vmi_storage_iops_write_total` | Write operations per drive (counter) |
| `kubevirt_vmi_storage_read_times_seconds_total` | Total read latency per drive (counter) |
| `kubevirt_vmi_storage_write_times_seconds_total` | Total write latency per drive (counter) |
| `kubevirt_vmi_storage_flush_requests_total` | Flush requests per drive (counter) |
| `kubevirt_vmi_storage_flush_times_seconds_total` | Total flush latency per drive (counter) |

#### Info and Phase

| Metric | Description |
|--------|-------------|
| `kubevirt_vmi_info` | VMI metadata (gauge, always 1; rich label set) |
| `kubevirt_vmi_phase_count` | Count of VMIs by phase |
| `kubevirt_vmi_non_evictable` | Whether the VMI is non-evictable |
| `kubevirt_vmi_launcher_memory_overhead_bytes` | Memory overhead of the virt-launcher pod |

### Labels

Common labels on all `kubevirt_vmi_*` metrics:

| Label | Description | Example values |
|-------|-------------|----------------|
| `name` | VM name | `my-rhel-vm`, `webserver-01` |
| `namespace` | VM namespace | `default`, `my-vms` |
| `node` | Node running the VMI | `worker-0`, `worker-1` |
| `pod` | Virt-launcher pod name | `virt-launcher-my-vm-abcde` |
| `owner` | Owner reference | `VirtualMachine/my-vm` |

Labels specific to certain metric groups:

| Label | Applies to | Description | Example values |
|-------|-----------|-------------|----------------|
| `interface` | Network metrics | Virtual NIC name | `default`, `net1` |
| `drive` | Storage metrics | Virtual disk name | `vda`, `vdb`, `ua-cloudinit` |
| `id` | `kubevirt_vmi_vcpu_seconds_total` | vCPU index | `0`, `1`, `2` |
| `state` | `kubevirt_vmi_vcpu_seconds_total` | vCPU state | `running`, `halted` |
| `phase` | `kubevirt_vmi_info` | VMI lifecycle phase | `Running`, `Pending`, `Succeeded` |
| `os` | `kubevirt_vmi_info` | Guest OS identifier | `linux`, `windows` |
| `flavor` | `kubevirt_vmi_info` | Instance type / flavor | `small`, `medium`, `large` |
| `workload` | `kubevirt_vmi_info` | Workload type | `server`, `desktop` |

---

## MTV / Forklift Migration Metrics

### Metrics

| Metric | Description |
|--------|-------------|
| `mtv_migrations_status_total` | Migration counts by status (succeeded/failed/running) |
| `mtv_plans_status` | Plan-level status counts |
| `mtv_migration_data_transferred_bytes` | Total bytes migrated per plan |
| `mtv_migration_net_throughput` | Migration network throughput |
| `mtv_migration_storage_throughput` | Migration storage throughput |
| `mtv_migration_duration_seconds` | Migration duration per plan |
| `mtv_plan_alert_status` | Alerts on migration plans |
| `mtv_workload_migrations_status_total` | Per-workload migration status (per plan + status) |
| `kubevirt_vmi_migrations_in_pending_phase` | KubeVirt VMI migrations in pending phase |
| `kubevirt_vmi_migrations_in_running_phase` | KubeVirt VMI migrations in progress |

### Labels

All `mtv_*` metrics share these labels for filtering and grouping:

| Label | Description | Example values |
|-------|-------------|----------------|
| `provider` | Source provider type | `vsphere`, `ovirt`, `openstack`, `ova`, `ec2` |
| `mode` | Migration mode | `Cold`, `Warm` |
| `target` | Target cluster | `Local` (host cluster) or remote cluster name |
| `owner` | User who owns the migration | `admin@example.com` |
| `plan` | Migration plan UUID | `363ce137-dace-4fb4-b815-759c214c9fec` |
| `namespace` | Forklift operator namespace | `konveyor-forklift`, `openshift-mtv` |
| `status` | Migration/plan status (on status metrics) | `Succeeded`, `Failed`, `Executing` |
