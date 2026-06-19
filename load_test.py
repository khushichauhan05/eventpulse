#!/usr/bin/env python3
"""
EventPulse Load Testing Script
Generates realistic transactions to test fraud detection system
"""

import requests
import json
import random
import time
import threading
from datetime import datetime, timedelta
from typing import List, Dict, Tuple
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed

# ============================================================================
# CONFIGURATION
# ============================================================================

API_ENDPOINT = "http://localhost:8080/events"
ALERTS_ENDPOINT = "http://localhost:8080/alerts"

# Load testing parameters
TRANSACTIONS_PER_SECOND = 50
DURATION_SECONDS = 60
FRAUD_PERCENTAGE = 5

# Threading
NUM_THREADS = 10

# ============================================================================
# TRANSACTION DATA GENERATORS
# ============================================================================

class TransactionGenerator:
    """Generates realistic transaction data"""

    MERCHANTS = {
        "amazon": {"category": "ecommerce", "risk": 0},
        "walmart": {"category": "retail", "risk": 0},
        "target": {"category": "retail", "risk": 0},
        "ebay": {"category": "ecommerce", "risk": 0.5},
        "unknown_online": {"category": "ecommerce", "risk": 1},
        "starbucks": {"category": "food", "risk": 0},
        "mcd": {"category": "food", "risk": 0},
        "pizza_hut": {"category": "food", "risk": 0},
        "uber": {"category": "transport", "risk": 0.5},
        "lyft": {"category": "transport", "risk": 0.5},
        "unknown_recipient": {"category": "transfer", "risk": 3},
        "crypto_exchange": {"category": "crypto", "risk": 2},
        "western_union": {"category": "transfer", "risk": 2},
    }

    HIGH_RISK_COUNTRIES = {
        "NG": "Nigeria",
        "GH": "Ghana",
        "KE": "Kenya",
        "RU": "Russia",
        "IR": "Iran",
        "KP": "North Korea",
    }

    LOW_RISK_COUNTRIES = {
        "US": "USA",
        "CA": "Canada",
        "MX": "Mexico",
        "GB": "United Kingdom",
        "DE": "Germany",
        "FR": "France",
    }

    EVENT_TYPES = {
        "purchase": "Online purchase",
        "withdrawal": "ATM withdrawal",
        "transfer": "Money transfer",
        "international_wire": "International wire transfer",
        "cryptocurrency": "Cryptocurrency purchase",
    }

    def __init__(self):
        self.user_ids = [f"user_{i:06d}" for i in range(1, 101)]
        self.transaction_count = 0

    def generate_normal_transaction(self) -> Dict:
        """Generate a normal (low-risk) transaction"""
        return {
            "user_id": random.choice(self.user_ids),
            "event_type": "purchase",
            "amount": round(random.uniform(20, 500), 2),
            "merchant": random.choice(["amazon", "walmart", "target", "starbucks", "mcd"]),
            "country": random.choice(list(self.LOW_RISK_COUNTRIES.keys())),
            "timestamp": self._realistic_timestamp(hour_range=(8, 22)),
            "device_ip": self._generate_ip(),
            "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
        }

    def generate_suspicious_transaction(self) -> Dict:
        """Generate a suspicious (high-risk) transaction"""

        factors = random.choices([
            "high_amount",
            "international",
            "night_time",
            "high_risk_country",
            "crypto",
            "unknown_merchant"
        ], k=random.randint(2, 4))

        amount = 50000 if "high_amount" in factors else round(random.uniform(100, 1000), 2)

        if "crypto" in factors:
            merchant = "crypto_exchange"
            amount = round(random.uniform(5000, 50000), 2)
        elif "unknown_merchant" in factors:
            merchant = random.choice(["unknown_online", "unknown_recipient"])
        else:
            merchant = random.choice(list(self.MERCHANTS.keys()))

        country = (random.choice(list(self.HIGH_RISK_COUNTRIES.keys()))
                  if "high_risk_country" in factors
                  else random.choice(list(self.LOW_RISK_COUNTRIES.keys())))

        hour_range = (0, 6) if "night_time" in factors else (8, 22)

        event_type = (random.choice(["international_wire", "cryptocurrency"])
                     if "international" in factors
                     else "purchase")

        return {
            "user_id": random.choice(self.user_ids),
            "event_type": event_type,
            "amount": amount,
            "merchant": merchant,
            "country": country,
            "timestamp": self._realistic_timestamp(hour_range=hour_range),
            "device_ip": self._generate_ip(),
            "user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 13_3 like Mac OS X)",
            "factors": factors,
        }

    def generate_transaction(self, is_fraud: bool = False) -> Dict:
        """Generate a random transaction"""
        self.transaction_count += 1

        if is_fraud:
            trans = self.generate_suspicious_transaction()
            trans["is_fraud"] = True
        else:
            trans = self.generate_normal_transaction()
            trans["is_fraud"] = False

        return trans

    @staticmethod
    def _realistic_timestamp(hour_range: Tuple[int, int] = (8, 22)) -> str:
        """Generate realistic timestamp"""
        hour = random.randint(hour_range[0], hour_range[1])
        minute = random.randint(0, 59)
        second = random.randint(0, 59)

        now = datetime.utcnow()
        ts = now.replace(hour=hour, minute=minute, second=second)

        return ts.strftime("%Y-%m-%dT%H:%M:%SZ")

    @staticmethod
    def _generate_ip() -> str:
        """Generate realistic IP address"""
        return f"{random.randint(1,255)}.{random.randint(0,255)}.{random.randint(0,255)}.{random.randint(0,255)}"


