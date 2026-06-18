# EventPulse Kubernetes Upgrade Status

**Branch**: `kubernetes-upgrade`  
**Base**: v1.0.0 (Docker Compose stable)  
**Phase**: Phase 1 — PostgreSQL Deployment

---

## ✅ Completed

### Infrastructure Foundation

- [x] **Kubernetes manifests directory structure**
  - `k8s/namespace.yaml` — EventPulse namespace isolation
  - `k8s/configmap.yaml` — Shared, non-sensitive configuration
  - `k8s/secrets.yaml` — Sensitive credentials template (placeholders only)
  - Per-service directories: `api-gateway/`, `analytics-service/`, `alert-service/`
  - Supporting directories: `postgres/`, `kafka/`, `monitoring/`, `ingress/`

### Configuration Management

- [x] **ConfigMap** (`k8s/configmap.yaml`)
  - Kafka broker addresses (using K8s DNS names)
  - All Kafka topic names and consumer groups
  - Service names, ports, and health ports
  - PostgreSQL connection metadata (host, port, db, user)
  - Log level configuration
  - Injected as environment variables in all services

- [x] **Secret template** (`k8s/secrets.yaml`)
  - PostgreSQL password placeholder
  - Full DATABASE_DSN placeholder (for Go code)
  - Grafana password placeholder
  - Clear documentation on how to create real secrets (kubectl, .env file, external operators)
  - **No real secrets committed to repository**

### Kubernetes Manifests

- [x] **PostgreSQL** (Modularized & Production-Ready)
  - `k8s/postgres/postgres-pvc.yaml` — 20Gi PersistentVolumeClaim with annotations
  - `k8s/postgres/postgres-init-cm.yaml` — SQL initialization ConfigMap
  - `k8s/postgres/postgres-deployment.yaml` — Single-replica Deployment (PostgreSQL 16)
    - Mounts PVC at `/var/lib/postgresql/data` (data survives pod restarts)
    - Mounts init ConfigMap at `/docker-entrypoint-initdb.d`
    - Credentials injected from Secret (POSTGRES_PASSWORD)
    - Environment from ConfigMap (POSTGRES_DB, POSTGRES_USER)
  - `k8s/postgres/postgres-service.yaml` — ClusterIP Service (postgres.eventpulse.svc.cluster.local:5432)
  - Liveness probe: `pg_isready` every 10s (restart after 3 failures)
  - Readiness probe: `pg_isready` every 5s (remove from service after 3 failures)
  - Resource requests: 250m CPU, 256Mi memory; Limits: 1000m CPU, 1Gi memory
  - Strategy: Recreate (required for RWO PVC)

- [x] **Kafka**
  - Single-node Deployment (Apache Kafka, KRaft mode)
  - PersistentVolumeClaim (50Gi)
  - Service (ClusterIP, internal DNS)
  - Auto topic creation enabled
  - Liveness probe (broker API check)
  - Readiness probe (topic list)
  - Resource requests/limits: 500m CPU, 512Mi memory (min); 2000m CPU, 2Gi (max)

- [x] **API Gateway**
  - Deployment with 2 replicas
  - ConfigMap + Secret environment variables
  - Service (ClusterIP)
  - Port 8080 for HTTP API and Prometheus metrics
  - Liveness & readiness probes (HTTP GET /health)
  - Resource requests/limits: 100m CPU, 128Mi memory (min); 500m CPU, 512Mi (max)

- [x] **Analytics Service**
  - Deployment with 2 replicas (Kafka consumer group scalability)
  - Ports: 8080 (app), 8081 (health + metrics)
  - Service (ClusterIP)
  - ConfigMap + Secret environment variables
  - Liveness & readiness probes (HTTP GET /health:8081)
  - Resource requests/limits: 100m CPU, 128Mi memory (min); 500m CPU, 512Mi (max)

- [x] **Alert Service**
  - Deployment with 2 replicas (Kafka consumer group scalability)
  - Ports: 8080 (app), 8082 (health + metrics)
  - Service (ClusterIP)
  - ConfigMap + Secret environment variables
  - Liveness & readiness probes (HTTP GET /health:8082)
  - Resource requests/limits: 100m CPU, 128Mi memory (min); 500m CPU, 512Mi (max)

- [x] **Prometheus**
  - Deployment for metrics collection
  - ConfigMap with scrape configuration (Kubernetes service discovery)
  - PersistentVolumeClaim (10Gi, 7-day retention)
  - Service (ClusterIP)
  - RBAC: ServiceAccount + ClusterRole + ClusterRoleBinding
  - Auto-discovers pods in eventpulse namespace via label selectors

