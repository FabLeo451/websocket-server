package system

import (
	"ekhoes-server/auth"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

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
