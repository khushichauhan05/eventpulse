# EventPulse Load Test - Quick Start

## ⚡ 2-Minute Setup

### 1. Install Python Package
```bash
pip install requests
```

### 2. Start Port Forwarding (3 separate terminals)
```bash
# Terminal 1
kubectl port-forward -n eventpulse svc/api-gateway 8080:8080

# Terminal 2
kubectl port-forward -n eventpulse svc/grafana 3000:3000

# Terminal 3
kubectl port-forward -n eventpulse svc/prometheus 9090:9090
```

### 3. Verify Setup
```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy",...}
```

---

## 🚀 Run Load Tests

### Light Load (15 seconds)
```bash
python3 run_profile.py light
```
**Use for:** Short video clips, testing

### Medium Load (30 seconds)
```bash
python3 run_profile.py medium
```
**Use for:** 30-second YouTube videos

### Heavy Load (60 seconds)
```bash
python3 run_profile.py heavy
```
**Use for:** Auto-scaling demo, full system test

### Spike Load (45 seconds)
```bash
python3 run_profile.py spike
```
**Use for:** Stress testing, burst scenarios

### Sustained Load (5 minutes)
```bash
python3 run_profile.py sustained
```
**Use for:** Production behavior demo

---

## 🔴 Fraud Pattern Demos

### Geographic Anomaly
```bash
python3 fraud_pattern_demo.py geographic_anomaly
```
Shows: Transaction in US → Large wire to Nigeria

### Structuring
```bash
python3 fraud_pattern_demo.py structuring
```
Shows: Multiple small transactions to evade detection

### Velocity Abuse
```bash
python3 fraud_pattern_demo.py velocity_abuse
```
Shows: Multiple transactions in short timeframe

### Night Spike
```bash
python3 fraud_pattern_demo.py night_spike
```
Shows: High-value transaction at 3 AM

---

## 📊 Monitor While Testing

### Grafana Dashboard
```
http://localhost:3000
username: admin
password: admin
```

### Prometheus Queries
```
http://localhost:9090
Query: rate(api_requests_total[1m])
```

### Alerts API
```bash
curl http://localhost:8080/alerts | jq .
```

### Kubernetes Pods
```bash
kubectl get pods -n eventpulse -w
```

---

## 📹 Video Recording Tips

### For 15-Second Clip
1. Run: `python3 run_profile.py light`
2. Show terminal output
3. Open Grafana dashboard
4. Show alerts being generated

### For 30-Second Clip
1. Run: `python3 run_profile.py medium`
2. Alternate between terminal, Grafana, Prometheus
3. Show both normal and fraud transactions
4. Show metrics climbing

### For 60-Second Scaling Demo
1. Open Kubernetes pods view
2. Run: `python3 run_profile.py heavy`
3. Watch pods scale 2→5→10
4. Show Grafana metrics responding
5. Show latency staying under 200ms

---

## ✅ Expected Results

### Light Profile (15s)
- Transactions: ~150
- Fraud Alerts: 8-10
- Success Rate: 100%
- P99 Latency: 45ms

### Medium Profile (30s)
- Transactions: ~1,500
- Fraud Alerts: 75-100
- Success Rate: 100%
- P99 Latency: 95ms

### Heavy Profile (60s)
- Transactions: ~12,000
- Fraud Alerts: 960+
- Success Rate: 99.9%
- P99 Latency: 180ms
- Pods: Scales to 10

---

## 🔧 Troubleshooting

### Error: Cannot connect to API Gateway
```bash
# Check if port forwarding is running
kubectl port-forward -n eventpulse svc/api-gateway 8080:8080
```

### No fraud alerts?
```bash
# Check Alert Service logs
kubectl logs -f alert-service-xxx -n eventpulse
```

### Slow performance?
- Reduce TPS (edit load_test.py)
- Check pod status: `kubectl top pods -n eventpulse`
- Use lighter profile

### Clear old alerts
```bash
kubectl exec -it postgres-0 -n eventpulse -- \
  psql -U postgres -d eventpulse -c "DELETE FROM alerts;"
```

---

## 📚 Full Documentation

See `DEMO_VIDEO_GUIDE.md` for complete video recording guide

---

## 🎯 One-Command Demo

```bash
python3 run_profile.py medium
```

That's it! Watch fraud detection at scale. 🚀
