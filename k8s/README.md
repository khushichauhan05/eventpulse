# EventPulse Kubernetes Foundation

This directory contains Kubernetes manifests for deploying EventPulse to a Kubernetes cluster. This is a foundation-only setup — no Helm, no Ingress, no HPA, no monitoring changes.

## Directory Structure

```
k8s/
├── README.md                    # This file
├── namespace.yaml               # EventPulse namespace
├── configmap.yaml               # Non-sensitive configuration (shared)
├── secrets.yaml                 # Sensitive credentials (template only)
│
├── postgres/
│   └── postgres.yaml            # PostgreSQL Deployment, Service, PVC, ConfigMap
│
├── kafka/
│   └── kafka.yaml               # Kafka Deployment, Service, PVC
│
├── api-gateway/
│   └── api-gateway.yaml         # API Gateway Deployment, Service (2 replicas)
│
├── analytics-service/
│   └── analytics-service.yaml   # Analytics Service Deployment, Service (2 replicas)
│
├── alert-service/
│   └── alert-service.yaml       # Alert Service Deployment, Service (2 replicas)
│
├── monitoring/
│   └── prometheus.yaml          # Prometheus Deployment, Service, ConfigMap, RBAC
│
└── ingress/
    └── (reserved for future Ingress controller resources)
```

## File Descriptions

### Core Configuration

#### `namespace.yaml`
- Creates the `eventpulse` namespace
- All resources are deployed in this namespace for isolation

#### `configmap.yaml`
- Contains **non-sensitive** environment variables shared across all services
- Includes:
  - Kafka broker addresses (using Kubernetes DNS names)
  - Kafka topic names
  - Consumer group IDs
  - Service names and ports
  - PostgreSQL connection details (host, port, db name, user only)
  - Log level
- Mounted as environment variables in all service deployments

#### `secrets.yaml`
- **Template only** — contains placeholders for sensitive data
- Includes:
  - PostgreSQL password
  - Full PostgreSQL DSN (used by Go applications)
  - Grafana admin password
- **Never commit real secrets to this file**
- See "Secrets Management" below for production approaches

### Infrastructure

#### `postgres/postgres.yaml`
- **Deployment**: Single-replica PostgreSQL 16
- **ConfigMap**: SQL initialization script (creates `alerts` table with schema)
- **PersistentVolumeClaim**: 20Gi storage
- **Service**: `postgres.eventpulse.svc.cluster.local:5432`
- **Probes**:
  - Liveness: pg_isready check every 10s
  - Readiness: pg_isready check every 5s
- **Resource limits**: 250m CPU request, 1Gi limit; 256Mi memory request, 1Gi limit

#### `kafka/kafka.yaml`
- **Deployment**: Single-node Apache Kafka
- **PersistentVolumeClaim**: 50Gi storage
- **Service**: `kafka.eventpulse.svc.cluster.local:9092`
- **Features**:
  - KRaft mode (no ZooKeeper)
  - Auto topic creation enabled
  - Liveness probe checks broker API
  - Readiness probe lists topics
- **Resource limits**: 500m CPU request, 2Gi limit; 512Mi memory request, 2Gi limit

### Services

#### `api-gateway/api-gateway.yaml`
- **Deployment**: 2 replicas (for load distribution)
- **Container port**: 8080 (HTTP API + Prometheus metrics)
- **Image**: `eventpulse-api-gateway:latest` (built locally)
- **Environment**: All vars from ConfigMap + DATABASE_DSN from Secret
- **Liveness probe**: HTTP GET `/health:8080` every 10s (fail after 3 tries)
- **Readiness probe**: HTTP GET `/health:8080` every 5s (fail after 3 tries)
- **Service**: `api-gateway.eventpulse.svc.cluster.local:8080`

#### `analytics-service/analytics-service.yaml`
- **Deployment**: 2 replicas (for Kafka consumer group scalability)
- **Container ports**: 
  - 8080: Main application port
  - 8081: Health + Prometheus metrics
- **Image**: `eventpulse-analytics-service:latest`
- **Environment**: All vars from ConfigMap + DATABASE_DSN from Secret
- **Liveness probe**: HTTP GET `/health:8081` every 10s
- **Readiness probe**: HTTP GET `/health:8081` every 5s
- **Service**: `analytics-service.eventpulse.svc.cluster.local:8081`
- **Consumer group**: `analytics-group` (configurable via env)

#### `alert-service/alert-service.yaml`
- **Deployment**: 2 replicas (for Kafka consumer group scalability)
- **Container ports**:
  - 8080: Main application port
  - 8082: Health + Prometheus metrics
