#!/usr/bin/env python3
"""
Run EventPulse load test profiles
Usage: python3 run_profile.py <profile_name>
"""

import sys
from load_profiles import run_profile, print_profiles

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print_profiles()
        print("\nUsage: python3 run_profile.py <profile_name>")
        print("\nExamples:")
        print("  python3 run_profile.py light")
        print("  python3 run_profile.py medium")
        print("  python3 run_profile.py heavy")
        print("  python3 run_profile.py spike")
        print("  python3 run_profile.py sustained")
        sys.exit(1)

    profile_name = sys.argv[1]
    run_profile(profile_name)
