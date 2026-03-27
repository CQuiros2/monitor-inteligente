# 🖥️ Monitor Inteligente de Sistema con Detección de Anomalías

**Universidad Latina de Costa Rica — Sede San Pedro**  
Facultad de Ingenierías en TICs  
BISOFT-34 Administración de Sistemas Operativos y Redes — I Cuatrimestre 2026

**Integrantes:**
- Kiany Arroliga Martínez
- Kirsten Naomi Vargas Quirós
- Cristian Josué Quirós Lobo

---

## 📋 Descripción

Sistema de monitoreo de recursos del sistema operativo Linux con detección automática de anomalías usando inteligencia artificial. El sistema recopila métricas de CPU, memoria, red y procesos en tiempo real, las transmite a un servidor central vía sockets TCP, y aplica el algoritmo **Isolation Forest** para identificar comportamientos anómalos.

---

## 🏗️ Arquitectura

```
[Agente Go] ──TCP:9000──► [Servidor Go] ──► [Detector Python/IA]
                               │
                           HTTP:8080
                               │
                         [Dashboard Web]
```

- **Agente** (`/agent`): proceso Go que lee `/proc` y envía métricas cada 5 segundos
- **Servidor** (`/server`): proceso Go que recibe conexiones concurrentes y detecta anomalías
- **Detector** (`/detector`): módulo Python con Isolation Forest para detección de anomalías

---

## 🚀 Inicio Rápido con Docker

### Requisitos
- [Docker Desktop](https://www.docker.com/products/docker-desktop) instalado

### Pasos

```bash
# 1. Clonar el repositorio
git clone https://github.com/[usuario]/monitor-inteligente
cd monitor-inteligente

# 2. Levantar el sistema completo
cd docker
docker-compose up --build

# 3. Abrir el dashboard en el navegador
# http://localhost:8080

# 4. Ver alertas en formato JSON
# http://localhost:8080/status
```

### Detener el sistema

```bash
docker-compose down
```

---

## 🛠️ Ejecución Manual (sin Docker)

### Requisitos
- Go 1.22+
- Python 3.11+

### Servidor

```bash
cd server
go run .
```

### Agente (en otra terminal)

```bash
cd agent
SERVER_ADDR=localhost:9000 go run .
```

### Detector (entrenar modelo)

```bash
cd detector
pip install -r requirements.txt
python train.py --simulated
```

---

## 📊 Métricas Recolectadas

| Métrica | Fuente | Descripción |
|---------|--------|-------------|
| CPU | `/proc/stat` | Porcentaje de uso de CPU |
| Memoria | `/proc/meminfo` | Porcentaje de RAM usada |
| Red | `/proc/net/dev` | KB recibidos por interfaz principal |
| Procesos | `/proc/[pid]` | Cantidad de procesos activos |

---

## 🔍 Detección de Anomalías

El sistema usa **Isolation Forest** (scikit-learn) con los siguientes parámetros:

- `n_estimators`: 100 árboles
- `contamination`: 5% esperado de anomalías
- Entrenamiento inicial con datos simulados de comportamiento normal
- Fallback a umbrales fijos si no hay modelo entrenado (CPU > 90%, RAM > 92%)

---

## 📁 Estructura del Proyecto

```
monitor-inteligente/
├── agent/
│   ├── main.go          # Punto de entrada del agente
│   └── collector.go     # Lectura de métricas desde /proc
├── server/
│   ├── main.go          # Servidor TCP + API HTTP + detector
│   └── handler.go       # Manejo de conexiones de agentes
├── detector/
│   ├── detector.py      # Modelo Isolation Forest
│   ├── train.py         # Script de entrenamiento
│   └── requirements.txt
├── docker/
│   ├── Dockerfile.agent
│   ├── Dockerfile.server
│   └── docker-compose.yml
└── README.md
```

---

## 📄 Licencia

MIT License — ver archivo [LICENSE](LICENSE)