### Deployment & Documentation

- [x] **DEPLOYMENT_GUIDE.md** — Phase-by-phase PostgreSQL deployment guide
  - Step-by-step kubectl commands with explanations
  - Namespace creation
  - ConfigMap application
  - Secret creation (3 options: kubectl, .env file, Sealed Secrets)
  - PVC verification
  - Deployment rollout monitoring
  - Service verification
  - **Comprehensive verification steps**:
    - Pod status checks
    - Log inspection (expect PostgreSQL ready messages)
    - Health probe verification
    - Database connectivity tests (from temporary pod)
    - Alert table schema verification
    - PVC binding verification
    - Port forwarding for local testing
    - **Pod restart survival test**: Proves data persists after pod deletion
  - Troubleshooting guide
  - Production considerations
  - Quick reference commands

- [x] **k8s/README.md** — Comprehensive guide covering:
  - Directory structure and file descriptions
  - Configuration management (ConfigMap, Secrets, external options)
  - Deployment instructions (prerequisites, quick start, port forwarding)
  - Network architecture (DNS, service discovery, storage)
  - Health & readiness probes
  - Scaling (manual and HPA placeholders)
  - Persistence and cleanup
  - Troubleshooting guide
  - References

- [x] **KUBERNETES_STATUS.md** — This file
  - Phase-by-phase status
  - What's done, what's next
  - Decision rationale

---

## ❌ Not Implemented (Out of Scope)

Per requirements, the following are **not** implemented in this phase:

### Networking & Ingress

- ❌ Kubernetes Ingress resource (no `/ingress/*.yaml` content)
- ❌ Ingress controller (Nginx, Traefik, etc.)
- ❌ External DNS configuration
- ❌ TLS/SSL certificates
- ❌ cert-manager integration

### Advanced Features

- ❌ Horizontal Pod Autoscaling (HPA)
- ❌ Helm charts (manifests only)
- ❌ Service Mesh (Istio, Linkerd)
- ❌ GitOps (ArgoCD, Flux)
- ❌ External Secrets Operator

### Monitoring Enhancements

- ❌ Grafana Kubernetes integration
- ❌ Additional monitoring (no changes to prometheus/grafana behavior)
- ❌ Alerting rules (Prometheus)
- ❌ Dashboards (reserved for future)

### Security & RBAC

- ❌ Network Policies
- ❌ Pod Security Policies / Pod Security Standards
- ❌ Custom RBAC roles (basic Prometheus discovery only)
- ❌ Workload Identity / IRSA

### Storage & Databases

- ❌ StatefulSets (Kafka remains Deployment for foundation)
- ❌ Multi-node Kafka cluster
- ❌ PostgreSQL replicas or HA setup
- ❌ External storage (cloud-specific provisioners)

---

## Key Decisions

### 1. ConfigMap + Secret Split
**Decision**: Non-sensitive config → ConfigMap; credentials → Secret  
**Rationale**: Kubernetes best practice. Enables CI/CD to inject only the Secret, while ConfigMap can be version-controlled.

### 2. Service DNS over Hardcoded IPs
**Decision**: All inter-service communication uses Kubernetes DNS (e.g., `kafka.eventpulse.svc.cluster.local`)  
**Rationale**: Portable across clusters. Works in dev, staging, production without config changes.

### 3. Deployment (not StatefulSet) for Kafka
**Decision**: Kafka runs as a Deployment, not StatefulSet  
**Rationale**: Foundation-only. StatefulSet needed for multi-node cluster with persistent identities. Can upgrade later.

### 4. Single Replicas for PostgreSQL & Kafka
**Decision**: Both run with 1 replica  
**Rationale**: Foundation simplicity. In production, use PostgreSQL managed services (RDS, Cloud SQL) and Kafka operators.

### 5. Multiple Replicas for Services
**Decision**: api-gateway, analytics-service, alert-service each have 2 replicas  
**Rationale**: Demonstrates load distribution and readiness for HPA. Kafka consumer groups handle distributed processing.

### 6. Prometheus ServiceAccount with ClusterRole
**Decision**: RBAC for pod discovery without cluster-admin  
**Rationale**: Least privilege. Prometheus only needs to list/watch pods in eventpulse namespace.

### 7. No Helm
**Decision**: Pure Kubernetes YAML manifests  
**Rationale**: Per requirements. Helm can be added in a future phase without rewriting these files.

---

## Configuration Mapping

From **Docker Compose** → to **Kubernetes**:

| Docker Compose | Kubernetes ConfigMap | Kubernetes Secret | Notes |
|---|---|---|---|
| `KAFKA_BROKERS=kafka:9092` | `kafka.eventpulse.svc.cluster.local:9092` | — | DNS-based discovery |
| `KAFKA_TOPIC_RAW=events.raw` | ✓ | — | Topic names non-sensitive |
| `DATABASE_DSN=host=postgres ...` | — | ✓ (full DSN) | Password in secret |
| `POSTGRES_PASSWORD` | — | ✓ | Sensitive, not in code |
| `LOG_LEVEL=INFO` | ✓ | — | Configuration |
| `PORT=8080`, `HEALTH_PORT=8081` | ✓ | — | Service definitions |

---

## Docker Compose Compatibility

**Status**: ✅ **Unchanged**

- No modifications to `docker-compose.yml`
- Docker Compose stack continues to work as-is
- Both stacks can coexist (e.g., Docker Compose for dev, K8s for production)
- Manifest files reference `eventpulse-*:latest` images (same as Docker builds)

---

## Deployment Readiness

### Before deploying to Kubernetes:

1. **Build Docker images**:
   ```bash
   docker build -f services/api-gateway/Dockerfile -t eventpulse-api-gateway:latest .
   docker build -f services/analytics-service/Dockerfile -t eventpulse-analytics-service:latest .
   docker build -f services/alert-service/Dockerfile -t eventpulse-alert-service:latest .
   ```

2. **Push to registry** (for production):
   ```bash
   docker tag eventpulse-api-gateway:latest myregistry.azurecr.io/eventpulse-api-gateway:v1.0.0
   docker push myregistry.azurecr.io/eventpulse-api-gateway:v1.0.0
   ```
   (Then update `imagePullPolicy` and image names in manifests)

3. **Create secrets**:
   ```bash
   kubectl create secret generic eventpulse-secrets \
     --from-literal=POSTGRES_PASSWORD='...' \
     --from-literal=DATABASE_DSN='...' \
     --from-literal=GRAFANA_PASSWORD='...' \
     -n eventpulse
   ```

4. **Apply manifests** (see k8s/README.md for full order)

5. **Verify readiness**:
   ```bash
   kubectl get pods -n eventpulse
   kubectl logs deployment/api-gateway -n eventpulse
   ```

---

## Next Phases

### Phase 2: Networking (Future)
- Add Ingress for external traffic
- TLS/cert-manager integration
- External DNS configuration

### Phase 3: Advanced Features (Future)
- HorizontalPodAutoscaler
- Sealed Secrets or external secret operator
- Network Policies

### Phase 4: Observability (Future)
- Grafana Kubernetes deployment
- Prometheus alerting rules
- Distributed tracing (Jaeger)

### Phase 5: Production Hardening (Future)
- Pod Disruption Budgets
- Pod Security Standards
- Resource quotas and limits per namespace
- Backup/restore strategy for PVCs

---

## Testing

**Not yet performed**:
- ❌ kubectl apply against test cluster
- ❌ Pod startup verification
- ❌ Inter-service communication tests
- ❌ Kafka topic creation
- ❌ API endpoint testing via port-forward
- ❌ Prometheus scrape verification

**Recommended next step**: Apply to a local Kubernetes cluster (minikube, kind) and verify all deployments reach "Ready" state.

---

## Files Summary

### Root Level
- `KUBERNETES_STATUS.md` — Project status tracking
- `DEPLOYMENT_GUIDE.md` — **NEW** Phase-by-phase deployment instructions (650+ lines)

### k8s/ Directory
- `namespace.yaml` — 12 lines
- `configmap.yaml` — 32 lines
- `secrets.yaml` — 30 lines (template with documentation)
- `postgres/` directory:
  - `postgres-pvc.yaml` — **NEW** 25 lines (PVC with annotations)
  - `postgres-init-cm.yaml` — **NEW** 45 lines (init SQL schema)
  - `postgres-deployment.yaml` — **NEW** 185 lines (deployment with probes)
  - `postgres-service.yaml` — **NEW** 35 lines (ClusterIP service)
- `kafka/kafka.yaml` — 110 lines (unchanged, but split coming in Phase 2)
- `api-gateway/api-gateway.yaml` — 100 lines (unchanged, deployed in Phase 3)
- `analytics-service/analytics-service.yaml` — 125 lines (unchanged, deployed in Phase 3)
- `alert-service/alert-service.yaml` — 125 lines (unchanged, deployed in Phase 3)
- `monitoring/prometheus.yaml` — 200+ lines (unchanged, deployed in Phase 4)
- `README.md` — 600+ lines (comprehensive guide)

