# EventPulse Kubernetes Deployment Guide

This guide walks through deploying EventPulse to Kubernetes step by step, starting with PostgreSQL.

**Status**: Phase 5 — Horizontal Pod Autoscaling  
**Branch**: kubernetes-upgrade  
**Target**: Production-ready Kubernetes setup

---

## Phase 1: PostgreSQL Deployment

PostgreSQL is the stateful component where alerts are persisted. This phase focuses on getting it running with persistent storage and proper health checks.

### Prerequisites

- Kubernetes cluster (1.24+)
- `kubectl` configured and authenticated
- EventPulse namespace exists (or will be created)
- Persistent volume provisioner (StorageClass)

### Step 1: Create the Namespace

```bash
kubectl apply -f k8s/namespace.yaml
```

**Verify**:
```bash
kubectl get namespace eventpulse
# Expected output:
# NAME         STATUS   AGE
# eventpulse   Active   5s
```

### Step 2: Create ConfigMap (Non-Sensitive Config)

The ConfigMap contains all non-sensitive configuration that will be reused by all services.

```bash
kubectl apply -f k8s/configmap.yaml
```

**Verify**:
```bash
kubectl get configmap -n eventpulse eventpulse-config
kubectl describe configmap eventpulse-config -n eventpulse
```

**View ConfigMap data**:
```bash
kubectl get configmap eventpulse-config -n eventpulse -o yaml | grep -A 50 "data:"
```

### Step 3: Create Secret (Sensitive Credentials)

Secrets contain database passwords and connection strings. These **must be created separately** and never committed to the repository.

#### Option A: kubectl create (Development)

```bash
kubectl create secret generic eventpulse-secrets \
  --from-literal=POSTGRES_PASSWORD='dev-password-change-me' \
  --from-literal=DATABASE_DSN='host=postgres.eventpulse.svc.cluster.local port=5432 user=admin password=dev-password-change-me dbname=eventpulse sslmode=disable' \
  --from-literal=GRAFANA_PASSWORD='dev-password-change-me' \
  -n eventpulse
```

#### Option B: From .env file (CI/CD)

Create `.env.secrets` locally (never commit):
```
POSTGRES_PASSWORD=your-secure-password
DATABASE_DSN=host=postgres.eventpulse.svc.cluster.local port=5432 user=admin password=your-secure-password dbname=eventpulse sslmode=disable
GRAFANA_PASSWORD=your-secure-password
```

Then:
```bash
kubectl create secret generic eventpulse-secrets \
  --from-env-file=.env.secrets \
  -n eventpulse
```

#### Option C: From YAML with base64 (Production - Use Sealed Secrets)

For production, use Sealed Secrets or external secret operator (not covered here).

**Verify**:
```bash
kubectl get secret -n eventpulse eventpulse-secrets
kubectl describe secret eventpulse-secrets -n eventpulse

# Decode secret (for debugging only):
kubectl get secret eventpulse-secrets -n eventpulse -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d
```

### Step 4: Create PostgreSQL Persistent Volume Claim

The PVC allocates 20Gi of storage that persists across pod restarts.

```bash
kubectl apply -f k8s/postgres/postgres-pvc.yaml
```

**Verify**:
```bash
kubectl get pvc -n eventpulse
# Expected output:
# NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
# postgres-pvc    Bound    pvc-xxxxx                                  20Gi       RWO            standard       10s

kubectl describe pvc postgres-pvc -n eventpulse
```

**Check underlying PersistentVolume**:
```bash
kubectl get pv
# The PVC should be bound to a PV
```

### Step 5: Create PostgreSQL Initialization ConfigMap

This ConfigMap contains the SQL schema that runs on first startup.

```bash
kubectl apply -f k8s/postgres/postgres-init-cm.yaml
```

**Verify**:
```bash
kubectl get configmap -n eventpulse postgres-init
kubectl get configmap postgres-init -n eventpulse -o yaml | grep -A 30 "data:"
```

### Step 6: Deploy PostgreSQL

Now deploy the PostgreSQL pod with the Deployment manifest.

```bash
kubectl apply -f k8s/postgres/postgres-deployment.yaml
```

**Verify deployment created**:
```bash
kubectl get deployment -n eventpulse
# Expected output:
# NAME       READY   UP-TO-DATE   AVAILABLE   AGE
# postgres   0/1     1            0           5s

# Watch rollout progress:
kubectl rollout status deployment/postgres -n eventpulse -w
# Wait for: "deployment "postgres" successfully rolled out"
```

**Check pod status**:
```bash
kubectl get pods -n eventpulse
# Wait for STATUS = Running and READY = 1/1
# This may take 20-30 seconds while PostgreSQL initializes

# Detailed pod info:
kubectl describe pod -n eventpulse -l app=postgres
```

### Step 7: Create PostgreSQL Service

The Service provides a stable DNS name and load balances traffic to the pod.

```bash
kubectl apply -f k8s/postgres/postgres-service.yaml
```

**Verify service created**:
```bash
kubectl get service -n eventpulse
# Expected output:
# NAME       TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
# postgres   ClusterIP   10.xxx.xxx.xxx   <none>        5432/TCP   5s

kubectl get service postgres -n eventpulse -o wide
```

---

## Verification: PostgreSQL is Running

After all manifests are applied, run these verification steps:

### 1. Check Pod Status

```bash
kubectl get pods -n eventpulse -l app=postgres
```

**Expected output**:
```
NAME                        READY   STATUS    RESTARTS   AGE
postgres-xxxxx              1/1     Running   0          2m
```

If STATUS is not "Running":
```bash
kubectl describe pod -n eventpulse -l app=postgres
# Look for "Events:" section for error details
```

