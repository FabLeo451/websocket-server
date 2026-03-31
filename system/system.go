package system

import (
	"ekhoes-server/auth"
	"encoding/json"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

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
