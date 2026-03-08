# Monitoring Plan: BabyMilk App — Prometheus + Grafana on Kubernetes

## Problem Statement

The BabyMilk application (Go web app + PostgreSQL + Nginx) currently has **zero observability**. There are no metrics, no dashboards, and no alerting. This plan defines a full-stack monitoring solution using the industry-standard Prometheus + Grafana stack deployed on Kubernetes (K3d/K8s), covering application metrics, database monitoring, Nginx proxy monitoring, and Kubernetes infrastructure health.

## Architecture Overview

```
┌─────────────┐    scrape     ┌─────────────────┐
│  Prometheus  │◄─────────────│ Go App (/metrics)│
│              │              └─────────────────┘
│              │    scrape     ┌─────────────────┐
│              │◄─────────────│ postgres_exporter│
│              │              └─────────────────┘
│              │    scrape     ┌─────────────────┐
│              │◄─────────────│ nginx_exporter   │
│              │              └─────────────────┘
│              │    scrape     ┌─────────────────┐
│              │◄─────────────│ kube-state-metrics│
│              │              └─────────────────┘
│              │    scrape     ┌─────────────────┐
│              │◄─────────────│ node-exporter    │
│              │              └─────────────────┘
└──────┬───┬──┘
       │   │
       │   └──────► ┌──────────────┐
       │            │ AlertManager │ ──► Slack / Email
       │            └──────────────┘
       ▼
┌──────────────┐
│   Grafana    │ ──► Dashboards
└──────────────┘
```

## Deployment Approach

Use **kube-prometheus-stack** Helm chart (formerly prometheus-operator) which bundles:
- Prometheus Operator
- Prometheus
- Grafana
- AlertManager
- kube-state-metrics
- node-exporter
- Default K8s dashboards and alerting rules

This is the industry-standard one-chart deployment for Kubernetes monitoring.

---

## 1. Go Application Instrumentation

### Library
Add `github.com/prometheus/client_golang` to `go.mod`.

### Metrics — Why We Need Them and Their Impact

#### `http_requests_total` (Counter) — Labels: `method`, `endpoint`, `status_code`
- **Why:** This is the foundational metric for any web service. Without it, you have no way to know how much traffic the API receives, which endpoints are most popular, or whether errors are increasing. Today, the only way to know something is wrong is when a user reports it.
- **Impact:** Enables calculating request rate (`rate(http_requests_total[5m])`), error rate (`rate(...{status_code=~"5.."}[5m])`), and traffic distribution by endpoint. For BabyMilk, this reveals if `/api/feedings` GET requests dominate (expected — the app polls for data), or if unexpected 4xx/5xx errors are appearing. A sudden spike in 500s on `POST /api/feedings` would tell you the DB is rejecting writes before any user complains about lost feeding records.

#### `http_request_duration_seconds` (Histogram) — Labels: `method`, `endpoint`
- **Why:** The app currently has zero visibility into whether API responses are fast or slow. The Go `net/http` server doesn't log latency by default. A slow PostgreSQL query (e.g., the `GetDailyTotalsByMonth` with timezone conversion) could silently degrade UX without anyone noticing.
- **Impact:** Enables p50/p95/p99 latency percentiles per endpoint. For BabyMilk, the histogram buckets (5ms–2.5s) are tuned for a CRUD API — if p95 of `GET /api/feedings/daily` exceeds 500ms, it directly impacts the dashboard chart load time. Also enables latency heatmaps to spot distribution shifts (e.g., all requests moving from 10ms to 200ms after a bad migration).

#### `http_requests_in_flight` (Gauge)
- **Why:** The app has no concurrency limiting. If something causes requests to pile up (e.g., a slow DB query holding connections), you need real-time visibility into how many requests are being processed simultaneously.
- **Impact:** A sustained value above 10 (matching `db.SetMaxOpenConns(10)` in `db.go`) means requests are likely waiting for DB connections. This metric directly correlates with user-perceived "hanging" behavior. Without it, you'd see symptoms (slow responses) but not the cause (request queuing).

