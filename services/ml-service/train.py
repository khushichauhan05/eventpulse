"""
Train an Isolation Forest fraud detection model on synthetic payment data.
Saves model artifacts to model_artifacts/ for use by the inference service.

Feature set:
  amount_log          - log1p(amount), handles wide dollar range without outlier distortion
  event_type_risk     - categorical risk weight per event type (0.1 withdrawal → 0.9)
  hour_sin / hour_cos - cyclical encoding of UTC hour so midnight wraps to 00:00
  user_velocity_1h    - transaction count for this user in last hour (from Redis at inference)
  user_amount_sum_log - log1p of total amount transacted by this user in last hour
"""

import json
import math
import os

import joblib
import numpy as np
from sklearn.ensemble import IsolationForest
from sklearn.preprocessing import StandardScaler

SEED = 42
rng = np.random.default_rng(SEED)

FEATURE_NAMES = [
    "amount_log",
    "event_type_risk",
    "hour_sin",
    "hour_cos",
    "user_velocity_1h",
    "user_amount_sum_log",
]

# ── synthetic normal transactions ────────────────────────────────────────────
N_NORMAL = 12_000

normal_amounts = rng.lognormal(mean=5.8, sigma=1.4, size=N_NORMAL)          # ~$100–$8 k
normal_type_risk = rng.beta(a=1.5, b=5, size=N_NORMAL)                      # skewed low
normal_hours = rng.integers(7, 23, size=N_NORMAL).astype(float)             # business hours
normal_velocity = rng.poisson(lam=1.3, size=N_NORMAL).astype(float)         # 1–3 / hr
normal_sum = normal_amounts * (normal_velocity + 1) * rng.uniform(0.8, 1.2, size=N_NORMAL)

# ── synthetic fraud – three distinct patterns ────────────────────────────────
N_FRAUD_A = 300   # large single transaction
N_FRAUD_B = 400   # velocity burst (card testing)
N_FRAUD_C = 300   # night-time + unusual type

# Pattern A: abnormally large amounts
fa_amounts = rng.lognormal(mean=11.0, sigma=0.6, size=N_FRAUD_A)            # $60 k–$500 k
fa_type_risk = rng.beta(a=5, b=1.5, size=N_FRAUD_A)
fa_hours = rng.integers(0, 24, size=N_FRAUD_A).astype(float)
fa_velocity = rng.poisson(lam=2, size=N_FRAUD_A).astype(float)
fa_sum = fa_amounts * (fa_velocity + 1)

# Pattern B: rapid small transactions (card testing / account takeover)
fb_amounts = rng.lognormal(mean=3.5, sigma=0.8, size=N_FRAUD_B)            # $5–$100
fb_type_risk = rng.beta(a=3, b=2, size=N_FRAUD_B)
fb_hours = rng.integers(0, 24, size=N_FRAUD_B).astype(float)
fb_velocity = rng.poisson(lam=22, size=N_FRAUD_B).astype(float)            # many / hr
fb_sum = fb_amounts * (fb_velocity + 1)

# Pattern C: night + high-risk type + moderate-large amount
fc_amounts = rng.lognormal(mean=8.5, sigma=1.0, size=N_FRAUD_C)
fc_type_risk = rng.beta(a=6, b=1, size=N_FRAUD_C)
fc_hours = rng.choice(list(range(0, 5)) + list(range(22, 24)), size=N_FRAUD_C).astype(float)
fc_velocity = rng.poisson(lam=6, size=N_FRAUD_C).astype(float)
fc_sum = fc_amounts * (fc_velocity + 1)

# ── assemble dataset ─────────────────────────────────────────────────────────
def make_features(amounts, type_risks, hours, velocities, sums):
    hour_sin = np.sin(2 * math.pi * hours / 24)
    hour_cos = np.cos(2 * math.pi * hours / 24)
    return np.column_stack([
        np.log1p(amounts),
        type_risks,
        hour_sin,
        hour_cos,
        velocities,
        np.log1p(sums),
    ])

X_normal = make_features(normal_amounts, normal_type_risk, normal_hours, normal_velocity, normal_sum)
X_fraud_a = make_features(fa_amounts, fa_type_risk, fa_hours, fa_velocity, fa_sum)
X_fraud_b = make_features(fb_amounts, fb_type_risk, fb_hours, fb_velocity, fb_sum)
X_fraud_c = make_features(fc_amounts, fc_type_risk, fc_hours, fc_velocity, fc_sum)

X_all = np.vstack([X_normal, X_fraud_a, X_fraud_b, X_fraud_c])

# ── scale ────────────────────────────────────────────────────────────────────
scaler = StandardScaler()
X_scaled = scaler.fit_transform(X_all)

# ── train ────────────────────────────────────────────────────────────────────
total = len(X_all)
n_fraud = N_FRAUD_A + N_FRAUD_B + N_FRAUD_C
contamination = round(n_fraud / total, 4)

model = IsolationForest(
    n_estimators=300,
    max_samples="auto",
    contamination=contamination,
    random_state=SEED,
    n_jobs=-1,
)
model.fit(X_scaled)

# ── save artifacts ───────────────────────────────────────────────────────────
os.makedirs("model_artifacts", exist_ok=True)

joblib.dump(model, "model_artifacts/model.pkl")
joblib.dump(scaler, "model_artifacts/scaler.pkl")

with open("model_artifacts/feature_names.json", "w") as f:
    json.dump(FEATURE_NAMES, f)

# Training distribution mean (used for fallback deviation-based explanation)
np.save("model_artifacts/train_mean_scaled.npy", X_scaled.mean(axis=0))
np.save("model_artifacts/train_std_scaled.npy", X_scaled.std(axis=0))

# Quick sanity check
scores = model.score_samples(X_scaled)
normal_scores = scores[:N_NORMAL]
fraud_scores = scores[N_NORMAL:]
print(f"Trained on {total} samples ({n_fraud} fraud, contamination={contamination})")
print(f"Normal score mean: {normal_scores.mean():.4f}  Fraud score mean: {fraud_scores.mean():.4f}")
print("Artifacts saved to model_artifacts/")
