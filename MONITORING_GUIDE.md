# EventPulse Monitoring Stack (Prometheus & Grafana) Guide

Complete guide for deploying and using Prometheus and Grafana to monitor EventPulse services.

**Objective**: Collect metrics from all services and visualize performance, errors, and scaling behavior.

---

## What is Prometheus & Grafana?

**Prometheus**:
- Time-series database for metrics
- Scrapes HTTP endpoints at regular intervals (15s default)
- Stores metrics for 7 days
- Provides query language (PromQL) for analysis

**Grafana**:
- Visualization platform for metrics
- Creates dashboards and alerts
- Connects to Prometheus as data source
- Web-based UI for exploration

**Together**:
```
EventPulse Services
    ↓ (expose /metrics)
Prometheus (scrapes, stores)
    ↓ (queries)
Grafana (visualizes)
    ↓
User Dashboard
```

---

## Prerequisites

- All EventPulse services deployed (Phases 1-5)
- Services have `/metrics` endpoints:
  - api-gateway:8080/metrics
  - analytics-service:8081/metrics
  - alert-service:8082/metrics
- Prometheus scrape annotations on pods (already added)
- Persistent volume provisioner (StorageClass)

---

## Phase 6: Deploy Monitoring Stack

### Step 1: Deploy Prometheus

```bash
kubectl apply -f k8s/monitoring/prometheus-deployment.yaml
```

**What this creates**:
- Namespace: eventpulse (already exists)
- PersistentVolumeClaim: 10Gi for metrics storage
- ConfigMap: prometheus.yml (scrape configuration)
- ServiceAccount, ClusterRole, ClusterRoleBinding (RBAC)
- Deployment: Prometheus (1 replica)
- Service: prometheus:9090

**Verify deployment**:
```bash
kubectl get deployment -n eventpulse prometheus
kubectl rollout status deployment/prometheus -n eventpulse -w
```

**Expected output**:
```
NAME         READY   UP-TO-DATE   AVAILABLE   AGE
prometheus   1/1     1            1           30s
```

### Step 2: Deploy Grafana

```bash
kubectl apply -f k8s/monitoring/grafana-deployment.yaml
```

**What this creates**:
- PersistentVolumeClaim: 5Gi for dashboards/config
- ConfigMap: grafana.ini (configuration)
- Deployment: Grafana (1 replica)
- Service: grafana:3000

**Verify deployment**:
```bash
kubectl get deployment -n eventpulse grafana
kubectl rollout status deployment/grafana -n eventpulse -w
```

**Expected output**:
```
NAME      READY   UP-TO-DATE   AVAILABLE   AGE
grafana   1/1     1            1           30s
```

### Step 3: Verify Both Services

```bash
kubectl get pods -n eventpulse -l app=prometheus,app=grafana
kubectl get service -n eventpulse prometheus grafana
```

**Expected output**:
```
NAME                     READY   STATUS    RESTARTS   AGE
prometheus-xxxxx         1/1     Running   0          1m
grafana-yyyyy            1/1     Running   0          1m

NAME         TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
prometheus   ClusterIP   10.xxx.xxx.xxx   <none>        9090/TCP   1m
grafana      ClusterIP   10.xxx.xxx.xxx   <none>        3000/TCP   1m
```

---

## Verification: Monitoring is Running

### Check Prometheus

**Port-forward to Prometheus**:
```bash
kubectl port-forward -n eventpulse svc/prometheus 9090:9090
```

**Open browser**:
```
http://localhost:9090
```

**Verify targets**:
- Click "Status" → "Targets"
- Should see:
  - api-gateway (1/1 UP)
  - analytics-service (1/1 UP)
  - alert-service (1/1 UP)
  - kubernetes-apiservers (1/1 UP)
  - kubernetes-nodes (1/1 UP)

**Query metrics**:
- Click "Graph" tab
- Enter PromQL query (see Metric Queries below)
- Example: `eventpulse_events_published_total`
- Click "Execute" → "Graph"

### Check Grafana

**Port-forward to Grafana**:
```bash
kubectl port-forward -n eventpulse svc/grafana 3000:3000
```

**Open browser**:
```
http://localhost:3000
```

**Login**:
- Username: admin
- Password: admin

**Verify Prometheus datasource**:
- Click "Configuration" (gear icon) → "Data Sources"
- Should see "Prometheus" with status "green" (OK)

---

## Metrics Collection

### What Metrics Are Collected?

**API Gateway** (api-gateway:8080/metrics):
```
eventpulse_events_published_total         # Total events published to Kafka
eventpulse_events_published_duration_ms   # Event publish latency
eventpulse_http_request_duration_ms       # HTTP request latency
eventpulse_http_requests_total            # Total HTTP requests
eventpulse_errors_total                   # Total errors
```

