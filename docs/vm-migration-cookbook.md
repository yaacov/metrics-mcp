# VM & Migration Cookbook

Targeted recipes for monitoring individual Forklift/MTV migrations, per-disk I/O during data transfer, and KubeVirt virtual machine resources (CPU, memory, network, storage).

For general cluster monitoring recipes see [Query Cookbook](query-cookbook.md). For metric names and labels see [Metrics Reference](metrics-reference.md). For PromQL syntax see [PromQL Reference](promql-reference.md).

---

## Monitoring a Specific Migration

### Step 1 -- Identify the plan

```bash
kubectl get plan -A
```

Note the plan **name** and **namespace**. The plan name is used in MTV metric labels (`plan_name`) and in pod naming conventions.

### Step 2 -- Find migration pods

Migration pods live in the target namespace (where VMs will run), not the Forklift operator namespace. Pod naming differs by source provider:

**vSphere migrations** -- pods carry a `plan` label:

```bash
kubectl get pods -n TARGET_NAMESPACE -l plan
```

**oVirt / OpenStack migrations** -- populator pods:

```bash
kubectl get pods -n TARGET_NAMESPACE | grep '^populate-'
```

**CDI importer pods** (any provider):

```bash
kubectl get pods -n TARGET_NAMESPACE | grep '^importer-'
```

**All data-transfer pods at once:**

```bash
kubectl get pods -n TARGET_NAMESPACE | grep -E '^(populate-|importer-|virt-v2v|cdi-upload)'
```

### Step 3 -- Query pod-level network I/O

Use the discovered pod names in PromQL. Replace `POD1`, `POD2` with actual names:

```bash
kubectl metrics query \
  --query 'sum by (pod)(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"POD1|POD2"}[5m]))'

kubectl metrics query \
  --query 'sum by (pod)(rate(container_network_transmit_bytes_total{namespace="TARGET_NAMESPACE",pod=~"POD1|POD2"}[5m]))'
```

Track pod network over time during the migration:

```bash
kubectl metrics query-range \
  --query 'sum by (pod)(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"POD1|POD2"}[5m]))' \
  --query 'sum by (pod)(rate(container_network_transmit_bytes_total{namespace="TARGET_NAMESPACE",pod=~"POD1|POD2"}[5m]))' \
  --name rx_bytes_per_sec --name tx_bytes_per_sec \
  --start "-1h" --step "30s"
```

### Step 4 -- Query plan-level throughput

MTV exposes aggregate throughput per plan. The label is `plan_name` (the human-readable plan name):

```bash
kubectl metrics query \
  --query 'mtv_migration_net_throughput{plan_name="MY_PLAN"}'

kubectl metrics query \
  --query 'mtv_migration_storage_throughput{plan_name="MY_PLAN"}'
```

Data transferred so far:

```bash
kubectl metrics query \
  --query 'mtv_migration_data_transferred_bytes{plan_name="MY_PLAN"}'
```

Duration of the migration:

```bash
kubectl metrics query \
  --query 'mtv_migration_duration_seconds{plan_name="MY_PLAN"}'
```

### Step 5 -- Track data transfer over time

For a completed migration, use absolute timestamps from the plan object:

```bash
kubectl get migration -n TARGET_NAMESPACE -o yaml | grep -E 'started|completed'
```

Then query the transfer window:

```bash
kubectl metrics query-range \
  --query 'mtv_migration_data_transferred_bytes{plan_name="MY_PLAN"}' \
  --start "2025-06-15T10:00:00Z" --end "2025-06-15T12:30:00Z" --step "60s"

kubectl metrics query-range \
  --query 'mtv_migration_net_throughput{plan_name="MY_PLAN"}' \
  --query 'mtv_migration_storage_throughput{plan_name="MY_PLAN"}' \
  --name net_throughput --name storage_throughput \
  --start "2025-06-15T10:00:00Z" --end "2025-06-15T12:30:00Z" --step "60s"
```

---

## Per-Disk Metrics During Migration

Multi-disk VMs produce one DataVolume (and one transfer pod) per disk.

### List DataVolumes for a migration

```bash
kubectl get datavolume -n TARGET_NAMESPACE
```

