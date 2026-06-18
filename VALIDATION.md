# EventPulse Kubernetes Validation Guide

Complete end-to-end validation of the EventPulse Kubernetes deployment pipeline.

**Objective**: Verify the complete data flow from API Gateway through Kafka, Analytics Service, Alert Service, to PostgreSQL and back.

---

## Prerequisites

- All services deployed: PostgreSQL, Kafka, API Gateway, Analytics Service, Alert Service
- All pods in READY 1/1 state: `kubectl get pods -n eventpulse`
- All services running: `kubectl get service -n eventpulse`
- ConfigMap and Secrets created: `kubectl get configmap,secret -n eventpulse`

---

## Validation Pipeline

```
1. Check service health & readiness
2. Send transaction event via POST /events
3. Verify event in Kafka (events.raw topic)
4. Monitor Analytics Service processing
5. Verify processed event in Kafka (events.processed topic)
6. Monitor Alert Service generation
7. Query PostgreSQL for stored alerts
8. Retrieve alerts via GET /alerts
```

---

## Step 1: Verify All Services Are Healthy

### 1.1 Check Pod Status

```bash
kubectl get pods -n eventpulse -o wide
```

**Expected output**:
```
NAME                               READY   STATUS    RESTARTS   AGE     IP            NODE
postgres-xxxxx                     1/1     Running   0          10m     10.244.x.x    <node>
kafka-xxxxx                        1/1     Running   0          8m      10.244.x.x    <node>
api-gateway-xxxxx                  1/1     Running   0          5m      10.244.x.x    <node>
api-gateway-yyyyy                  1/1     Running   0          5m      10.244.x.x    <node>
analytics-service-xxxxx            1/1     Running   0          5m      10.244.x.x    <node>
analytics-service-yyyyy            1/1     Running   0          5m      10.244.x.x    <node>
alert-service-xxxxx                1/1     Running   0          5m      10.244.x.x    <node>
alert-service-yyyyy                1/1     Running   0          5m      10.244.x.x    <node>
```

**Verification**: All pods show READY 1/1 and STATUS Running

### 1.2 Check Service Endpoints

```bash
kubectl get endpoints -n eventpulse
```

**Expected output**:
```
NAME                 ENDPOINTS                           AGE
postgres             10.244.x.x:5432                     10m
kafka                10.244.x.x:9092,10.244.x.x:9093     8m
api-gateway          10.244.x.x:8080,10.244.x.x:8080     5m
analytics-service    10.244.x.x:8081,10.244.x.x:8081     5m
alert-service        10.244.x.x:8082,10.244.x.x:8082     5m
```

**Verification**: All services have endpoints (pods behind them)

### 1.3 Check Health Endpoints

```bash
# API Gateway health
kubectl run -it --rm health-check --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://api-gateway:8080/health | jq .

# Analytics Service health
kubectl run -it --rm health-check --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://analytics-service:8081/health | jq .

# Alert Service health
kubectl run -it --rm health-check --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://alert-service:8082/health | jq .
```

**Expected output** (each service):
```json
{
  "status": "ok",
  "service": "<service-name>",
  "timestamp": "2026-06-18T20:00:00Z"
}
```

**Verification**: All services report status "ok"

---

## Step 2: Send Transaction Event

### 2.1 Post a High-Risk Event (>$10,000)

```bash
kubectl run -it --rm event-sender --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s -X POST http://api-gateway:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user_001",
    "event_type": "purchase",
    "amount": 75000
  }' | jq .
```

**Expected output**:
```json
{
  "message": "Event Published"
}
```

**Verification**: Event accepted and published to Kafka

### 2.2 Check API Gateway Logs

```bash
kubectl logs -n eventpulse deployment/api-gateway --tail=20
```

**Expected output** (last 20 lines):
```
{"time":"2026-06-18T20:00:00.123Z","level":"INFO","msg":"published event","service":"api-gateway","user_id":"test_user_001","amount":75000,"event_id":"xxxxxxxx"}
```

**Verification**: Event logging shows successful publish with user_id and amount

---

## Step 3: Verify Event in Kafka (events.raw)

### 3.1 List Kafka Topics

```bash
kubectl run -it --rm kafka-check --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 \
  --list
```