# ============================================================================
# LOAD TEST EXECUTOR
# ============================================================================

class LoadTester:
    """Executes load test against EventPulse API"""

    def __init__(self, endpoint: str, tps: int, duration: int, fraud_pct: int):
        self.endpoint = endpoint
        self.tps = tps
        self.duration = duration
        self.fraud_pct = fraud_pct
        self.generator = TransactionGenerator()

        self.total_sent = 0
        self.total_success = 0
        self.total_failed = 0
        self.fraud_count = 0
        self.start_time = None
        self.errors: List[str] = []
        self.response_times: List[float] = []

        self.lock = threading.Lock()
        self.NUM_THREADS = 10

    def send_transaction(self, transaction: Dict) -> Tuple[bool, float, str]:
        """Send single transaction to API"""

        is_fraud = transaction.pop("is_fraud", False)
        transaction.pop("factors", [])

        payload = {
            "user_id": transaction["user_id"],
            "event_type": transaction["event_type"],
            "amount": transaction["amount"],
        }

        try:
            start = time.time()
            response = requests.post(
                self.endpoint,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=5
            )
            elapsed = (time.time() - start) * 1000

            with self.lock:
                self.total_sent += 1
                self.response_times.append(elapsed)

                if is_fraud:
                    self.fraud_count += 1

            if response.status_code in [200, 201, 202]:
                with self.lock:
                    self.total_success += 1

                event_id = response.json().get("event_id", "unknown")
                return (True, elapsed, event_id)
            else:
                with self.lock:
                    self.total_failed += 1

                return (False, elapsed, f"HTTP {response.status_code}")

        except Exception as e:
            with self.lock:
                self.total_failed += 1
                self.errors.append(str(e))

            return (False, 0, f"Error: {str(e)}")

    def worker_thread(self, thread_id: int):
        """Worker thread that sends transactions"""

        transactions_per_thread = (self.tps * self.duration) // self.NUM_THREADS + 1
        delay_between_requests = (self.NUM_THREADS / self.tps) if self.tps > 0 else 0

        for i in range(transactions_per_thread):
            if time.time() - self.start_time > self.duration:
                break

            is_fraud = random.randint(1, 100) <= self.fraud_pct
            transaction = self.generator.generate_transaction(is_fraud=is_fraud)
            success, elapsed, result = self.send_transaction(transaction)

            status = "" if success else ""
            fraud_indicator = "" if transaction.get("is_fraud") else " "

            print(f"[T{thread_id}] {status} {fraud_indicator} "
                  f"Amount: ${transaction.get('amount', 0):>8.2f} "
                  f"Merchant: {transaction.get('merchant', ''):<15} "
                  f"Latency: {elapsed:>6.1f}ms")

            time.sleep(delay_between_requests)

    def run(self):
        """Execute load test"""

        print("\n" + "="*90)
        print(" EventPulse Load Test Started")
        print("="*90)
        print(f"Endpoint: {self.endpoint}")
        print(f"Target TPS: {self.tps}")
        print(f"Duration: {self.duration}s")
        print(f"Fraud Percentage: {self.fraud_pct}%")
        print(f"Threads: {self.NUM_THREADS}")
        print("="*90 + "\n")

        self.start_time = time.time()

        with ThreadPoolExecutor(max_workers=self.NUM_THREADS) as executor:
            futures = []

            for thread_id in range(self.NUM_THREADS):
                future = executor.submit(self.worker_thread, thread_id)
                futures.append(future)

            for future in as_completed(futures):
                try:
                    future.result()
                except Exception as e:
                    print(f" Thread error: {e}")

        elapsed_time = time.time() - self.start_time
        self.print_results(elapsed_time)

    def print_results(self, elapsed_time: float):
        """Print final statistics"""

        print("\n" + "="*90)
        print(" Load Test Results")
        print("="*90)

        print(f"\n⏱  Execution Time: {elapsed_time:.2f}s")
        print(f" Total Transactions Sent: {self.total_success + self.total_failed}")
        print(f" Successful: {self.total_success}")
        print(f" Failed: {self.total_failed}")
        print(f" Fraud Detected: {self.fraud_count}")

        if self.response_times:
            print(f"\n Response Time Statistics:")
            response_times_sorted = sorted(self.response_times)
            print(f"   Min: {min(response_times_sorted):.2f}ms")
            print(f"   Max: {max(response_times_sorted):.2f}ms")
            print(f"   Avg: {sum(response_times_sorted)/len(response_times_sorted):.2f}ms")
            print(f"   P50 (Median): {response_times_sorted[len(response_times_sorted)//2]:.2f}ms")
            print(f"   P95: {response_times_sorted[int(len(response_times_sorted)*0.95)]:.2f}ms")
            print(f"   P99: {response_times_sorted[int(len(response_times_sorted)*0.99)]:.2f}ms")

        actual_tps = self.total_success / elapsed_time if elapsed_time > 0 else 0
        print(f"\n Actual TPS: {actual_tps:.2f} (target: {self.tps})")

        if self.errors:
            print(f"\n  Errors ({len(self.errors)}):")
            for error in self.errors[:5]:
                print(f"   - {error}")
            if len(self.errors) > 5:
                print(f"   ... and {len(self.errors)-5} more")

        print("="*90 + "\n")


