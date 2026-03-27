package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
)

// Metrics es la misma estructura que recibe del agente
type Metrics struct {
	Hostname  string  `json:"hostname"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	Network   float64 `json:"network"`
	Processes int     `json:"processes"`
	Timestamp int64   `json:"timestamp"`
}

// handleAgent maneja la conexión de un agente individual en su propia goroutine
func handleAgent(conn net.Conn, detector *AnomalyDetector) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("[SERVIDOR] Agente conectado desde: %s", remoteAddr)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var metrics Metrics
		if err := json.Unmarshal([]byte(line), &metrics); err != nil {
			log.Printf("[SERVIDOR] Error parseando métricas de %s: %v", remoteAddr, err)
			continue
		}

		log.Printf("[SERVIDOR] Recibido de %s — CPU: %.1f%% | MEM: %.1f%% | PROCS: %d",
			metrics.Hostname, metrics.CPU, metrics.Memory, metrics.Processes)

		// Enviar al detector de anomalías
		detector.Evaluate(metrics)
	}

	log.Printf("[SERVIDOR] Agente desconectado: %s", remoteAddr)
}