#### `db_query_duration_seconds` (Histogram) — Labels: `query_name`
- **Why:** The app runs complex PostgreSQL queries with timezone conversions (`AT TIME ZONE`) and aggregations (`SUM`, `GROUP BY`). These queries can degrade as the `feedings` table grows, but currently there's no way to detect slow queries without manually checking PostgreSQL logs.
- **Impact:** Labels like `query_name: "get_daily_totals"` or `"get_feedings_by_month"` let you pinpoint exactly which query is slow. If `GetDailyTotalsByMonth` latency increases from 5ms to 500ms after 10K records, this metric catches it. Without it, you'd only see elevated `http_request_duration_seconds` and have to guess which layer is slow.

#### `db_connections_open` / `db_connections_idle` / `db_connections_max_open` (Gauges)
- **Why:** The app configures `MaxOpenConns=10` and `MaxIdleConns=5` in `db.go`. If open connections hit 10, new requests block waiting for a free connection. Today, this would manifest as random latency spikes with zero diagnostic information.
- **Impact:** Plotting these three together shows pool health at a glance: `open` approaching `max_open` = connection pressure; `idle` dropping to 0 = sustained load; `open` stuck at `max_open` = pool exhaustion causing request queuing. This is especially critical because the app has no connection timeout — a saturated pool would cause requests to hang indefinitely.

#### `feedings_created_total` / `feedings_deleted_total` (Counters)
- **Why:** Business-level metrics that answer "is the app being used as expected?" Technical metrics alone can't tell you if users are actually recording feedings or if the data is being accidentally deleted.
- **Impact:** A flat `feedings_created_total` rate could mean the app is unused or broken from the user's perspective (even if technically returning 200 OK on empty responses). A spike in `feedings_deleted_total` could indicate accidental bulk deletion. For a baby feeding tracker, missing data is critical — parents rely on complete records.

#### `app_info` (Gauge) — Labels: `version`, `go_version`
- **Why:** After a deployment, you need to confirm which version is actually running. Without this, you have to exec into the container or check image tags manually.
- **Impact:** Low overhead, high diagnostic value. Enables Grafana annotations showing when deployments happened, and correlating metric changes with code versions. When you see a latency regression, `app_info` instantly answers "did we just deploy something new?"

### Implementation Notes
- Add a Prometheus middleware wrapping the existing `corsMiddleware` in `cmd/server/main.go`
- Expose `/metrics` endpoint using `promhttp.Handler()`
- Wrap `sql.DB` stats collector using `db.Stats()` in a periodic goroutine or custom collector
- Add Kubernetes annotations to the app Deployment for Prometheus auto-discovery:
  ```yaml
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8000"
    prometheus.io/path: "/metrics"
  ```

---

## 2. PostgreSQL Monitoring

### Exporter
Deploy **postgres_exporter** (`prometheuscommunity/postgres-exporter`) as a sidecar or separate Deployment.

### Key Metrics — Why We Need Them and Their Impact

#### `pg_up` — Database Reachability
- **Why:** The single most important database metric. If PostgreSQL is down, the entire BabyMilk app is non-functional — every API endpoint queries the DB. The Go app currently just returns 500 errors with no external visibility.
- **Impact:** Enables the `DatabaseDown` critical alert. Without it, you'd only discover a DB outage when users report "the app is broken." With it, you get paged within 1 minute of the DB going unreachable. This is a boolean metric — either the DB is up or it isn't — so it's zero-ambiguity.

#### `pg_stat_database_tup_fetched` / `tup_inserted` / `tup_updated` / `tup_deleted` — Row Operation Rates
- **Why:** These show the actual workload PostgreSQL is handling. The BabyMilk app does frequent reads (`GetFeedings`, `GetDailyTotals`) and occasional writes (`CreateFeeding`, `UpdateFeeding`). Without these metrics, you can't distinguish between "the DB is slow because of heavy reads" vs "the DB is slow because of lock contention from writes."
- **Impact:** The rate of these counters reveals the read/write ratio and workload pattern. For BabyMilk, you'd expect `tup_fetched` to be ~100x higher than `tup_inserted` (frequent list views vs occasional feedings). A sudden spike in `tup_deleted` could indicate accidental bulk deletion. A drop in `tup_fetched` to zero means the app stopped querying — possibly crashed.

