#!/usr/bin/env python3
"""
Predefined load test profiles for different scenarios
"""

from load_test import LoadTester, NUM_THREADS

PROFILES = {
    "light": {
        "name": "Light Load (Demo - 15 Second Clip)",
        "description": "Gentle load for short video clips",
        "tps": 10,
        "duration": 15,
        "fraud_percentage": 5,
    },

    "medium": {
        "name": "Medium Load (30-Second Clip)",
        "description": "Realistic load for 30-second video segment",
        "tps": 50,
        "duration": 30,
        "fraud_percentage": 5,
    },

    "heavy": {
        "name": "Heavy Load (60-Second Scaling Demo)",
        "description": "Heavy load to show auto-scaling",
        "tps": 200,
        "duration": 60,
        "fraud_percentage": 8,
    },

    "spike": {
        "name": "Traffic Spike (Stress Test)",
        "description": "Sudden spike to show HPA in action",
        "tps": 500,
        "duration": 45,
        "fraud_percentage": 10,
    },

    "sustained": {
        "name": "Sustained Load (Production - 5 Minutes)",
        "description": "Sustained load over 5 minutes",
        "tps": 150,
        "duration": 300,
        "fraud_percentage": 4,
    },
}


def print_profiles():
    """Print all available profiles"""
    print("\n📋 Available Load Test Profiles:\n")
    for key, profile in PROFILES.items():
        print(f"  {key}:")
        print(f"    Name: {profile['name']}")
        print(f"    Description: {profile['description']}")
        print(f"    TPS: {profile['tps']}")
        print(f"    Duration: {profile['duration']}s")
        print(f"    Fraud %: {profile['fraud_percentage']}%\n")


def run_profile(profile_name: str):
    """Run a specific profile"""

    if profile_name not in PROFILES:
        print(f"❌ Profile '{profile_name}' not found")
        print_profiles()
        return

    profile = PROFILES[profile_name]

    print(f"\n▶️  Running Profile: {profile['name']}")
    print(f"   {profile['description']}\n")

    tester = LoadTester(
        endpoint="http://localhost:8080/events",
        tps=profile["tps"],
        duration=profile["duration"],
        fraud_pct=profile["fraud_percentage"]
    )
    tester.NUM_THREADS = min(10, max(2, profile["tps"] // 20))
    tester.run()
