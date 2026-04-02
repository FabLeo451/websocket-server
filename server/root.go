package server

import (
	"ekhoes-server/assets"
	"ekhoes-server/config"
	"ekhoes-server/module"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

var tmpl = template.Must(template.ParseFS(assets.TemplatesFS, "root.htm"))

type RootData struct {
	Package      string
	Version      string
	InstanceName string
	BuildTime    string
	StartTime    string
	UpTime       string
	Database     string
	Cache        string
	Modules      string
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

	//formattedStartTime := config.Runtime.StartTime.UTC().Format("2006-01-02 15:04:05") + " UTC"

	data := RootData{
		Package:      config.Name(),
		Version:      config.Version(),
		InstanceName: config.Runtime.InstanceName,
		BuildTime:    config.BuildTime(),
		StartTime:    config.Runtime.StartTime.Format(time.RFC3339),
		UpTime:       humanizeDuration(time.Since(config.Runtime.StartTime)),
		Database:     config.Runtime.Database,
		Cache:        config.Runtime.Cache,
		Modules:      module.GetLoadedModules(),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}
