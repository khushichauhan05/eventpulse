@echo off
REM EventPulse Demo Setup Script for Windows
REM This script sets up everything needed for the demo video

echo.
echo ╔════════════════════════════════════════════════════════════════╗
echo ║                                                                ║
echo ║            EventPulse Demo Setup Script                        ║
echo ║                                                                ║
echo ╚════════════════════════════════════════════════════════════════╝
echo.

REM Check if Python is installed
python --version >nul 2>&1
if errorlevel 1 (
    echo ❌ ERROR: Python is not installed or not in PATH
    echo Please install Python 3.8+ from https://www.python.org
    pause
    exit /b 1
)

echo ✅ Python is installed
python --version
echo.

REM Install requests package
echo Installing required packages...
pip install requests -q
if errorlevel 1 (
    echo ❌ ERROR: Failed to install requests
    pause
    exit /b 1
)
echo ✅ Requests package installed
echo.

REM Check if kubectl is available
kubectl version --client >nul 2>&1
if errorlevel 1 (
    echo ❌ ERROR: kubectl is not installed or not in PATH
    echo Please install kubectl
    pause
    exit /b 1
)

echo ✅ kubectl is installed
kubectl version --client
echo.

REM Check API connectivity
echo Checking API Gateway connectivity...
timeout /t 2 /nobreak >nul

for /f %%i in ('curl -s -o /dev/null -w "%%{http_code}" http://localhost:8080/health 2^>nul') do set HTTP_CODE=%%i

if "%HTTP_CODE%"=="200" (
    echo ✅ API Gateway is running on http://localhost:8080
) else (
    echo ❌ ERROR: Cannot connect to API Gateway
    echo Make sure port forwarding is running:
    echo.
    echo kubectl port-forward -n eventpulse svc/api-gateway 8080:8080
    echo.
    pause
    exit /b 1
)
echo.

REM Show available profiles
echo 📋 Available Load Test Profiles:
echo.
echo   1. light    - 15 seconds, light load (good for short clips)
echo   2. medium   - 30 seconds, medium load (30-second YouTube)
echo   3. heavy    - 60 seconds, heavy load (auto-scaling demo)
echo   4. spike    - 45 seconds, traffic spike (stress test)
echo   5. sustained - 300 seconds, production load (5 minutes)
echo.

REM Display usage
echo 📝 Usage Examples:
echo.
echo   For light load:
echo   python3 run_profile.py light
echo.
echo   For fraud pattern demo:
echo   python3 fraud_pattern_demo.py geographic_anomaly
echo.
echo   Available fraud patterns:
echo   - geographic_anomaly
echo   - structuring
echo   - velocity_abuse
echo   - night_spike
echo.

REM Display instructions
echo ✅ Setup Complete!
echo.
echo Next steps:
echo   1. Open http://localhost:3000 in browser (Grafana)
echo   2. Open http://localhost:9090 in browser (Prometheus)
echo   3. Open PowerShell and run: python3 run_profile.py light
echo   4. Watch the fraud detection in action!
echo.
echo For detailed guide, see: DEMO_VIDEO_GUIDE.md
echo.

pause