**Expected output**:
```
__consumer_offsets
alerts
events.dlq
events.processed
events.raw
```

**Verification**: All required topics exist

### 3.2 Check events.raw Topic

```bash
kubectl run -it --rm kafka-check --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 \
  --describe \
  --topic events.raw
```

**Expected output**:
```
Topic: events.raw	TopicId: xxxxx	PartitionCount: 1	ReplicationFactor: 1
	Topic: events.raw	Partition: 0	Leader: 1	Replicas: 1	Isr: 1
```

**Verification**: Topic exists with 1 partition, leader is broker 1

### 3.3 Consume Raw Event

```bash
kubectl run -it --rm kafka-consumer --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server kafka:9092 \
  --topic events.raw \
  --from-beginning \
  --max-messages 1
```

**Expected output**:
```
{"user_id":"test_user_001","event_type":"purchase","amount":75000,"event_id":"xxxxxxxx"}
```

**Verification**: Raw event exists in Kafka with correct structure

---

## Step 4: Monitor Analytics Service Processing

### 4.1 Check Analytics Service Logs

```bash
kubectl logs -n eventpulse deployment/analytics-service --tail=30
```

**Expected output**:
```
{"time":"2026-06-18T20:00:00.234Z","level":"INFO","msg":"processed event","service":"analytics-service","user_id":"test_user_001","risk_score":90}
```

**Verification**: Service processed event and calculated risk score (should be 90 for amount > 10000)

### 4.2 Exec into Analytics Pod to Check Offsets

```bash
# Get a pod name
kubectl get pods -n eventpulse -l app=analytics-service -o jsonpath='{.items[0].metadata.name}'

# Get the pod name and use it
POD=$(kubectl get pods -n eventpulse -l app=analytics-service -o jsonpath='{.items[0].metadata.name}')

# Check consumer group status
kubectl exec -it -n eventpulse $POD -- \
  /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server kafka:9092 \
  --group analytics-group \
  --describe
```

**Expected output**:
```
GROUP           TOPIC          PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG             CONSUMER-ID
analytics-group events.raw     0          1               1               0               <consumer-id>
```

**Verification**: Consumer group has consumed the message (LAG = 0 means caught up)

---

## Step 5: Verify Processed Event in Kafka (events.processed)

### 5.1 Consume Processed Event

```bash
kubectl run -it --rm kafka-consumer --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server kafka:9092 \
  --topic events.processed \
  --from-beginning \
  --max-messages 1
```

**Expected output**:
```
{"user_id":"test_user_001","event_type":"purchase","amount":75000,"risk_score":90,"event_id":"xxxxxxxx"}
```

**Verification**: Processed event contains risk_score (90 for high-risk transaction)

### 5.2 Verify Topic Partitions

```bash
kubectl run -it --rm kafka-check --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka:9092 \
  --describe \
  --topic events.processed
```

**Expected output**:
```
Topic: events.processed	TopicId: yyyyy	PartitionCount: 1	ReplicationFactor: 1
	Topic: events.processed	Partition: 0	Leader: 1	Replicas: 1	Isr: 1
```

**Verification**: Topic exists and broker 1 is leader

---

## Step 6: Monitor Alert Service Generation

### 6.1 Check Alert Service Logs

```bash
kubectl logs -n eventpulse deployment/alert-service --tail=30
```

**Expected output**:
```
{"time":"2026-06-18T20:00:00.345Z","level":"INFO","msg":"alert generated","service":"alert-service","user_id":"test_user_001","risk_score":90,"event_id":"xxxxxxxx"}
```

**Verification**: Service generated alert for high-risk event (risk_score >= 80)

### 6.2 Check Alert Service Consumer Group

```bash
POD=$(kubectl get pods -n eventpulse -l app=alert-service -o jsonpath='{.items[0].metadata.name}')

kubectl exec -it -n eventpulse $POD -- \
  /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server kafka:9092 \
  --group alert-group \
  --describe
```

**Expected output**:
```
GROUP       TOPIC              PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG             CONSUMER-ID
alert-group events.processed    0          1               1               0               <consumer-id>
```

**Verification**: Consumer group consumed the processed event (LAG = 0)