**Analytics Service** (analytics-service:8081/metrics):
```
eventpulse_events_processed_total         # Total events processed
eventpulse_events_processed_duration_ms   # Processing latency
eventpulse_risk_scores_total              # Risk scores calculated
eventpulse_kafka_consumer_lag             # Kafka consumer lag
```

**Alert Service** (alert-service:8082/metrics):
```
eventpulse_alerts_generated_total         # Total alerts generated
eventpulse_alerts_generated_duration_ms   # Alert generation latency
eventpulse_high_risk_transactions         # High-risk transaction count
eventpulse_database_writes_total          # Database write count
eventpulse_database_write_errors_total    # Database write errors
```

**Kubernetes** (built-in):
```
container_cpu_usage_seconds_total          # Pod CPU usage
container_memory_usage_bytes               # Pod memory usage
kube_pod_status_phase                      # Pod status (Running, Pending, etc.)
kube_deployment_status_replicas            # Deployment replica counts
```

### PromQL Query Examples

**Total events published**:
```promql
rate(eventpulse_events_published_total[5m])
```

**Average event publish latency**:
```promql
rate(eventpulse_events_published_duration_ms_sum[5m]) / rate(eventpulse_events_published_duration_ms_count[5m])
```

**Kafka consumer lag (Analytics)**:
```promql
eventpulse_kafka_consumer_lag{service="analytics-service"}
```

**Total alerts generated per minute**:
```promql
rate(eventpulse_alerts_generated_total[1m])
```

**Error rate (API Gateway)**:
```promql
rate(eventpulse_errors_total{service="api-gateway"}[5m])
```

**Pod CPU usage**:
```promql
rate(container_cpu_usage_seconds_total{pod=~"api-gateway.*"}[5m]) * 100
```

**Pod memory usage**:
```promql
container_memory_usage_bytes{pod=~"api-gateway.*"} / 1024 / 1024
```

**HPA replica count**:
```promql
kube_deployment_status_replicas{deployment=~"api-gateway|analytics-service|alert-service"}
```

---

## Creating Dashboards

### Dashboard 1: EventPulse Overview

**Purpose**: High-level system health and throughput

**Panels** (create each panel in Grafana):

1. **Events Published Per Minute**
   - Query: `rate(eventpulse_events_published_total[1m])`
   - Type: Graph
   - Unit: short (events/sec)

2. **Alerts Generated Per Minute**
   - Query: `rate(eventpulse_alerts_generated_total[1m])`
   - Type: Graph
   - Unit: short

3. **Total Events in Pipeline**
   - Query: `eventpulse_events_published_total - eventpulse_events_processed_total`
   - Type: Stat
   - Unit: short

4. **Kafka Consumer Lag**
   - Query: `eventpulse_kafka_consumer_lag`
   - Type: Graph (one line per service)
   - Unit: short (messages)

5. **Error Rate (API Gateway)**
   - Query: `rate(eventpulse_errors_total{service="api-gateway"}[5m]) * 100`
   - Type: Graph
   - Unit: percent

6. **Current Replicas**
   - Query: `kube_deployment_status_replicas{deployment=~"api-gateway|analytics-service|alert-service"}`
   - Type: Graph (one line per service)
   - Unit: short

### Dashboard 2: API Gateway Performance

**Purpose**: Deep dive into API Gateway metrics

**Panels**:

1. **Request Rate (RPS)**
   - Query: `rate(eventpulse_http_requests_total{service="api-gateway"}[1m])`
   - Type: Graph
   - Unit: short (requests/sec)

2. **Request Latency (p50, p95, p99)**
   - Query:
     - p50: `histogram_quantile(0.50, rate(eventpulse_http_request_duration_ms_bucket[5m]))`
     - p95: `histogram_quantile(0.95, rate(eventpulse_http_request_duration_ms_bucket[5m]))`
     - p99: `histogram_quantile(0.99, rate(eventpulse_http_request_duration_ms_bucket[5m]))`
   - Type: Graph
   - Unit: ms

3. **Events Published Per Minute**
   - Query: `rate(eventpulse_events_published_total[1m])`
   - Type: Graph
   - Unit: short

4. **Event Publish Latency**
   - Query: `rate(eventpulse_events_published_duration_ms_sum[5m]) / rate(eventpulse_events_published_duration_ms_count[5m])`
   - Type: Graph
   - Unit: ms

5. **Error Count**
   - Query: `rate(eventpulse_errors_total{service="api-gateway"}[1m])`
   - Type: Graph
   - Unit: short

6. **Pod CPU Usage**
   - Query: `rate(container_cpu_usage_seconds_total{pod=~"api-gateway.*"}[5m]) * 100`
   - Type: Graph
   - Unit: percent

### Dashboard 3: Kafka & Analytics

**Purpose**: Monitor Kafka message flow and analytics processing

