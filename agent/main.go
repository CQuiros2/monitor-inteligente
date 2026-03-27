package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// Metrics representa las métricas recolectadas del sistema operativo
type Metrics struct {
	Hostname  string  `json:"hostname"`
	CPU       float64 `json:"cpu"`       // Porcentaje de uso de CPU
	Memory    float64 `json:"memory"`    // Porcentaje de memoria usada
	Network   float64 `json:"network"`   // KB recibidos por la interfaz principal
	Processes int     `json:"processes"` // Número de procesos activos
	Timestamp int64   `json:"timestamp"` // Unix timestamp
}

func main() {
	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "server:9000"
	}

	hostname, _ := os.Hostname()
	log.Printf("[AGENTE] Iniciando en host: %s", hostname)
	log.Printf("[AGENTE] Conectando al servidor: %s", serverAddr)

	// Reintentar conexión hasta que el servidor esté listo
	var conn net.Conn
	var err error
	for {
		conn, err = net.Dial("tcp", serverAddr)
		if err == nil {
			break
		}
		log.Printf("[AGENTE] Esperando servidor... reintentando en 3s")
		time.Sleep(3 * time.Second)
	}
	defer conn.Close()
	log.Printf("[AGENTE] Conectado al servidor exitosamente")

	collectAndSend(conn, hostname)
}

func collectAndSend(conn net.Conn, hostname string) {
	for {
		metrics := Metrics{
			Hostname:  hostname,
			CPU:       readCPU(),
			Memory:    readMemory(),
			Network:   readNetwork(),
			Processes: countProcesses(),
			Timestamp: time.Now().Unix(),
		}

		data, err := json.Marshal(metrics)
		if err != nil {
			log.Printf("[AGENTE] Error serializando métricas: %v", err)
			continue
		}

		// Enviamos con salto de línea como delimitador
		_, err = fmt.Fprintf(conn, "%s\n", string(data))
		if err != nil {
			log.Printf("[AGENTE] Error enviando métricas: %v", err)
			return
		}

		log.Printf("[AGENTE] Métricas enviadas — CPU: %.1f%% | MEM: %.1f%% | PROCS: %d",
			metrics.CPU, metrics.Memory, metrics.Processes)

		time.Sleep(5 * time.Second)
	}
}
