# Monitoring Stack

This document turns `metrics-alerting.md` into deployable infrastructure. It defines a practical Prometheus + Grafana + Alertmanager setup for a single-VPS production MVP, with a clear path to managed monitoring later.

---

## Scope

Canonical metric names, SLOs, dashboards, and alert meanings live in `metrics-alerting.md`. This file covers deployment wiring:

```text
Prometheus scrape config
Grafana provisioning
Alertmanager routing
Docker Compose monitoring profile
internal-only /metrics access
retention/storage
backup of dashboard/rule config
```

`/metrics` must remain internal-only.

---

## Topology

Single VPS baseline:

```text
Go API internal listener :9090
Worker/asynq metrics     :9090 or separate internal port
Postgres exporter        :9187 internal
Redis exporter           :9121 internal
Node exporter            :9100 internal
Prometheus               :9091 bound to 127.0.0.1 or private network
Grafana                  :3000 behind auth/VPN/tunnel
Alertmanager             :9093 bound internal/private
```

Access options:

```text
local SSH tunnel
Cloudflare Access protected tunnel
private VPN
```

Do not expose Prometheus, Alertmanager, or `/metrics` publicly.

---

## Docker Compose Monitoring Profile

```yaml
# docker-compose.monitoring.yml
services:
  prometheus:
    image: prom/prometheus:v2.55.1
    restart: unless-stopped
    command:
      - --config.file=/etc/prometheus/prometheus.yml
      - --storage.tsdb.path=/prometheus
      - --storage.tsdb.retention.time=30d
      - --web.enable-lifecycle
    ports:
      - "127.0.0.1:9091:9090"
    volumes:
      - ./docker/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - ./docker/prometheus/rules:/etc/prometheus/rules:ro
      - prometheus_data:/prometheus
    networks:
      - app
      - monitoring

  alertmanager:
    image: prom/alertmanager:v0.27.0
    restart: unless-stopped
    command:
      - --config.file=/etc/alertmanager/alertmanager.yml
      - --storage.path=/alertmanager
    ports:
      - "127.0.0.1:9093:9093"
    volumes:
      - ./docker/alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro
      - alertmanager_data:/alertmanager
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:11.4.0
    restart: unless-stopped
    environment:
      GF_SECURITY_ADMIN_USER: ${GRAFANA_ADMIN_USER:-admin}
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASSWORD:?set-grafana-password}
      GF_USERS_ALLOW_SIGN_UP: "false"
      GF_ANALYTICS_REPORTING_ENABLED: "false"
    ports:
      - "127.0.0.1:3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./docker/grafana/provisioning:/etc/grafana/provisioning:ro
      - ./docker/grafana/dashboards:/var/lib/grafana/dashboards:ro
    networks:
      - monitoring

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.16.0
    restart: unless-stopped
    environment:
      DATA_SOURCE_NAME: ${POSTGRES_EXPORTER_DSN:?set-postgres-exporter-dsn}
    networks:
      - app
      - monitoring

  redis-exporter:
    image: oliver006/redis_exporter:v1.67.0
    restart: unless-stopped
    environment:
      REDIS_ADDR: redis://redis:6379
      REDIS_PASSWORD: ${REDIS_PASSWORD:?set-redis-password}
    networks:
      - app
      - monitoring

  node-exporter:
    image: prom/node-exporter:v1.8.2
    restart: unless-stopped
    pid: host
    command:
      - --path.rootfs=/host
    volumes:
      - /:/host:ro,rslave
    networks:
      - monitoring

volumes:
  prometheus_data:
  alertmanager_data:
  grafana_data:

networks:
  # Use the same Docker network as the app stack. In `deployment.md` the app network is
  # named `backend`; set the actual external name with `docker network ls`.
  app:
    external: true
    name: yourapp_backend
  monitoring:
```

If your main `docker-compose.yml` uses a different network name, align `networks.app.name` accordingly. Do not leave the network name implicit; otherwise Prometheus/exporters may not reach `api`, `worker`, `postgres`, or `redis`.

---

## Prometheus Config

```yaml
# docker/prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - /etc/prometheus/rules/*.yml

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']

scrape_configs:
  - job_name: yourapp-api
    metrics_path: /metrics
    static_configs:
      - targets: ['api:9090']

  - job_name: yourapp-worker
    metrics_path: /metrics
    static_configs:
      - targets: ['worker:9090']

  - job_name: postgres
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: redis
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: node
    static_configs:
      - targets: ['node-exporter:9100']
```

For systemd deployments, scrape `127.0.0.1:<internal_port>` from Prometheus on the same host or use private network addresses. Do not bind app `/metrics` to `0.0.0.0` unless protected by firewall/private network.

---

## Alert Rules

Start with rules from `metrics-alerting.md` and place them in:

```text
docker/prometheus/rules/yourapp.yml
```

Add infrastructure rules:

```yaml
groups:
  - name: infrastructure
    rules:
      - alert: InstanceDiskSpaceLow
        expr: (node_filesystem_avail_bytes{fstype!~"tmpfs|overlay"} / node_filesystem_size_bytes{fstype!~"tmpfs|overlay"}) < 0.10
        for: 10m
        labels:
          severity: page
        annotations:
          summary: "Disk space below 10%"

      - alert: InstanceMemoryPressure
        expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 0.90
        for: 15m
        labels:
          severity: ticket
        annotations:
          summary: "Memory usage above 90%"

      - alert: PrometheusTargetDown
        expr: up == 0
        for: 2m
        labels:
          severity: page
        annotations:
          summary: "Prometheus target down: {{ $labels.job }}"
```