**Panels**:

1. **Raw Events Consumed Per Minute**
   - Query: `rate(eventpulse_events_processed_total{service="analytics-service"}[1m])`
   - Type: Graph
   - Unit: short

2. **Analytics Processing Latency**
   - Query: `rate(eventpulse_events_processed_duration_ms_sum[5m]) / rate(eventpulse_events_processed_duration_ms_count[5m])`
   - Type: Graph
   - Unit: ms

3. **Kafka Consumer Lag (Analytics)**
   - Query: `eventpulse_kafka_consumer_lag{consumer_group="analytics-group"}`
   - Type: Stat
   - Unit: short (messages behind)

4. **Risk Scores Calculated**
   - Query: `rate(eventpulse_risk_scores_total[1m])`
   - Type: Graph
   - Unit: short (scores/sec)

5. **Kafka Consumer Lag (Alert)**
   - Query: `eventpulse_kafka_consumer_lag{consumer_group="alert-group"}`
   - Type: Stat
   - Unit: short

6. **Processing Throughput Trend**
   - Query: `rate(eventpulse_events_processed_total[5m])`
   - Type: Graph
   - Legend: `{{ service }}`

### Dashboard 4: Alert Service & Database

**Purpose**: Monitor alert generation and database operations

**Panels**:

1. **Alerts Generated Per Minute**
   - Query: `rate(eventpulse_alerts_generated_total[1m])`
   - Type: Graph
   - Unit: short

2. **Alert Generation Latency**
   - Query: `rate(eventpulse_alerts_generated_duration_ms_sum[5m]) / rate(eventpulse_alerts_generated_duration_ms_count[5m])`
   - Type: Graph
   - Unit: ms

3. **High Risk Transactions**
   - Query: `rate(eventpulse_high_risk_transactions[1m])`
   - Type: Stat
   - Unit: short

4. **Database Writes Per Minute**
   - Query: `rate(eventpulse_database_writes_total[1m])`
   - Type: Graph
   - Unit: short

5. **Database Write Errors**
   - Query: `rate(eventpulse_database_write_errors_total[1m])`
   - Type: Graph
   - Unit: short
   - Alert: If > 0, red background

6. **Kafka Lag (Alert Service)**
   - Query: `eventpulse_kafka_consumer_lag{consumer_group="alert-group"}`
   - Type: Stat
   - Unit: short

---

## Monitoring Metrics During Load Tests

### Monitor API Gateway Under Load

```bash
# Terminal 1: Generate load
ab -n 10000 -c 100 http://localhost/events

# Terminal 2: Watch Prometheus metrics
# Open http://localhost:9090
# Query: rate(eventpulse_events_published_total[1m])
# Watch the rate increase as load increases
```

**Expected behavior**:
```
t=0s:   0 events/sec
t=5s:   100 events/sec
t=10s:  180 events/sec (increasing)
t=20s:  250 events/sec (peak load)
t=30s:  200 events/sec (load backing off)
```

### Monitor Kafka Consumer Lag

**Send events and watch lag**:

```bash
# Terminal 1: Send 1000 events
for i in {1..1000}; do
  curl -s -X POST http://localhost:8000/events \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"user_$i\",\"event_type\":\"purchase\",\"amount\":$((50000 + RANDOM % 50000))}" &
done
wait

# Terminal 2: Query Prometheus
# http://localhost:9090
# Query: eventpulse_kafka_consumer_lag
# Watch lag increase, then decrease as services catch up
```

**Expected behavior**:
```
t=0s:   Lag=0
t=5s:   Lag=800 (1000 events - 200 processed)
t=10s:  Lag=400
t=15s:  Lag=50
t=20s:  Lag=0 (fully caught up)
```

### Monitor Alert Generation

**Observe alerts being generated**:

```bash
# Terminal 1: Generate high-risk events
for i in {1..50}; do
  curl -s -X POST http://localhost:8000/events \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"user_$i\",\"event_type\":\"purchase\",\"amount\":100000}" &
done
wait

# Terminal 2: Query Prometheus
# http://localhost:9090
# Query: rate(eventpulse_alerts_generated_total[1m])
# Query: eventpulse_high_risk_transactions
# Watch metrics increase
```

**Expected behavior**:
```
Alerts per minute: 5-10 (depending on latency)
High-risk transactions: 50 (all should generate alerts)
```

---

## Grafana Dashboard Creation (Step-by-Step)

### Create New Dashboard

1. Click "+" (plus icon) in left sidebar
2. Click "Dashboard"
3. Click "Add new panel"

### Add Panel

1. **Query Tab**: Enter PromQL query
   - Example: `rate(eventpulse_events_published_total[1m])`
2. **Visualization Tab**: Choose chart type
   - Graph: Time series visualization
   - Stat: Current value
   - Gauge: Dial/progress bar
   - Table: Tabular data