# ============================================================================
# ALERT MONITORING
# ============================================================================

class AlertMonitor:
    """Monitor generated alerts in real-time"""

    def __init__(self, alerts_endpoint: str):
        self.endpoint = alerts_endpoint
        self.last_alert_count = 0

    def check_alerts(self):
        """Check how many alerts have been generated"""
        try:
            response = requests.get(self.endpoint, timeout=5)
            if response.status_code == 200:
                alerts = response.json().get("alerts", [])
                new_alerts = len(alerts) - self.last_alert_count

                if new_alerts > 0:
                    self.last_alert_count = len(alerts)
                    latest_alert = alerts[-1] if alerts else None

                    print(f"\n NEW FRAUD ALERT! ({len(alerts)} total)")
                    if latest_alert:
                        print(f"   Event: {latest_alert.get('event_id')}")
                        print(f"   User: {latest_alert.get('user_id')}")
                        print(f"   Risk Score: {latest_alert.get('risk_score')}")
                        print(f"   Message: {latest_alert.get('alert_message')}\n")

                return len(alerts)
        except Exception as e:
            pass

        return self.last_alert_count


# ============================================================================
# MAIN ENTRY POINT
# ============================================================================

def main():
    """Main entry point"""

    print("""
    ╔════════════════════════════════════════════════════════════════╗
    ║                                                                ║
    ║            EventPulse Load Testing & Demo Script              ║
    ║                                                                ║
    ║  This script generates realistic transactions to test          ║
    ║  EventPulse fraud detection at scale.                          ║
    ║                                                                ║
    ╚════════════════════════════════════════════════════════════════╝
    """)

    try:
        response = requests.get("http://localhost:8080/health", timeout=5)
        print(" API Gateway is running\n")
    except:
        print(" ERROR: Cannot connect to API Gateway at http://localhost:8080")
        print("   Make sure EventPulse is running:")
        print("   $ kubectl port-forward -n eventpulse svc/api-gateway 8080:8080")
        sys.exit(1)

    tester = LoadTester(
        endpoint=API_ENDPOINT,
        tps=TRANSACTIONS_PER_SECOND,
        duration=DURATION_SECONDS,
        fraud_pct=FRAUD_PERCENTAGE
    )

    tester.NUM_THREADS = NUM_THREADS

    monitor = AlertMonitor(ALERTS_ENDPOINT)

    def monitor_alerts():
        while tester.start_time and time.time() - tester.start_time < DURATION_SECONDS + 5:
            monitor.check_alerts()
            time.sleep(2)

    monitor_thread = threading.Thread(target=monitor_alerts, daemon=True)
    monitor_thread.start()

    try:
        tester.run()
    except KeyboardInterrupt:
        print("\n  Load test interrupted by user")

    print("\n Final Alert Summary:")
    final_alerts = monitor.check_alerts()
    print(f"Total Fraud Alerts Generated: {final_alerts}")


if __name__ == "__main__":
    main()