Each DataVolume name typically includes the VM name and disk identifier (e.g. `my-vm-disk-0`, `my-vm-disk-1`). The corresponding transfer pod is named `importer-DVNAME-RANDOM` or `populate-DVNAME-RANDOM`.

### Per-disk transfer rate

Query network traffic for each disk's pod individually:

```bash
kubectl metrics query \
  --query 'sum by (pod)(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"importer-my-vm-disk-0.*"}[5m]))'

kubectl metrics query \
  --query 'sum by (pod)(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"importer-my-vm-disk-1.*"}[5m]))'
```

### Compare disks side-by-side over time

Use multiple `--query` / `--name` flags to plot both disks in a single range query:

```bash
kubectl metrics query-range \
  --query 'sum(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"importer-my-vm-disk-0.*"}[5m]))' \
  --query 'sum(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"importer-my-vm-disk-1.*"}[5m]))' \
  --name disk_0_rx --name disk_1_rx \
  --start "-1h" --step "30s"
```

### Storage-side I/O (Ceph / ODF)

If the target storage is ODF, check Ceph pool write rates during the migration window to confirm data is landing:

```bash
kubectl metrics query-range \
  --query 'rate(ceph_pool_wr_bytes[5m])' \
  --start "-1h" --step "60s" --group-by pool_id
```

---

## KubeVirt VM Monitoring

Recipes for monitoring any running KubeVirt virtual machine -- whether freshly migrated or long-running. All `kubevirt_vmi_*` metrics use the `name` label for the VM name and `namespace` for the VM namespace.

### Listing running VMs

```bash
kubectl metrics query --query 'kubevirt_vmi_phase_count{phase="Running"}' --group-by namespace
```

### CPU usage

Total CPU cores consumed by a VM:

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_cpu_usage_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])'
```

System vs. user CPU breakdown:

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_cpu_system_usage_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])'

kubectl metrics query \
  --query 'rate(kubevirt_vmi_cpu_user_usage_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])'
```

Per-vCPU usage (the `id` label identifies each vCPU):

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_vcpu_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by id
```

Top 10 VMs by CPU (preset):

```bash
kubectl metrics preset --name vm_cpu_usage
kubectl metrics preset --name vm_cpu_usage --namespace VM_NAMESPACE
```

### Memory usage

Current resident memory:

```bash
kubectl metrics query \
  --query 'kubevirt_vmi_memory_resident_bytes{name="VM_NAME",namespace="VM_NAMESPACE"}'
```

Available vs. used memory:

```bash
kubectl metrics query \
  --query 'kubevirt_vmi_memory_available_bytes{name="VM_NAME",namespace="VM_NAMESPACE"}'

kubectl metrics query \
  --query 'kubevirt_vmi_memory_used_bytes{name="VM_NAME",namespace="VM_NAMESPACE"}'
```

Memory usage percentage:

```bash
kubectl metrics query \
  --query 'kubevirt_vmi_memory_used_bytes{name="VM_NAME",namespace="VM_NAMESPACE"} / kubevirt_vmi_memory_domain_bytes{name="VM_NAME",namespace="VM_NAMESPACE"} * 100'
```

Top 10 VMs by memory (preset):

```bash
kubectl metrics preset --name vm_memory_usage
```

### Network traffic

Per-interface receive and transmit rates (the `interface` label identifies each NIC):

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_network_receive_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by interface

kubectl metrics query \
  --query 'rate(kubevirt_vmi_network_transmit_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by interface
```

Network errors and dropped packets:

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_network_receive_errors_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])'

kubectl metrics query \
  --query 'rate(kubevirt_vmi_network_receive_packets_dropped_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])'
```

Top 10 VMs by network (preset):

```bash
kubectl metrics preset --name vm_network_rx
kubectl metrics preset --name vm_network_tx
```

### Disk I/O

Per-drive read and write throughput (the `drive` label identifies each virtual disk):

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_storage_read_traffic_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by drive

kubectl metrics query \
  --query 'rate(kubevirt_vmi_storage_write_traffic_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by drive
```

IOPS per drive:

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_storage_iops_read_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by drive

kubectl metrics query \
  --query 'rate(kubevirt_vmi_storage_iops_write_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by drive
