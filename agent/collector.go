package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// readCPU calcula el porcentaje de uso de CPU leyendo /proc/stat
func readCPU() float64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				return 0
			}
			user, _      := strconv.ParseFloat(fields[1], 64)
			nice, _      := strconv.ParseFloat(fields[2], 64)
			system, _    := strconv.ParseFloat(fields[3], 64)
			idle, _      := strconv.ParseFloat(fields[4], 64)
			iowait, _    := strconv.ParseFloat(fields[5], 64)
			irq, _       := strconv.ParseFloat(fields[6], 64)
			softirq, _   := strconv.ParseFloat(fields[7], 64)

			total := user + nice + system + idle + iowait + irq + softirq
			used  := total - idle - iowait
			if total == 0 {
				return 0
			}
			return (used / total) * 100
		}
	}
	return 0
}

// readMemory retorna el porcentaje de memoria usada leyendo /proc/meminfo
func readMemory() float64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()

	var total, available float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseFloat(fields[1], 64)
		switch fields[0] {
		case "MemTotal:":
			total = val
		case "MemAvailable:":
			available = val
		}
	}
	if total == 0 {
		return 0
	}
	return ((total - available) / total) * 100
}

// readNetwork retorna los bytes recibidos por la interfaz de red principal (eth0 o enp0s3)
func readNetwork() float64 {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Saltamos las dos primeras líneas de encabezado
	scanner.Scan()
	scanner.Scan()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		// Ignorar loopback
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 1 {
			continue
		}
		rxBytes, _ := strconv.ParseFloat(fields[0], 64)
		return rxBytes / 1024 // KB
	}
	return 0
}

// countProcesses cuenta los procesos activos leyendo /proc
func countProcesses() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			if _, err := strconv.Atoi(e.Name()); err == nil {
				count++
			}
		}
	}
	return count
}