- **Image**: `eventpulse-alert-service:latest`
- **Environment**: All vars from ConfigMap + DATABASE_DSN from Secret
- **Liveness probe**: HTTP GET `/health:8082` every 10s
- **Readiness probe**: HTTP GET `/health:8082` every 5s
- **Service**: `alert-service.eventpulse.svc.cluster.local:8082`
- **Consumer group**: `alert-group` (configurable via env)

### Monitoring

#### `monitoring/prometheus.yaml`
- **Deployment**: Single-replica Prometheus
- **ConfigMap**: Scrape configuration with Kubernetes service discovery
- **PersistentVolumeClaim**: 10Gi storage
- **Service**: `prometheus.eventpulse.svc.cluster.local:9090`
- **RBAC**: ServiceAccount + ClusterRole + ClusterRoleBinding for pod discovery
- **Features**:
  - Auto-discovers all pods in `eventpulse` namespace
  - Scrapes `/metrics` endpoint from all services
  - Retains metrics for 7 days
  - Label-based job discovery

## Configuration Management

### Environment Variables

**Non-sensitive (ConfigMap)**:
```
KAFKA_BROKERS
KAFKA_TOPIC_*
KAFKA_*_GROUP
LOG_LEVEL
*_PORT
*_SERVICE_NAME
POSTGRES_HOST
POSTGRES_PORT
POSTGRES_DB
POSTGRES_USER
```

**Sensitive (Secret)**:
```
POSTGRES_PASSWORD
DATABASE_DSN
GRAFANA_PASSWORD
```

All services use Kubernetes `valueFrom` to inject:
- ConfigMap keys → environment variables
- Secret keys → environment variables

### Secrets Management

**For Development**:
```bash
kubectl create secret generic eventpulse-secrets \
  --from-literal=POSTGRES_PASSWORD='dev-password' \
  --from-literal=DATABASE_DSN='host=postgres.eventpulse.svc.cluster.local port=5432 user=admin password=dev-password dbname=eventpulse sslmode=disable' \
  --from-literal=GRAFANA_PASSWORD='dev-password' \
  -n eventpulse
```

**For CI/CD**:
- Use GitHub Actions secrets to populate a `.env.secrets` file
- Create the secret during deployment: `kubectl create secret generic eventpulse-secrets --from-env-file=.env.secrets -n eventpulse`

**For Production**:
- Use external secret management:
  - **Sealed Secrets**: `kubectl apply -f secret.sealed.yaml`
  - **HashiCorp Vault**: Operator integration or webhook
  - **AWS Secrets Manager**: Using IRSA (IAM Roles for Service Accounts)
  - **Azure Key Vault**: Using workload identity
  - **Google Secret Manager**: Using Workload Identity

## Deployment

### Prerequisites

1. Kubernetes cluster (1.24+)
2. `kubectl` configured to access the cluster
3. Docker images built and available:
   - `eventpulse-api-gateway:latest`
   - `eventpulse-analytics-service:latest`
   - `eventpulse-alert-service:latest`

### Quick Start (Development)

```bash
# 1. Create namespace
kubectl apply -f k8s/namespace.yaml

# 2. Create ConfigMap
kubectl apply -f k8s/configmap.yaml

# 3. Create Secrets (development)
kubectl create secret generic eventpulse-secrets \
  --from-literal=POSTGRES_PASSWORD='postgres123' \
  --from-literal=DATABASE_DSN='host=postgres.eventpulse.svc.cluster.local port=5432 user=admin password=postgres123 dbname=eventpulse sslmode=disable' \
  --from-literal=GRAFANA_PASSWORD='grafana123' \
  -n eventpulse

# 4. Deploy infrastructure (PostgreSQL + Kafka)
kubectl apply -f k8s/postgres/postgres.yaml
kubectl apply -f k8s/kafka/kafka.yaml

# 5. Wait for readiness (5-10 minutes)
kubectl rollout status deployment/postgres -n eventpulse
kubectl rollout status deployment/kafka -n eventpulse

# 6. Deploy services
kubectl apply -f k8s/api-gateway/api-gateway.yaml
kubectl apply -f k8s/analytics-service/analytics-service.yaml
kubectl apply -f k8s/alert-service/alert-service.yaml

# 7. Deploy monitoring
kubectl apply -f k8s/monitoring/prometheus.yaml

# 8. Verify
kubectl get pods -n eventpulse
kubectl get svc -n eventpulse
```

### Port Forwarding (Local Testing)

```bash
# API Gateway (POST /events, GET /alerts)
kubectl port-forward -n eventpulse svc/api-gateway 8080:8080

# Prometheus
kubectl port-forward -n eventpulse svc/prometheus 9090:9090

# Direct pod access
kubectl port-forward -n eventpulse pod/postgres-xxx 5432:5432
kubectl port-forward -n eventpulse pod/kafka-xxx 9092:9092
```