---

## Alertmanager Config

Example webhook/email skeleton:

```yaml
# docker/alertmanager/alertmanager.yml
global:
  resolve_timeout: 5m

route:
  receiver: default
  group_by: ['alertname', 'job']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - matchers:
        - severity="page"
      receiver: pager
      repeat_interval: 30m

receivers:
  - name: default
    webhook_configs:
      - url: 'https://alerts.example.invalid/default'
        send_resolved: true

  - name: pager
    webhook_configs:
      - url: 'https://alerts.example.invalid/pager'
        send_resolved: true
```

Alertmanager does not expand shell-style `${ALERT_WEBHOOK_URL}` placeholders by itself. Generate this file from a template during deployment, mount it from a secret, or use a notifier sidecar that reads secrets. Do not commit real webhook URLs.

---

## Grafana Provisioning

Datasource:

```yaml
# docker/grafana/provisioning/datasources/prometheus.yml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
```

Dashboard provider:

```yaml
# docker/grafana/provisioning/dashboards/dashboards.yml
apiVersion: 1

providers:
  - name: yourapp
    orgId: 1
    folder: YourApp
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /var/lib/grafana/dashboards
```

Dashboards to provision:

```text
yourapp-api.json
yourapp-db-redis.json
yourapp-workers.json
yourapp-billing.json
yourapp-public-cache.json
yourapp-email.json
yourapp-backups.json
```

Panel requirements are listed in `metrics-alerting.md`.

---

## Exporter Credentials

Postgres exporter should use a least-privilege monitoring user.

Example:

```sql
CREATE USER monitoring_user WITH PASSWORD 'change-me';
GRANT pg_monitor TO monitoring_user;
```

DSN:

```dotenv
POSTGRES_EXPORTER_DSN=postgresql://monitoring_user:change-me@postgres:5432/yourdb?sslmode=disable
```

This user must not be `app_user`, `platform_user`, or `migrator_user`.

---

## Access Control

Grafana:

- Behind Cloudflare Access/VPN/SSH tunnel.
- Strong admin password via secret manager.
- Disable signups.
- Do not expose anonymously.

Prometheus/Alertmanager:

- Bound to `127.0.0.1` or private Docker network.
- Access via SSH tunnel or private network only.

Example SSH tunnel:

```bash
ssh -L 3000:127.0.0.1:3000 -L 9091:127.0.0.1:9091 user@vps
```

Open locally:

```text
Grafana:    http://localhost:3000
Prometheus: http://localhost:9091
```

---

## Retention and Storage

Baseline:

```text
Prometheus retention: 30 days
Grafana dashboards: git-provisioned JSON, not only UI-created
Alert rules: git-managed YAML
Long-term metrics: optional managed Prometheus/Thanos later
```

Monitor Prometheus disk usage; it can fill a small VPS disk.

---

## Backup of Monitoring Config

Back up:

```text
docker/prometheus/prometheus.yml
docker/prometheus/rules/*.yml
docker/alertmanager/alertmanager.yml template
docker/grafana/provisioning/**
docker/grafana/dashboards/*.json
```

Do not rely on Grafana UI-only dashboards. Export and commit dashboard JSON without secrets.

Prometheus time-series data is usually not business-critical; config is.

---

## Deployment Commands

```bash
# Start monitoring stack
docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d prometheus alertmanager grafana postgres-exporter redis-exporter node-exporter

# Validate targets through SSH tunnel or on host
curl -s http://127.0.0.1:9091/-/healthy
curl -s http://127.0.0.1:9091/api/v1/targets | jq '.data.activeTargets[] | {job: .labels.job, health: .health}'

# Reload Prometheus after rule/config changes if lifecycle enabled
curl -X POST http://127.0.0.1:9091/-/reload
```

---

## Production Verification

```text
[ ] /metrics not reachable from public origin
[ ] Prometheus target `yourapp-api` up
[ ] Prometheus target `yourapp-worker` up
[ ] Postgres exporter up
[ ] Redis exporter up
[ ] Node exporter up
[ ] Grafana dashboard loads via private access
[ ] Alertmanager test alert reaches notification channel
[ ] APIHigh5xxRate rule visible
[ ] RedisUnavailable rule visible
[ ] BackupMissing rule visible
[ ] Dashboards/rules stored in git
```

---

## Managed Monitoring Option

For larger teams, replace self-hosted Prometheus/Grafana with:

```text
Grafana Cloud
Prometheus remote_write
Cloudflare analytics/logpush
managed incident/pager provider
```

Rules remain the same:

- `/metrics` internal or privately scraped.
- low-cardinality labels.
- no PII/secrets in labels/logs.
- alerts link to `incident-response.md` runbooks.

---

## Definition of Done

```text
[ ] Monitoring Compose/profile or managed equivalent documented
[ ] Prometheus scrapes API/worker/Postgres/Redis/node
[ ] Alertmanager route configured with secret-managed webhooks
[ ] Grafana datasource and dashboards provisioned from git
[ ] Monitoring endpoints private
[ ] Retention/storage configured
[ ] Test alert delivered
[ ] Alert descriptions link to incident runbooks
[ ] Config backup/export procedure documented
```
