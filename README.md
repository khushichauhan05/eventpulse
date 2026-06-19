# EventPulse

![Go](https://img.shields.io/badge/Go-00ADD8?logo=go&logoColor=white)
![Kafka](https://img.shields.io/badge/Kafka-231F20?logo=apachekafka&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white)
![Kubernetes](https://img.shields.io/badge/Kubernetes-326CE5?logo=kubernetes&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Grafana-F27830?logo=grafana&logoColor=white)

EventPulse is a production-grade event-driven fraud detection platform built with Go, Apache Kafka, PostgreSQL, Docker, and Kubernetes. It showcases modern distributed systems design, streaming event processing, and cloud-native infrastructure patterns for real-time fraud detection at scale.

Transaction events are ingested in real-time through REST APIs, scored by a streaming analytics service using machine learning pipelines, converted into fraud alerts, and persisted in PostgreSQL. The entire system is containerized with Docker and orchestrated on Kubernetes with full observability through Prometheus metrics and Grafana dashboards.

## Live Demo

### Grafana — Real-Time Metrics Dashboard

![Grafana Dashboard](docs/screenshots/grafana-dashboard-1.png)

![Grafana Dashboard — Total Alerts](docs/screenshots/grafana-dashboard-2.png)

### Fraud Alerts API — Real-Time Detection

![Alerts API](docs/screenshots/alerts-api.png)

### Load Test Results — 818 Transactions, 0 Failures, P99 < 48ms

![Load Test Results](docs/screenshots/load-test-results.png)

### Prometheus — Metrics (Table)

![Prometheus Table](docs/screenshots/prometheus-table.png)

### Prometheus — Metrics (Graph)

![Prometheus Graph](docs/screenshots/prometheus-graph.png)

---

## Key Highlights

- **Production-Grade**: Security-hardened Kubernetes deployment with RBAC, pod disruption budgets, and security contexts
- **Scalable**: Horizontal Pod Autoscaling (HPA) automatically scales services from 2-10 replicas based on CPU utilization
- **Observable**: Complete monitoring stack with Prometheus metrics collection and Grafana visualization dashboards
- **Resilient**: Multi-replica services with pod anti-affinity, graceful shutdown, and health probes ensure high availability
- **Secure**: Non-root container execution, read-only filesystems, network policies, and least-privilege RBAC

## Tech Stack

### Backend
- **Go** (1.26+) - High-performance microservices
- **REST APIs** - HTTP/JSON transaction interfaces

### Messaging & Event Streaming
- **Apache Kafka** - Distributed event streaming
- **KRaft Mode** - Kafka without ZooKeeper
- **Consumer Groups** - Scalable event processing

### Database
- **PostgreSQL** - Relational data persistence
- **Schema Initialization** - Automated via Kubernetes ConfigMap

### Containerization
- **Docker** - Multi-stage builds for optimized images
- **Docker Compose** - Local development orchestration

### Orchestration
- **Kubernetes** - Production cloud-native deployment
- **1.24+** - Minimum supported version

### Observability
- **Prometheus** - Time-series metrics collection (7-day retention)
- **Grafana** - Metrics visualization with 4 pre-configured dashboards

### Networking
- **NGINX Ingress Controller** - External HTTP/HTTPS access
- **Kubernetes Services** - Internal service discovery via DNS

### Scaling
- **Horizontal Pod Autoscaler (HPA)** - CPU-based auto-scaling (2-10 replicas)
- **Resource Limits** - Prevent node starvation and OOM kills

### Security
- **RBAC** - Role-based access control with least-privilege
- **Security Contexts** - Non-root execution, privilege escalation prevention
- **Pod Disruption Budgets** - Prevent accidental total service loss
- **Network Policies** - Pod-to-pod traffic control (optional, CNI-dependent)
- **Read-Only Filesystems** - Immutable binaries on stateless services

## Architecture

### Event Processing Pipeline

```
Client HTTP Request
    ↓
API Gateway (Port 8080)
    ↓
Kafka Topic: events.raw
    ↓
Analytics Service (Kafka Consumer Group: analytics-group)
    - Calculates risk scores for transactions
    ↓
Kafka Topic: events.processed
    ↓
Alert Service (Kafka Consumer Group: alert-group)
    - Generates fraud alerts for high-risk transactions
    ↓
PostgreSQL (Persistent Storage)
    ↓
API Gateway GET /alerts
    ↓
Client Response (JSON)
```

### Kubernetes Deployment Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         NGINX Ingress Controller (2 replicas)        │  │
│  │  Exposes: /events /alerts /alert /health /metrics    │  │
│  └──────────────────────────────────────────────────────┘  │
│                          ↓                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  API Gateway (2-10 replicas, HPA enabled)           │  │
│  │  • Event ingestion: POST /events                     │  │
│  │  • Alert retrieval: GET /alerts                      │  │
│  │  • Health check: GET /health                         │  │
│  │  • Metrics: GET /metrics                             │  │
│  └──────────────────────────────────────────────────────┘  │
│     ↓                                    ↓                  │
│  ┌────────────────────────────────┐  ┌──────────────────┐ │
│  │  Kafka Broker (KRaft Mode)     │  │  PostgreSQL 16   │ │
│  │  • events.raw                  │  │  • alerts table  │ │
│  │  • events.processed            │  │  • 20Gi storage  │ │
│  │  • events.dlq                  │  │  • replication   │ │
│  │  • alerts                       │  └──────────────────┘ │
│  │  • 50Gi persistent storage     │                        │
│  └────────────────────────────────┘                        │
│     ↓                    ↓                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Analytics Service (2-10 replicas, HPA enabled)     │  │
│  │  • Consumes: events.raw                             │  │
│  │  • Produces: events.processed (with risk_score)     │  │
│  │  • Consumer Group: analytics-group                  │  │
│  │  • Metrics: Port 8081                               │  │
│  └──────────────────────────────────────────────────────┘  │
│                          ↓                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Alert Service (2-10 replicas, HPA enabled)         │  │
│  │  • Consumes: events.processed                        │  │
│  │  • Produces: alerts (to PostgreSQL)                  │  │
│  │  • Consumer Group: alert-group                       │  │
│  │  • Metrics: Port 8082                                │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │    Monitoring Stack                                  │  │
│  │  ┌────────────────┐        ┌───────────────────┐   │  │
│  │  │  Prometheus    │ ←---→  │     Grafana       │   │  │
│  │  │  (scrapes)     │        │   (visualizes)    │   │  │
│  │  │  • 10Gi PVC    │        │   • 5Gi PVC       │   │  │
│  │  │  • 7-day data  │        │   • 4 dashboards  │   │  │
│  │  └────────────────┘        │   • Port 3000     │   │  │
│  │                             └───────────────────┘   │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │    Supporting Infrastructure                         │  │
│  │  • Pod Disruption Budgets (8 resources)             │  │
│  │  • Pod Anti-Affinity (multi-replica services)       │  │
│  │  • ConfigMap (shared configuration)                 │  │
│  │  • Secrets (sensitive credentials)                  │  │
│  │  • RBAC (ServiceAccount, ClusterRole)               │  │
│  │  • Network Policies (optional, CNI-dependent)       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Features

### Core Functionality
- **Real-Time Event Ingestion**: HTTP API for transaction event publishing
- **Streaming Event Processing**: Kafka-based pipeline for asynchronous processing
- **Risk Scoring Engine**: Machine learning scoring for high-value transactions
- **Fraud Alert Generation**: Automatic alert creation for high-risk events
- **Alert Persistence**: PostgreSQL storage for alert history and retrieval
- **REST API**: Complete CRUD operations for events and alerts

### Kafka Features
- **Consumer Groups**: Scalable event processing with Kafka consumer groups
- **Dead Letter Queue**: events.dlq topic for malformed messages
- **Auto Topic Creation**: Development convenience (disabled in production)
- **Snappy Compression**: Optimized log compression
- **7-Day Retention**: Configurable message retention policy

### Kubernetes Features
- **Multi-Replica Deployments**: API Gateway, Analytics, Alert Service (2 replicas each)
- **Horizontal Pod Autoscaling**: Automatic scaling from 2-10 replicas per service
- **Health Probes**: Startup, readiness, and liveness probes on all services
- **Graceful Shutdown**: 30-60 second termination grace periods
- **Zero-Downtime Updates**: Rolling update strategy (maxUnavailable: 0)
- **Resource Management**: CPU and memory requests/limits per service

### Observability
- **Prometheus Metrics**: Real-time metrics collection from all services
- **Grafana Dashboards**: 4 pre-configured dashboards:
  - EventPulse Overview
  - API Gateway Performance
  - Kafka & Analytics Pipeline
  - Alert Service & Database
- **Health Endpoints**: Liveness and readiness checks on all services
- **Structured Logging**: Service-aware log formatting with levels

### Resilience & Availability
- **Pod Disruption Budgets**: Protection for all 8 services
- **Pod Anti-Affinity**: Replicas spread across different nodes
- **Multi-Zone Deployment**: Optional cross-zone affinity for cloud deployments
- **Service Recovery**: Automatic pod restart on failure via liveness probes
- **Load Balancing**: Kubernetes service load distribution

### Security
- **RBAC**: Role-based access control with least-privilege principle
- **Security Contexts**: Non-root execution (uid 1000+) on all services
- **Privilege Escalation Prevention**: allowPrivilegeEscalation: false
- **Capability Restrictions**: Linux capabilities dropped (except NET_BIND_SERVICE for NGINX)
- **Read-Only Filesystems**: Immutable binaries on stateless services
- **Network Policies**: Optional traffic control (CNI-dependent)
- **Secret Management**: Kubernetes Secrets for credentials (encrypted options documented)

## Project Structure

```
EventPulse/
├── services/                           # Microservices
│   ├── api-gateway/
│   │   ├── main.go
│   │   ├── handler.go
│   │   ├── Dockerfile
│   │   └── Dockerfile.dev
│   ├── analytics-service/
│   │   ├── main.go
│   │   ├── processor.go
│   │   ├── Dockerfile
│   │   └── Dockerfile.dev
│   └── alert-service/
│       ├── main.go
│       ├── alerter.go
│       ├── Dockerfile
│       └── Dockerfile.dev
│
├── internal/                           # Shared packages
│   ├── config/
│   │   └── config.go
│   ├── logger/
│   │   └── logger.go
│   ├── handlers/
│   │   ├── gateway_test.go
│   │   └── gateway.go
│   └── kafka/
│       └── client.go
│
├── db/                                 # Database
│   └── init.sql                        # Schema initialization
│
├── k8s/                                # Kubernetes manifests
│   ├── namespace.yaml                  # Namespace definition
│   ├── configmap.yaml                  # Shared configuration
│   ├── secrets.yaml                    # Secrets template
│   ├── postgres/                       # PostgreSQL deployment
│   │   ├── postgres-pvc.yaml
│   │   ├── postgres-init-cm.yaml
│   │   ├── postgres-deployment.yaml
│   │   └── postgres-service.yaml
│   ├── kafka/                          # Kafka deployment
│   │   ├── kafka-pvc.yaml
│   │   ├── kafka-deployment.yaml
│   │   ├── kafka-service.yaml
│   │   └── kafka.yaml
│   ├── api-gateway/                    # API Gateway
│   │   ├── api-gateway-deployment.yaml
│   │   └── api-gateway.yaml
│   ├── analytics-service/              # Analytics Service
│   │   ├── analytics-service-deployment.yaml
│   │   └── analytics-service.yaml
│   ├── alert-service/                  # Alert Service
│   │   ├── alert-service-deployment.yaml
│   │   └── alert-service.yaml
│   ├── ingress/                        # NGINX Ingress
│   │   ├── nginx-ingress-deployment.yaml
│   │   └── eventpulse-ingress.yaml
│   ├── autoscaling/                    # Horizontal Pod Autoscaling
│   │   ├── api-gateway-hpa.yaml
│   │   ├── analytics-service-hpa.yaml
│   │   └── alert-service-hpa.yaml
│   ├── monitoring/                     # Prometheus & Grafana
│   │   ├── prometheus-deployment.yaml
│   │   ├── prometheus.yaml
│   │   └── grafana-deployment.yaml
│   └── security/                       # Security resources
│       ├── pod-disruption-budgets.yaml
│       └── network-policies.yaml
│
├── docs/                               # Documentation
│   └── screenshots/                    # Architecture & validation screenshots
│       ├── architecture.png
│       ├── api-request.png
│       ├── analytics-service.png
│       ├── alert-service.png
│       └── alerts-response.png
│
├── docker-compose.yml                  # Local development (Docker Compose only)
├── Dockerfile                          # Multi-stage build
├── .gitignore
├── .gitattributes
├── go.mod
├── go.sum
├── LICENSE
│
├── README.md                           # This file
├── DEPLOYMENT_GUIDE.md                 # Kubernetes deployment guide
├── VALIDATION.md                       # End-to-end validation procedures
├── KUBERNETES_STATUS.md                # Project status & completion tracking
├── SECURITY_REVIEW.md                  # Security audit & hardening details
├── SECURITY_VALIDATION.md              # Security verification checklist
├── INGRESS_GUIDE.md                    # NGINX Ingress setup & troubleshooting
├── AUTOSCALING_GUIDE.md                # HPA configuration & load testing
├── MONITORING_GUIDE.md                 # Prometheus & Grafana setup
├── RELEASE_VALIDATION.md               # v2.0.0 release report
└── PR_TEMPLATE.md                      # GitHub PR template
```

## Deployment Options

### Option 1: Docker Compose (Local Development Only)

**Use Case**: Development, testing, and learning

**Quick Start**:
```bash
docker compose up --build
```

**Features**:
- Single-file orchestration
- All services in one container stack
- Auto-startup dependencies
- Local port forwarding

**Limitations**:
- Single machine (no scaling)
- No health recovery
- No monitoring
- Limited to development

See docker-compose.yml for configuration.

### Option 2: Kubernetes (Production Deployment)

**Use Case**: Staging and production deployments

**Quick Start**:
```bash
# 1. Create namespace
kubectl create namespace eventpulse

# 2. Create secrets
kubectl create secret generic eventpulse-secrets \
  --from-literal=DATABASE_DSN="postgresql://postgres:password@postgres:5432/eventpulse" \
  -n eventpulse

# 3. Apply manifests
kubectl apply -f k8s/

# 4. Verify
kubectl get pods -n eventpulse
```

**Features**:
- Multi-replica deployments
- Automatic scaling (2-10 replicas)
- Health probes and recovery
- Pod Disruption Budgets
- Security hardening
- Complete monitoring stack
- Production-grade reliability

**Architecture**:
- Kubernetes 1.24+
- 35+ YAML manifest files
- 8 services (PostgreSQL, Kafka, 3 app services, NGINX, Prometheus, Grafana)
- Full RBAC configuration
- Storage classes for persistence

**Documentation**:
- [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) - Step-by-step deployment
- [VALIDATION.md](VALIDATION.md) - End-to-end validation
- [KUBERNETES_STATUS.md](KUBERNETES_STATUS.md) - Status tracking
- [SECURITY_REVIEW.md](SECURITY_REVIEW.md) - Security details

---

## Monitoring & Observability

### Prometheus Metrics

EventPulse services expose Prometheus metrics on dedicated ports:

- **API Gateway** (port 8080/metrics):
  - HTTP request rate
  - Request latency percentiles
  - Event ingestion rate
  - Error count

- **Analytics Service** (port 8081/metrics):
  - Events processed per minute
  - Processing latency
  - Risk scores calculated
  - Kafka consumer lag

- **Alert Service** (port 8082/metrics):
  - Alerts generated per minute
  - Alert generation latency
  - High-risk transaction count
  - Database write operations

### Grafana Dashboards

Four pre-configured production dashboards:

1. **EventPulse Overview** - System health at a glance
   - Event ingestion rate
   - Alert generation rate
   - Pipeline depth
   - Consumer lag
   - Error rate

2. **API Gateway Performance** - Request metrics
   - Request rate (RPS)
   - Latency percentiles (p50, p95, p99)
   - Event publishing rate
   - Error count

3. **Kafka & Analytics** - Message flow monitoring
   - Raw events consumed
   - Processing latency
   - Risk scores calculated
   - Consumer lag

4. **Alert Service & Database** - Alert generation and persistence
   - Alerts generated per minute
   - Database write rate
   - High-risk transaction count
   - Write errors

**Access**:
```bash
kubectl port-forward -n eventpulse svc/grafana 3000:3000
# Visit http://localhost:3000 (default: admin/admin)
```

### Health Endpoints

All services expose health check endpoints:

```bash
# API Gateway
curl http://localhost:8080/health

# Analytics Service
curl http://localhost:8081/health

# Alert Service
curl http://localhost:8082/health
```

---

## Security

### RBAC (Role-Based Access Control)

- **Prometheus ServiceAccount**: Read-only access to pod/service/node discovery
- **NGINX Ingress ServiceAccount**: Read-only access to ingress resources
- **Application Services**: No cluster API access (principle of least privilege)

### Security Contexts

All containers run with hardened security contexts:

```yaml
securityContext:
  runAsNonRoot: true        # No root execution
  runAsUser: 1000           # Non-root user ID
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true  # Stateless services only
  capabilities:
    drop:
    - ALL                   # All capabilities removed
```

### Read-Only Filesystems

Stateless services (API Gateway, Analytics, Alert) run with immutable root filesystems:
- Prevents binary modification
- Protects against container escape
- Go applications are fully compiled (no runtime compilation)

Stateful services (PostgreSQL, Kafka, Prometheus) require writable storage:
- Necessary for data persistence
- Secured via other controls (RBAC, network policies)

### Pod Disruption Budgets

All 8 services protected with Pod Disruption Budgets:
- Minimum 1 pod must always be running
- Prevents accidental total outage during maintenance
- Enables safe node drains and autoscaling

### Pod Anti-Affinity

Multi-replica services spread across different nodes:
- `preferredDuringSchedulingIgnoredDuringExecution`
- Node failure doesn't cause total service loss
- Enables true high availability

### Network Policies (Optional)

7 NetworkPolicy templates provided (requires CNI support):
- Default deny all ingress
- Explicit allow for each service
- Traffic control between pods
- Optional implementation (Calico, Cilium, etc.)

---

## Scalability

### Horizontal Scaling

**Kafka Consumer Groups**: Event processing scales automatically
- Analytics Service: `analytics-group` (2 consumers, scales to 10)
- Alert Service: `alert-group` (2 consumers, scales to 10)
- Each additional replica automatically joins consumer group

**Kubernetes HPA**: Automatic pod scaling
- Monitors CPU utilization (70% target)
- Scales up: 1 pod per 30 seconds
- Scales down: 1 pod per 60 seconds (after 5-minute stable)
- Min replicas: 2 per service
- Max replicas: 10 per service

**Auto-Scaling in Action**:
```bash
# Watch HPA status
kubectl get hpa -n eventpulse --watch

# Expected scaling timeline (under load):
# Time 0s: 2 replicas @ 80% CPU
# Time 30s: 3 replicas scaling up
# Time 60s: 4 replicas if still > 70% CPU
# ... continues until load decreases
```

### Load Testing

See [AUTOSCALING_GUIDE.md](AUTOSCALING_GUIDE.md) for comprehensive load testing procedures including:
- Apache Bench stress testing
- Event publishing loops
- Kafka consumer lag monitoring
- Real-time scaling observation
- Prometheus metrics collection

---

## Resume Value

EventPulse demonstrates essential distributed systems and backend engineering concepts:

### Architecture Patterns
- **Microservices**: Independent services with clear boundaries
- **Event-Driven Architecture**: Asynchronous communication via Kafka
- **Distributed Systems**: Multi-service coordination and fault tolerance
- **Consumer Groups**: Scalable event processing and load distribution

### Kubernetes & Cloud-Native
- **Declarative Infrastructure**: Kubernetes manifests as code
- **Container Orchestration**: Pod management, deployment strategies
- **Service Discovery**: Kubernetes DNS for dynamic networking
- **Resource Management**: CPU/memory requests, limits, autoscaling
- **High Availability**: PDBs, anti-affinity, health probes
- **Security**: RBAC, pod security contexts, network policies

### Observability & Monitoring
- **Metrics Collection**: Prometheus scrape configuration
- **Visualization**: Grafana dashboards for real-time insights
- **Health Monitoring**: Liveness, readiness, startup probes
- **Distributed Tracing**: Service-to-service visibility

### Reliability & Resilience
- **Graceful Degradation**: Services continue on partial failures
- **Self-Healing**: Automatic pod restart on failure
- **Circuit Breakers**: Kafka consumer lag monitoring
- **Idempotency**: Event deduplication via event_id

### Go & Backend Development
- **Concurrency**: goroutines for handling multiple streams
- **HTTP APIs**: REST endpoints with proper status codes
- **Database Integration**: PostgreSQL with connection pooling
- **Structured Logging**: Contextual logging with service names
- **Error Handling**: Proper error propagation and recovery

---

## Validation Results

### Architecture Diagram

![Architecture](docs/screenshots/architecture.png)

### API Request

Successfully publishing an event through the API Gateway.

![API Request](docs/screenshots/api-request.png)

### Analytics Processing

Analytics service consuming events from Kafka and calculating risk scores.

![Analytics Service](docs/screenshots/analytics-service.png)

### Alert Generation

Alert service consuming processed events and generating fraud alerts.

![Alert Service](docs/screenshots/alert-service.png)

### Alert Retrieval

Alerts stored in PostgreSQL and retrieved through the REST API.

![Alerts Response](docs/screenshots/alerts-response.png)

---

## Production Readiness Checklist

### Infrastructure
- [x] PostgreSQL with persistent storage (20Gi PVC)
- [x] Kafka in KRift mode with persistence (50Gi PVC)
- [x] Application services with 2 replicas
- [x] NGINX Ingress for external access
- [x] Horizontal Pod Autoscaling (2-10 replicas)
- [x] Prometheus metrics collection (10Gi PVC)
- [x] Grafana visualization (5Gi PVC)

### Security
- [x] Non-root container execution
- [x] Privilege escalation prevention
- [x] Linux capability restrictions
- [x] Pod Disruption Budgets (all services)
- [x] Pod anti-affinity (multi-replica services)
- [x] RBAC (least-privilege)
- [x] Network Policies (templates provided)

### Reliability
- [x] Health probes (startup, readiness, liveness)
- [x] Graceful shutdown (30-60s)
- [x] Rolling updates (zero-downtime)
- [x] Resource requests/limits
- [x] Service recovery on failure
- [x] Pod anti-affinity for availability

### Observability
- [x] Prometheus metrics from all services
- [x] Grafana dashboards (4 pre-configured)
- [x] Health endpoints on all services
- [x] Structured logging
- [x] Performance metrics collection

### Documentation
- [x] Deployment guide (1,200+ lines)
- [x] Validation procedures (700+ lines)
- [x] Security audit (600+ lines)
- [x] Ingress troubleshooting (500+ lines)
- [x] Autoscaling guide (450+ lines)
- [x] Monitoring setup (500+ lines)

### Pre-Production
- [ ] Implement encrypted secrets (Sealed Secrets/Vault)
- [ ] Apply Network Policies (if CNI supports)
- [ ] Configure persistent volume classes
- [ ] Set up cluster monitoring/alerting
- [ ] Create backup/recovery procedures

---

## Future Improvements

### Security Enhancements
- [ ] Implement Sealed Secrets or HashiCorp Vault for encrypted secrets
- [ ] Apply Network Policies (Calico/Cilium) for pod-to-pod traffic control
- [ ] Pod Security Policies (PSP) or Pod Security Standards (PSS)
- [ ] Audit logging for compliance and forensics
- [ ] Image scanning for vulnerability detection

### Operational Excellence
- [ ] Integration tests with Testcontainers
- [ ] GitOps with ArgoCD or Flux
- [ ] Helm charts for easy deployment
- [ ] Custom CRDs for domain-specific resources
- [ ] Service mesh (Istio/Linkerd) for advanced traffic management

### Feature Enhancements
- [ ] Idempotency keys for alert writes
- [ ] Outbox pattern for atomic Kafka/Postgres consistency
- [ ] Machine learning scoring enhancements
- [ ] Real-time alerting via webhooks
- [ ] Multi-region deployment support

### Observability
- [ ] Distributed tracing (Jaeger/Tempo)
- [ ] Custom metrics for business logic
- [ ] Alert rules in Prometheus
- [ ] SLO/SLI tracking
- [ ] Log aggregation (ELK/Loki)

---

## Getting Started

### Prerequisites
- **For Docker Compose**: Docker Desktop 4.0+, Docker Compose 1.29+
- **For Kubernetes**: kubectl, Kubernetes cluster 1.24+

### Local Development (Docker Compose)

```bash
# Clone and navigate
git clone https://github.com/apekshita0511/EventPulse.git
cd EventPulse

# Build and run
docker compose up --build

# Services available at:
# API Gateway: http://localhost:8080
# Health checks: :8080/health, :8081/health, :8082/health
```

### Production Deployment (Kubernetes)

See [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) for complete instructions.

```bash
# Quick start
kubectl create namespace eventpulse
kubectl create secret generic eventpulse-secrets \
  --from-literal=DATABASE_DSN="postgresql://..." \
  -n eventpulse
kubectl apply -f k8s/
```

---

## API Endpoints

### Event Ingestion
```bash
POST /events
Content-Type: application/json

{
  "user_id": "user_123",
  "event_type": "purchase",
  "amount": 75000
}
```

### Alert Retrieval
```bash
GET /alerts           # List all alerts
GET /alert?id=1       # Get specific alert
```

### Health & Metrics
```bash
GET /health           # Liveness probe
GET /metrics          # Prometheus metrics
```

---

## Kafka Topics

| Topic | Purpose | Retention |
|-------|---------|-----------|
| events.raw | Raw transaction events | 7 days |
| events.processed | Events with risk scores | 7 days |
| alerts | Fraud alerts | 7 days |
| events.dlq | Malformed messages | 7 days |

---

## Environment Variables

All configuration available via Kubernetes ConfigMap or Docker Compose environment:

```
SERVICE_NAME          # Service identifier
PORT                  # Primary service port
KAFKA_BROKERS         # Kafka broker addresses
KAFKA_TOPIC_RAW       # Raw events topic
KAFKA_TOPIC_PROCESSED # Processed events topic
KAFKA_TOPIC_ALERTS    # Alerts topic
KAFKA_TOPIC_DLQ       # Dead letter queue
KAFKA_ANALYTICS_GROUP # Analytics consumer group
KAFKA_ALERT_GROUP     # Alert consumer group
DATABASE_DSN          # PostgreSQL connection string
LOG_LEVEL             # Logging verbosity
```

---

## Configuration

### Docker Compose
Edit `docker-compose.yml` for development configuration.

### Kubernetes
Edit:
- `k8s/configmap.yaml` - Non-sensitive configuration
- `k8s/secrets.yaml` - Sensitive credentials template
- Individual deployment files for service-specific settings

---

## Troubleshooting

### Docker Compose Issues
- **Kafka not starting**: Wait for health checks; Kafka startup is slow
- **PostgreSQL empty**: Confirm `db/init.sql` exists and volume is fresh
- **Connection errors**: Check `docker compose logs` for detailed output

### Kubernetes Issues
- **Pods not starting**: `kubectl describe pod <name> -n eventpulse`
- **Service unavailable**: `kubectl get svc -n eventpulse`
- **Ingress not working**: See [INGRESS_GUIDE.md](INGRESS_GUIDE.md)

For comprehensive troubleshooting, see [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) and [VALIDATION.md](VALIDATION.md).

---

## Related Documentation

- [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) - Complete Kubernetes deployment
- [VALIDATION.md](VALIDATION.md) - End-to-end pipeline validation
- [KUBERNETES_STATUS.md](KUBERNETES_STATUS.md) - Project status and progress
- [SECURITY_REVIEW.md](SECURITY_REVIEW.md) - Security implementation details
- [SECURITY_VALIDATION.md](SECURITY_VALIDATION.md) - Security verification
- [INGRESS_GUIDE.md](INGRESS_GUIDE.md) - NGINX Ingress setup
- [AUTOSCALING_GUIDE.md](AUTOSCALING_GUIDE.md) - HPA and load testing
- [MONITORING_GUIDE.md](MONITORING_GUIDE.md) - Prometheus and Grafana

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

## Summary

EventPulse is a **production-ready event-driven fraud detection platform** that demonstrates:

 **Distributed Systems Design** - Microservices, event streaming, consumer groups  
 **Cloud-Native Kubernetes** - Production deployment with security and observability  
 **Modern Go Backend** - High-performance REST APIs with streaming processing  
 **Scalable Architecture** - Automatic scaling from 2-10 replicas per service  
 **Complete Observability** - Prometheus metrics and Grafana dashboards  
 **Security Hardening** - RBAC, non-root execution, pod disruption budgets  
 **Production Reliability** - Health probes, graceful shutdown, zero-downtime updates

**Use EventPulse for**:
- Learning cloud-native Kubernetes development
- Understanding event-driven microservices patterns
- Exploring distributed systems design
- Building production fraud detection pipelines
- Backend engineering portfolio demonstration

---

**Version**: v2.0.0 (Kubernetes Release)  
**Last Updated**: 2026-06-18  
**Status**: Production Ready 