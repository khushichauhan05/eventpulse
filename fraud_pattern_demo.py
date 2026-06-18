#!/usr/bin/env python3
"""
Demonstrate specific fraud patterns
Usage: python3 fraud_pattern_demo.py <pattern>
"""

import requests
import json
import time
import sys

PATTERNS = {
    "structuring": {
        "description": "Multiple small transactions to evade detection",
        "transactions": [
            {"amount": 9500, "merchant": "bank", "description": "Deposit 1"},
            {"amount": 9500, "merchant": "bank", "description": "Deposit 2"},
            {"amount": 9500, "merchant": "bank", "description": "Deposit 3"},
        ]
    },
    "velocity_abuse": {
        "description": "Multiple transactions in short timeframe",
        "transactions": [
            {"amount": 1000, "merchant": "amazon", "description": "Purchase 1"},
            {"amount": 2000, "merchant": "ebay", "description": "Purchase 2"},
            {"amount": 3000, "merchant": "unknown_online", "description": "Purchase 3"},
        ]
    },
    "geographic_anomaly": {
        "description": "Impossible geographic transaction sequence",
        "transactions": [
            {"amount": 50, "country": "US", "merchant": "starbucks", "description": "US Transaction"},
            {"amount": 5000, "country": "NG", "merchant": "western_union", "description": "Nigeria Wire"},
        ]
    },
    "night_spike": {
        "description": "High-value transactions at unusual times",
        "transactions": [
            {"amount": 100000, "merchant": "crypto_exchange", "description": "3AM Crypto Buy"},
        ]
    },
}


def demo_fraud_pattern(pattern_name: str):
    """Demonstrate a specific fraud pattern"""

    pattern = PATTERNS.get(pattern_name)
    if not pattern:
        print(f"❌ Unknown pattern: {pattern_name}")
        print("\nAvailable patterns:")
        for key in PATTERNS:
            print(f"  - {key}: {PATTERNS[key]['description']}")
        return

    print(f"\n🔴 Demonstrating: {pattern['description']}\n")

    for i, trans in enumerate(pattern['transactions'], 1):
        transaction = {
            "user_id": "fraud_demo_user",
            "event_type": "purchase",
            "amount": trans.get("amount", 1000),
            "merchant": trans.get("merchant", "unknown"),
            "country": trans.get("country", "US"),
            "timestamp": "2026-06-19T15:02:00Z",
        }

        print(f"[{i}] Sending: {trans.get('description', 'Transaction')}")
        print(f"    Amount: ${transaction['amount']}")
        print(f"    Merchant: {transaction['merchant']}")
        print(f"    Country: {transaction['country']}")

        try:
            response = requests.post(
                "http://localhost:8080/events",
                json=transaction,
                timeout=5
            )

            if response.status_code in [200, 202]:
                event_id = response.json().get("event_id")
                print(f"    ✅ Accepted: {event_id}\n")
            else:
                print(f"    ❌ Failed: {response.status_code}\n")

        except Exception as e:
            print(f"    ❌ Error: {e}\n")

        time.sleep(1)

    print("\n⏳ Waiting for fraud detection...\n")
    time.sleep(5)

    try:
        response = requests.get("http://localhost:8080/alerts")
        alerts = response.json().get("alerts", [])

        fraud_alerts = [a for a in alerts if a.get("user_id") == "fraud_demo_user"]

        if fraud_alerts:
            print(f"🚨 {len(fraud_alerts)} Fraud Alert(s) Generated:\n")
            for alert in fraud_alerts:
                print(f"   Event ID: {alert['event_id']}")
                print(f"   Risk Score: {alert['risk_score']}/10")
                print(f"   Message: {alert['alert_message']}")
                print(f"   Timestamp: {alert['created_at']}\n")
        else:
            print("⚠️  No fraud alerts generated yet\n")
    except Exception as e:
        print(f"❌ Error fetching alerts: {e}\n")


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("🎯 EventPulse Fraud Pattern Demonstrator\n")
        print("Available patterns:")
        for key, pattern in PATTERNS.items():
            print(f"  {key}: {pattern['description']}")
        print("\nUsage: python3 fraud_pattern_demo.py <pattern>")
        print("\nExamples:")
        print("  python3 fraud_pattern_demo.py structuring")
        print("  python3 fraud_pattern_demo.py velocity_abuse")
        print("  python3 fraud_pattern_demo.py geographic_anomaly")
        print("  python3 fraud_pattern_demo.py night_spike")
        sys.exit(1)

    pattern_name = sys.argv[1]
    demo_fraud_pattern(pattern_name)
