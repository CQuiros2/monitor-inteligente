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

Sistema de monitoreo de recursos del sistema operativo Linux con detección automática de anomalías usando inteligencia artificial. El sistema recopila métricas de CPU, memoria, red y procesos en tiempo real, las transmite a un servidor central vía sockets TCP, y aplica el algoritmo **Isolation Forest** con umbral dinámico (2x el promedio histórico) para identificar comportamientos anómalos con severidad **HIGH** y **MED**.

El dashboard web en tiempo real muestra gráficas históricas, panel de alertas con timestamps e incluye un **botón de simulación de anomalías** para demostración en vivo.

---

## 🏗️ Arquitectura

```
[Agente Go] ──TCP:9000──► [Servidor Go] ──► [Detector Isolation Forest]
                               │
                           HTTP:8080
                               │
                      [Dashboard Visual]
                    gráficas · alertas · demo
```

- **Agente** (`/agent`): proceso Go que lee `/proc` y envía métricas cada 5 segundos
- **Servidor** (`/server`): proceso Go que recibe conexiones concurrentes, detecta anomalías y sirve el dashboard
- **Detector** (`/detector`): módulo Python con Isolation Forest para detección de anomalías

---

## 🚀 Inicio Rápido con Docker

### Requisitos
- [Docker Desktop](https://www.docker.com/products/docker-desktop) instalado

### Pasos

```bash
# 1. Clonar el repositorio
git clone https://github.com/CQuiros2/monitor-inteligente
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

## 📊 Dashboard

El dashboard en `http://localhost:8080` incluye:

- **4 tarjetas de métricas** — CPU, Memoria, Red y Procesos con indicadores de color (verde/rojo)
- **Gráfica de CPU** en tiempo real con historial de 60 muestras
- **Panel de alertas** con badges HIGH/MED y timestamps
- **Gráfica de Memoria + Red** combinadas
- **Estado del sistema** con indicadores por componente
- **⚡ Botón de simulación** — inyecta métricas de CPU alta (88-98%) para demostrar la detección en vivo

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
- Umbral dinámico: alerta cuando CPU supera **2x el promedio histórico**
- Severidad **HIGH** si CPU > 90% · severidad **MED** si CPU > 85%
- Fallback a umbrales fijos si no hay modelo entrenado

### Endpoints disponibles

| Endpoint | Descripción |
|----------|-------------|
| `GET /` | Dashboard visual en tiempo real |
| `GET /status` | Estado y alertas en formato JSON |
| `GET /stress/start` | Inicia simulación de anomalía de CPU |
| `GET /stress/stop` | Detiene la simulación |

---

## 📁 Estructura del Proyecto

```
monitor-inteligente/
├── agent/
│   ├── main.go          # Punto de entrada del agente
│   └── collector.go     # Lectura de métricas desde /proc
├── server/
│   ├── main.go          # Servidor TCP + dashboard + detector + stress endpoints
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