#### `pg_stat_database_numbackends` — Active Connections
- **Why:** The Go app sets `MaxOpenConns=10`. PostgreSQL also has its own `max_connections` default (100). If connections exceed what the app or DB can handle, requests start queuing or failing.
- **Impact:** This metric bridges the app and DB monitoring. If `numbackends` rises close to the Go app's pool max (10), you know the app is saturating its own pool. If it goes higher (other clients connecting), you know something unexpected is accessing the DB. Combined with the Go-side `db_connections_open` metric, you get a complete picture of connection usage from both sides.

#### `pg_stat_database_deadlocks` — Deadlock Count
- **Why:** The BabyMilk app allows concurrent creates, updates, and deletes on the `feedings` table. While deadlocks are rare for simple CRUD, they can occur under concurrent updates to the same row. A deadlock causes one transaction to fail with an error that surfaces as a 500 to the user.
- **Impact:** Even one deadlock per hour is worth investigating. The `rate(pg_stat_database_deadlocks[5m]) > 0` alert catches this early. Without monitoring, deadlocks appear as intermittent "internal server error" responses that are nearly impossible to reproduce manually.

#### `pg_stat_database_blks_hit` / `blks_read` — Cache Hit Ratio
- **Why:** PostgreSQL uses shared buffers to cache frequently accessed data. If the `feedings` table grows large and can't fit in cache, every query hits disk, dramatically increasing latency.
- **Impact:** Cache hit ratio = `blks_hit / (blks_hit + blks_read)`. A healthy DB has >99% hit ratio. If this drops below 90%, it means PostgreSQL is doing excessive disk reads — you need to increase `shared_buffers` or the table is too large for the allocated memory. For BabyMilk's `GetFeedings` with timezone conversion, poor cache performance would directly slow every page load.

#### `pg_database_size_bytes` — Database Size
- **Why:** The `feedings` table grows over time as parents log feedings daily. While growth is slow (each record is ~100 bytes), over months/years it accumulates. Without monitoring, you won't know when disk space becomes a concern.
- **Impact:** Enables proactive capacity planning. The `DatabaseGrowing > 1GB` info alert gives you advance notice to expand storage or implement data archival. Combined with `pg_stat_user_tables_n_live_tup` (row counts), you can estimate growth rate and predict when you'll need more space.

#### `pg_stat_activity_count` — Connections by State
- **Why:** Not all connections are equal. Connections stuck in `idle in transaction` are holding locks and blocking others. Connections in `active` state are doing work. This metric breaks down what connections are actually doing.
- **Impact:** If you see many `idle in transaction` connections, it indicates the Go app isn't properly closing transactions (potential bug). If you see many `waiting` connections, it indicates lock contention. This is deeper diagnostic information than `numbackends` alone.

### Custom Queries (optional)
Monitor the `feedings` table specifically:
```yaml
pg_feedings:
  query: "SELECT count(*) as count FROM feedings"
  metrics:
    - count:
        usage: "GAUGE"
        description: "Total feeding records"
```

---

## 3. Nginx Monitoring

### Exporter
Deploy **nginx-prometheus-exporter** (`nginx/nginx-prometheus-exporter`).

### Prerequisites
Enable the Nginx stub_status module by adding to `deploy/nginx.conf`:
```nginx
location /stub_status {
    stub_status;
    allow 127.0.0.1;
    deny all;
}
```

### Key Metrics — Why We Need Them and Their Impact

