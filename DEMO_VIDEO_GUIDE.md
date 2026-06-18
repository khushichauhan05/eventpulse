# EventPulse Demo Video - Complete Guide

## 📹 Quick Start

### Step 1: Install Python Dependencies

```bash
pip install requests
```

### Step 2: Setup Kubernetes Port Forwarding

Open **PowerShell** or **Terminal** and run these commands in separate windows:

```bash
# Window 1: API Gateway
kubectl port-forward -n eventpulse svc/api-gateway 8080:8080

# Window 2: Grafana
kubectl port-forward -n eventpulse svc/grafana 3000:3000

# Window 3: Prometheus
kubectl port-forward -n eventpulse svc/prometheus 9090:9090
```

### Step 3: Verify Everything Works

```bash
# Test API
curl http://localhost:8080/health

# Should return:
# {"status":"healthy",...}
```

---

## 🎬 Recording Sessions

### Session 1: Light Demo (15 seconds)

Perfect for short LinkedIn/Twitter clips

```bash
python3 run_profile.py light
```

**What you'll see:**
- ~150 transactions sent
- 7-8 fraud alerts generated
- Latency: 40-60ms
- Success rate: 100%

**Recording tips:**
- Show the output in terminal
- Open Grafana dashboard (http://localhost:3000)
- Open Alerts API (http://localhost:8080/alerts)

---

### Session 2: Medium Demo (30 seconds)

Perfect for 30-second YouTube video segment

```bash
python3 run_profile.py medium
```

**What you'll see:**
- ~1,500 transactions sent
- 75 fraud alerts generated
- Latency: P99 < 150ms
- Real-time metrics in Grafana

**Recording flow:**
1. Start load test
2. Show transactions flowing in terminal
3. Switch to Grafana dashboard
4. Show metrics climbing
5. End with Alerts API

---

### Session 3: Heavy Demo (60 seconds)

Perfect for auto-scaling demonstration

```bash
python3 run_profile.py heavy
```

**What you'll see:**
- 12,000+ transactions sent
- 960+ fraud alerts
- Kubernetes pods scaling 2→5→10 replicas
- Latency stays under 200ms despite heavy load
- CPU/Memory usage climbing

**Recording steps:**
1. Open terminal with load test
2. Open Kubernetes pods view
3. Open Grafana dashboard
4. Watch pods scale up in real-time
5. Show metrics responding to load

---

### Session 4: Fraud Pattern Demos (1-2 minutes each)

Specific fraud patterns in action

```bash
# Geographic Anomaly
python3 fraud_pattern_demo.py geographic_anomaly

# Structuring (layering)
python3 fraud_pattern_demo.py structuring

# Velocity Abuse
python3 fraud_pattern_demo.py velocity_abuse

# Night Spike
python3 fraud_pattern_demo.py night_spike
```

**Recording approach:**
- Show each transaction being sent
- Show fraud alerts being generated
- Explain the fraud pattern
- Show risk score calculation

---

## 📊 Complete Video Sequence (5-6 minutes)

### 0:00-0:30: Architecture Overview
- Show diagram or screenshot
- Explain components
- Narrate: "EventPulse is a fraud detection system..."

### 0:30-1:30: Normal Transaction Demo
```bash
# Run light profile in background
python3 run_profile.py light &
```
- Show normal transactions
- Show low-risk scores
- Show no alerts generated

### 1:30-2:30: Fraud Pattern Demo
```bash
# Run geographic anomaly
python3 fraud_pattern_demo.py geographic_anomaly
```
- Show high-risk transactions
- Show fraud alerts being generated
- Explain risk scoring

### 2:30-4:00: Heavy Load & Scaling Demo
```bash
# Run heavy profile
python3 run_profile.py heavy
```
- Show 200 TPS load
- Watch Kubernetes scale
- Show Grafana metrics
- Show pod count increasing

### 4:00-4:30: Monitoring Dashboard
- Show Grafana with all metrics
- Show Prometheus queries
- Explain observability

### 4:30-5:00: Summary
- Show architecture again
- Recap features
- Call to action (GitHub link)

---

## 🖥️ Desktop Setup for Recording

### Terminal Layout
```
┌─────────────────────────────────────────────┐
│          Terminal 1: Load Test              │
│  $ python3 run_profile.py medium            │
│                                             │
│  [T0] ✅ Amount: $45.50 Latency: 42.3ms   │
│  [T1] ✅ 🔴 Amount: $95000 Latency: 45.1ms│
│  [T2] ✅ Amount: $120.00 Latency: 41.8ms  │
└─────────────────────────────────────────────┘
```

### Browser Tabs
```
Tab 1: http://localhost:3000
   → Grafana Dashboard (shows live metrics)

Tab 2: http://localhost:9090
   → Prometheus (advanced queries)

Tab 3: http://localhost:8080/alerts
   → Alerts API (shows fraud alerts)

Tab 4: Kubernetes Dashboard
   → Watch pods scaling
```

---

## 📝 Narration Script

### When Running Light Profile:

```
"Here we're sending low-risk transactions to EventPulse.
Each transaction gets a risk score. These are scoring 1-3 out of 10.
Notice the response time: 42 milliseconds. The system responds instantly.
No fraud alerts are generated because these transactions are clean."
```

### When Running Heavy Profile:

```
"Now let's see the system under real load.
We're sending 200 transactions per second.
Watch the Kubernetes dashboard - the system is automatically scaling.
Started with 2 replicas, now at 5, continuing to 10 as load increases.

Meanwhile, the fraud detection is working perfectly.
We're generating 12+ alerts per minute for high-risk transactions.
The latency stays under 150 milliseconds despite the load.
Prometheus shows all services healthy and responding."
```

### When Running Fraud Demo:

```
"Let me demonstrate a specific fraud pattern.
This is a geographic anomaly - a transaction in the US followed immediately
by a large international wire transfer to Nigeria.
Impossible for the same user in seconds.
EventPulse detects this instantly.
Risk score: 9.2 out of 10.
Alert generated and stored in PostgreSQL."
```

---

## 🔧 Troubleshooting

### Error: Cannot connect to API Gateway

```bash
# Make sure port forwarding is running
kubectl port-forward -n eventpulse svc/api-gateway 8080:8080

# Test connectivity
curl http://localhost:8080/health
```

### Error: Slow response times

- Reduce TPS in profiles
- Check Kubernetes pod status: `kubectl get pods -n eventpulse`
- Check CPU/Memory: `kubectl top pods -n eventpulse`

### No fraud alerts being generated

- Wait 5-10 seconds for processing
- Increase FRAUD_PERCENTAGE in load_test.py
- Check Alert Service logs: `kubectl logs -f alert-service-xxx -n eventpulse`

### Terminal too slow

- Reduce verbosity or redirect to file
- Use lighter profile (light instead of heavy)
- Run on faster machine or reduce TPS

---

## 🎥 Recording Software Recommendations

### Windows
- **OBS Studio** (free): Best option, professional results
- **ScreenFlow**: Mac only
- **Camtasia**: Paid, excellent editing

### All Platforms
- **OBS Studio**: Free, open-source, professional
- **SimpleScreenRecorder**: Linux

### Settings
- Resolution: 1920x1080
- Frame Rate: 60 FPS
- Zoom: 125-150% (for readability)
- Bitrate: 10-20 Mbps
- Audio: Microphone + System Audio

---

## 📊 Expected Results by Profile

| Profile | TPS | Duration | Alerts | P99 Latency | Pods |
|---------|-----|----------|--------|------------|------|
| light | 10 | 15s | 8-10 | 45ms | 2 |
| medium | 50 | 30s | 75-100 | 95ms | 3-4 |
| heavy | 200 | 60s | 960-1200 | 180ms | 6-10 |
| spike | 500 | 45s | 2250-2500 | 250ms | 10 |
| sustained | 150 | 300s | 1800-2400 | 150ms | 5-8 |

---

## 🚀 Pro Tips

1. **Run a test first**: Run the profile once before recording to warm up the system
2. **Clear old alerts**: `kubectl exec -it postgres-0 -n eventpulse -- psql -U postgres -d eventpulse -c "DELETE FROM alerts;"`
3. **Monitor in background**: Keep Grafana visible while running load tests
4. **Record in sections**: Don't try to do entire video in one take
5. **Use keyboard shortcuts**: Zoom terminal for better visibility during recording
6. **Test audio**: Record a few seconds of test narration before full recording
7. **Have backups**: Run each profile 2-3 times, pick the best run for final video

---

## 📚 File Reference

- `load_test.py` - Main load testing engine
- `load_profiles.py` - Predefined load scenarios
- `run_profile.py` - Easy profile launcher
- `fraud_pattern_demo.py` - Specific fraud patterns
- `DEMO_VIDEO_GUIDE.md` - This file

---

## ✅ Pre-Recording Checklist

- [ ] Python 3.8+ installed
- [ ] `pip install requests` completed
- [ ] Kubernetes cluster running
- [ ] Port forwarding set up (3 windows)
- [ ] API health check passes
- [ ] Browser tabs open (Grafana, Prometheus, Alerts)
- [ ] Terminal theme set to dark
- [ ] Screen resolution 1920x1080
- [ ] Zoom at 125-150%
- [ ] Microphone tested
- [ ] OBS/ScreenFlow configured
- [ ] Test recording made and verified

---

## 🎬 Let's Make This Video!

You're ready to create a professional EventPulse demo video.

Start with the light profile for a quick test:

```bash
python3 run_profile.py light
```

Then move to heavier profiles as you get comfortable with the flow.

**Good luck! 🚀**
