package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

type Alert struct {
	Hostname  string    `json:"hostname"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Severity  string    `json:"severity"`
	Time      time.Time `json:"time"`
}

type AnomalyDetector struct {
	mu        sync.Mutex
	alerts    []Alert
	latest    map[string]Metrics
	history   map[string][]Metrics
	stressing bool
	stopStress chan struct{}
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		alerts:  []Alert{},
		latest:  make(map[string]Metrics),
		history: make(map[string][]Metrics),
	}
}

func (d *AnomalyDetector) Evaluate(m Metrics) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.latest[m.Hostname] = m
	d.history[m.Hostname] = append(d.history[m.Hostname], m)
	if len(d.history[m.Hostname]) > 60 {
		d.history[m.Hostname] = d.history[m.Hostname][1:]
	}
	samples := d.history[m.Hostname]
	if len(samples) < 10 {
		return
	}
	cpuAvg, memAvg := average(samples)
	if m.CPU > cpuAvg*2.0 || m.CPU > 85.0 {
		sev := "MED"
		if m.CPU > 90.0 {
			sev = "HIGH"
		}
		d.alerts = append(d.alerts, Alert{
			Hostname: m.Hostname, Metric: "CPU",
			Value: m.CPU, Threshold: cpuAvg * 2.0,
			Severity: sev, Time: time.Now(),
		})
		if len(d.alerts) > 20 {
			d.alerts = d.alerts[1:]
		}
		log.Printf("🚨 [ANOMALÍA] Host: %s | CPU: %.1f%% (avg: %.1f%%) [%s]", m.Hostname, m.CPU, cpuAvg, sev)
	}
	if m.Memory > memAvg*1.8 || m.Memory > 88.0 {
		sev := "MED"
		if m.Memory > 92.0 {
			sev = "HIGH"
		}
		d.alerts = append(d.alerts, Alert{
			Hostname: m.Hostname, Metric: "MEMORY",
			Value: m.Memory, Threshold: memAvg * 1.8,
			Severity: sev, Time: time.Now(),
		})
		if len(d.alerts) > 20 {
			d.alerts = d.alerts[1:]
		}
		log.Printf("🚨 [ANOMALÍA] Host: %s | MEM: %.1f%% (avg: %.1f%%) [%s]", m.Hostname, m.Memory, memAvg, sev)
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

func (d *AnomalyDetector) metricsData() map[string]interface{} {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Merge all hosts into a single combined timeline
	// Find the longest history and use that as base, overlaying stress-demo on top
	maxLen := 0
	for _, host := range d.history {
		if len(host) > maxLen {
			maxLen = len(host)
		}
	}

	cpuHistory := make([]float64, maxLen)
	memHistory := make([]float64, maxLen)
	netHistory := make([]float64, maxLen)

	// Fill with real agent data first
	for hostname, host := range d.history {
		if hostname == "stress-demo" {
			continue
		}
		offset := maxLen - len(host)
		for i, s := range host {
			cpuHistory[offset+i] = s.CPU
			memHistory[offset+i] = s.Memory
			netHistory[offset+i] = s.Network
		}
		break
	}

	// Overlay stress-demo data at the end (most recent)
	if stressHost, ok := d.history["stress-demo"]; ok && len(stressHost) > 0 {
		offset := maxLen - len(stressHost)
		if offset < 0 {
			offset = 0
		}
		for i, s := range stressHost {
			idx := offset + i
			if idx < maxLen {
				cpuHistory[idx] = s.CPU
				memHistory[idx] = s.Memory
				netHistory[idx] = s.Network
			}
		}
	}

	// Latest: prefer stress-demo when stressing, otherwise real agent
	latest := map[string]interface{}{"cpu": 0.0, "memory": 0.0, "network": 0.0, "processes": 0, "hostname": "—"}
	if d.stressing {
		if m, ok := d.latest["stress-demo"]; ok {
			latest = map[string]interface{}{
				"cpu": m.CPU, "memory": m.Memory,
				"network": m.Network, "processes": m.Processes,
				"hostname": m.Hostname,
			}
		}
	} else {
		for hostname, m := range d.latest {
			if hostname == "stress-demo" {
				continue
			}
			latest = map[string]interface{}{
				"cpu": m.CPU, "memory": m.Memory,
				"network": m.Network, "processes": m.Processes,
				"hostname": m.Hostname,
			}
			break
		}
		// fallback to any latest if no real agent
		if latest["hostname"] == "—" {
			for _, m := range d.latest {
				latest = map[string]interface{}{
					"cpu": m.CPU, "memory": m.Memory,
					"network": m.Network, "processes": m.Processes,
					"hostname": m.Hostname,
				}
				break
			}
		}
	}
	recentAlerts := d.alerts
	if len(recentAlerts) > 8 {
		recentAlerts = recentAlerts[len(recentAlerts)-8:]
	}
	return map[string]interface{}{
		"status": "running", "alert_count": len(d.alerts),
		"alerts": recentAlerts, "latest": latest,
		"cpu_history": cpuHistory, "mem_history": memHistory, "net_history": netHistory,
		"stressing": d.stressing,
	}
}

// startStress inyecta métricas de CPU alta para simular una anomalía
func (d *AnomalyDetector) startStress() {
	d.mu.Lock()
	if d.stressing {
		d.mu.Unlock()
		return
	}
	d.stressing = true
	d.stopStress = make(chan struct{})
	d.mu.Unlock()

	log.Printf("🔥 [STRESS] Simulación de anomalía iniciada")
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-d.stopStress:
				log.Printf("✅ [STRESS] Simulación detenida")
				return
			case <-ticker.C:
				// Inyectar métrica de CPU alta con algo de ruido aleatorio
				fakeCPU := 88.0 + rand.Float64()*10.0
				fakeMem := 45.0 + rand.Float64()*5.0
				fakeMetric := Metrics{
					Hostname:  "stress-demo",
					CPU:       fakeCPU,
					Memory:    fakeMem,
					Network:   500 + rand.Float64()*200,
					Processes: 160 + rand.Intn(20),
					Timestamp: time.Now().Unix(),
				}
				d.Evaluate(fakeMetric)
			}
		}
	}()
}

func (d *AnomalyDetector) stopStressing() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stressing && d.stopStress != nil {
		close(d.stopStress)
		d.stressing = false
	}
}

func main() {
	detector := NewAnomalyDetector()

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(detector.metricsData())
	})

	// Endpoint para iniciar simulación de anomalía
	http.HandleFunc("/stress/start", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		detector.startStress()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "stress started"})
	})

	// Endpoint para detener simulación
	http.HandleFunc("/stress/stop", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		detector.stopStressing()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "stress stopped"})
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(dashboardHTML))
	})

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

	log.Printf("[SERVIDOR] Dashboard disponible en http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("[SERVIDOR] Error HTTP: %v", err)
	}
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<title>Monitor Inteligente — BISOFT-34</title>
<style>
@import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@300;400;700;800&display=swap');
*{margin:0;padding:0;box-sizing:border-box;}
:root{--g:#39ff14;--c:#00e5ff;--a:#ffb300;--r:#ff4444;--bg:#0c0c0c;--bg2:#111;--bd:#222;--tx:#c8c8c8;}
body{font-family:'JetBrains Mono',monospace;background:var(--bg);color:var(--tx);height:100vh;display:flex;flex-direction:column;overflow:hidden;}
body::after{content:'';position:fixed;inset:0;pointer-events:none;background:repeating-linear-gradient(0deg,transparent,transparent 2px,rgba(0,0,0,.025) 2px,rgba(0,0,0,.025) 4px);}
.topbar{height:36px;background:var(--bg2);border-bottom:1px solid var(--bd);display:flex;align-items:center;padding:0 16px;gap:8px;flex-shrink:0;}
.dot{width:11px;height:11px;border-radius:50%;}
.tb-title{color:#555;font-size:11px;flex:1;margin-left:6px;}
.pill{border:1px solid rgba(57,255,20,.3);color:var(--g);font-size:9px;padding:2px 8px;border-radius:2px;background:rgba(57,255,20,.05);letter-spacing:1px;}
.pill.red{border-color:rgba(255,68,68,.3);color:var(--r);background:rgba(255,68,68,.05);}
.pill.amber{border-color:rgba(255,179,0,.3);color:var(--a);background:rgba(255,179,0,.05);}
/* STRESS BUTTON */
.stress-btn{
  padding:5px 14px;border-radius:3px;font-family:'JetBrains Mono',monospace;
  font-size:10px;font-weight:700;cursor:pointer;border:none;
  letter-spacing:.5px;transition:all .2s;
}
.stress-btn.start{background:rgba(255,68,68,.15);color:var(--r);border:1px solid rgba(255,68,68,.4);}
.stress-btn.start:hover{background:rgba(255,68,68,.3);}
.stress-btn.stop{background:rgba(57,255,20,.12);color:var(--g);border:1px solid rgba(57,255,20,.35);}
.stress-btn.stop:hover{background:rgba(57,255,20,.22);}
.main{display:flex;flex-direction:column;padding:10px 14px 10px;gap:9px;}
.row{display:grid;gap:9px;}
.r4{grid-template-columns:repeat(4,1fr);}
.r21{grid-template-columns:2fr 1fr;}
.r11{grid-template-columns:1fr 1fr;}
.card{background:var(--bg2);border:1px solid var(--bd);border-radius:4px;padding:10px 13px;}
.chart-card{background:var(--bg2);border:1px solid var(--bd);border-radius:4px;padding:10px 13px;height:220px;display:flex;flex-direction:column;}
.chart-card canvas{flex:1;display:block;width:100%!important;height:0!important;}
.card-head{font-size:9px;color:#444;letter-spacing:1px;text-transform:uppercase;margin-bottom:8px;display:flex;align-items:center;gap:5px;}
.blink{width:6px;height:6px;border-radius:50%;background:var(--g);animation:bl 1.2s infinite;}
@keyframes bl{0%,100%{opacity:1;}50%{opacity:.2;}}
.metric{border-top:2px solid var(--g);}
.metric.warn{border-top-color:var(--a);}
.metric.danger{border-top-color:var(--r);}
.metric.info{border-top-color:var(--c);}
.m-val{font-size:26px;font-weight:800;color:#fff;line-height:1;}
.m-unit{font-size:11px;color:#555;margin-left:2px;}
.m-sub{font-size:9px;color:#444;margin-top:4px;}
.m-sub.alert{color:var(--r);}
canvas{display:block;width:100%!important;}
.alert-row{display:flex;align-items:center;gap:7px;padding:5px 0;border-bottom:1px solid #161616;font-size:10px;}
.alert-row:last-child{border-bottom:none;}
.badge{font-size:8px;padding:2px 6px;border-radius:2px;font-weight:700;min-width:32px;text-align:center;}
.badge.HIGH{background:rgba(255,68,68,.15);color:var(--r);border:1px solid rgba(255,68,68,.3);}
.badge.MED{background:rgba(255,179,0,.12);color:var(--a);border:1px solid rgba(255,179,0,.25);}
.badge.LOW{background:rgba(57,255,20,.08);color:var(--g);border:1px solid rgba(57,255,20,.2);}
.al-msg{flex:1;color:#888;}
.al-msg b{color:#ccc;}
.al-time{color:#333;font-size:9px;}
.st-row{display:flex;align-items:center;gap:7px;padding:4px 0;border-bottom:1px solid #161616;font-size:10px;}
.st-row:last-child{border-bottom:none;}
.st-dot{width:7px;height:7px;border-radius:50%;flex-shrink:0;}
.st-label{flex:1;color:#888;}
.st-val{color:#444;font-size:9px;}
</style>
</head>
<body>
<div class="topbar">
  <div class="dot" style="background:#ff5f57;"></div>
  <div class="dot" style="background:#febc2e;"></div>
  <div class="dot" style="background:#28c840;"></div>
  <div class="tb-title">monitor-inteligente &mdash; Universidad Latina de Costa Rica &mdash; BISOFT-34 &mdash; I Cuatrimestre 2026</div>
  <button id="stress-btn" class="stress-btn start" onclick="toggleStress()">⚡ SIMULAR ANOMALÍA</button>
  <div style="width:10px;"></div>
  <div class="pill" id="status-pill">ENGINE RUNNING</div>
</div>
<div class="main">
  <div class="row r4">
    <div class="card metric" id="card-cpu">
      <div class="card-head"><div class="blink"></div>CPU USAGE</div>
      <div><span class="m-val" id="val-cpu">—</span><span class="m-unit">%</span></div>
      <div class="m-sub" id="sub-cpu">cargando...</div>
    </div>
    <div class="card metric info" id="card-mem">
      <div class="card-head"><div class="blink" style="background:var(--c);"></div>MEMORY</div>
      <div><span class="m-val" id="val-mem">—</span><span class="m-unit">%</span></div>
      <div class="m-sub" id="sub-mem">cargando...</div>
    </div>
    <div class="card metric" id="card-net">
      <div class="card-head"><div class="blink" style="background:var(--a);"></div>NETWORK RX</div>
      <div><span class="m-val" id="val-net">—</span><span class="m-unit">KB</span></div>
      <div class="m-sub">bytes recibidos / interfaz</div>
    </div>
    <div class="card metric" id="card-proc">
      <div class="card-head"><div class="blink" style="background:#888;"></div>PROCESOS</div>
      <div><span class="m-val" id="val-proc">—</span></div>
      <div class="m-sub">activos en el sistema</div>
    </div>
  </div>
  <div class="row r21">
    <div class="chart-card">
      <div class="card-head"><div class="blink"></div>CPU — historial tiempo real</div>
      <canvas id="chart-cpu"></canvas>
    </div>
    <div class="card" style="height:160px;overflow:hidden;display:flex;flex-direction:column;">
      <div class="card-head"><div class="blink" style="background:var(--r);"></div>ALERTAS &nbsp;<span id="alert-count" style="color:var(--r);">0</span></div>
      <div id="alerts-list" style="overflow-y:auto;flex:1;"><div class="alert-row"><div style="color:#333;font-size:10px;">Sin alertas detectadas</div></div></div>
    </div>
  </div>
  <div class="row r11">
    <div class="chart-card">
      <div class="card-head"><div class="blink" style="background:var(--a);"></div>MEMORIA + RED — historial</div>
      <canvas id="chart-mem"></canvas>
    </div>
    <div class="card" style="height:220px;">
      <div class="card-head"><div class="blink" style="background:var(--c);"></div>ESTADO DEL SISTEMA</div>
      <div id="status-grid"></div>
    </div>
  </div>
</div>
<script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.4.0/chart.umd.min.js"></script>
<script>
const G='#39ff14',C='#00e5ff',A='#ffb300',R='#ff4444',DIM='#333';
const cpuChart=new Chart(document.getElementById('chart-cpu'),{type:'line',data:{labels:Array(60).fill(''),datasets:[{label:'CPU %',data:Array(60).fill(null),borderColor:G,backgroundColor:'rgba(57,255,20,.06)',borderWidth:1.5,pointRadius:0,fill:true,tension:.3}]},options:{responsive:true,maintainAspectRatio:false,animation:{duration:300},plugins:{legend:{display:false}},scales:{x:{ticks:{display:false},grid:{color:'#181818'}},y:{min:0,max:100,ticks:{color:DIM,font:{size:9},callback:v=>v+'%'},grid:{color:'#181818'}}}}});
const memChart=new Chart(document.getElementById('chart-mem'),{type:'line',data:{labels:Array(60).fill(''),datasets:[{label:'MEM %',data:Array(60).fill(null),borderColor:C,backgroundColor:'rgba(0,229,255,.05)',borderWidth:1.5,pointRadius:0,fill:true,tension:.3},{label:'NET KB',data:Array(60).fill(null),borderColor:A,backgroundColor:'transparent',borderWidth:1.2,pointRadius:0,fill:false,tension:.3,yAxisID:'y2'}]},options:{responsive:true,maintainAspectRatio:false,animation:{duration:300},plugins:{legend:{labels:{color:DIM,font:{size:9},boxWidth:10}}},scales:{x:{ticks:{display:false},grid:{color:'#181818'}},y:{min:0,max:100,ticks:{color:C,font:{size:9},callback:v=>v+'%'},grid:{color:'#181818'}},y2:{position:'right',ticks:{color:A,font:{size:9}},grid:{display:false}}}}});
let lastAlertCount=-1,isStressing=false;
function fmtTime(iso){const d=new Date(iso);return d.getHours().toString().padStart(2,'0')+':'+d.getMinutes().toString().padStart(2,'0')+':'+d.getSeconds().toString().padStart(2,'0');}
function stRow(ok,label,val){return '<div class="st-row"><div class="st-dot" style="background:'+(ok?G:R)+';"></div><div class="st-label">'+label+'</div><div class="st-val">'+val+'</div></div>';}

async function toggleStress(){
  const btn=document.getElementById('stress-btn');
  if(!isStressing){
    await fetch('/stress/start');
    isStressing=true;
    btn.textContent='⛔ DETENER SIMULACIÓN';
    btn.className='stress-btn stop';
    document.getElementById('status-pill').textContent='🔥 STRESS MODE';
    document.getElementById('status-pill').className='pill amber';
  } else {
    await fetch('/stress/stop');
    isStressing=false;
    btn.textContent='⚡ SIMULAR ANOMALÍA';
    btn.className='stress-btn start';
    document.getElementById('status-pill').textContent='ENGINE RUNNING';
    document.getElementById('status-pill').className='pill';
  }
}

async function tick(){
  try{
    const d=await(await fetch('/status')).json();
    const l=d.latest||{};
    const cpu=+(l.cpu||0),mem=+(l.memory||0),net=+(l.network||0),proc=+(l.processes||0);
    document.getElementById('val-cpu').textContent=cpu.toFixed(1);
    document.getElementById('sub-cpu').textContent=cpu>85?'🚨 ANOMALIA DETECTADA':('host: '+(l.hostname||'—'));
    document.getElementById('sub-cpu').className='m-sub'+(cpu>85?' alert':'');
    document.getElementById('card-cpu').className='card metric'+(cpu>85?' danger':cpu>60?' warn':'');
    document.getElementById('val-mem').textContent=mem.toFixed(1);
    document.getElementById('sub-mem').textContent=mem>88?'🚨 ANOMALIA DETECTADA':('host: '+(l.hostname||'—'));
    document.getElementById('sub-mem').className='m-sub'+(mem>88?' alert':'');
    document.getElementById('card-mem').className='card metric'+(mem>88?' danger':mem>70?' warn':' info');
    document.getElementById('val-net').textContent=net.toFixed(0);
    document.getElementById('val-proc').textContent=proc;
    if(d.cpu_history&&d.cpu_history.length){
      cpuChart.data.datasets[0].data=d.cpu_history.slice(-60);cpuChart.update('none');
      memChart.data.datasets[0].data=(d.mem_history||[]).slice(-60);
      memChart.data.datasets[1].data=(d.net_history||[]).slice(-60);
      memChart.update('none');
    }
    const alerts=d.alerts||[];
    document.getElementById('alert-count').textContent=d.alert_count||0;
    if((d.alert_count||0)!==lastAlertCount){
      lastAlertCount=d.alert_count||0;
      const el=document.getElementById('alerts-list');
      el.innerHTML=alerts.length===0?'<div class="alert-row"><div style="color:#333;font-size:10px;">Sin alertas detectadas</div></div>':[...alerts].reverse().slice(0,8).map(a=>'<div class="alert-row"><div class="badge '+a.severity+'">'+a.severity+'</div><div class="al-msg"><b>'+a.metric+'</b> '+a.value.toFixed(1)+'% &gt; '+a.threshold.toFixed(1)+'%</div><div class="al-time">'+fmtTime(a.time)+'</div></div>').join('');
    }
    // sync stress button state with server
    if(d.stressing!==undefined&&d.stressing!==isStressing){
      isStressing=d.stressing;
      const btn=document.getElementById('stress-btn');
      btn.textContent=isStressing?'⛔ DETENER SIMULACIÓN':'⚡ SIMULAR ANOMALÍA';
      btn.className='stress-btn '+(isStressing?'stop':'start');
    }
    document.getElementById('status-grid').innerHTML=
      stRow(true,'Agente TCP',l.hostname||'—')+
      stRow(true,'HTTP dashboard',':8080')+
      stRow(true,'TCP listener',':9000')+
      stRow(!d.stressing,'Modo',d.stressing?'🔥 STRESS ACTIVO':'normal')+
      stRow(true,'Isolation Forest',(d.alert_count||0)>0?'alertas: '+d.alert_count:'normal')+
      stRow(cpu<85,'CPU',cpu>85?'🚨 ANOMALIA':'normal')+
      stRow(mem<88,'MEM',mem>88?'🚨 ANOMALIA':'normal')+
      stRow(true,'Muestras',(d.cpu_history||[]).length+' pts');
    if(!isStressing){
      document.getElementById('status-pill').textContent='ENGINE RUNNING';
      document.getElementById('status-pill').className='pill';
    }
  }catch(e){
    document.getElementById('status-pill').textContent='OFFLINE';
    document.getElementById('status-pill').className='pill red';
  }
}
tick();setInterval(tick,3000);
</script>
</body>
</html>`
