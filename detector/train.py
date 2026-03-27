"""
train.py — Script para entrenar el modelo con datos reales del sistema
Universidad Latina de Costa Rica — BISOFT-34

Uso:
  1. Correr primero el agente durante 10-15 minutos en condiciones normales
  2. Ejecutar: python train.py --file metricas.json
  3. El modelo queda guardado como model.pkl y scaler.pkl
"""

import argparse
import json
import os
import sys
from detector import train_model, generate_training_data


def main():
    parser = argparse.ArgumentParser(description="Entrenamiento del modelo de detección de anomalías")
    parser.add_argument("--file", type=str, help="Archivo JSON con métricas históricas")
    parser.add_argument("--simulated", action="store_true", help="Usar datos simulados para entrenamiento")
    args = parser.parse_args()

    if args.simulated or not args.file:
        print("[TRAIN] Usando datos simulados para entrenamiento...")
        data = generate_training_data(300)
        train_model(data)
        print("[TRAIN] Listo. Modelo entrenado con datos simulados.")
        return

    if not os.path.exists(args.file):
        print(f"[TRAIN] Error: archivo no encontrado: {args.file}")
        sys.exit(1)

    with open(args.file, "r") as f:
        data = json.load(f)

    print(f"[TRAIN] Cargadas {len(data)} muestras desde {args.file}")
    train_model(data)
    print("[TRAIN] Modelo entrenado exitosamente.")


if __name__ == "__main__":
    main()
