# EventPulse NGINX Ingress Installation & Troubleshooting Guide

Complete guide for installing and managing NGINX Ingress Controller for EventPulse.

**Objective**: Expose EventPulse API Gateway endpoints through a single HTTP entrypoint using NGINX Ingress Controller.

---

## What is Ingress?

**Ingress** is a Kubernetes resource that manages external HTTP/HTTPS access to services.

Instead of:
```
Client → api-gateway:8080 (direct port access)
```

With Ingress:
```
Client → HTTP/HTTPS (port 80/443) → NGINX Ingress Controller → api-gateway:8080
```

**Benefits**:
- Single entrypoint for all API endpoints
- Path-based routing (/events, /alerts, /health)
- TLS/HTTPS termination
- Rate limiting, CORS handling
- Load balancing across replicas

---

## Prerequisites

- Kubernetes cluster running (minikube, EKS, GKE, AKS, etc.)
- EventPulse services deployed (PostgreSQL, Kafka, API Gateway, etc.)
- kubectl CLI configured
- Ingress Controller support (most K8s distributions have this)

---

## Phase 4a: Install NGINX Ingress Controller

### Step 1: Install NGINX Ingress Controller

Deploy the NGINX Ingress Controller in the `ingress-nginx` namespace:

```bash
kubectl apply -f k8s/ingress/nginx-ingress-deployment.yaml
```

**What this creates**:
- Namespace: `ingress-nginx`
- ServiceAccount: `nginx-ingress`
- ClusterRole & ClusterRoleBinding (RBAC)
- Deployment: `nginx-ingress-controller` (2 replicas)
- Service: `nginx-ingress` (LoadBalancer type)
- ConfigMap: `nginx-config` (NGINX settings)
- IngressClass: `nginx` (defines controller)

### Step 2: Verify Controller Deployment

```bash
kubectl get deployment -n ingress-nginx
```

**Expected output**:
```
NAME                       READY   UP-TO-DATE   AVAILABLE   AGE
nginx-ingress-controller   2/2     2            2           30s
```

Watch rollout progress:
```bash
kubectl rollout status deployment/nginx-ingress-controller -n ingress-nginx -w
```

**Expected output**:
```
deployment "nginx-ingress-controller" successfully rolled out
```

### Step 3: Verify NGINX Pods

```bash
kubectl get pods -n ingress-nginx -o wide
```

**Expected output**:
```
NAME                                        READY   STATUS    RESTARTS   AGE   IP            NODE
nginx-ingress-controller-xxxxx              1/1     Running   0          1m    10.244.x.x    <node>
nginx-ingress-controller-yyyyy              1/1     Running   0          1m    10.244.x.x    <node>
```

### Step 4: Check Ingress Controller Service

```bash
kubectl get service -n ingress-nginx
```

**Expected output**:
```
NAME            TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)                      AGE
nginx-ingress   LoadBalancer   10.xxx.xxx.xxx   <pending>     80:30xxx/TCP,443:30xxx/TCP   1m
```

**Note on EXTERNAL-IP**:
- **Local/Minikube**: May show `<pending>` or localhost IP (use port-forward for testing)
- **Cloud (AWS/GCP/Azure)**: Will provision external Load Balancer IP
- **On-premises**: Requires MetalLB or similar

### Step 5: Get Controller Logs

```bash
kubectl logs -n ingress-nginx deployment/nginx-ingress-controller -f
```

**Expected output** (first few lines):
```
I1118 15:30:00.123456       1 controller.go:178] "msg"="ingress-nginx controller started" "hash"="..."
I1118 15:30:00.234567       1 controller.go:209] "msg"="watching for ingress class"
```

---

## Phase 4b: Deploy EventPulse Ingress

### Step 1: Apply EventPulse Ingress

Route external traffic to API Gateway:

```bash
kubectl apply -f k8s/ingress/eventpulse-ingress.yaml
```

**What this creates**:
- Ingress resource: `eventpulse-ingress`
- Routes `/events`, `/alerts`, `/alert`, `/metrics`, `/health` to api-gateway:8080
- CORS enabled for cross-origin requests
- Rate limiting (100 requests/sec per IP, 50 connections per IP)