### Viewing Logs

```bash
# Service logs
kubectl logs -n eventpulse deployment/api-gateway
kubectl logs -n eventpulse deployment/analytics-service
kubectl logs -n eventpulse deployment/alert-service

# Stream logs
kubectl logs -n eventpulse deployment/api-gateway -f

# Logs from specific pod
kubectl logs -n eventpulse pod/api-gateway-xxx
```

## Network Architecture

### Service Discovery (Kubernetes DNS)

All services communicate via Kubernetes service DNS:

```
api-gateway         → postgres.eventpulse.svc.cluster.local:5432
api-gateway         → kafka.eventpulse.svc.cluster.local:9092
analytics-service   → kafka.eventpulse.svc.cluster.local:9092
alert-service       → kafka.eventpulse.svc.cluster.local:9092
alert-service       → postgres.eventpulse.svc.cluster.local:5432
prometheus          → api-gateway:8080/metrics (via service discovery)
prometheus          → analytics-service:8081/metrics (via service discovery)
prometheus          → alert-service:8082/metrics (via service discovery)
```

### Storage

- **PostgreSQL**: PVC `postgres-pvc` (20Gi, `ReadWriteOnce`)
- **Kafka**: PVC `kafka-pvc` (50Gi, `ReadWriteOnce`)
- **Prometheus**: PVC `prometheus-pvc` (10Gi, `ReadWriteOnce`)

These use the default StorageClass. For production, specify a StorageClass explicitly.

## Health & Readiness

All deployments include:

- **Liveness probe**: Restarts unhealthy pods
  - API Gateway: HTTP GET `/health` on port 8080
  - Analytics Service: HTTP GET `/health` on port 8081
  - Alert Service: HTTP GET `/health` on port 8082
  - PostgreSQL: `pg_isready` command
  - Kafka: Broker API version check

- **Readiness probe**: Marks pods ready for traffic
  - Services: HTTP GET `/health` (same as liveness)
  - PostgreSQL: `pg_isready` command
  - Kafka: `kafka-topics.sh --list`

Initial delays staggered to allow services time to start before probes begin.

## Scaling

### Horizontal Pod Autoscaling (Not Implemented)

To add HPA later:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: analytics-service-hpa
  namespace: eventpulse
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: analytics-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 75
```

### Manual Scaling

```bash
kubectl scale deployment/api-gateway --replicas=3 -n eventpulse
```

## Persistence

All stateful services use PersistentVolumes:

| Service | Size | Mode | Purpose |
|---------|------|------|---------|
| PostgreSQL | 20Gi | RWO | Data storage, alerts table |
| Kafka | 50Gi | RWO | Message log, brokers |
| Prometheus | 10Gi | RWO | Time-series metrics |

**Important**: PVCs are cluster-scoped. When deleting deployments, PVCs persist and data is retained.

To **clean up storage**:
```bash
kubectl delete pvc postgres-pvc kafka-pvc prometheus-pvc -n eventpulse
```

## Next Steps

To add features:

1. **Ingress**: Add `k8s/ingress/ingress.yaml` with Nginx/Traefik
2. **TLS**: Add cert-manager and certificates
3. **Monitoring**: Add Grafana `k8s/monitoring/grafana.yaml`
4. **Service Mesh**: Istio, Linkerd (service-to-service mTLS, observability)
5. **Secrets**: Sealed Secrets or external secret operator
6. **GitOps**: ArgoCD or Flux for declarative deployments
7. **HPA**: Kubernetes Metrics Server + HorizontalPodAutoscaler

## Troubleshooting

### Pods not starting

```bash
kubectl describe pod <pod-name> -n eventpulse
kubectl logs <pod-name> -n eventpulse
```

### Services not communicating

Check DNS:
```bash
kubectl run -it --rm debug --image=busybox --restart=Never -n eventpulse -- sh
# nslookup kafka.eventpulse.svc.cluster.local
# nc -zv postgres.eventpulse.svc.cluster.local 5432
```

### Metrics not scraping

Check Prometheus targets:
```bash
kubectl port-forward -n eventpulse svc/prometheus 9090:9090
# Open http://localhost:9090/targets
```

### Storage issues

```bash
kubectl get pvc -n eventpulse
kubectl get pv
```

## References

- Kubernetes docs: https://kubernetes.io/docs/
- Deployments: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
- ConfigMaps & Secrets: https://kubernetes.io/docs/concepts/configuration/configmap/
- Probes: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
- Service Discovery: https://kubernetes.io/docs/concepts/services-networking/service/