3. **Panel Title & Options**:
   - Title: descriptive name
   - Unit: (ms, short, percent, etc.)
   - Legend: show/hide
4. **Click "Save"**

### Save Dashboard

1. Click "Save" (top right)
2. Enter Dashboard Name: "EventPulse Overview"
3. Choose Folder: "EventPulse"
4. Click "Save"

---

## Alerting (Optional)

### Alert When Event Rate Drops

1. In Prometheus: http://localhost:9090/alerts
2. Create Alert Rule (advanced use case):

```promql
# Alert if no events published for 5 minutes
rate(eventpulse_events_published_total[5m]) == 0
```

### Alert When Kafka Lag is High

```promql
# Alert if lag > 1000 messages
eventpulse_kafka_consumer_lag > 1000
```

### Alert When Error Rate High

```promql
# Alert if error rate > 1%
rate(eventpulse_errors_total[5m]) / rate(eventpulse_http_requests_total[5m]) > 0.01
```

---

## Troubleshooting

### Prometheus shows no targets

```bash
kubectl describe pod -n eventpulse prometheus-xxxxx
# Look for "Error" or "Warning" events

kubectl logs -n eventpulse deployment/prometheus
# Look for scrape errors
```

**Common issues**:
- Service not accessible (check network policies)
- Metrics endpoint not responding (check `/metrics` endpoint)
- Scrape configuration incorrect (check prometheus.yml)

### Grafana can't connect to Prometheus

**Verify Prometheus is running**:
```bash
kubectl port-forward -n eventpulse svc/prometheus 9090:9090
curl http://localhost:9090/-/ready
```

**Verify datasource in Grafana**:
1. Configuration → Data Sources
2. Click "Prometheus"
3. URL should be: `http://prometheus:9090`
4. Click "Test"

**If test fails**: Check Prometheus pod logs

### No metrics data showing in Grafana

**Verify metrics are being scraped**:
```bash
# Query Prometheus directly
curl http://prometheus:9090/api/v1/query?query=eventpulse_events_published_total
```

**If no data**:
1. Check if services have `/metrics` endpoints
2. Verify Prometheus scrape targets (Status → Targets)
3. Wait 60+ seconds for initial scrape

### High Prometheus memory usage

**Issue**: Prometheus storing too much data

**Solution**:
1. Reduce retention (edit prometheus-deployment.yaml):
   ```yaml
   - --storage.tsdb.retention.time=3d  # 3 days instead of 7
   ```
2. Delete old data:
   ```promql
   DELETE SERIES WHERE metric_name=~".*"
   # (Be careful, this deletes ALL data)
   ```

---

## Quick Reference

| Task | Command/URL |
|------|------------|
| Deploy Prometheus | `kubectl apply -f k8s/monitoring/prometheus-deployment.yaml` |
| Deploy Grafana | `kubectl apply -f k8s/monitoring/grafana-deployment.yaml` |
| Check Prometheus status | `kubectl logs -n eventpulse deployment/prometheus` |
| Check Grafana status | `kubectl logs -n eventpulse deployment/grafana` |
| Access Prometheus UI | `kubectl port-forward -n eventpulse svc/prometheus 9090:9090` then `http://localhost:9090` |
| Access Grafana UI | `kubectl port-forward -n eventpulse svc/grafana 3000:3000` then `http://localhost:3000` |
| View scrape targets | `http://localhost:9090/targets` (after port-forward) |
| Query metrics | `http://localhost:9090/graph` → Enter PromQL query |
| View dashboards | Grafana home page (after port-forward) |
| Check pod metrics | `kubectl top pods -n eventpulse` |

---

## Production Checklist

- [ ] Prometheus storage sized for retention period
- [ ] Grafana default password changed
- [ ] Dashboards created and tested
- [ ] Alerting rules configured
- [ ] Backup strategy for Prometheus/Grafana data
- [ ] Metrics ingestion rate understood
- [ ] Scaling behavior monitored during load tests
- [ ] Error rates monitored and within acceptable range
- [ ] Kafka consumer lag monitored and acceptable
- [ ] Database performance metrics tracked

---

## Summary

 Prometheus deployed and scraping all EventPulse services  
 Grafana connected to Prometheus datasource  
 Metrics collected: requests, events, alerts, errors, lag  
 4 dashboards documented for visualization  
 Load testing monitoring procedures provided  
 Troubleshooting guide for common issues  

**Complete Kubernetes Stack Ready**:
-  Phase 1: PostgreSQL
-  Phase 2: Kafka
-  Phase 3: Application Services
-  Phase 4: NGINX Ingress
-  Phase 5: HPA
-  Phase 6: Monitoring (Prometheus + Grafana)

**Next**: Production hardening (network policies, pod security, RBAC fine-tuning)