### Step 2: Verify Ingress Creation

```bash
kubectl get ingress -n eventpulse
```

**Expected output**:
```
NAME                   CLASS   HOSTS   ADDRESS       PORTS   AGE
eventpulse-ingress     nginx   *       10.244.x.x    80      10s
```

Detailed view:
```bash
kubectl describe ingress eventpulse-ingress -n eventpulse
```

**Expected output**:
```
Name:             eventpulse-ingress
Namespace:        eventpulse
Address:          10.244.x.x
Ingress Class:    nginx
Host:             *
Rules:
  Path  Backend         Service Port
  ----  -------         ------- ----
  /health                api-gateway:8080
  /events                api-gateway:8080
  /alerts                api-gateway:8080
  /alert                 api-gateway:8080
  /metrics               api-gateway:8080
```

### Step 3: Check NGINX Configuration

NGINX has reloaded its configuration:

```bash
kubectl logs -n ingress-nginx deployment/nginx-ingress-controller --tail=10
```

**Expected output** (look for):
```
I1118 15:30:10.345678       1 controller.go:1567] "msg"="NGINX reload triggered" "ingress"="eventpulse/eventpulse-ingress"
```

---

## Verification: Ingress Routing

### Test 1: Health Check (Local Testing)

For local/minikube testing, use port-forward:

```bash
# Terminal 1: Port-forward the NGINX service
kubectl port-forward -n ingress-nginx svc/nginx-ingress 8000:80
```

```bash
# Terminal 2: Test health endpoint
curl -v http://localhost:8000/health
```

**Expected output**:
```
HTTP/1.1 200 OK
Content-Type: application/json
...
{"status":"ok","service":"api-gateway","timestamp":"2026-06-18T21:00:00Z"}
```

### Test 2: Send Event via Ingress

```bash
curl -X POST http://localhost:8000/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "ingress_test",
    "event_type": "purchase",
    "amount": 50000
  }'
```

**Expected output**:
```json
{"message":"Event Published"}
```

### Test 3: Retrieve Alerts via Ingress

```bash
curl http://localhost:8000/alerts
```

**Expected output**:
```json
[
  {
    "id": 1,
    "user_id": "ingress_test",
    "risk_score": 90,
    "message": "HIGH RISK TRANSACTION DETECTED",
    "created_at": "2026-06-18T21:00:00Z"
  }
]
```

### Test 4: CORS Preflight Request

```bash
curl -v -X OPTIONS http://localhost:8000/events \
  -H "Origin: http://example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type"
```

**Expected output** (200 OK with CORS headers):
```
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

### Test 5: Rate Limiting

Send rapid requests to test rate limiting (100 requests/sec):

```bash
for i in {1..150}; do
  curl -s http://localhost:8000/health &
done
wait
```

**Expected**: Some requests may get 429 (Too Many Requests) if rate limit exceeded.

### Test 6: Metrics Endpoint

```bash
curl http://localhost:8000/metrics | grep eventpulse_ | head -5
```

**Expected output** (sample):
```
# HELP eventpulse_events_published_total Total events published.
# TYPE eventpulse_events_published_total counter
eventpulse_events_published_total{service="api-gateway"} 5
eventpulse_alerts_generated_total{service="alert-service"} 1
```

---

## Production Deployment

### For Cloud Environments (EKS, GKE, AKS)

```bash
# Deploy NGINX Ingress
kubectl apply -f k8s/ingress/nginx-ingress-deployment.yaml

# Get the external load balancer IP
kubectl get service -n ingress-nginx nginx-ingress

# Wait for EXTERNAL-IP to appear (may take 2-5 minutes)
# Then deploy EventPulse Ingress
kubectl apply -f k8s/ingress/eventpulse-ingress.yaml
```

**Result**: External IP will be the public-facing entry point.

### Enable TLS/HTTPS (Production)

1. **Create a certificate Secret**:
```bash
# Using self-signed certificate (for testing)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Create Kubernetes Secret
kubectl create secret tls eventpulse-tls \
  --cert=cert.pem \
  --key=key.pem \
  -n eventpulse
