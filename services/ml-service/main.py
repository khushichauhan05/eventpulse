"""
EventPulse ML Scoring Service
Real-time fraud detection using Isolation Forest with SHAP explanations.
"""

import json
import logging
import time
from contextlib import asynccontextmanager

import joblib
import numpy as np
import shap
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import Response
from prometheus_client import Counter, Histogram, generate_latest
from pydantic import BaseModel, Field

from features import build_feature_vector

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)

# ── human-readable labels for each feature ───────────────────────────────────
FEATURE_LABELS: dict[str, str] = {
    "amount_log": "Transaction Amount",
    "event_type_risk": "Transaction Type Risk",
    "hour_sin": "Time-of-Day Anomaly",
    "hour_cos": "Time-of-Day Cycle",
    "user_velocity_1h": "Transaction Frequency (1h)",
    "user_amount_sum_log": "Recent Volume (1h)",
}

# ── model artifacts (loaded at startup) ──────────────────────────────────────
model = None
scaler = None
feature_names: list[str] = []
shap_explainer = None
train_mean_scaled: np.ndarray | None = None
train_std_scaled: np.ndarray | None = None
USE_SHAP = False


def _load_artifacts():
    global model, scaler, feature_names, shap_explainer, train_mean_scaled, train_std_scaled, USE_SHAP

    model = joblib.load("model_artifacts/model.pkl")
    scaler = joblib.load("model_artifacts/scaler.pkl")
    with open("model_artifacts/feature_names.json") as f:
        feature_names = json.load(f)
    train_mean_scaled = np.load("model_artifacts/train_mean_scaled.npy")
    train_std_scaled = np.load("model_artifacts/train_std_scaled.npy")

    try:
        shap_explainer = shap.TreeExplainer(model)
        # Warm-up: verify it works on a dummy sample
        dummy = np.zeros((1, len(feature_names)))
        _ = shap_explainer.shap_values(dummy)
        USE_SHAP = True
        logger.info("SHAP TreeExplainer loaded successfully")
    except Exception as exc:
        logger.warning("SHAP unavailable, using deviation-based explanations: %s", exc)
        USE_SHAP = False

    logger.info("Model artifacts loaded. Features: %s", feature_names)


@asynccontextmanager
async def lifespan(app: FastAPI):
    _load_artifacts()
    yield


app = FastAPI(
    title="EventPulse Fraud Detection API",
    description="Real-time ML scoring for payment transactions using Isolation Forest",
    version="2.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["GET", "POST"],
    allow_headers=["*"],
)

# ── prometheus metrics ────────────────────────────────────────────────────────
SCORE_REQUESTS = Counter("ml_score_requests_total", "Total scoring requests")
HIGH_RISK_EVENTS = Counter("ml_high_risk_events_total", "High-risk events flagged")
ML_FALLBACKS = Counter("ml_shap_fallbacks_total", "Requests using deviation fallback instead of SHAP")
INFERENCE_LATENCY = Histogram(
    "ml_inference_latency_seconds",
    "End-to-end scoring latency (feature engineering + model + explanation)",
    buckets=[0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25],
)
RISK_SCORE_DIST = Histogram(
    "ml_risk_score_distribution",
    "Distribution of output risk scores",
    buckets=[10, 20, 30, 40, 50, 60, 70, 80, 90, 100],
)


# ── request / response schemas ────────────────────────────────────────────────
class ScoreRequest(BaseModel):
    user_id: str = Field(..., example="user_42")
    amount: float = Field(..., gt=0, example=92400.0)
    event_type: str = Field(..., example="withdrawal")


class ScoreResponse(BaseModel):
    risk_score: int = Field(..., ge=0, le=100, description="0 = safe, 100 = certain fraud")
    confidence: float = Field(..., ge=0.0, le=1.0)
    is_high_risk: bool
    model: str
    explanation: dict[str, float] = Field(
        ..., description="Top feature contributions to the risk score (positive = increases risk)"
    )


