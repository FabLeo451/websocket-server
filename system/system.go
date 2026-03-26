package system

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"time"
	"websocket-server/assets"
	"websocket-server/auth"
	"websocket-server/config"
	"websocket-server/websocket"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

var buildTime string

var tmpl = template.Must(template.ParseFS(assets.TemplatesFS, "root.htm"))

type RootData struct {
	Package      string
	Version      string
	InstanceName string
	BuildTime    string
	StartTime    string
	UpTime       string
	Database     string
}

func humanizeDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func GetRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	/*
	   tmpl, err := template.ParseFiles("template/root.htm")
	   if err != nil {
	       http.Error(w, "File not found", 404)
	       return
	   }
	*/

	formattedStartTime := config.Runtime.StartTime.UTC().Format("2006-01-02 15:04:05") + " UTC"

	data := RootData{
		Package:      config.Name(),
		Version:      config.Version(),
		InstanceName: config.Runtime.InstanceName,
		BuildTime:    buildTime,
		StartTime:    formattedStartTime,
		UpTime:       humanizeDuration(time.Since(config.Runtime.StartTime)),
		Database:     config.Runtime.Database,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

type SystemInfo struct {
	Hostname      string  `json:"hostname"`
	OS            string  `json:"os"`
	Platform      string  `json:"platform"`
	PlatformVer   string  `json:"platform_version"`
	KernelVersion string  `json:"kernel_version"`
	CPULoad       float64 `json:"cpu_load"`
	RAMUsed       uint64  `json:"ram_used"`
	RAMTotal      uint64  `json:"ram_total"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
}

func GetSystemInfo(w http.ResponseWriter, r *http.Request) {
	_, err := auth.CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Info host
	hostInfo, _ := host.Info()

	// CPU load (ultimi 1 secondo)
	cpuLoad, _ := cpu.Percent(time.Second, false)

	// RAM
	vm, _ := mem.VirtualMemory()

	// Disco root ("/" su Linux/macOS, "C:\" su Windows)
	usage, _ := disk.Usage("/")

	info := SystemInfo{
		Hostname:      hostInfo.Hostname,
		OS:            hostInfo.OS,
		Platform:      hostInfo.Platform,
		PlatformVer:   hostInfo.PlatformVersion,
		KernelVersion: hostInfo.KernelVersion,
		CPULoad:       cpuLoad[0],
		RAMUsed:       vm.Used,
		RAMTotal:      vm.Total,
		DiskUsed:      usage.Used,
		DiskTotal:     usage.Total,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}

type ProcInfo struct {
	PID  int32
	User string
	CPU  float64
	Name string
}

func TopCpuProcesses(w http.ResponseWriter, r *http.Request) {
	_, err := auth.CheckAuthorization(r)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	procs, err := process.Processes()
	if err != nil {
		http.Error(w, "Can't get processes", http.StatusInternalServerError)
		return
	}

	list := make([]ProcInfo, 0, len(procs))

	// Prima lettura CPU (necessaria per inizializzare i delta)
	for _, p := range procs {
		p.CPUPercent()
	}

	// Attendi un breve intervallo per misurare l'uso reale
	time.Sleep(500 * time.Millisecond)

	// Seconda lettura CPU (valore reale)
	for _, p := range procs {
		cpu, err := p.CPUPercent()
		if err != nil {
			continue
		}

		name, _ := p.Name()
		user, _ := p.Username()

		list = append(list, ProcInfo{
			PID:  p.Pid,
			User: user,
			CPU:  cpu,
			Name: name,
		})
	}

	// Ordina per CPU discendente
	sort.Slice(list, func(i, j int) bool {
		return list[i].CPU > list[j].CPU
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if len(list) > 10 {
		json.NewEncoder(w).Encode(list[:10])
	} else {
		json.NewEncoder(w).Encode(list)
	}
}

func GetMetrics(w http.ResponseWriter, r *http.Request) {

	metrics := map[string]interface{}{
		"count": websocket.GetConnectionsCount(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}