```

2. **Update eventpulse-ingress.yaml** to enable TLS:
```yaml
spec:
  tls:
  - hosts:
    - example.com
    secretName: eventpulse-tls

  rules:
  - host: example.com
    http:
      paths:
      - path: /events
        backend:
          service:
            name: api-gateway
            port:
              number: 8080
```

3. **Apply updated Ingress**:
```bash
kubectl apply -f k8s/ingress/eventpulse-ingress.yaml
```

---

## Troubleshooting

### Issue: EXTERNAL-IP stays <pending>

```bash
kubectl get service -n ingress-nginx nginx-ingress
```

**Causes & Solutions**:
1. **Local/Minikube**: Use port-forward instead
   ```bash
   kubectl port-forward -n ingress-nginx svc/nginx-ingress 80:80
   ```

2. **Cloud cluster**: Provider may take 2-5 minutes to provision. Check:
   ```bash
   kubectl describe service -n ingress-nginx nginx-ingress
   # Look for "Events:" section for errors
   ```

3. **On-premises cluster**: Requires MetalLB or similar. Install:
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/main/config/manifests/metallb-native.yaml
   ```

### Issue: Ingress shows no ADDRESS

```bash
kubectl describe ingress eventpulse-ingress -n eventpulse
```

**Causes**:
1. **NGINX controller not ready**: Check controller logs
   ```bash
   kubectl logs -n ingress-nginx deployment/nginx-ingress-controller
   ```

2. **IngressClass not found**: Verify `nginx` class exists
   ```bash
   kubectl get ingressclass
   ```

3. **Service not found**: Verify api-gateway service exists
   ```bash
   kubectl get service -n eventpulse api-gateway
   ```

### Issue: Requests Timeout (504 Gateway Timeout)

```bash
curl -v http://localhost:8000/events
```

**Causes & Solutions**:
1. **API Gateway pod not running**:
   ```bash
   kubectl get pods -n eventpulse -l app=api-gateway
   ```

2. **Service endpoint missing**:
   ```bash
   kubectl get endpoints -n eventpulse api-gateway
   ```

3. **NGINX backend unreachable**: Check NGINX logs
   ```bash
   kubectl logs -n ingress-nginx deployment/nginx-ingress-controller
   # Look for "upstream timed out"
   ```

4. **Network policy blocking traffic**: If using NetworkPolicies, verify rules allow ingress-nginx → eventpulse namespace

### Issue: 404 Not Found for Valid Paths

```bash
curl -v http://localhost:8000/events
HTTP/1.1 404 Not Found
```

**Causes**:
1. **Path mismatch**: Verify path in Ingress matches actual API endpoint
   ```bash
   kubectl describe ingress eventpulse-ingress -n eventpulse
   # Check "Path" column
   ```

2. **API Gateway not responding on /events**:
   ```bash
   kubectl exec -it -n eventpulse <pod-name> -- \
     curl -s http://localhost:8080/events
   ```

3. **Ingress not reloaded**: Force NGINX reload
   ```bash
   # Edit Ingress to add/remove annotation (trigger reload)
   kubectl edit ingress eventpulse-ingress -n eventpulse
   ```

### Issue: CORS Errors

```
Access to XMLHttpRequest at 'http://localhost:8000/events' from origin 
'http://example.com' has been blocked by CORS policy
```

**Solution**: Verify CORS annotations in Ingress:
```bash
kubectl get ingress eventpulse-ingress -n eventpulse -o yaml | grep cors
```

Should show:
```yaml
nginx.ingress.kubernetes.io/enable-cors: "true"
nginx.ingress.kubernetes.io/cors-allow-origin: "*"
nginx.ingress.kubernetes.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
```

If missing, reapply Ingress:
```bash
kubectl apply -f k8s/ingress/eventpulse-ingress.yaml
```

### Issue: Rate Limiting Too Aggressive

```
curl http://localhost:8000/events
HTTP/1.1 429 Too Many Requests
```

**Solution**: Adjust rate limit in nginx-ingress-deployment.yaml:
```yaml
data:
  limit-rps: "100"          # Requests per second
  limit-connections: "50"   # Connections per IP
```