# ── scoring logic ─────────────────────────────────────────────────────────────
def _compute_explanation(X_scaled: np.ndarray) -> dict[str, float]:
    """
    Returns top-4 feature contributions as {human_label: contribution_score}.
    Positive values increase fraud risk; negative values reduce it.
    Uses SHAP TreeExplainer when available, falls back to Z-score deviation.
    """
    if USE_SHAP:
        try:
            raw_shap = shap_explainer.shap_values(X_scaled)  # (1, n_features)
            # IsolationForest score_samples: higher = more normal.
            # Negate so positive contribution = increases *risk*.
            contributions = {
                FEATURE_LABELS.get(f, f): round(-float(v), 4)
                for f, v in zip(feature_names, raw_shap[0])
            }
            return dict(sorted(contributions.items(), key=lambda x: abs(x[1]), reverse=True)[:4])
        except Exception as exc:
            logger.warning("SHAP inference failed, falling back: %s", exc)
            ML_FALLBACKS.inc()

    # Fallback: normalised deviation from training distribution
    deviations = (X_scaled[0] - train_mean_scaled) / (train_std_scaled + 1e-8)
    contributions = {
        FEATURE_LABELS.get(f, f): round(float(d), 4)
        for f, d in zip(feature_names, deviations)
    }
    return dict(sorted(contributions.items(), key=lambda x: abs(x[1]), reverse=True)[:4])


def _score_sample(feature_vector: list[float]) -> tuple[int, float, bool]:
    """
    Returns (risk_score 0-100, confidence 0-1, is_high_risk).
    Isolation Forest score_samples output range is roughly [-0.7, 0.0].
    We map: 0.0 → risk 0, -0.7 → risk 100.
    """
    X = np.array([feature_vector])
    X_scaled = scaler.transform(X)

    raw_score = float(model.score_samples(X_scaled)[0])  # negative; more negative = more anomalous
    prediction = model.predict(X_scaled)[0]               # -1 = anomaly, 1 = normal

    # Map to 0-100 risk score. Clamp so extreme values don't overflow.
    risk_score = int(np.clip(round(-raw_score * 130), 0, 100))

    # Confidence: how far from the decision boundary (0.5 = decision boundary)
    boundary_score = -0.5 / 130  # maps to risk_score ≈ 50
    confidence = float(np.clip(abs(raw_score - boundary_score) / 0.35, 0.0, 1.0))
    confidence = round(confidence, 3)

    is_high_risk = prediction == -1 or risk_score >= 70

    return risk_score, confidence, is_high_risk


# ── endpoints ─────────────────────────────────────────────────────────────────
@app.post("/score", response_model=ScoreResponse)
def score_event(req: ScoreRequest):
    t0 = time.perf_counter()
    SCORE_REQUESTS.inc()

    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")

    feature_vector, _ = build_feature_vector(req.user_id, req.amount, req.event_type)
    risk_score, confidence, is_high_risk = _score_sample(feature_vector)

    X_scaled = scaler.transform(np.array([feature_vector]))
    explanation = _compute_explanation(X_scaled)

    if is_high_risk:
        HIGH_RISK_EVENTS.inc()

    latency = time.perf_counter() - t0
    INFERENCE_LATENCY.observe(latency)
    RISK_SCORE_DIST.observe(risk_score)

    logger.info(
        "scored user=%s amount=%.2f type=%s risk=%d confidence=%.3f latency_ms=%.1f",
        req.user_id, req.amount, req.event_type, risk_score, confidence, latency * 1000,
    )

    return ScoreResponse(
        risk_score=risk_score,
        confidence=confidence,
        is_high_risk=is_high_risk,
        model="isolation_forest_v2",
        explanation=explanation,
    )


@app.get("/health")
def health():
    return {
        "status": "ok",
        "model_loaded": model is not None,
        "shap_enabled": USE_SHAP,
        "features": feature_names,
    }


@app.get("/metrics")
def metrics():
    return Response(generate_latest(), media_type="text/plain; version=0.0.4")