---

## Step 7: Verify Alert in PostgreSQL

### 7.1 Port-Forward to PostgreSQL

```bash
kubectl port-forward -n eventpulse svc/postgres 5432:5432 &
```

**Background process**: Now you can connect to PostgreSQL on localhost:5432

### 7.2 Connect to PostgreSQL

```bash
# Using psql from a pod
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local \
  -U admin \
  -d eventpulse \
  -c "SELECT id, user_id, risk_score, message, created_at FROM alerts ORDER BY id DESC LIMIT 1;"
```

**Expected output**:
```
 id | user_id       | risk_score |           message            |           created_at
----+---------------+------------+------------------------------+-------------------------------
  1 | test_user_001 |         90 | HIGH RISK TRANSACTION DETECTED | 2026-06-18 20:00:00.456+00
(1 row)
```

**Verification**: Alert stored in PostgreSQL with correct user_id, risk_score, and message

### 7.3 Check Alert Count

```bash
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local \
  -U admin \
  -d eventpulse \
  -c "SELECT COUNT(*) FROM alerts;"
```

**Expected output**:
```
 count
-------
     1
(1 row)
```

**Verification**: Alert count increased by 1

### 7.4 Check Alert Table Schema

```bash
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local \
  -U admin \
  -d eventpulse \
  -c "\d alerts"
```

**Expected output**:
```
                         Table "public.alerts"
   Column   |            Type             | Collation | Nullable | Default
------------+-----------------------------+-----------+----------+---------
 id         | integer                     |           | not null | nextval('alerts_id_seq'::regclass)
 event_id   | text                        |           | not null | ''::text
 user_id    | text                        |           | not null |
 risk_score | integer                     |           | not null |
 message    | text                        |           | not null |
 created_at | timestamp with time zone    |           | not null | now()
```

**Verification**: Table has correct schema with event_id uniqueness constraint

---

## Step 8: Retrieve Alerts via API

### 8.1 GET /alerts Endpoint

```bash
kubectl run -it --rm api-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://api-gateway:8080/alerts | jq .
```

**Expected output**:
```json
[
  {
    "id": 1,
    "user_id": "test_user_001",
    "risk_score": 90,
    "message": "HIGH RISK TRANSACTION DETECTED",
    "created_at": "2026-06-18T20:00:00.456Z"
  }
]
```

**Verification**: Alert returned via REST API with all expected fields

### 8.2 GET /alert by ID

```bash
kubectl run -it --rm api-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s "http://api-gateway:8080/alert?id=1" | jq .
```

**Expected output**:
```json
{
  "id": 1,
  "user_id": "test_user_001",
  "risk_score": 90,
  "message": "HIGH RISK TRANSACTION DETECTED",
  "created_at": "2026-06-18T20:00:00.456Z"
}
```

**Verification**: Single alert retrieval works correctly

### 8.3 Check API Gateway Access Logs

```bash
kubectl logs -n eventpulse deployment/api-gateway --tail=20 | grep "published event\|GET /alerts"
```

**Expected output**:
```
{"time":"2026-06-18T20:00:00.123Z","level":"INFO","msg":"published event",...}
{"time":"2026-06-18T20:00:00.789Z","level":"INFO","msg":"http request",...}
```

**Verification**: API Gateway logged both event publication and GET request

---

## Advanced Debugging Techniques

### Tail Logs from All Services

```bash
# Watch all service logs in real-time
kubectl logs -n eventpulse \
  -f \
  -l tier=services \
  --all-containers=true \
  --timestamps=true
```

**Use case**: Monitor all services simultaneously as events flow through

### Port-Forward Multiple Services

```bash
# Terminal 1: Forward API Gateway
kubectl port-forward -n eventpulse svc/api-gateway 8080:8080 &

# Terminal 2: Forward Analytics Service metrics
kubectl port-forward -n eventpulse svc/analytics-service 8081:8081 &

# Terminal 3: Forward Alert Service metrics
kubectl port-forward -n eventpulse svc/alert-service 8082:8082 &

# Terminal 4: Forward PostgreSQL
kubectl port-forward -n eventpulse svc/postgres 5432:5432 &

# Terminal 5: Forward Kafka
kubectl port-forward -n eventpulse svc/kafka 9092:9092 &
```

