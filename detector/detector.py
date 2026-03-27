"""
detector.py — Módulo de detección de anomalías con Isolation Forest
Universidad Latina de Costa Rica — BISOFT-34
Monitor Inteligente de Sistema
"""

import json
import sys
import numpy as np
from sklearn.ensemble import IsolationForest
from sklearn.preprocessing import StandardScaler
import joblib
import os

MODEL_PATH = "model.pkl"
SCALER_PATH = "scaler.pkl"


def train_model(data: list[dict]) -> None:
    """
    Entrena el modelo Isolation Forest con datos históricos normales del sistema.
    data: lista de dicts con claves cpu, memory, network, processes
    """
    if len(data) < 20:
        print("[DETECTOR] Se necesitan al menos 20 muestras para entrenar.")
        return

    X = np.array([[d["cpu"], d["memory"], d["network"], d["processes"]] for d in data])

    scaler = StandardScaler()
    X_scaled = scaler.fit_transform(X)

    # contamination=0.05 significa que esperamos ~5% de anomalías
    model = IsolationForest(n_estimators=100, contamination=0.05, random_state=42)
    model.fit(X_scaled)

    joblib.dump(model, MODEL_PATH)
    joblib.dump(scaler, SCALER_PATH)
    print(f"[DETECTOR] Modelo entrenado con {len(data)} muestras y guardado.")


def predict(metrics: dict) -> dict:
    """
    Evalúa una observación y retorna si es anómala o no.
    Retorna: {"anomaly": bool, "score": float}
    """
    if not os.path.exists(MODEL_PATH) or not os.path.exists(SCALER_PATH):
        # Si no hay modelo entrenado, usar umbral fijo como fallback
        is_anomaly = metrics.get("cpu", 0) > 90 or metrics.get("memory", 0) > 92
        return {"anomaly": is_anomaly, "score": -1.0, "method": "threshold"}

    model  = joblib.load(MODEL_PATH)
    scaler = joblib.load(SCALER_PATH)

    X = np.array([[
        metrics.get("cpu", 0),
        metrics.get("memory", 0),
        metrics.get("network", 0),
        metrics.get("processes", 0)
    ]])

    X_scaled = scaler.transform(X)
    prediction = model.predict(X_scaled)  # 1 = normal, -1 = anomalía
    score = model.score_samples(X_scaled)[0]

    return {
        "anomaly": bool(prediction[0] == -1),
        "score": float(score),
        "method": "isolation_forest"
    }


def generate_training_data(n_samples: int = 200) -> list[dict]:
    """
    Genera datos de entrenamiento simulando comportamiento normal del sistema.
    En producción, esto se reemplaza por datos reales recolectados durante
    un período de operación normal.
    """
    np.random.seed(42)
    data = []
    for _ in range(n_samples):
        data.append({
            "cpu":       float(np.random.normal(25, 10)),   # CPU promedio ~25%
            "memory":    float(np.random.normal(45, 8)),    # Memoria ~45%
            "network":   float(np.random.normal(500, 200)), # Red ~500 KB
            "processes": int(np.random.normal(150, 20))     # ~150 procesos
        })
    return data


if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Uso: python detector.py train | python detector.py predict '<json>'")
        sys.exit(1)

    command = sys.argv[1]

    if command == "train":
        print("[DETECTOR] Generando datos de entrenamiento...")
        training_data = generate_training_data(300)
        train_model(training_data)

    elif command == "predict":
        if len(sys.argv) < 3:
            print("Error: se requiere JSON de métricas")
            sys.exit(1)
        metrics = json.loads(sys.argv[2])
        result  = predict(metrics)
        print(json.dumps(result))

    else:
        print(f"Comando desconocido: {command}")
        sys.exit(1)