### 2. Check Pod Logs

```bash
kubectl logs -n eventpulse -l app=postgres
```

**Expected output** (last few lines):
```
...
PostgreSQL Database directory appears to contain a database; Skipping initialization

2026-06-18 20:00:00.000 UTC [1] LOG:  listening on IPv4 address "0.0.0.0", port 5432
2026-06-18 20:00:00.111 UTC [1] LOG:  listening on IPv6 address "::", port 5432
2026-06-18 20:00:00.222 UTC [1] LOG:  database system is ready to accept connections
```

If init script ran:
```
2026-06-18 20:00:00.000 UTC [1] LOG:  redo starts at 0/1000000
...
CREATE TABLE
CREATE INDEX
```

**Stream logs** (follow in real-time):
```bash
kubectl logs -n eventpulse -l app=postgres -f
# Press Ctrl+C to stop
```

### 3. Check Health Probe Status

```bash
kubectl describe pod -n eventpulse -l app=postgres | grep -A 10 "Probes:"
```

**Expected output**:
```
Liveness:       exec [/bin/sh -c pg_isready -U admin -d eventpulse] delay=30s timeout=5s period=10s #success=1 #failure=3
Readiness:      exec [/bin/sh -c pg_isready -U admin -d eventpulse] delay=10s timeout=2s period=5s #success=1 #failure=3
```

If probes are failing, pod will show `CrashLoopBackOff` or not reach `READY 1/1`.

### 4. Connect to PostgreSQL

**From within the cluster** (using a temporary pod):

```bash
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "SELECT version();"
```

**Expected output**:
```
Password for user admin: [enter the POSTGRES_PASSWORD]
                            version
 ─────────────────────────────────────────────────────────────────────────
 PostgreSQL 16.x on ... (Debian x.x)
(1 row)
```

**Check if alerts table was created**:

```bash
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "\dt"
```

**Expected output**:
```
         List of relations
 Schema |  Name  | Type  | Owner
────────────────────────────────
 public | alerts | table | admin
(1 row)
```

**View table schema**:

```bash
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "\d alerts"
```

**Expected output**:
```
                                Table "public.alerts"
   Column   |            Type             | Collation | Nullable |              Default
────────────────────────────────────────────────────────────────────────────────────
 id         | integer                     |           | not null | nextval('alerts_id_seq'::regclass)
 event_id   | text                        |           | not null | ''::text
 user_id    | text                        |           | not null |
 risk_score | integer                     |           | not null |
 message    | text                        |           | not null |
 created_at | timestamp with time zone    |           | not null | now()
```

### 5. Check Persistent Volume

```bash
kubectl get pvc -n eventpulse
kubectl get pv | grep postgres-pvc
```

**Expected**: PVC should be BOUND to a PV.

### 6. Port Forward for Local Testing

Connect to PostgreSQL from your local machine:

```bash
kubectl port-forward -n eventpulse svc/postgres 5432:5432 &
# Background the process with &
```

Then from your local machine:
```bash
psql -h localhost -U admin -d eventpulse
# Password: [enter POSTGRES_PASSWORD]
```

**Stop port forwarding**:
```bash
pkill -f "port-forward.*postgres"
# or find the process ID and kill it
```

### 7. Verify Pod Restarts Survive

**Delete the pod** (Kubernetes will recreate it):

```bash
kubectl delete pod -n eventpulse -l app=postgres
```

**Watch it restart**:
```bash
kubectl get pods -n eventpulse -l app=postgres -w
# You'll see:
# postgres-xxxx   1/1     Running   0          0s          <- Deleted
# postgres-yyyy   0/1     Pending   0          2s          <- Recreating
# postgres-yyyy   0/1     ContainerCreating   0          5s
# postgres-yyyy   1/1     Running   0          30s         <- Ready
```

**Connect to new pod** (data should still exist):

```bash
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "SELECT COUNT(*) FROM alerts;"
```

This proves the PVC persisted the data across pod deletion/recreation.

---

## Troubleshooting

### Pod stuck in CrashLoopBackOff

```bash
kubectl describe pod -n eventpulse -l app=postgres
# Look for "Error" or "LastState" details

kubectl logs -n eventpulse -l app=postgres --previous
# View logs from the crashed container
```

**Common causes**:
- Secret not created (check `POSTGRES_PASSWORD`)
- PVC not bound (check `kubectl get pvc`)
- Insufficient resources (check node capacity)

### Service can't reach PostgreSQL

```bash
# Test from another pod
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  sh -c "echo 'SELECT 1' | nc -w 2 postgres.eventpulse.svc.cluster.local 5432"

# Or try nslookup
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  nslookup postgres.eventpulse.svc.cluster.local
```

### PVC not creating PV

Some clusters require explicit StorageClass. Check:

```bash
kubectl get storageclass
# If none exist, create a default or modify postgres-pvc.yaml to specify one
```

---

## PostgreSQL is Ready

When all verifications pass, PostgreSQL is ready for the next phase: deploying the application services.

**Next**: Kafka deployment (Phase 2)

---

## Summary

| Component | File | Status |
|-----------|------|--------|
| Namespace | `k8s/namespace.yaml` | ✅ Applied |
| ConfigMap | `k8s/configmap.yaml` | ✅ Applied |
| Secret | (created via kubectl) | ✅ Created |
| PVC | `k8s/postgres/postgres-pvc.yaml` | ✅ Applied |
| Init ConfigMap | `k8s/postgres/postgres-init-cm.yaml` | ✅ Applied |
| Deployment | `k8s/postgres/postgres-deployment.yaml` | ✅ Applied |
| Service | `k8s/postgres/postgres-service.yaml` | ✅ Applied |

