FROM python:3.11-slim AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends gcc g++ && rm -rf /var/lib/apt/lists/*

COPY services/ml-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY services/ml-service/ .

# Train the model at image build time so the artifact is baked in.
RUN mkdir -p model_artifacts && python train.py

# ── runtime image ─────────────────────────────────────────────────────────────
FROM python:3.11-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin/uvicorn /usr/local/bin/uvicorn
COPY --from=builder /app /app

EXPOSE 8000

HEALTHCHECK --interval=10s --timeout=5s --retries=10 --start-period=15s \
  CMD curl -fsS http://localhost:8000/health || exit 1

CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
