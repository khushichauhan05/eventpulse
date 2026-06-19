# EventPulse Horizontal Pod Autoscaling (HPA) Guide

Complete guide for deploying and testing Horizontal Pod Autoscaling on EventPulse services.

**Objective**: Automatically scale application services (2-10 replicas) based on CPU utilization to handle variable workloads.

---

## What is Horizontal Pod Autoscaling (HPA)?

HPA automatically adjusts the number of pod replicas based on resource metrics (CPU, memory).

**Without HPA**:
```
Fixed 2 replicas → High load → Slow response, dropped requests
```

**With HPA**:
```
2 replicas → High CPU (>70%) → Scale to 3, 4, 5... 10 replicas → Lower latency
10 replicas → Low CPU (<50%) → Scale down to 2 replicas → Save resources
```

**Benefits**:
- Automatic response to traffic spikes
- Resource efficiency (scale down during low traffic)
- No manual intervention needed
- Better user experience (faster responses)

---

## Prerequisites

### 1. Kubernetes Metrics Server

HPA requires the Metrics Server to collect pod resource metrics. Check if installed:

```bash
kubectl get deployment -n kube-system metrics-server
```

**If not found, install**:

```bash
# For most Kubernetes distributions (minikube, EKS, GKE, AKS)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Verify installation
kubectl get deployment -n kube-system metrics-server
kubectl get apiservice v1beta1.metrics.k8s.io -o yaml
```

### 2. Resource Requests in Deployments

HPA uses CPU **requests** (not limits) as the baseline for percentage calculations.

**Current deployments have**:
```yaml
resources:
  requests:
    cpu: 100m      # HPA will calculate 70% of this (70m)
    memory: 128Mi
```

This is already configured 

### 3. Multiple Replicas Enabled

Deployments must support multiple replicas. Current state:
```yaml
spec:
  replicas: 2  # Can scale from 2-10 
```

---

## Phase 5: Deploy Horizontal Pod Autoscalers

### Step 1: Deploy HPA for API Gateway

```bash
kubectl apply -f k8s/autoscaling/api-gateway-hpa.yaml
```

**Verify creation**:
```bash
kubectl get hpa -n eventpulse api-gateway-hpa
kubectl describe hpa -n eventpulse api-gateway-hpa
```

**Expected output**:
```
NAME                   REFERENCE                    TARGETS        MINPODS   MAXPODS   REPLICAS   AGE
api-gateway-hpa        Deployment/api-gateway       <unknown>/70%   2         10        2          10s
```

**Note**: `TARGETS` shows `<unknown>` initially (metrics collecting, will show data in 30-60 seconds)

### Step 2: Deploy HPA for Analytics Service

```bash
kubectl apply -f k8s/autoscaling/analytics-service-hpa.yaml
```

**Verify**:
```bash
kubectl get hpa -n eventpulse analytics-service-hpa
```

### Step 3: Deploy HPA for Alert Service

```bash
kubectl apply -f k8s/autoscaling/alert-service-hpa.yaml
```

**Verify**:
```bash
kubectl get hpa -n eventpulse alert-service-hpa
```

### Step 4: Verify All HPAs

```bash
kubectl get hpa -n eventpulse
```

**Expected output**:
```
NAME                      REFERENCE                        TARGETS         MINPODS   MAXPODS   REPLICAS   AGE
api-gateway-hpa           Deployment/api-gateway           45%/70%         2         10        2          1m
analytics-service-hpa     Deployment/analytics-service     32%/70%         2         10        2          1m
alert-service-hpa         Deployment/alert-service         28%/70%         2         10        2          1m
```

**Interpretation**:
- `TARGETS`: Current CPU usage / Target threshold
- `MINPODS`: Minimum replicas (never scale below 2)
- `MAXPODS`: Maximum replicas (never scale above 10)
- `REPLICAS`: Current number of running pods

---

## Monitoring: Watch HPA Status

### Watch HPA in Real-Time

```bash
kubectl get hpa -n eventpulse -w
```

**Output** (updates as scaling happens):
```
NAME                      REFERENCE                        TARGETS         MINPODS   MAXPODS   REPLICAS   AGE
api-gateway-hpa           Deployment/api-gateway           45%/70%         2         10        2          1m
analytics-service-hpa     Deployment/analytics-service     32%/70%         2         10        2          1m
alert-service-hpa         Deployment/alert-service         28%/70%         2         10        2          1m
api-gateway-hpa           Deployment/api-gateway           85%/70%         2         10        2          2m  # CPU increased!
api-gateway-hpa           Deployment/api-gateway           85%/70%         2         10        3          2m  # Scaled to 3 pods
```

