# PromQL Reference

PromQL (Prometheus Query Language) selects and aggregates time-series data.
This page is a quick reference for use with `kubectl metrics query` and `kubectl metrics query-range`.

The same content is available from the CLI:

```bash
kubectl metrics help-promql
```

## Selectors

```
metric_name                          # all time series for this metric
metric_name{label="value"}          # exact label match
metric_name{label=~"pattern.*"}     # regex match
metric_name{label!="value"}         # exclude a label value
metric_name{label!~"test.*"}        # negative regex
metric_name{l1="a", l2="b"}        # combine multiple filters
```

```bash
kubectl metrics query --query 'up'
kubectl metrics query --query 'up{job="prometheus"}'
kubectl metrics query --query 'up{job=~"prom.*"}'
```

## Range Vectors

Range vectors select a window of samples. They are required by functions like `rate` and `increase`.

```
http_requests_total[5m]              # last 5 minutes of samples
http_requests_total[1h]              # last 1 hour
```

Range vectors cannot be returned directly — wrap them in a function:

```bash
kubectl metrics query --query 'rate(http_requests_total[5m])'
```

## Functions

### Rate and increase (for counters)

Counters only go up. Use `rate` or `increase` to get meaningful values:

```
rate(metric[5m])                     # per-second rate over 5 minutes
irate(metric[5m])                    # instant rate (last two samples)
increase(metric[1h])                 # total increase over 1 hour
```

```bash
kubectl metrics query --query 'rate(container_cpu_usage_seconds_total[5m])'
kubectl metrics query-range --query 'rate(http_requests_total[5m])' --start "-1h"
```

### Aggregation

```
sum(metric)                          # total across all series
avg(metric)                          # average across all series
min(metric)                          # minimum
max(metric)                          # maximum
count(metric)                        # count of series
```

Group by a label with `by`, or drop a label with `without`:

```
sum by (namespace)(metric)           # total grouped by namespace
avg by (pod)(rate(cpu[5m]))          # average rate grouped by pod
sum without (instance)(metric)       # sum, dropping the instance label
```

```bash
kubectl metrics query --query 'sum by (namespace)(rate(container_cpu_usage_seconds_total[5m]))'
kubectl metrics query --query 'avg by (pod)(rate(container_cpu_usage_seconds_total[5m]))'
```

### Sorting and limiting

```
topk(10, metric)                     # top 10 series by value
bottomk(5, metric)                   # bottom 5 series by value
sort_desc(metric)                    # sort descending
```

```bash
kubectl metrics query --query 'topk(10, sum by (namespace)(rate(container_network_receive_bytes_total[5m])))'
```

### Other functions

```
absent(metric)                       # returns 1 if metric has no series
changes(metric[1h])                  # number of value changes
delta(metric[1h])                    # difference over range (gauges only)
predict_linear(metric[1h], 3600)     # linear prediction 1 hour ahead
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))  # P99 latency from histogram
```

## Operators

### Arithmetic

```
metric_a + metric_b                  # addition
metric_a - metric_b                  # subtraction
metric_a / metric_b                  # division
metric_a * 100                       # scale a metric
1 - (available / total)              # compute used percentage
```

```bash
kubectl metrics query --query '100 * (1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m])))'
kubectl metrics query --query 'rate(ceph_osd_op_latency_sum[5m]) / rate(ceph_osd_op_latency_count[5m])'
```

### Comparison (filtering)

```
metric > 100                         # keep series where value > 100
metric == 0                          # keep series where value is 0
metric != 1                          # keep series where value is not 1
```

### Set operations

```
metric_a and metric_b                # intersection (series present in both)
metric_a or metric_b                 # union (series from either)
metric_a unless metric_b             # difference (in a but not b)
```

## Common Patterns

| Pattern | Description |
|---------|-------------|
| `rate(counter[5m])` | Per-second rate from a counter |
| `sum by (ns)(rate(bytes_total[5m]))` | Aggregate rate by namespace |
| `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))` | P99 latency from histogram |
| `changes(metric[1h])` | Number of value changes |
| `delta(metric[1h])` | Difference over range (gauges) |
| `predict_linear(metric[1h], 3600)` | Linear prediction 1 hour ahead |
| `topk(10, sort_desc(sum by (label)(metric)))` | Top 10 grouped totals |
| `100 - avg by (instance)(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100` | CPU utilization per node |

## Time Units

| Unit | Meaning |
|------|---------|
| `s` | seconds |
| `m` | minutes |
| `h` | hours |
| `d` | days |
| `w` | weeks |

Used in range vectors: `[5m]`, `[1h]`, `[7d]`

Used in `--start` / `--end` flags: `-30m`, `-1h`, `-7d`, `-2w`

Absolute timestamps use ISO-8601: `2025-06-15T10:00:00Z`

## CLI Flag Mapping

| PromQL concept | CLI flag |
|----------------|----------|
| The query itself | `--query` |
| Time window start | `--start` (e.g. `-1h`, ISO-8601) |
| Time window end | `--end` (default: now) |
| Resolution | `--step` (default: `60s`) |
| Post-query label filter | `--selector` (e.g. `namespace=prod,pod=~nginx.*`) |
| Group results into sub-tables | `--group-by` (e.g. `namespace`) |
| Flat row-per-sample output | `--no-pivot` |
