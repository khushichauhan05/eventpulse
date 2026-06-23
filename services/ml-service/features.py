"""
Feature engineering for real-time fraud scoring.

Velocity features (user_velocity_1h, user_amount_sum_log) are computed from
a Redis sorted-set sliding window. If Redis is unavailable the function
falls back to zeros so the pipeline stays live.
"""

import logging
import math
import os
import time

import redis

logger = logging.getLogger(__name__)

# Risk weight per event type — withdrawal/transfer carry inherently higher risk
EVENT_TYPE_RISK: dict[str, float] = {
    "purchase": 0.10,
    "payment": 0.20,
    "deposit": 0.15,
    "refund": 0.25,
    "transfer": 0.60,
    "withdrawal": 0.85,
    "payout": 0.70,
}
DEFAULT_TYPE_RISK = 0.50

REDIS_URL = os.getenv("REDIS_URL", "redis://redis:6379")
WINDOW_SECONDS = 3600  # 1-hour sliding window

_redis: redis.Redis | None = None


def _get_redis() -> redis.Redis | None:
    global _redis
    if _redis is not None:
        try:
            _redis.ping()
            return _redis
        except Exception:
            _redis = None

    try:
        _redis = redis.from_url(REDIS_URL, socket_connect_timeout=1, socket_timeout=1)
        _redis.ping()
        return _redis
    except Exception as exc:
        logger.warning("Redis unavailable, velocity features will be zero: %s", exc)
        return None


def _get_velocity(user_id: str, amount: float) -> tuple[float, float]:
    """
    Returns (tx_count_1h, amount_sum_1h) for user_id using a Redis sorted set.
    Adds the current transaction and prunes entries older than 1 hour atomically.
    Returns (0, 0) if Redis is unavailable.
    """
    r = _get_redis()
    if r is None:
        return 0.0, 0.0

    now = time.time()
    cutoff = now - WINDOW_SECONDS

    key_count = f"ep:v:count:{user_id}"
    key_sum = f"ep:v:sum:{user_id}"
    member = f"{now}"
    member_amt = f"{now}:{amount}"

    try:
        pipe = r.pipeline(transaction=False)
        pipe.zadd(key_count, {member: now})
        pipe.zadd(key_sum, {member_amt: now})
        pipe.zremrangebyscore(key_count, "-inf", cutoff)
        pipe.zremrangebyscore(key_sum, "-inf", cutoff)
        pipe.zcard(key_count)
        pipe.zrange(key_sum, 0, -1)
        pipe.expire(key_count, WINDOW_SECONDS + 120)
        pipe.expire(key_sum, WINDOW_SECONDS + 120)
        results = pipe.execute()

        count = float(results[4])
        amount_entries: list[bytes] = results[5]
        total_amount = sum(
            float(e.decode().split(":", 1)[1]) for e in amount_entries if b":" in e
        )
        return count, total_amount
    except Exception as exc:
        logger.warning("Redis velocity lookup failed for %s: %s", user_id, exc)
        return 0.0, 0.0


def build_feature_vector(user_id: str, amount: float, event_type: str) -> tuple[list[float], dict]:
    """
    Returns (feature_vector, raw_feature_dict).
    feature_vector has the same column order as FEATURE_NAMES in train.py.
    """
    import datetime

    hour = datetime.datetime.utcnow().hour
    velocity_count, velocity_sum = _get_velocity(user_id, amount)
    type_risk = EVENT_TYPE_RISK.get(event_type.lower(), DEFAULT_TYPE_RISK)

    raw = {
        "amount_log": math.log1p(amount),
        "event_type_risk": type_risk,
        "hour_sin": math.sin(2 * math.pi * hour / 24),
        "hour_cos": math.cos(2 * math.pi * hour / 24),
        "user_velocity_1h": velocity_count,
        "user_amount_sum_log": math.log1p(velocity_sum),
    }

    vector = [
        raw["amount_log"],
        raw["event_type_risk"],
        raw["hour_sin"],
        raw["hour_cos"],
        raw["user_velocity_1h"],
        raw["user_amount_sum_log"],
    ]
    return vector, raw