### Check HPA Events

```bash
kubectl describe hpa -n eventpulse api-gateway-hpa
```

**Look for "Events" section**:
```
Events:
  Type     Reason                   Age    From                       Message
  ----     ------                   ----   ----                       -------
  Normal   SuccessfulRescale        2m     horizontal-pod-autoscaler  New size: 3; reason: cpu resource utilization (85%) above target (70%)
  Normal   SuccessfulRescale        1m     horizontal-pod-autoscaler  New size: 4; reason: cpu resource utilization (72%) above target (70%)
  Normal   SuccessfulRescale        30s    horizontal-pod-autoscaler  New size: 3; reason: cpu resource utilization (45%) below target (70%)
```

### Check Deployment Replicas

```bash
kubectl get deployment -n eventpulse
```

**Current state**:
```
NAME                  READY   UP-TO-DATE   AVAILABLE   AGE
api-gateway           3/3     3            3           2m
analytics-service     2/2     2            2           2m
alert-service         2/2     2            2           2m
```

**Interpretation**: If HPA scaled to 3 replicas, you'll see 3 pods running.

---

## Load Testing: Force Scaling

### Test 1: Generate Load on API Gateway

Use Apache Bench (`ab`) or `hey` to generate sustained load:

**Option A: Using `ab` (Apache Bench)**

```bash
# Terminal 1: Port-forward API Gateway
kubectl port-forward -n ingress-nginx svc/nginx-ingress 8000:80

# Terminal 2: Generate load (100 concurrent requests, 10,000 total)
ab -n 10000 -c 100 http://localhost:8000/health
```

**Expected output**:
```
This is ApacheBench, Version 2.3
Benchmarking localhost (be patient)
Completed 1000 requests
Completed 2000 requests
...
Finished 10000 requests

Server Software:        nginx
Requests per second:    1,234.56 [#/sec]
Time per request:       81.09 [ms]
```

**Monitor HPA during load**:

```bash
# Terminal 3: Watch HPA scaling
kubectl get hpa -n eventpulse -w
```

**Expected behavior**:
```
api-gateway-hpa   Deployment/api-gateway   45%/70%   2   10   2   1m
api-gateway-hpa   Deployment/api-gateway   95%/70%   2   10   2   2m  # CPU spiked!
api-gateway-hpa   Deployment/api-gateway   95%/70%   2   10   3   2m  # Scaled to 3
api-gateway-hpa   Deployment/api-gateway   72%/70%   2   10   4   3m  # Scaled to 4
api-gateway-hpa   Deployment/api-gateway   68%/70%   2   10   4   4m  # Stabilized
```

**Option B: Using `hey` (Modern load testing)**

```bash
# Install hey (if not installed)
# go install github.com/rakyll/hey@latest

# Generate load with 100 concurrent requests for 60 seconds
hey -n 10000 -c 100 -z 60s http://localhost:8000/health
```

### Test 2: Generate Load on Analytics Service (Kafka Consumer)

Send many events to trigger processing:

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

**Expected**:
- Analytics Service CPU increases as it processes events
- HPA detects high CPU (>70%)
- Replicas scale from 2 → 3 → 4 (depending on load)

### Test 3: Generate Load on Alert Service (DB + Kafka)

Same as Test 2, but alert-service processes risk-scored events:

```bash
# Continued from Test 2, wait 5 seconds for events to flow through pipeline
sleep 5

# Monitor alert-service HPA
kubectl get hpa -n eventpulse alert-service-hpa -w
```

**Expected**:
- Alert Service writes to PostgreSQL (high CPU)
- HPA scales up to handle database write load
- Number of replicas depends on event throughput

---

## Scaling Verification

### Verify Upscaling (2 → More Replicas)

**Procedure**:
1. Generate load (any of the tests above)
2. Watch HPA and deployment replicas
3. Confirm replicas increase

```bash
# Terminal 1: Generate load
ab -n 10000 -c 100 http://localhost:8000/health

# Terminal 2: Watch scaling
watch kubectl get deployment,hpa -n eventpulse

# Expected: replicas go from 2 → 3 → 4 → 5... (depending on load)
```

### Verify Downscaling (Many → 2 Replicas)

**Procedure**:
1. Stop load (Ctrl+C in load generation terminal)
2. Wait 5-10 minutes for stabilization
3. Confirm replicas decrease back to 2