---

## Quick Reference: All Commands

### Phase 1: PostgreSQL

```bash
# Deploy PostgreSQL
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl create secret generic eventpulse-secrets \
  --from-literal=POSTGRES_PASSWORD='dev-password' \
  --from-literal=DATABASE_DSN='host=postgres.eventpulse.svc.cluster.local port=5432 user=admin password=dev-password dbname=eventpulse sslmode=disable' \
  --from-literal=GRAFANA_PASSWORD='dev-password' \
  -n eventpulse
kubectl apply -f k8s/postgres/postgres-pvc.yaml
kubectl apply -f k8s/postgres/postgres-init-cm.yaml
kubectl apply -f k8s/postgres/postgres-deployment.yaml
kubectl apply -f k8s/postgres/postgres-service.yaml

# Verify PostgreSQL
kubectl rollout status deployment/postgres -n eventpulse -w
kubectl get pods -n eventpulse -l app=postgres
kubectl logs -n eventpulse -l app=postgres -f
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "SELECT COUNT(*) FROM alerts;"

# Port forward for local testing
kubectl port-forward -n eventpulse svc/postgres 5432:5432
```

### Phase 2: Kafka

```bash
# Deploy Kafka
kubectl apply -f k8s/kafka/kafka-pvc.yaml
kubectl apply -f k8s/kafka/kafka-deployment.yaml
kubectl apply -f k8s/kafka/kafka-service.yaml

# Verify Kafka is running
kubectl rollout status deployment/kafka -n eventpulse -w
kubectl get pods -n eventpulse -l app=kafka
kubectl logs -n eventpulse -l app=kafka -f

# Create topics
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create --topic events.raw --partitions 1 --replication-factor 1
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create --topic events.processed --partitions 1 --replication-factor 1
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create --topic alerts --partitions 1 --replication-factor 1
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create --topic events.dlq --partitions 1 --replication-factor 1

# Verify topics
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 --list

# Check broker health
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092

# Port forward for local testing
kubectl port-forward -n eventpulse svc/kafka 9092:9092
```

---

## Phase 2: Kafka Deployment

Kafka is the message broker that orchestrates communication between services. This phase deploys Kafka in KRaft mode (no ZooKeeper) with persistent storage and health checks.

### Prerequisites

- PostgreSQL deployed and running (Phase 1 completed)
- EventPulse namespace and ConfigMap already exist
- Persistent volume provisioner (StorageClass)

### Step 1: Create Kafka Persistent Volume Claim

The PVC allocates 50Gi of storage for Kafka broker data.

```bash
kubectl apply -f k8s/kafka/kafka-pvc.yaml
```

**Verify**:
```bash
kubectl get pvc -n eventpulse kafka-pvc
# Expected output:
# NAME         STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
# kafka-pvc    Bound    pvc-xxxxx                                  50Gi       RWO            standard       10s

kubectl describe pvc kafka-pvc -n eventpulse
```

### Step 2: Deploy Kafka Broker

Deploy the Kafka broker in KRaft mode (single node, no ZooKeeper).

```bash
kubectl apply -f k8s/kafka/kafka-deployment.yaml
```

**Verify deployment created**:
```bash
kubectl get deployment -n eventpulse kafka
# Expected output:
# NAME    READY   UP-TO-DATE   AVAILABLE   AGE
# kafka   0/1     1            0           5s

# Watch rollout progress:
kubectl rollout status deployment/kafka -n eventpulse -w
# Wait for: "deployment "kafka" successfully rolled out"
```

**Check pod status**:
```bash
kubectl get pods -n eventpulse -l app=kafka
# Wait for STATUS = Running and READY = 1/1
# This may take 20-30 seconds while Kafka initializes

# Detailed pod info:
kubectl describe pod -n eventpulse -l app=kafka
```

### Step 3: Create Kafka Service

The Service provides a stable DNS name and load balances traffic to the broker.

```bash
kubectl apply -f k8s/kafka/kafka-service.yaml
```

**Verify service created**:
```bash
kubectl get service -n eventpulse kafka
# Expected output:
# NAME    TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)             AGE
# kafka   ClusterIP   10.xxx.xxx.xxx   <none>        9092/TCP,9093/TCP   5s

kubectl get service kafka -n eventpulse -o wide
```

---

## Verification: Kafka is Running

After all manifests are applied, run these verification steps:

### 1. Check Pod Status

```bash
kubectl get pods -n eventpulse -l app=kafka
```

**Expected output**:
```
NAME                     READY   STATUS    RESTARTS   AGE
kafka-xxxxx              1/1     Running   0          2m
```

If STATUS is not "Running":
```bash
kubectl describe pod -n eventpulse -l app=kafka
# Look for "Events:" section for error details
```

### 2. Check Pod Logs

```bash
kubectl logs -n eventpulse -l app=kafka
```

**Expected output** (key lines):
```
...
[BrokerServer id=1] started
[KafkaServer id=1] started
[SocketServer brokerId=1] Started 2 acceptors on port 9092
[SocketServer brokerId=1] Started 1 acceptor on port 9093
```

**Stream logs** (follow in real-time):
```bash
kubectl logs -n eventpulse -l app=kafka -f
# Press Ctrl+C to stop
```

### 3. Check Health Probe Status

```bash
kubectl describe pod -n eventpulse -l app=kafka | grep -A 10 "Probes:"
```

**Expected output**:
```
Liveness:       exec [/bin/bash -c /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 | grep ApiVersion] delay=30s timeout=5s period=10s #success=1 #failure=3
Readiness:      exec [/bin/bash -c /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list | head -1] delay=20s timeout=5s period=5s #success=1 #failure=3
```

### 4. Create Kafka Topics