**Use case**: Access all services locally for debugging

### Exec into Pod and Run Commands

```bash
# Get pod name
POD=$(kubectl get pods -n eventpulse -l app=api-gateway -o jsonpath='{.items[0].metadata.name}')

# Execute shell command in pod
kubectl exec -it -n eventpulse $POD -- /bin/sh

# Inside the pod, you can run:
# - curl http://localhost:8080/health
# - env | grep KAFKA_BROKERS
# - ps aux | grep api-gateway
```

**Use case**: Inspect environment variables, process state, network connectivity

### Check Metrics from Services

```bash
# Get Prometheus metrics from API Gateway
kubectl run -it --rm metrics --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://api-gateway:8080/metrics | grep eventpulse_

# Get Prometheus metrics from Analytics Service
kubectl run -it --rm metrics --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://analytics-service:8081/metrics | grep eventpulse_

# Get Prometheus metrics from Alert Service
kubectl run -it --rm metrics --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s http://alert-service:8082/metrics | grep eventpulse_
```

**Expected output**:
```
# HELP eventpulse_events_published_total Total events published to Kafka.
# TYPE eventpulse_events_published_total counter
eventpulse_events_published_total{service="api-gateway",topic="events.raw"} 1
eventpulse_events_processed_total{service="analytics-service"} 1
eventpulse_alerts_generated_total{service="alert-service"} 1
```

**Verification**: Metrics show correct event counts through pipeline

---

## Testing Low-Risk Events (Should Not Generate Alerts)

### Send a Low-Risk Event

```bash
kubectl run -it --rm event-sender --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s -X POST http://api-gateway:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user_002",
    "event_type": "view",
    "amount": 5000
  }' | jq .
```

**Expected output**:
```json
{
  "message": "Event Published"
}
```

### Verify No Alert Generated

```bash
# Check alert count (should still be 1)
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local \
  -U admin \
  -d eventpulse \
  -c "SELECT COUNT(*) FROM alerts WHERE user_id='test_user_002';"
```

**Expected output**:
```
 count
-------
     0
(1 row)
```

**Verification**: Low-risk event processed but no alert generated (risk_score < 80)

---

## Testing Idempotency (Duplicate Event Handling)

### Send Same Event Twice

```bash
# Send event once
kubectl run -it --rm event-sender --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s -X POST http://api-gateway:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user_003",
    "event_type": "purchase",
    "amount": 50000,
    "event_id": "duplicate_test_001"
  }' | jq .

# Send same event again (same event_id)
kubectl run -it --rm event-sender --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -s -X POST http://api-gateway:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "test_user_003",
    "event_type": "purchase",
    "amount": 50000,
    "event_id": "duplicate_test_001"
  }' | jq .
```

### Verify Alert Count

```bash
# Should have exactly 1 alert for test_user_003, not 2
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local \
  -U admin \
  -d eventpulse \
  -c "SELECT COUNT(*) FROM alerts WHERE user_id='test_user_003';"
```

**Expected output**:
```
 count
-------
     1
(1 row)
```

**Verification**: Duplicate event_id was rejected via PostgreSQL unique constraint (idempotency works)

---

## Troubleshooting Common Issues

### Issue: Pod in CrashLoopBackOff

```bash
# Check pod status
kubectl describe pod -n eventpulse <pod-name>

# Check logs from crashed container
kubectl logs -n eventpulse <pod-name> --previous

# Check events
kubectl get events -n eventpulse --sort-by='.lastTimestamp'
```

### Issue: Service Connection Refused

```bash
# Check if pod is ready
kubectl get pods -n eventpulse -l app=<service> -o wide

# Check DNS resolution from another pod
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  nslookup <service>.eventpulse.svc.cluster.local

# Test port connectivity
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  nc -zv <service> <port>
```

### Issue: Kafka Messages Not Being Consumed

```bash
# Check consumer group lag
kubectl run -it --rm kafka-check --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server kafka:9092 \
  --group <consumer-group> \
  --describe

# Check if topic has messages
kubectl run -it --rm kafka-check --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-run-class.sh \
  kafka.tools.GetOffsetShell