```bash
# After stopping load generation, wait and watch
watch kubectl get deployment,hpa -n eventpulse

# Timeline:
# t=0min: Stop load, still 5 pods (CPU dropping)
# t=1min: CPU < 70%, scaling begins
# t=5min: Wait for stabilization window (300 seconds)
# t=5min+: Replicas gradually reduce to 2 pods
```

### Monitor CPU Metrics

```bash
# View current CPU usage of all pods
kubectl top pods -n eventpulse -l tier=services
```

**Expected output**:
```
NAME                              CPU(cores)   MEMORY(Mi)
api-gateway-xxxxx                 45m          80Mi
api-gateway-yyyyy                 38m          75Mi
analytics-service-aaaaa           62m          95Mi
analytics-service-bbbbb           41m          88Mi
alert-service-ccccc               25m          102Mi
alert-service-ddddd               19m          99Mi
```

**Interpretation**:
- If any pod > 70m (70% of 100m request), HPA will scale up
- Average across all pods is used for scaling decision

### Monitor Deployment Status

```bash
kubectl get deployment -n eventpulse -w
```

**Watch for**:
- `DESIRED` = target replicas (set by HPA)
- `CURRENT` = running replicas (ramping up/down)
- `READY` = healthy replicas
- When all three match, scaling is complete

---

## Advanced Monitoring

### View HPA Scaling History

```bash
kubectl get events -n eventpulse --sort-by='.lastTimestamp' | grep -i hpa
```

**Output** (shows all scaling events):
```
eventpulse   Normal   SuccessfulRescale      4m    horizontal-pod-autoscaler   New size: 3; reason: cpu resource utilization (85%) above target (70%)
eventpulse   Normal   SuccessfulRescale      3m    horizontal-pod-autoscaler   New size: 4; reason: cpu resource utilization (72%) above target (70%)
eventpulse   Normal   SuccessfulRescale      2m    horizontal-pod-autoscaler   New size: 3; reason: cpu resource utilization (45%) below target (70%)
```

### Monitor HPA Decisions with kubectl explain

```bash
kubectl describe hpa api-gateway-hpa -n eventpulse
```

**Look for "Conditions"**:
```
Conditions:
  Type            Status  Reason            Message
  ----            ------  ------            -------
  AbleToScale     True    SucceededGetResourceMetric  successfully obtained the metric for the pod
  ScalingActive   True    ValidMetricsFound Available metrics found
  ScalingLimited  False   DesiredWithinRange the desired replica count is within acceptable range
```

### Check Metrics Server Health

```bash
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/eventpulse/pods | jq '.items[] | {name: .metadata.name, cpu: .containers[0].usage.cpu, memory: .containers[0].usage.memory}'
```

**This confirms metrics are being collected**

---

## Scaling Configuration Details

### Current HPA Settings

All three services use identical scaling configuration:

```yaml
minReplicas: 2              # Never go below 2 pods
maxReplicas: 10             # Never go above 10 pods
targetCPUUtilization: 70%   # Scale up when avg CPU > 70% of request (70m)
                            # Scale down when avg CPU < 50% of request (50m)

scaleUp:
  periodSeconds: 30         # Check every 30 seconds
  addPodsPerPeriod: 1       # Add 1 pod per period (gradual)

scaleDown:
  stabilization: 300s       # Wait 5 minutes before scaling down
  removePodsPerPeriod: 1    # Remove 1 pod per period (conservative)
```

### Tuning Scaling Behavior

To make scaling more aggressive (faster):

```yaml
scaleUp:
  periodSeconds: 15         # Check more frequently
  addPodsPerPeriod: 2       # Add 2 pods at once

scaleDown:
  stabilization: 60s        # Scale down faster
```

To make scaling more conservative (fewer changes):

```yaml
scaleUp:
  periodSeconds: 60         # Check less frequently
  addPodsPerPeriod: 1       # Add 1 pod at a time

scaleDown:
  stabilization: 600s       # Wait 10 minutes before scaling down
```

---

## Troubleshooting HPA

### Issue: HPA shows `<unknown>` for TARGETS

```bash
kubectl describe hpa api-gateway-hpa -n eventpulse
```

**Expected message**:
```
Warning  FailedGetResourceMetric  resource metrics not yet available
```

**Solution**: Wait 1-2 minutes for metrics to collect. Then check:

```bash
# Verify Metrics Server is running
kubectl get deployment -n kube-system metrics-server

# Verify metrics are available
kubectl get --raw /apis/metrics.k8s.io/v1beta1/namespaces/eventpulse/pods | jq .
```

### Issue: Pods don't scale even under high CPU

```bash
# Check HPA status
kubectl describe hpa api-gateway-hpa -n eventpulse

# Look for conditions and events
# Common causes:
# 1. Metrics Server not running
# 2. CPU requests not defined in deployment
# 3. Scale target ref (Deployment name) incorrect
```

**Solution**:
1. Verify Metrics Server: `kubectl get deployment -n kube-system metrics-server`
2. Verify CPU requests: `kubectl get deployment api-gateway -n eventpulse -o yaml | grep -A 5 resources`
3. Verify scale target: `kubectl get hpa api-gateway-hpa -n eventpulse -o yaml | grep scaleTargetRef`

### Issue: Excessive scaling (pods constantly added/removed)

**Cause**: Target CPU percentage too close to current usage.

**Solution**: Increase target percentage:
```yaml
target:
  type: Utilization
  averageUtilization: 80  # Instead of 70
```

Or increase stabilization window:
```yaml
scaleDown:
  stabilizationWindowSeconds: 600  # Wait 10 minutes before scaling
```

### Issue: Database connection pool exhausted

**Cause**: Alert Service scales too much, each pod opens PostgreSQL connections.

**Solution**:
1. Check current connections:
```bash
kubectl exec -it -n eventpulse <postgres-pod> -- \
  psql -U admin -d eventpulse \
  -c "SELECT count(*) FROM pg_stat_activity;"
```

2. Increase PostgreSQL max_connections:
```bash
# In postgres deployment, add to environment:
POSTGRES_INITDB_ARGS: "-c max_connections=500"
```

3. Lower HPA max replicas:
```yaml
maxReplicas: 5  # Instead of 10
```

---

## Cost Optimization

### Recommended Settings for Cost Control

```yaml
minReplicas: 2
maxReplicas: 5      # Lower max to reduce cloud provider charges
targetCPUUtilization: 80  # Higher target = fewer replicas needed

scaleDown:
  stabilizationWindowSeconds: 600  # Take time before scaling down
```

### Monitoring Replica Count

```bash
# Track replica count over time
kubectl get hpa -n eventpulse -o jsonpath='{.items[*].status.currentReplicas}' | xargs
# Output: 2 2 2 (3 services, 2 replicas each)

# After high load
kubectl get hpa -n eventpulse -o jsonpath='{.items[*].status.currentReplicas}' | xargs
# Output: 5 6 4 (scaled up)
```

---

## Production Checklist

- [ ] Metrics Server installed and running
- [ ] All deployments have CPU requests defined
- [ ] HPAs deployed for all services (api-gateway, analytics, alert)
- [ ] Load testing performed to verify scaling
- [ ] Scaling behavior monitored and tuned
- [ ] Downscaling verified after load reduction
- [ ] Database connections verified under max replicas
- [ ] Cost implications understood (max replicas × resource limits)
- [ ] Monitoring dashboards show replica counts
- [ ] Alert rules for scaling events (optional)

---

## Quick Reference

| Task | Command |
|------|---------|
| Deploy all HPAs | `kubectl apply -f k8s/autoscaling/*.yaml` |
| View HPA status | `kubectl get hpa -n eventpulse` |
| Watch HPA scaling | `kubectl get hpa -n eventpulse -w` |
| View HPA details | `kubectl describe hpa -n eventpulse <name>` |
| View scaling events | `kubectl get events -n eventpulse --sort-by='.lastTimestamp'` |
| View pod CPU usage | `kubectl top pods -n eventpulse -l tier=services` |
| View deployment replicas | `kubectl get deployment -n eventpulse` |
| Generate load (ab) | `ab -n 10000 -c 100 http://localhost/health` |
| Generate load (hey) | `hey -n 10000 -c 100 http://localhost/health` |
| Edit HPA settings | `kubectl edit hpa -n eventpulse <name>` |
| Delete HPA | `kubectl delete hpa -n eventpulse <name>` |

---

## Summary

 HPAs deployed for all 3 services (api-gateway, analytics-service, alert-service)  
 Scaling range: 2-10 replicas  
 Scaling metric: CPU utilization (70% target)  
 Verified with load testing procedures  
 Scaling events and metrics documented  
 Production-ready with monitoring  

**Next**: Phase 6 — Prometheus & Grafana (metrics visualization & alerting)