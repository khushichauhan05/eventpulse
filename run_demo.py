#!/usr/bin/env python3
"""
Simple demo launcher for EventPulse
Usage: python3 run_demo.py <duration> <tps> <fraud_percent>

Examples:
  python3 run_demo.py 15 10 5     # Light demo
  python3 run_demo.py 30 50 5     # Medium demo
  python3 run_demo.py 60 200 8    # Heavy demo
"""

import sys
from load_test import LoadTester

if __name__ == "__main__":
    if len(sys.argv) < 4:
        print("EventPulse Demo Launcher\n")
        print("Usage: python3 run_demo.py <duration> <tps> <fraud_percent>\n")
        print("Examples:")
        print("  python3 run_demo.py 15 10 5     # Light demo (15s, 10 TPS)")
        print("  python3 run_demo.py 30 50 5     # Medium demo (30s, 50 TPS)")
        print("  python3 run_demo.py 60 200 8    # Heavy demo (60s, 200 TPS)")
        print("  python3 run_demo.py 300 150 4   # Sustained (5 min, 150 TPS)")
        sys.exit(1)

    duration = int(sys.argv[1])
    tps = int(sys.argv[2])
    fraud_pct = int(sys.argv[3])

    print(f"\n Recording: {duration}s at {tps} TPS with {fraud_pct}% fraud\n")

    tester = LoadTester(
        endpoint="http://localhost:8080/events",
        tps=tps,
        duration=duration,
        fraud_pct=fraud_pct
    )
    tester.NUM_THREADS = min(10, max(2, tps // 20))
    tester.run()
