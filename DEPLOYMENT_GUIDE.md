# EventPulse Kubernetes Deployment Guide

This guide walks through deploying EventPulse to Kubernetes step by step, starting with PostgreSQL.

**Status**: Phase 1 — PostgreSQL Deployment  
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

```bash
# Deploy PostgreSQL (step by step)
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

# Verify
kubectl rollout status deployment/postgres -n eventpulse -w
kubectl get pods -n eventpulse
kubectl logs -n eventpulse -l app=postgres -f

# Test connectivity
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "SELECT version();"

# Port forward for local testing
kubectl port-forward -n eventpulse svc/postgres 5432:5432
```

---

## Production Considerations

- **StorageClass**: Specify a high-performance StorageClass for production (SSD, replicated)
- **PVC Size**: Adjust 20Gi based on expected alert volume
- **Replicas**: Use managed PostgreSQL service (RDS, Cloud SQL) for HA instead of single pod
- **Backups**: Configure automated backups of the PVC/PV
- **Secrets**: Use Sealed Secrets, Vault, or cloud-native secret management
- **Resource Limits**: Tune CPU/memory requests based on query patterns
- **Monitoring**: Add Prometheus exporter for PostgreSQL metrics