Topics must exist before producers/consumers can use them. Auto-creation is enabled for development, but explicit creation is recommended.

**Create topics from within cluster**:

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create \
  --topic events.raw \
  --partitions 1 \
  --replication-factor 1
```

**Expected output**:
```
Created topic events.raw.
```

Repeat for other topics:

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create \
  --topic events.processed \
  --partitions 1 \
  --replication-factor 1
```

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create \
  --topic alerts \
  --partitions 1 \
  --replication-factor 1
```

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --create \
  --topic events.dlq \
  --partitions 1 \
  --replication-factor 1
```

### 5. Verify Topics

**List all topics**:

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
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

**Describe a topic**:

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --describe \
  --topic events.raw
```

**Expected output**:
```
Topic: events.raw	TopicId: xxxxxxxxxxxxxxxxxxxxx	PartitionCount: 1	ReplicationFactor: 1	Configs:
	Topic: events.raw	Partition: 0	Leader: 1	Replicas: 1	Isr: 1
```

### 6. Test Broker Health

**Check broker API versions**:

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-broker-api-versions.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092
```

**Expected output**:
```
ApiVersion: 0 Min: 0 Max: 4
ApiVersion: 1 Min: 0 Max: 11
ApiVersion: 2 Min: 0 Max: 4
...
```

### 7. Verify Pod Restart Survival

**Delete the pod** (Kubernetes will recreate it):

```bash
kubectl delete pod -n eventpulse -l app=kafka
```

**Watch it restart**:
```bash
kubectl get pods -n eventpulse -l app=kafka -w
# You'll see pod recreate and reach Running state
```

**List topics again** (data should still exist):

```bash
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --list
```

This proves the PVC persisted Kafka broker logs across pod deletion/recreation.

---

## Troubleshooting: Kafka

### Pod stuck in CrashLoopBackOff

```bash
kubectl describe pod -n eventpulse -l app=kafka
# Look for "Error" or "LastState" details

kubectl logs -n eventpulse -l app=kafka --previous
# View logs from the crashed container
```

**Common causes**:
- PVC not bound (check `kubectl get pvc`)
- Insufficient disk space (check PVC capacity)
- Insufficient memory (check node capacity)
- Port conflict (rare in Kubernetes)

### Topics not creating

```bash
# Try to list topics (will show failure reason)
kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
  /opt/kafka/bin/kafka-topics.sh \
  --bootstrap-server kafka.eventpulse.svc.cluster.local:9092 \
  --list
```

**Common causes**:
- Broker not ready (wait for readiness probe)
- Network connectivity issue (test from another pod)
- Broker configuration error (check logs)

### Services can't reach Kafka

```bash
# Test from another pod
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  sh -c "echo 'Test' | nc -w 2 kafka.eventpulse.svc.cluster.local 9092"

# Or try DNS lookup
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  nslookup kafka.eventpulse.svc.cluster.local
```

### Broker not responding

```bash
# Check broker process
kubectl exec -it -n eventpulse -c kafka pod/$(kubectl get pod -n eventpulse -l app=kafka -o jsonpath='{.items[0].metadata.name}') -- \
  ps aux | grep java

# Check Kafka logs for errors
kubectl logs -n eventpulse -l app=kafka -f
```

### PVC not binding

```bash
kubectl get pvc -n eventpulse kafka-pvc
kubectl get pv | grep kafka-pvc

# Check StorageClass
kubectl get storageclass
```

---

## Kafka is Ready

When all verifications pass, Kafka is ready for the next phase: deploying the application services.

---

## Phase 3: Application Services Deployment

Deploy the three EventPulse microservices: API Gateway, Analytics Service, and Alert Service.

### Prerequisites

- PostgreSQL deployed and running (Phase 1 completed)
- Kafka deployed with topics created (Phase 2 completed)
- EventPulse namespace and ConfigMap already exist
- Docker images built and available locally:
  - `eventpulse-api-gateway:latest`
  - `eventpulse-analytics-service:latest`
  - `eventpulse-alert-service:latest`

### Step 1: Deploy API Gateway

The API Gateway exposes REST endpoints for event ingestion and alert retrieval.

```bash
kubectl apply -f k8s/api-gateway/api-gateway-deployment.yaml
```

**Verify deployment**:
```bash
kubectl get deployment -n eventpulse api-gateway
kubectl rollout status deployment/api-gateway -n eventpulse -w
```

### Step 2: Deploy Analytics Service

The Analytics Service consumes raw events and computes risk scores.

```bash
kubectl apply -f k8s/analytics-service/analytics-service-deployment.yaml
```

**Verify deployment**:
```bash
kubectl get deployment -n eventpulse analytics-service
kubectl rollout status deployment/analytics-service -n eventpulse -w
```

### Step 3: Deploy Alert Service

The Alert Service consumes risk-scored events and generates fraud alerts.

```bash
kubectl apply -f k8s/alert-service/alert-service-deployment.yaml
```

**Verify deployment**:
```bash
kubectl get deployment -n eventpulse alert-service
kubectl rollout status deployment/alert-service -n eventpulse -w
```

---

## Verification: All Services Running

### 1. Check All Pods

```bash
kubectl get pods -n eventpulse
```

**Expected output**:
```
NAME                                   READY   STATUS    RESTARTS   AGE
postgres-xxxxx                         1/1     Running   0          5m
kafka-xxxxx                            1/1     Running   0          3m
api-gateway-xxxxx                      1/1     Running   0          30s
api-gateway-yyyyy                      1/1     Running   0          30s
analytics-service-xxxxx                1/1     Running   0          30s
analytics-service-yyyyy                1/1     Running   0          30s
alert-service-xxxxx                    1/1     Running   0          30s
alert-service-yyyyy                    1/1     Running   0          30s
```

