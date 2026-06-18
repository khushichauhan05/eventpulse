#!/bin/bash
# EventPulse Demo Setup Script for Linux/Mac

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                                                                ║"
echo "║            EventPulse Demo Setup Script                        ║"
echo "║                                                                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo "❌ ERROR: Python 3 is not installed"
    echo "Please install Python 3.8+ from https://www.python.org"
    exit 1
fi

echo "✅ Python is installed"
python3 --version
echo ""

# Install requests package
echo "Installing required packages..."
pip3 install requests -q
if [ $? -ne 0 ]; then
    echo "❌ ERROR: Failed to install requests"
    exit 1
fi
echo "✅ Requests package installed"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ ERROR: kubectl is not installed"
    echo "Please install kubectl"
    exit 1
fi

echo "✅ kubectl is installed"
kubectl version --client
echo ""

# Check API connectivity
echo "Checking API Gateway connectivity..."
sleep 2

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)

if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ API Gateway is running on http://localhost:8080"
else
    echo "❌ ERROR: Cannot connect to API Gateway"
    echo "Make sure port forwarding is running:"
    echo ""
    echo "kubectl port-forward -n eventpulse svc/api-gateway 8080:8080"
    echo ""
    exit 1
fi
echo ""

# Show available profiles
echo "📋 Available Load Test Profiles:"
echo ""
echo "   1. light    - 15 seconds, light load (good for short clips)"
echo "   2. medium   - 30 seconds, medium load (30-second YouTube)"
echo "   3. heavy    - 60 seconds, heavy load (auto-scaling demo)"
echo "   4. spike    - 45 seconds, traffic spike (stress test)"
echo "   5. sustained - 300 seconds, production load (5 minutes)"
echo ""

# Display usage
echo "📝 Usage Examples:"
echo ""
echo "   For light load:"
echo "   python3 run_profile.py light"
echo ""
echo "   For fraud pattern demo:"
echo "   python3 fraud_pattern_demo.py geographic_anomaly"
echo ""
echo "   Available fraud patterns:"
echo "   - geographic_anomaly"
echo "   - structuring"
echo "   - velocity_abuse"
echo "   - night_spike"
echo ""

# Display instructions
echo "✅ Setup Complete!"
echo ""
echo "Next steps:"
echo "   1. Open http://localhost:3000 in browser (Grafana)"
echo "   2. Open http://localhost:9090 in browser (Prometheus)"
echo "   3. Open terminal and run: python3 run_profile.py light"
echo "   4. Watch the fraud detection in action!"
echo ""
echo "For detailed guide, see: DEMO_VIDEO_GUIDE.md"
echo ""