**Total**: ~2,000 lines of manifests + documentation (including DEPLOYMENT_GUIDE.md)

---

## Checklist for Phase 1: PostgreSQL Deployment

### Foundation (Phase 0) — Completed ✅
- [x] Directory structure created
- [x] Namespace defined
- [x] ConfigMap with all non-sensitive configuration
- [x] Secret template with guidance
- [x] Comprehensive k8s/README.md

### Phase 1 — PostgreSQL Deployment — Completed ✅
- [x] PostgreSQL modularized into separate files
  - [x] `postgres-pvc.yaml` — PVC with annotations
  - [x] `postgres-init-cm.yaml` — SQL schema initialization
  - [x] `postgres-deployment.yaml` — Deployment with probes, health checks
  - [x] `postgres-service.yaml` — ClusterIP service
- [x] PVC: 20Gi persistent storage, ReadWriteOnce, survives pod restarts
- [x] ConfigMap: init.sql runs on first startup (creates alerts table + indexes)
- [x] Secret integration: POSTGRES_PASSWORD from Secret, DATABASE_DSN injected
- [x] Liveness probe: `pg_isready` every 10s (restart on failure)
- [x] Readiness probe: `pg_isready` every 5s (deregister from service on failure)
- [x] Pod restart survival: Data persists via PVC
- [x] **DEPLOYMENT_GUIDE.md** (650+ lines)
  - Step-by-step kubectl apply commands with explanations
  - 7 deployment steps with verification after each
  - Comprehensive verification commands
  - **Pod restart survival test procedure**
  - Troubleshooting guide
  - Production considerations
  - Quick reference command list
- [x] Docker Compose remains untouched

### Next Phases — TODO
- [ ] Phase 2: Kafka deployment (split into files, deployment guide)
- [ ] Phase 3: Application services (api-gateway, analytics, alert)
- [ ] Phase 4: Monitoring (Prometheus, Grafana)
- [ ] Phase 5: Ingress & networking
- [ ] Phase 6: HPA, autoscaling
- [ ] Phase 7: Production hardening

---

## Status: Phase 1 Complete — PostgreSQL Deployment Ready

**PostgreSQL is ready for deployment.**

### What's Included

1. **Modular, Production-Ready Manifests**:
   - PVC: Persistent storage that survives pod restarts
   - Init ConfigMap: Schema creation (alerts table + indexes)
   - Deployment: Pod with health checks, resource limits, restart policy
   - Service: Internal DNS and load balancing

2. **Comprehensive Deployment Guide** (DEPLOYMENT_GUIDE.md):
   - 7-step deployment process with validation after each step
   - 6 detailed verification procedures:
     - Pod status and logs
     - Health probe status
     - Database connectivity from within cluster
     - Alert table schema verification
     - PVC binding verification
     - **Pod restart survival test** (proves data persistence)
   - Troubleshooting section
   - Quick reference commands

3. **Production Features**:
   - Liveness & readiness probes configured
   - Resource requests/limits set
   - ConfigMap for non-sensitive config
   - Secret for credentials
   - Recreate strategy for PVC compatibility
   - Comments explaining all fields

### Verification Commands

```bash
# Deploy (Step 1-7 from DEPLOYMENT_GUIDE.md)
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl create secret generic eventpulse-secrets --from-literal=POSTGRES_PASSWORD='...' ...
kubectl apply -f k8s/postgres/postgres-pvc.yaml
kubectl apply -f k8s/postgres/postgres-init-cm.yaml
kubectl apply -f k8s/postgres/postgres-deployment.yaml
kubectl apply -f k8s/postgres/postgres-service.yaml

# Verify (from DEPLOYMENT_GUIDE.md section "Verification: PostgreSQL is Running")
kubectl rollout status deployment/postgres -n eventpulse -w
kubectl get pods -n eventpulse
kubectl logs -n eventpulse -l app=postgres -f
kubectl run -it --rm psql-client --image=postgres:16 --restart=Never -n eventpulse -- \
  psql -h postgres.eventpulse.svc.cluster.local -U admin -d eventpulse -c "SELECT COUNT(*) FROM alerts;"

# Test pod restart survival
kubectl delete pod -n eventpulse -l app=postgres
kubectl get pods -n eventpulse -w  # Wait for new pod to be Running
# Repeat psql query — data should still exist
```

Next: Phase 2 — Kafka Deployment (using same modular approach)