### 2. Check Service Logs

**API Gateway logs** (watch for startup messages):
```bash
kubectl logs -n eventpulse deployment/api-gateway -f
# Expect: "api gateway started" message
```

**Analytics Service logs**:
```bash
kubectl logs -n eventpulse deployment/analytics-service -f
# Expect: "analytics service started"
```

**Alert Service logs**:
```bash
kubectl logs -n eventpulse deployment/alert-service -f
# Expect: "alert service started"
```

### 3. Check Services

```bash
kubectl get service -n eventpulse
```

**Expected output**:
```
NAME                 TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
postgres             ClusterIP   10.xxx.xxx.xxx   <none>        5432/TCP   5m
kafka                ClusterIP   10.xxx.xxx.xxx   <none>        9092/TCP   3m
api-gateway          ClusterIP   10.xxx.xxx.xxx   <none>        8080/TCP   1m
analytics-service    ClusterIP   10.xxx.xxx.xxx   <none>        8081/TCP   1m
alert-service        ClusterIP   10.xxx.xxx.xxx   <none>        8082/TCP   1m
```

### 4. Health Checks

**API Gateway health**:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  wget -qO- http://api-gateway:8080/health
# Expected: {"status":"ok"}
```

**Analytics Service health**:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  wget -qO- http://analytics-service:8081/health
# Expected: {"status":"ok","...}
```

**Alert Service health**:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  wget -qO- http://alert-service:8082/health
# Expected: {"status":"ok","...}
```

### 5. Test End-to-End Flow

**Send a transaction event**:
```bash
kubectl run -it --rm curl-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl -X POST http://api-gateway:8080/events \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test_user","event_type":"purchase","amount":75000}'
# Expected: {"message":"Event Published"}
```

**Wait for processing** (3-5 seconds for full pipeline):
```bash
sleep 5
```

**Check generated alerts**:
```bash
kubectl run -it --rm curl-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl http://api-gateway:8080/alerts
# Expected: JSON array with alert(s) for high-risk transaction
```

### 6. Check Prometheus Metrics

**API Gateway metrics**:
```bash
kubectl run -it --rm curl-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl http://api-gateway:8080/metrics | grep eventpulse_events_published_total
# Shows event publishing counts
```

**Analytics Service metrics**:
```bash
kubectl run -it --rm curl-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl http://analytics-service:8081/metrics | grep eventpulse_events_processed_total
# Shows event processing counts
```

**Alert Service metrics**:
```bash
kubectl run -it --rm curl-test --image=curlimages/curl --restart=Never -n eventpulse -- \
  curl http://alert-service:8082/metrics | grep eventpulse_alerts_generated_total
# Shows alert generation counts
```

---

## Troubleshooting: Application Services

### Pod not starting (CrashLoopBackOff)

```bash
kubectl describe pod -n eventpulse <pod-name>
kubectl logs -n eventpulse <pod-name> --previous
```

**Common causes**:
- ConfigMap or Secret not created (check: `kubectl get configmap/secret -n eventpulse`)
- Database not accessible (check PostgreSQL pod)
- Kafka not accessible (check Kafka pod and topics)
- Image not available locally (build with `docker build`)

### Pod running but not ready

```bash
kubectl describe pod -n eventpulse -l app=<service-name>
# Check: Probes section for startup/readiness failures
```

**Common causes**:
- Health endpoint not responding (application startup slow)
- Kafka/PostgreSQL not reachable yet (wait 5-10 seconds)
- Port mismatch in ConfigMap (check port values)

### Services can't communicate

```bash
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- \
  nslookup api-gateway.eventpulse.svc.cluster.local
