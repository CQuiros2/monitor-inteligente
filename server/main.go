package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// Alert representa una alerta generada por el detector
type Alert struct {
	Hostname  string    `json:"hostname"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Time      time.Time `json:"time"`
}

// AnomalyDetector evalúa métricas y genera alertas con umbrales dinámicos simples
type AnomalyDetector struct {
	mu     sync.Mutex
	alerts []Alert
	// Historial de promedios por host para detección basada en desviación
	history map[string][]Metrics
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		alerts:  []Alert{},
		history: make(map[string][]Metrics),
	}
}

// Evaluate analiza las métricas recibidas y genera alertas si detecta anomalías
func (d *AnomalyDetector) Evaluate(m Metrics) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Guardar en historial (máx 100 muestras por host)
	d.history[m.Hostname] = append(d.history[m.Hostname], m)
	if len(d.history[m.Hostname]) > 100 {
		d.history[m.Hostname] = d.history[m.Hostname][1:]
	}

	// Necesitamos al menos 10 muestras para calcular umbrales dinámicos
	samples := d.history[m.Hostname]
	if len(samples) < 10 {
		return
	}

	cpuAvg, memAvg := average(samples)

	// Anomalía si el valor actual supera 2x el promedio histórico O supera umbrales fijos
	if m.CPU > cpuAvg*2.0 || m.CPU > 90.0 {
		alert := Alert{
			Hostname:  m.Hostname,
			Metric:    "CPU",
			Value:     m.CPU,
			Threshold: cpuAvg * 2.0,
			Time:      time.Now(),
		}
		d.alerts = append(d.alerts, alert)
		log.Printf("🚨 [ANOMALÍA] Host: %s | CPU: %.1f%% (promedio: %.1f%%)", m.Hostname, m.CPU, cpuAvg)
	}

	if m.Memory > memAvg*1.8 || m.Memory > 92.0 {
		alert := Alert{
			Hostname:  m.Hostname,
			Metric:    "MEMORY",
			Value:     m.Memory,
			Threshold: memAvg * 1.8,
			Time:      time.Now(),
		}
		d.alerts = append(d.alerts, alert)
		log.Printf("🚨 [ANOMALÍA] Host: %s | MEM: %.1f%% (promedio: %.1f%%)", m.Hostname, m.Memory, memAvg)
	}
}

func average(samples []Metrics) (float64, float64) {
	var cpuSum, memSum float64
	for _, s := range samples {
		cpuSum += s.CPU
		memSum += s.Memory
	}
	n := float64(len(samples))
	return cpuSum / n, memSum / n
}

func main() {
	detector := NewAnomalyDetector()

	// API HTTP para consultar el estado
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		detector.mu.Lock()
		defer detector.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "running",
			"alert_count": len(detector.alerts),
			"alerts":      detector.alerts,
		})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
  <title>Monitor Inteligente de Sistema</title>
  <meta http-equiv="refresh" content="5">
  <style>
    body { font-family: Arial, sans-serif; background: #f4f4f4; padding: 20px; }
    h1 { color: #6AB42D; }
    .alert { background: #ffe0e0; border-left: 4px solid #e00; padding: 10px; margin: 8px 0; border-radius: 4px; }
    .ok { color: #6AB42D; font-weight: bold; }
    table { width: 100%%; border-collapse: collapse; background: white; }
    th { background: #6AB42D; color: white; padding: 8px; }
    td { padding: 8px; border-bottom: 1px solid #ddd; }
  </style>
</head>
<body>
  <h1>🖥️ Monitor Inteligente de Sistema</h1>
  <p>Universidad Latina de Costa Rica — BISOFT-34</p>
  <p>Página se actualiza cada 5 segundos. Ver alertas en <a href="/status">/status</a></p>
</body>
</html>`)
	})

	// Servidor TCP para agentes (goroutine separada)
	go func() {
		ln, err := net.Listen("tcp", ":9000")
		if err != nil {
			log.Fatalf("[SERVIDOR] Error iniciando TCP: %v", err)
		}
		log.Printf("[SERVIDOR] Escuchando agentes en :9000")
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("[SERVIDOR] Error aceptando conexión: %v", err)
				continue
			}
			go handleAgent(conn, detector)
		}
	}()

	// Servidor HTTP
	log.Printf("[SERVIDOR] Dashboard disponible en http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("[SERVIDOR] Error HTTP: %v", err)
	}
}