```

Average I/O latency per operation (seconds):

```bash
kubectl metrics query \
  --query 'rate(kubevirt_vmi_storage_read_times_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m]) / rate(kubevirt_vmi_storage_iops_read_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by drive

kubectl metrics query \
  --query 'rate(kubevirt_vmi_storage_write_times_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m]) / rate(kubevirt_vmi_storage_iops_write_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --group-by drive
```

Top 10 VMs by storage (presets):

```bash
kubectl metrics preset --name vm_storage_read
kubectl metrics preset --name vm_storage_write
kubectl metrics preset --name vm_storage_iops
```

### VM resource trends over time

Combine CPU, memory, and network into a single range query for a VM dashboard:

```bash
kubectl metrics query-range \
  --query 'rate(kubevirt_vmi_cpu_usage_seconds_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m])' \
  --query 'kubevirt_vmi_memory_used_bytes{name="VM_NAME",namespace="VM_NAMESPACE"}' \
  --query 'sum(rate(kubevirt_vmi_network_receive_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m]))' \
  --query 'sum(rate(kubevirt_vmi_network_transmit_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE"}[5m]))' \
  --name cpu_cores --name memory_bytes --name net_rx --name net_tx \
  --start "-1h" --step "60s"
```

Disk I/O trends for a specific drive:

```bash
kubectl metrics query-range \
  --query 'rate(kubevirt_vmi_storage_read_traffic_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE",drive="DRIVE"}[5m])' \
  --query 'rate(kubevirt_vmi_storage_write_traffic_bytes_total{name="VM_NAME",namespace="VM_NAMESPACE",drive="DRIVE"}[5m])' \
  --name read_bytes_per_sec --name write_bytes_per_sec \
  --start "-1h" --step "60s"
```

---

## Advanced Techniques

### Discovering migration pods

Migration pod names follow the pattern `{plan-name}-{vm-id}-{random}`. Find them before querying network metrics:

```bash
# VMware / general migration pods (carry a "plan" label)
kubectl get pods -n TARGET_NAMESPACE -l plan

# oVirt / OpenStack populator pods
kubectl get pods -n TARGET_NAMESPACE | grep '^populate-'
```

Then use the discovered names in PromQL:

```bash
kubectl metrics query \
  --query 'sum by (pod)(rate(container_network_receive_bytes_total{namespace="TARGET_NAMESPACE",pod=~"POD1|POD2"}[5m]))'
```

### Short-lived pod network metrics

Container-level network metrics (`container_network_*`) require cAdvisor to establish tracking, which takes 1--2 collection cycles (~10--20s). Pods running under ~60 seconds may complete before tracking starts. CPU and memory metrics are unaffected.

**Node-level fallback** -- find which node ran the pod, then query node RX/TX during the migration window:

```bash
kubectl metrics query-range \
  --query 'instance:node_network_receive_bytes_excluding_lo:rate1m{instance=~"NODE_NAME.*"}' \
  --query 'instance:node_network_transmit_bytes_excluding_lo:rate1m{instance=~"NODE_NAME.*"}' \
  --name node_rx --name node_tx \
  --start "2025-06-15T10:00:00Z" --end "2025-06-15T10:30:00Z" --step "30s"
```

Compare against baseline before and after the migration window to isolate transfer traffic.

**CPU activity** -- confirm the pod was active:

```bash
kubectl metrics query-range \
  --query 'rate(container_cpu_usage_seconds_total{pod="POD_NAME",namespace="NAMESPACE"}[1m])' \
  --start "2025-06-15T10:00:00Z" --end "2025-06-15T10:30:00Z" --step "30s"
```

### Historical queries for completed migrations

For a migration that already finished, use absolute timestamps rather than relative offsets like `-1h`:

```bash
kubectl metrics query-range \
  --query 'sum by (pod)(rate(container_network_receive_bytes_total{namespace="NAMESPACE"}[5m]))' \
  --start "2025-06-15T10:00:00Z" --end "2025-06-15T12:30:00Z" --step "60s"
```

Get the exact timestamps from the migration plan (e.g. via `kubectl get migration -n NAMESPACE -o yaml`).