```

**Common causes**:
- DNS resolution issue (rare in K8s)
- Service not created (check: `kubectl get service -n eventpulse`)
- Network policies blocking traffic (if configured)

### Metrics not scraping

Check Prometheus targets:
```bash
# Port-forward Prometheus
kubectl port-forward -n eventpulse svc/prometheus 9090:9090
# Open http://localhost:9090/targets
```

**Common causes**:
- Prometheus scrape configuration missing (check k8s/monitoring/prometheus.yaml)
- Endpoint annotations not on pods (should have `prometheus.io/scrape: "true"`)

---

## Full Stack Deployment

All services ready:

```bash
# Phase 1: PostgreSQL
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl create secret generic eventpulse-secrets --from-literal=... -n eventpulse
kubectl apply -f k8s/postgres/*.yaml

# Phase 2: Kafka
kubectl apply -f k8s/kafka/*.yaml

# Create topics
for topic in events.raw events.processed alerts events.dlq; do
  kubectl run -it --rm kafka-client --image=apache/kafka:latest --restart=Never -n eventpulse -- \
    /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 \
    --create --topic $topic --partitions 1 --replication-factor 1
done

# Phase 3: Application Services
kubectl apply -f k8s/api-gateway/api-gateway-deployment.yaml
kubectl apply -f k8s/analytics-service/analytics-service-deployment.yaml
kubectl apply -f k8s/alert-service/alert-service-deployment.yaml

# Verify
kubectl get pods -n eventpulse
kubectl get service -n eventpulse
```

---

## Phase 4: NGINX Ingress Deployment

NGINX Ingress Controller provides a single HTTP entrypoint for all EventPulse API endpoints. Instead of accessing api-gateway directly (port 8080), clients connect to the Ingress (port 80/443).

### Prerequisites

- All services deployed: PostgreSQL, Kafka, Application Services (Phases 1-3)
- Kubernetes cluster with Ingress support (most distributions include this)
- kubectl configured

### Step 1: Deploy NGINX Ingress Controller

The NGINX Ingress Controller runs in the `ingress-nginx` namespace and manages all Ingress resources.

```bash
kubectl apply -f k8s/ingress/nginx-ingress-deployment.yaml
```

**What this creates**:
- Namespace: ingress-nginx
- ServiceAccount, ClusterRole, ClusterRoleBinding (RBAC)
- Deployment: nginx-ingress-controller (2 replicas)
- Service: nginx-ingress (LoadBalancer)
- ConfigMap: nginx-config (NGINX settings)
- IngressClass: nginx

**Verify deployment**:
```bash
kubectl get deployment -n ingress-nginx
kubectl rollout status deployment/nginx-ingress-controller -n ingress-nginx -w

# Expected output:
# deployment "nginx-ingress-controller" successfully rolled out
```

**Check pods**:
```bash
kubectl get pods -n ingress-nginx
# Expected: 2 pods, both READY 1/1, STATUS Running
```

**Check service**:
```bash
kubectl get service -n ingress-nginx nginx-ingress
# Expected: LoadBalancer service on ports 80 (HTTP) and 443 (HTTPS)
```

### Step 2: Deploy EventPulse Ingress

The Ingress resource routes external traffic to the API Gateway service.

```bash
kubectl apply -f k8s/ingress/eventpulse-ingress.yaml
```

**What this creates**:
- Ingress resource: eventpulse-ingress
- Routes: /events, /alerts, /alert, /health, /metrics → api-gateway:8080
- CORS enabled (cross-origin requests allowed)
- Rate limiting (100 req/sec per IP)
- Path-based routing (all paths go to same backend)

**Verify Ingress**:
```bash
kubectl get ingress -n eventpulse
# Expected output:
# NAME                   CLASS   HOSTS   ADDRESS       PORTS   AGE
# eventpulse-ingress     nginx   *       10.244.x.x    80      10s

kubectl describe ingress eventpulse-ingress -n eventpulse
# Should show all 5 paths pointing to api-gateway:8080
```

---

## Verification: Ingress Routing

### Step 1: Port-Forward for Local Testing

For local/minikube environments, use port-forward to access the Ingress:

```bash
kubectl port-forward -n ingress-nginx svc/nginx-ingress 8000:80
```

**Note**: Runs in foreground. Open another terminal for tests.

### Step 2: Test Health Endpoint

```bash
curl -v http://localhost:8000/health
```

**Expected output**:
```
HTTP/1.1 200 OK
Content-Type: application/json
...
{"status":"ok","service":"api-gateway","timestamp":"..."}
```

### Step 3: Test Event Publication

```bash
curl -X POST http://localhost:8000/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "ingress_test",
    "event_type": "purchase",
    "amount": 75000
  }'
```

**Expected output**:
```json
{"message":"Event Published"}
```

### Step 4: Test Alert Retrieval

```bash
curl http://localhost:8000/alerts | jq .
```

**Expected output**:
```json
[
  {
    "id": 1,
    "user_id": "ingress_test",
    "risk_score": 90,
    "message": "HIGH RISK TRANSACTION DETECTED",
    "created_at": "..."
  }
]
```

### Step 5: Test CORS

```bash
curl -i -X OPTIONS http://localhost:8000/events \
  -H "Origin: http://example.com" \
  -H "Access-Control-Request-Method: POST"
```

**Expected output** (200 OK with CORS headers):
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
```

### Step 6: Test Rate Limiting

```bash
# Send 150 rapid requests (limit is 100/sec)
for i in {1..150}; do curl -s http://localhost:8000/health & done
wait
```

**Result**: Some requests may get 429 (Too Many Requests) if rate limit exceeded.

---

## Troubleshooting: Ingress

### Pod not starting

```bash
kubectl logs -n ingress-nginx deployment/nginx-ingress-controller -f
```

**Common issues**:
- RBAC permissions (check ServiceAccount bindings)
- Image not available (check container runtime)
- Port conflict (port 80/443 already in use)

### Ingress shows no ADDRESS

```bash
kubectl describe ingress eventpulse-ingress -n eventpulse
```

**Causes**:
- NGINX controller not ready (check controller logs)
- IngressClass not found (verify `kubectl get ingressclass`)
- Service backend missing (check `kubectl get service -n eventpulse api-gateway`)

### Request timeout (504 Gateway Timeout)

```bash
curl -v http://localhost:8000/events
```

**Causes**:
- API Gateway pod not running (check `kubectl get pods -n eventpulse`)
- Service endpoint missing (check `kubectl get endpoints -n eventpulse api-gateway`)
- Network unreachable (check NGINX logs)

### 404 Not Found

```bash
curl -v http://localhost:8000/invalid-path
```

**Solution**:
- Verify path in Ingress matches actual API endpoint
- Check `kubectl describe ingress eventpulse-ingress -n eventpulse` for paths

### CORS Errors

**Solution**: Verify CORS annotations are applied
```bash
kubectl get ingress eventpulse-ingress -n eventpulse -o yaml | grep cors
```

Should show enabled CORS with `*` origin

---

## Cloud Deployment (AWS/GCP/Azure)

### Get External Load Balancer IP

```bash
kubectl get service -n ingress-nginx nginx-ingress
```

After 2-5 minutes, `EXTERNAL-IP` will show the cloud-provided IP:

```
NAME            TYPE           CLUSTER-IP       EXTERNAL-IP      PORT(S)
nginx-ingress   LoadBalancer   10.xxx.xxx.xxx   203.0.113.50     80:30xxx/TCP
```

**Access API**:
```bash
curl http://203.0.113.50/events
curl http://203.0.113.50/alerts
```

### Enable HTTPS/TLS

1. Create certificate Secret:
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
kubectl create secret tls eventpulse-tls --cert=cert.pem --key=key.pem -n eventpulse
```

2. Update eventpulse-ingress.yaml to enable TLS:
```yaml
spec:
  tls:
  - hosts:
    - api.example.com
    secretName: eventpulse-tls
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /events
        backend:
          service:
            name: api-gateway
            port:
              number: 8080
```

3. Reapply:
```bash
kubectl apply -f k8s/ingress/eventpulse-ingress.yaml
```

---

## Summary

✅ NGINX Ingress Controller installed and running  
✅ EventPulse Ingress routes all paths to API Gateway  
✅ Single entrypoint on port 80 (or 443 with TLS)  
✅ CORS and rate limiting configured  
✅ Ready for production traffic  

See **INGRESS_GUIDE.md** for detailed troubleshooting and advanced configuration.

---

## Phase 5: Horizontal Pod Autoscaling (HPA)

Horizontal Pod Autoscaling automatically adjusts the number of pod replicas based on CPU utilization. Services scale from 2-10 replicas to handle variable workloads.

### Prerequisites

1. **Metrics Server** must be running:
```bash
kubectl get deployment -n kube-system metrics-server
```

If not found, install:
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

2. **CPU Requests** defined (already in deployments):
```yaml
resources:
  requests:
    cpu: 100m      # HPA calculates 70% of this = 70m threshold
```

### Step 1: Deploy API Gateway HPA

```bash
kubectl apply -f k8s/autoscaling/api-gateway-hpa.yaml
```

**Verify**:
```bash
kubectl get hpa -n eventpulse api-gateway-hpa
# Expected output:
# NAME                REFERENCE                 TARGETS      MINPODS  MAXPODS  REPLICAS  AGE
# api-gateway-hpa     Deployment/api-gateway    45%/70%      2        10       2         10s
```

### Step 2: Deploy Analytics Service HPA

```bash
kubectl apply -f k8s/autoscaling/analytics-service-hpa.yaml
```

**Verify**:
```bash
kubectl get hpa -n eventpulse analytics-service-hpa
```

### Step 3: Deploy Alert Service HPA

```bash
kubectl apply -f k8s/autoscaling/alert-service-hpa.yaml
```

**Verify**:
```bash
kubectl get hpa -n eventpulse alert-service-hpa
```

### Step 4: View All HPAs

```bash
kubectl get hpa -n eventpulse
```

**Expected output**:
```
NAME                      REFERENCE                        TARGETS        MINPODS  MAXPODS  REPLICAS  AGE
api-gateway-hpa           Deployment/api-gateway           45%/70%        2        10       2         1m
analytics-service-hpa     Deployment/analytics-service     32%/70%        2        10       2         1m
alert-service-hpa         Deployment/alert-service         28%/70%        2        10       2         1m
```

---

## Verification: HPA Scaling

### Monitor Current Status

```bash
kubectl get hpa -n eventpulse -w
```

This watches HPA status in real-time. Metrics appear after 30-60 seconds.

### Check Pod Counts

```bash
kubectl get deployment -n eventpulse
```

**Current state** (2 replicas each):
```
NAME               READY  UP-TO-DATE  AVAILABLE
api-gateway        2/2    2           2
analytics-service  2/2    2           2
alert-service      2/2    2           2
```

### View CPU Usage

```bash
kubectl top pods -n eventpulse -l tier=services
```

**Expected output**:
```
NAME                           CPU(cores)   MEMORY(Mi)
api-gateway-xxxxx              45m          80Mi
api-gateway-yyyyy              38m          75Mi
analytics-service-aaaaa        62m          95Mi
analytics-service-bbbbb        41m          88Mi
alert-service-ccccc            25m          102Mi
alert-service-ddddd            19m          99Mi
```

---

## Load Testing: Force Scaling

### Test 1: Generate Load with Apache Bench

**Terminal 1**: Port-forward NGINX Ingress
```bash
kubectl port-forward -n ingress-nginx svc/nginx-ingress 8000:80
```

**Terminal 2**: Generate load (100 concurrent requests, 10,000 total)
```bash
ab -n 10000 -c 100 http://localhost:8000/health
```

**Terminal 3**: Watch HPA scaling
```bash
kubectl get hpa -n eventpulse -w
```

**Expected behavior** (watch Terminal 3):
```
api-gateway-hpa  Deployment/api-gateway  45%/70%  2  10  2  1m
api-gateway-hpa  Deployment/api-gateway  95%/70%  2  10  2  2m   # CPU spiked!
api-gateway-hpa  Deployment/api-gateway  95%/70%  2  10  3  2m   # Scaled to 3 pods
api-gateway-hpa  Deployment/api-gateway  72%/70%  2  10  4  3m   # Scaled to 4 pods
api-gateway-hpa  Deployment/api-gateway  68%/70%  2  10  4  4m   # Stabilized at 4
```

### Test 2: Send Events (Tests All Services)

```bash
# Terminal 1: Watch HPA
kubectl get hpa -n eventpulse -w

# Terminal 2: Send 1000 events rapidly
for i in {1..1000}; do
  curl -s -X POST http://localhost:8000/events \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"user_$i\",\"event_type\":\"purchase\",\"amount\":$((50000 + RANDOM % 50000))}" &
done
wait
```

**Expected**: All services scale up as events flow through pipeline

### Test 3: Monitor Scaling Events

```bash
kubectl get events -n eventpulse --sort-by='.lastTimestamp' | grep -i hpa
```

**Example output**:
```
eventpulse  Normal  SuccessfulRescale  2m  horizontal-pod-autoscaler  New size: 3; reason: cpu resource utilization (85%) above target (70%)
eventpulse  Normal  SuccessfulRescale  1m  horizontal-pod-autoscaler  New size: 4; reason: cpu resource utilization (72%) above target (70%)
```

---

## Scaling Verification

### Upscaling (2 → More Replicas)

1. Generate load (use Test 1 or 2 above)
2. Confirm CPU increases
3. Verify replica count increases

```bash
# Check current replicas
kubectl get deployment api-gateway -n eventpulse
# Should show READY > 2
```

### Downscaling (Many → 2 Replicas)

1. Stop load generation (Ctrl+C in load test terminal)
2. Wait 5-10 minutes for metrics to stabilize
3. Confirm replicas gradually decrease

```bash
# Watch downscaling (after load stops)
watch kubectl get deployment -n eventpulse

# Timeline:
# t=0min:   Stop load, still 5 pods (high CPU)
# t=5min:   CPU drops below 70%, scaling begins
# t=10min:  Replicas gradually reduce to 2 pods
# t=15min:  Back to minimum (2 replicas)
```

### HPA Description

```bash
kubectl describe hpa api-gateway-hpa -n eventpulse
```

**Look for "Conditions"**:
```
Conditions:
  Type            Status  Reason            Message
  AbleToScale     True    SucceededGetResourceMetric  successfully obtained the metric
  ScalingActive   True    ValidMetricsFound Available metrics found
  ScalingLimited  False   DesiredWithinRange desired replica count within acceptable range
```

---

## HPA Configuration Details

### Current Settings (All Services)

```yaml
minReplicas: 2          # Never below 2 pods
maxReplicas: 10         # Never above 10 pods
targetCPU: 70%          # Scale when avg CPU > 70m (70% of 100m request)
scaleUp: 1 pod/30s      # Add 1 pod every 30 seconds
scaleDown: 1 pod/60s    # Remove 1 pod every 60 seconds (after 5 min stable)
```

### Tuning Scaling Behavior

To make scaling **faster** (aggressive):
```yaml
scaleUp:
  periodSeconds: 15      # Check more frequently
  addPodsPerPeriod: 2    # Add 2 pods at once
```

To make scaling **slower** (conservative):
```yaml
scaleUp:
  periodSeconds: 60      # Check less frequently
  addPodsPerPeriod: 1    # Add 1 pod at a time
scaleDown:
  stabilization: 600s    # Wait 10 minutes before scaling
```

---

## Troubleshooting HPA

### Metrics show `<unknown>`

```bash
kubectl describe hpa api-gateway-hpa -n eventpulse
# Should see: "FailedGetResourceMetric resource metrics not yet available"
```

**Solution**: Wait 1-2 minutes for Metrics Server to collect data

### Pods don't scale even under high CPU

**Verify Metrics Server**:
```bash
kubectl get deployment -n kube-system metrics-server
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/eventpulse/pods | jq .
```

**Verify HPA target**:
```bash
kubectl get hpa api-gateway-hpa -n eventpulse -o yaml | grep scaleTargetRef
# Should match: Deployment/api-gateway
```

### Excessive scaling (pods constantly added/removed)

**Cause**: Target CPU too close to current usage

**Solution**: Increase target or stabilization window
```yaml
averageUtilization: 80  # Instead of 70
scaleDown:
  stabilizationWindowSeconds: 600  # 10 minutes instead of 5
```

---

## Summary

✅ HPAs deployed for all 3 services (2-10 replicas)  
✅ Scaling based on CPU utilization (70% threshold)  
✅ Load testing procedures documented  
✅ Scaling verification step-by-step  
✅ Troubleshooting guide included  

See **AUTOSCALING_GUIDE.md** for comprehensive load testing and advanced tuning.

**Next**: Phase 6 — Prometheus & Grafana (monitoring and dashboards)

---

## Production Considerations

### PostgreSQL

- **StorageClass**: Specify a high-performance StorageClass for production (SSD, replicated)
- **PVC Size**: Adjust 20Gi based on expected alert volume
- **Replicas**: Use managed PostgreSQL service (RDS, Cloud SQL) for HA instead of single pod
- **Backups**: Configure automated backups of the PVC/PV
- **Secrets**: Use Sealed Secrets, Vault, or cloud-native secret management
- **Resource Limits**: Tune CPU/memory requests based on query patterns
- **Monitoring**: Add Prometheus exporter for PostgreSQL metrics

### Kafka

- **StorageClass**: Specify high-performance SSD StorageClass (Kafka is I/O intensive)
- **PVC Size**: Adjust 50Gi based on:
  - Event throughput (events/sec)
  - Number of topics and partitions
  - Retention policy (time-based or size-based)
- **Replicas**: Deploy 3+ Kafka brokers in a StatefulSet (not a Deployment)
  - Each broker needs a separate PVC
  - Use Kafka broker scale-out (multiple replicas)
  - Set `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=3`
  - Set `KAFKA_DEFAULT_REPLICATION_FACTOR=3`
- **Auto Topic Creation**: Disable in production (`KAFKA_AUTO_CREATE_TOPICS_ENABLE=false`)
  - Create topics manually with proper partition count
  - Set retention policies explicitly
- **Monitoring**: Add JMX exporter for Kafka metrics (CPU, network, disk, lag)
- **Backups**: Implement disaster recovery:
  - Regular PVC snapshots
  - Cross-region replication
  - Partition offset backup
- **Security**:
  - Enable SASL authentication
  - Use TLS for encryption
  - Implement network policies
- **Optimization**:
  - Tune broker settings: `num.network.threads`, `num.io.threads`
  - Adjust segment size and retention based on workload
  - Use compression (snappy or lz4) for log segments