Update values and reapply:
```bash
kubectl apply -f k8s/ingress/nginx-ingress-deployment.yaml
```

### Issue: NGINX Controller Logs Show Errors

```bash
kubectl logs -n ingress-nginx deployment/nginx-ingress-controller -f
```

**Common errors**:
- `upstream timed out` → Backend service unreachable
- `worker_connections are not enough` → Need more connections (increase in nginx-config)
- `invalid backend` → Service/port not found

**Debug**:
```bash
# Check NGINX configuration inside pod
kubectl exec -it -n ingress-nginx <pod-name> -- \
  cat /etc/nginx/nginx.conf | grep -A 10 "eventpulse"
```

---

## Advanced Configuration

### Custom NGINX Settings

Edit nginx-ingress-deployment.yaml ConfigMap:

```yaml
data:
  # Increase worker connections for high traffic
  worker-connections: "4096"

  # Disable access logs if high volume
  access-log: "off"

  # Enable compression
  gzip: "true"
  gzip-types: "text/plain text/css application/json application/javascript"
```

### Path Rewriting

If your backend expects different paths, add to eventpulse-ingress.yaml:

```yaml
annotations:
  nginx.ingress.kubernetes.io/rewrite-target: /api/$2

paths:
- path: /api(/|$)(.*)
  pathType: ImplementationSpecific
```

### URL Redirect (HTTP → HTTPS)

```yaml
annotations:
  nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
```

### Circuit Breaker

```yaml
annotations:
  nginx.ingress.kubernetes.io/upstream-fail-timeout: "30s"
  nginx.ingress.kubernetes.io/upstream-max-fails: "3"
```

---

## Monitoring NGINX Ingress

### Check Ingress Controller Metrics

```bash
kubectl port-forward -n ingress-nginx deployment/nginx-ingress-controller 10254:10254

# In another terminal
curl http://localhost:10254/metrics | grep nginx_ingress_controller
```

**Metrics to monitor**:
- `nginx_ingress_controller_requests_total` — Total requests
- `nginx_ingress_controller_request_duration_seconds` — Latency
- `nginx_ingress_controller_success` — Success rate

### View NGINX Access Logs

```bash
kubectl logs -n ingress-nginx deployment/nginx-ingress-controller -f | grep "GET /events"
```

### Ingress Events

```bash
kubectl get events -n eventpulse --sort-by='.lastTimestamp'
# Look for Ingress-related events
```

---

## Cleanup

### Remove EventPulse Ingress

```bash
kubectl delete ingress eventpulse-ingress -n eventpulse
```

### Remove NGINX Ingress Controller

```bash
kubectl delete -f k8s/ingress/nginx-ingress-deployment.yaml
```

**Note**: This deletes the entire ingress-nginx namespace

---

## Quick Reference

| Task | Command |
|------|---------|
| Install NGINX | `kubectl apply -f k8s/ingress/nginx-ingress-deployment.yaml` |
| Check controller | `kubectl get deployment -n ingress-nginx` |
| Apply EventPulse Ingress | `kubectl apply -f k8s/ingress/eventpulse-ingress.yaml` |
| Verify Ingress | `kubectl get ingress -n eventpulse` |
| Port-forward (local testing) | `kubectl port-forward -n ingress-nginx svc/nginx-ingress 80:80` |
| Test health | `curl http://localhost/health` |
| View NGINX logs | `kubectl logs -n ingress-nginx deployment/nginx-ingress-controller -f` |
| View Ingress events | `kubectl describe ingress eventpulse-ingress -n eventpulse` |
| Check rate limits | `curl -v http://localhost/events` (repeat 100+ times) |
| Edit Ingress | `kubectl edit ingress eventpulse-ingress -n eventpulse` |
| Delete Ingress | `kubectl delete ingress eventpulse-ingress -n eventpulse` |

---

## Summary

 NGINX Ingress Controller installed and configured  
 EventPulse Ingress routes all paths to API Gateway  
 CORS, rate limiting, and proxy settings enabled  
 Single entrypoint for all API endpoints  
 Ready for testing and production deployment  

**Next**: Phase 5 — Production Hardening & Monitoring