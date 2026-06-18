# EventPulse Kubernetes Upgrade Status

**Branch**: `kubernetes-upgrade`  
**Base**: v1.0.0 (Docker Compose stable)  
**Phase**: Foundation & Configuration Setup

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

- [x] **PostgreSQL**
  - Single-replica Deployment (PostgreSQL 16)
  - PersistentVolumeClaim (20Gi)
  - ConfigMap with SQL initialization (alerts table schema)
  - Service (ClusterIP, internal DNS)
  - Liveness & readiness probes
  - Resource requests/limits: 250m CPU, 256Mi memory (min); 1000m CPU, 1Gi (max)

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

### Documentation

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
- `KUBERNETES_STATUS.md` — This file

### k8s/ Directory
- `namespace.yaml` — 12 lines
- `configmap.yaml` — 32 lines
- `secrets.yaml` — 30 lines (template with documentation)
- `postgres/postgres.yaml` — 115 lines
- `kafka/kafka.yaml` — 110 lines
- `api-gateway/api-gateway.yaml` — 100 lines
- `analytics-service/analytics-service.yaml` — 125 lines
- `alert-service/alert-service.yaml` — 125 lines
- `monitoring/prometheus.yaml` — 200+ lines
- `README.md` — 600+ lines (comprehensive guide)

**Total**: ~1,300 lines of manifests + documentation

---

## Checklist for Foundation Completion

- [x] Directory structure created
- [x] Namespace defined
- [x] ConfigMap with all non-sensitive configuration
- [x] Secret template with guidance
- [x] PostgreSQL manifest (Deployment + PVC + Service + ConfigMap)
- [x] Kafka manifest (Deployment + PVC + Service)
- [x] API Gateway manifest (Deployment + Service)
- [x] Analytics Service manifest (Deployment + Service)
- [x] Alert Service manifest (Deployment + Service)
- [x] Prometheus manifest (Deployment + ConfigMap + Service + RBAC)
- [x] Comprehensive k8s/README.md with deployment steps
- [x] KUBERNETES_STATUS.md tracking progress
- [x] Docker Compose remains untouched
- [x] No Helm, Ingress, HPA, or monitoring changes
- [x] Ready for PR review

---

## Status: Ready for Review

The Kubernetes foundation is complete and documented. All manifests are declarative, follow Kubernetes best practices, and are ready to be deployed to a Kubernetes cluster with minimal configuration (secrets creation).

Next: Await feedback before proceeding to Phase 2 (Ingress/Networking).