#### `nginx_connections_active` — Active Connections
- **Why:** Nginx is the entry point for all traffic — both static assets (HTML/JS/CSS) and API proxy requests to the Go app. If active connections spike, it could mean a traffic surge, a DDoS, or a slow backend causing connections to pile up.
- **Impact:** This is the "traffic pressure" indicator for the frontend layer. For BabyMilk, normal active connections should be very low (<10 for a personal app). If this suddenly jumps to 100+, something is wrong — either the Go backend is slow (requests backing up in Nginx's proxy queue) or unexpected traffic is hitting the server. The `NginxConnectionSpike > 100` alert catches this.

#### `nginx_connections_accepted` / `nginx_connections_handled` — Connection Processing
- **Why:** If `accepted > handled`, Nginx is dropping connections — meaning some users are getting connection refused errors that are completely invisible without metrics.
- **Impact:** The difference between these two counters reveals dropped connections. For a healthy Nginx, they should be equal. Any divergence means Nginx hit a worker or connection limit and is silently rejecting traffic. This is critical because dropped connections don't generate access logs — they're completely invisible without this metric.

#### `nginx_connections_reading` / `writing` / `waiting` — Connection States
- **Why:** These break down what active connections are actually doing. High `writing` means Nginx is waiting for the Go backend to produce responses. High `waiting` means many keep-alive connections are idle.
- **Impact:** For BabyMilk, high `writing` relative to `reading` would indicate the Go backend is slow to respond (correlate with `http_request_duration_seconds`). High `waiting` is normal for keep-alive but excessive values waste Nginx worker resources. This helps distinguish frontend vs backend bottlenecks.

#### `nginx_http_requests_total` — Total Requests
- **Why:** This covers ALL requests including static assets, not just API calls measured by the Go app. The Go `http_requests_total` only sees `/api/*` requests, but Nginx also serves `index.html`, `app.js`, `style.css`, and `favicon.svg`.
- **Impact:** Comparing Nginx request rate with Go app request rate reveals how much traffic is static vs dynamic. If Nginx shows 100 req/s but Go only sees 10 req/s, 90% of traffic is static — good caching indicator. Also serves as an independent cross-check: if Nginx shows requests but Go doesn't, the proxy is misconfigured or the Go app crashed.

---

## 4. Kubernetes Infrastructure Monitoring

Provided automatically by kube-prometheus-stack:

### kube-state-metrics — Why We Need Them and Their Impact

#### `kube_pod_status_phase` — Pod Phase
- **Why:** Tells you whether your pods are actually Running, stuck in Pending (can't be scheduled — insufficient resources), or Failed. Without this, K8s could silently fail to schedule pods and you'd only notice when the app stops responding.
- **Impact:** A pod stuck in Pending means Kubernetes can't find a node with enough resources. A pod in Failed means it crashed and isn't being restarted. For BabyMilk with its 3 services (app, db, frontend), any pod not in Running = partial or full outage.

#### `kube_pod_container_status_restarts_total` — Container Restarts
- **Why:** The Go app crashes with `log.Fatalf` on DB connection failure or migration errors. Kubernetes will restart it, but frequent restarts indicate a persistent problem. Without this metric, the app could be crash-looping every 30 seconds and auto-recovering — appearing "fine" but losing all in-flight requests on each restart.
- **Impact:** The `PodCrashLooping` critical alert fires when restart rate > 0 over 15 minutes. Each restart means: lost in-flight requests, cold DB connection pool, re-running migrations. Even if K8s auto-heals, crash loops cause intermittent errors that are maddening to debug without restart count data.

#### `kube_deployment_status_replicas_available` / `kube_deployment_spec_replicas` — Replica Health
- **Why:** If you scale to 2+ replicas and one fails, the deployment is degraded but might still serve traffic. Without comparing available vs desired replicas, you won't know you're running at reduced capacity.
- **Impact:** The `DeploymentReplicaMismatch` info alert fires when available ≠ desired for >10 minutes. This catches scenarios like: failed rolling updates (new version crashes but old pods are already terminated), persistent volume conflicts (only one pod can mount a PVC), or node failures taking out replicas.

#### `kube_pod_container_resource_requests` / `limits` — Resource Configuration
- **Why:** Helps verify that resource requests/limits are properly configured. Without requests, K8s can't make good scheduling decisions. Without limits, a runaway process can starve other pods.
- **Impact:** Visualizing actual usage vs configured requests/limits reveals whether resources are right-sized. If the Go app consistently uses 50MB but has a 512MB limit, you're wasting cluster capacity. If it uses 490MB of a 512MB limit, you're one spike away from OOMKill.

### node-exporter — Why We Need Them and Their Impact

#### `node_cpu_seconds_total` — CPU Usage
- **Why:** The K3d/K8s nodes have finite CPU. If the node is CPU-saturated, all pods suffer — not just the ones using the most CPU. This metric is about node-level health, not individual container health.
- **Impact:** Node CPU at 90%+ means everything on that node is being throttled. For a K3d single-node cluster, this means the entire stack (app + DB + Nginx) slows down simultaneously. The `HighCPUUsage` alert gives you time to react before CPU saturation causes cascading failures.

#### `node_memory_MemAvailable_bytes` — Available Memory
- **Why:** When a node runs out of memory, the Linux OOM killer starts terminating processes — often choosing PostgreSQL (the biggest memory consumer) first. This kills the DB without a clean shutdown, potentially corrupting data.
- **Impact:** Memory pressure is the #1 cause of unexpected pod evictions in Kubernetes. For BabyMilk, PostgreSQL's `shared_buffers` and the Go runtime's heap both consume memory. Monitoring available memory prevents the catastrophic scenario of OOM-killed PostgreSQL with potential data loss.

#### `node_filesystem_avail_bytes` — Disk Space
- **Why:** PostgreSQL stores data on disk. Container logs accumulate on disk. Prometheus metrics are stored on disk. When a node runs out of disk, Kubernetes evicts pods and PostgreSQL stops accepting writes.
- **Impact:** The `DiskSpaceLow > 80%` warning alert gives you time to clean up or expand storage. For BabyMilk, the main disk consumers are: PostgreSQL data (`pgdata` volume), Prometheus metrics (10Gi PVC), and container logs. Without monitoring, you'd discover disk issues when the app starts returning "read-only file system" errors.

#### `node_network_receive_bytes_total` / `transmit_bytes_total` — Network I/O
- **Why:** Network saturation is rare for a small app but indicates problems when it occurs — e.g., a misconfigured backup job flooding the network, or an attack generating excessive traffic.
- **Impact:** Primarily useful for correlation: if you see high latency and high network I/O simultaneously, the network is the bottleneck. For BabyMilk, network I/O should be very low (small JSON payloads). Unexpected spikes help identify anomalous traffic patterns.

---

## 5. Retention Policy — Why These Tiers and Their Impact

Data retention is a tradeoff between storage cost and historical visibility. Too short and you can't investigate last week's incident; too long and you're paying for data nobody looks at.

| Tier | Storage | Retention | Resolution | Purpose | Why This Setting |
|---|---|---|---|---|---|
| **Hot** (Prometheus local) | PVC (SSD recommended) | 15 days | Full (15s scrape interval) | Real-time monitoring & alerting | 15 days covers two weekends + a full work week, enough to compare "this week vs last week" patterns. 15s scrape interval balances granularity vs storage — 30s would miss short latency spikes, 5s would triple storage cost for minimal benefit. |
| **Warm** (optional — Thanos/Cortex) | Object storage (S3/MinIO) | 90 days | Downsampled to 5 min | Trend analysis | 90 days lets you answer "is the DB growing faster this quarter?" and "did latency trend up after last month's deploy?" 5-min resolution is fine for trends — you don't need 15s granularity for month-old data. |
| **Cold** (optional — long-term) | Object storage (S3/MinIO) | 1 year | Downsampled to 1 hour | Capacity planning | 1 year enables year-over-year comparison and long-term capacity planning. For BabyMilk, this is optional — mainly useful if you want to see "how did usage change as the baby grew." |

### Recommended Starting Config (Prometheus)
```yaml
prometheus:
  prometheusSpec:
    retention: 15d
    retentionSize: "5GB"
    scrapeInterval: 15s
    evaluationInterval: 15s
    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 10Gi
```

### Notes
- For a small personal app like BabyMilk, 15 days hot retention is sufficient
- Add Thanos sidecar later if long-term trending becomes needed
- Estimated storage: ~50MB/day for this stack → 10Gi PVC is generous

---

## 6. Grafana Dashboards — Why Each Dashboard and Panel Matters

The dashboards follow a **layered approach**: start with the App Overview for symptoms, drill into DB/Nginx for root causes, and check K8s Resources for infrastructure issues. This mirrors how you'd actually troubleshoot: "users say it's slow" → check app latency → see high DB query time → investigate DB connections.

### Dashboard 1: Application Overview
**Purpose:** Your first stop when something seems wrong. Answers: "Is the app healthy? How fast is it? Are there errors?"

| Panel | Visualization | PromQL | Why This Panel |
|---|---|---|---|
| Request Rate | Time series | `rate(http_requests_total[5m])` | Shows traffic volume over time. A sudden drop = possible outage. A spike = potential overload. Establishes the baseline "normal" traffic pattern. |
| Request Latency (p50/p95/p99) | Time series | `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))` | p50 = typical user experience. p95 = "slow but tolerable." p99 = worst-case latency. If p95 diverges from p50, a subset of requests is slow (likely a specific endpoint or DB query). |
| Error Rate (%) | Stat + time series | `rate(http_requests_total{status_code=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100` | The single most actionable number. >0% means something is broken. >5% triggers the critical alert. The stat panel shows the current value instantly; the time series shows when errors started. |
| Requests In Flight | Gauge | `http_requests_in_flight` | Real-time concurrency indicator. If this exceeds `MaxOpenConns` (10), requests are definitely queuing. Helps decide if you need to scale horizontally or fix a slow dependency. |
| Request Duration Heatmap | Heatmap | `rate(http_request_duration_seconds_bucket[5m])` | Shows the DISTRIBUTION of latency, not just percentiles. Reveals bimodal patterns (e.g., "most requests are 5ms but some cluster at 500ms") that percentiles can hide. |
| Feedings Created/Deleted | Time series | `rate(feedings_created_total[1h])` | Business health check. If creation rate drops to zero during typical usage hours, the app might be broken from the user's perspective even if technical metrics look fine. |
| Top Endpoints by Latency | Table | `topk(10, histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])))` | Immediately pinpoints which endpoint is slow when overall latency increases. Saves debugging time by narrowing the investigation to a specific route. |

### Dashboard 2: Database Health
**Purpose:** Drill-down for DB-related issues. Use when the App Overview shows high latency or errors.

| Panel | Visualization | PromQL | Why This Panel |
|---|---|---|---|
| DB Up/Down | Stat (green/red) | `pg_up` | Instant visual confirmation. Green = running. Red = outage. Paired with the `DatabaseDown` alert for at-a-glance status. |
| Active Connections | Time series | `pg_stat_database_numbackends{datname="babymilk"}` | Shows connection usage over time. Correlate with request rate to understand if connections scale with traffic or are leaking. |
| Go App DB Pool (open/idle/max) | Time series | `db_connections_open`, `db_connections_idle`, `db_connections_max_open` | The app-side view of DB connections. Open approaching max = saturation. Idle dropping to zero = sustained heavy load. Max is a flat line for reference. |
| Cache Hit Ratio | Gauge | `pg_stat_database_blks_hit / (...)` | Single number showing DB memory efficiency. Below 95% = performance problem. Below 90% = urgent. Helps decide if `shared_buffers` needs increasing. |
| Rows per Second (CRUD) | Time series | `rate(pg_stat_database_tup_inserted[5m])` etc. | Workload characterization. Shows whether the DB is read-heavy or write-heavy, and how the pattern changes over time. |
| Database Size | Stat | `pg_database_size_bytes{datname="babymilk"}` | Current disk usage. Simple but essential for capacity planning. Pairs with the `DatabaseGrowing` info alert. |
| Query Duration | Histogram | `rate(db_query_duration_seconds_bucket[5m])` | App-instrumented query timing by query name. Directly answers "which query is slow?" when DB latency increases. |
| Deadlocks | Time series | `rate(pg_stat_database_deadlocks[5m])` | Should normally be flat at zero. Any spike is immediately visible and worth investigating. |

### Dashboard 3: Nginx Proxy
**Purpose:** Frontend layer health. Use when users report the site is slow or unresponsive but the API seems fine.

| Panel | Visualization | PromQL | Why This Panel |
|---|---|---|---|
| Active Connections | Gauge | `nginx_connections_active` | Current load on the proxy. Low for a personal app; high values indicate problems. |
| Request Rate | Time series | `rate(nginx_http_requests_total[5m])` | Total traffic including static assets. Compare with Go app request rate to understand static vs dynamic traffic split. |
| Connection States | Stacked area | `nginx_connections_reading`, `writing`, `waiting` | Visual breakdown of what connections are doing. High `writing` = backend is slow. High `waiting` = many idle keep-alive connections. |
| Accepted vs Handled | Time series | `rate(nginx_connections_accepted[5m])` vs `rate(nginx_connections_handled[5m])` | Should overlap perfectly. Any gap = Nginx is dropping connections (critical silent failure). |

### Dashboard 4: Kubernetes Resources
**Purpose:** Infrastructure health. Use when everything seems slow or pods are restarting.

| Panel | Visualization | PromQL | Why This Panel |
|---|---|---|---|
| Pod Status | Table | `kube_pod_status_phase{namespace="babymilk"}` | At-a-glance pod health. All should be "Running." Any other state = investigation needed. |
| Container Restarts | Time series | `rate(kube_pod_container_status_restarts_total[1h])` | Trend of restarts over time. Helps distinguish "one-time crash" from "crash loop." |
| CPU Usage vs Request | Time series | `container_cpu_usage_seconds_total` vs `kube_pod_container_resource_requests` | Shows if pods are using more CPU than requested (risk of throttling) or much less (over-provisioned). |
| Memory Usage vs Limit | Time series | `container_memory_working_set_bytes` vs `kube_pod_container_resource_limits` | Usage approaching limit = OOMKill risk. Usage well below limit = potential to reduce limits and free cluster resources. |
| Node CPU % | Gauge | `1 - avg(rate(node_cpu_seconds_total{mode="idle"}[5m]))` | Overall node saturation. >80% means all pods on the node are affected by CPU contention. |
| Node Memory % | Gauge | `1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes` | Overall memory pressure. High values = OOM risk for the entire node. |
| Disk Usage % | Gauge | `1 - node_filesystem_avail_bytes / node_filesystem_size_bytes` | Pairs with `DiskSpaceLow` alert. Visual indicator of disk consumption trend. |
| Network I/O | Time series | `rate(node_network_receive_bytes_total[5m])` | Baseline network usage. Useful for spotting anomalies like backup jobs or unexpected traffic. |

---

## 7. Alerting Rules — Why Each Alert and Its Impact

Alerts are organized by severity. The key principle: **alert on symptoms users would notice, not on every metric fluctuation.**

### Critical (Page immediately) — Immediate user-facing impact

| Alert | Condition | For | Why This Alert | Impact If Missed |
|---|---|---|---|---|
| `AppDown` | `up{job="babymilk-app"} == 0` | 1m | The Go app process is unreachable. No API calls work. | Parents can't log feedings. Complete app outage with no record-keeping. |
| `DatabaseDown` | `pg_up == 0` | 1m | PostgreSQL is unreachable. Every API endpoint returns 500. | Even if the app is "up," all operations fail. Data loss risk if the DB crashed uncleanly. |
| `HighErrorRate` | Error rate > 5% | 5m | More than 1 in 20 requests is failing. Users are experiencing errors. The 5-min `for` duration avoids alerting on single retries or brief deploy windows. | Users see intermittent "something went wrong" errors. Feeding data may be lost if POST requests fail silently on the frontend. |
| `PodCrashLooping` | `rate(kube_pod_container_status_restarts_total[15m]) > 0` | 5m | A container is repeatedly crashing and restarting. The app appears to work between crashes but drops in-flight requests on each restart. | Intermittent failures that are extremely hard to debug without this alert. Users experience random timeouts that "fix themselves" seconds later. |

### Warning (Notify, no page) — Degraded performance or approaching limits

| Alert | Condition | For | Why This Alert | Impact If Missed |
|---|---|---|---|---|
| `HighLatencyP95` | p95 latency > 1s | 5m | The app is slow for 5% of requests. Users notice sluggish behavior. 1s threshold chosen because the BabyMilk API should respond in <100ms for CRUD operations. | Users perceive the app as "laggy." The feeding logging workflow feels broken even though it technically works. |
| `HighDBConnections` | `pg_stat_database_numbackends > 8` | 5m | 80% of the Go app's `MaxOpenConns=10` are in use. One more burst of traffic saturates the pool. | Next traffic spike exhausts the pool, causing requests to hang indefinitely (no timeout configured). |
| `DBPoolExhausted` | `db_connections_open >= db_connections_max_open` | 2m | The connection pool is fully saturated RIGHT NOW. New requests are queuing. Short 2-min window because this is actively causing latency. | All new requests block waiting for a connection. Cascading failure: Nginx proxy timeout → user sees errors. |
| `HighMemoryUsage` | Container memory > 80% of limit | 10m | The container is approaching its memory limit. At 100%, Kubernetes OOM-kills it (immediate crash). 10-min window filters out temporary spikes (e.g., GC pressure). | Sudden OOMKill terminates the process without graceful shutdown. For PostgreSQL, this risks data corruption. |
| `HighCPUUsage` | Container CPU > 80% of request | 10m | Sustained high CPU means the container is at capacity. Request latency increases due to CPU throttling. | Gradual performance degradation. Users experience slower responses but the app doesn't crash — a "silent" failure mode. |
| `DiskSpaceLow` | Node disk > 80% full | 15m | Kubernetes starts evicting pods at ~85% disk usage. PostgreSQL stops accepting writes when disk is full. 15-min window avoids alerting during temporary spikes (e.g., log rotation). | At 100%: PostgreSQL goes read-only, Prometheus stops storing metrics, container logs fail. Recovery requires manual intervention. |
| `NginxConnectionSpike` | `nginx_connections_active` > 100 | 5m | For a personal app, 100 concurrent connections is anomalous — likely a traffic spike, misconfiguration, or abuse. | Without alerting, excessive connections could exhaust Nginx worker connections, causing the frontend to become unresponsive. |
| `HighDBDeadlocks` | `rate(pg_stat_database_deadlocks[5m]) > 0` | 5m | Any deadlock in a simple CRUD app indicates a bug or unexpected concurrent access pattern. Deadlocks cause transaction failures that surface as 500 errors. | Intermittent 500 errors on write operations that are nearly impossible to reproduce without knowing deadlocks are occurring. |

### Info (Log only) — Awareness, no action needed immediately

| Alert | Condition | For | Why This Alert | Impact If Missed |
|---|---|---|---|---|
| `DeploymentReplicaMismatch` | Available ≠ desired replicas | 10m | A rolling update may have stalled or a node failure reduced capacity. Not critical if at least one replica is serving, but worth investigating. | Running at reduced capacity without awareness. Next failure could cause complete outage. |
| `DatabaseGrowing` | DB size > 1GB | — | Proactive capacity signal. For a baby feeding tracker, 1GB means either the app has been running a very long time or something is writing excessive data. | Disk fills up gradually until the critical `DiskSpaceLow` alert fires — but by then you have less time to react. |

---

## 8. Implementation Todos

### Todo 1: Install kube-prometheus-stack via Helm
- Add Helm chart to K8s deployment manifests or scripts
- Configure `values.yaml` with retention, storage, and scrape config
- Deploy to a `monitoring` namespace

### Todo 2: Instrument Go application
- Add `prometheus/client_golang` dependency
- Create metrics middleware for HTTP handlers
- Expose `/metrics` endpoint
- Add DB pool stats collector
- Add business metrics (feedings_created_total, etc.)

### Todo 3: Deploy postgres_exporter
- Create K8s Deployment + Service for postgres_exporter
- Configure ServiceMonitor for Prometheus discovery
- Add custom query for feedings table count

### Todo 4: Configure Nginx for monitoring
- Add `stub_status` location to `deploy/nginx.conf`
- Deploy nginx-prometheus-exporter sidecar
- Create ServiceMonitor

### Todo 5: Create Grafana dashboards
- Build 4 dashboards (App, DB, Nginx, K8s) as JSON ConfigMaps
- Import via Grafana provisioning or dashboard-as-code

### Todo 6: Configure alerting rules
- Create PrometheusRule CRDs for critical/warning/info alerts
- Configure AlertManager with notification channels (Slack/email)

### Todo 7: Test and validate
- Verify all scrape targets are up in Prometheus Targets page
- Confirm dashboards populate with live data
- Trigger test alerts

---

## 9. Files to Create/Modify

| File | Action | Description |
|---|---|---|
| `go.mod` | Modify | Add `prometheus/client_golang` |
| `cmd/server/main.go` | Modify | Add `/metrics` endpoint + middleware |
| `internal/app/metrics.go` | Create | Metric definitions + middleware |
| `deploy/nginx.conf` | Modify | Add `stub_status` location |
| `deploy/k8s/monitoring/` | Create | Directory for monitoring manifests |
| `deploy/k8s/monitoring/kube-prometheus-values.yaml` | Create | Helm values for kube-prometheus-stack |
| `deploy/k8s/monitoring/postgres-exporter.yaml` | Create | postgres_exporter Deployment + ServiceMonitor |
| `deploy/k8s/monitoring/nginx-exporter.yaml` | Create | nginx-exporter sidecar config |
| `deploy/k8s/monitoring/alerting-rules.yaml` | Create | PrometheusRule CRDs |
| `deploy/k8s/monitoring/dashboards/` | Create | Grafana dashboard JSON ConfigMaps |
| `scripts/monitoring-setup.sh` | Create | Script to deploy monitoring stack |
