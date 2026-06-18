# EventPulse Kubernetes Deployment Guide

This guide walks through deploying EventPulse to Kubernetes step by step, starting with PostgreSQL.

**Status**: Phase 2 — Kafka Deployment  
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

**Next**: Phase 3 — Application Services (API Gateway, Analytics Service, Alert Service)

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
