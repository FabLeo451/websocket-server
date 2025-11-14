package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"websocket-server/herenow"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Callback func([]string) int

type Command struct {
	f    Callback
	args string
	help string
}

var flagVersion = false

func getRoot(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//fmt.Printf("%s: %s %s %s\n", r.RemoteAddr, r.UserAgent(), r.Method, r.URL)
	//io.WriteString(w, "This is my website!\n")

	response, _ := json.Marshal(conf.Package)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

type Host struct {
	Mem  *mem.VirtualMemoryStat
	Disk *disk.UsageStat
}

func getMetrics(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	metrics := map[string]interface{}{
		"activeConnections": herenow.GetActiveConnectionsCount(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

func DynamicCORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		// Preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

/**
 * Start service
 */
func Start(args []string) int {

	fmt.Printf("\n   *** %s %s ***\n\n", conf.Package.Name, conf.Package.Version)

	if !herenow.Init() {
		return 1
	}

	log.Printf("Starting service on port %d...\n", conf.Port)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(DynamicCORSMiddleware)

	// Static routes
	r.Get("/", getRoot)
	r.Get("/metrics", getMetrics)
	r.Post("/login", herenow.Login)
	r.Post("/logout", herenow.Logout)

	// Alternative: method prefix syntax (chi supports it too)
	r.Method("GET", "/connect", http.HandlerFunc(herenow.HandleConnection))

	// /hotspot routes
	r.Route("/hotspot", func(r chi.Router) {
		// GET /hotspot
		r.Get("/", herenow.GetHotspot)

		// POST /hotspot
		r.Post("/", herenow.PostHotspot)

		// Routes with /hotspot/{id}
		r.Route("/{id}", func(r chi.Router) {
			// GET /hotspot/{id}
			r.Get("/", herenow.GetHotspot)

			// PUT /hotspot/{id}
			r.Put("/", herenow.PutHotspot)

			// DELETE /hotspot/{id}
			r.Delete("/", herenow.DeleteHotspot)

			// POST/DELETE /hotspot/{id}/like
			r.Post("/like", herenow.LikeHotspot)
			r.Delete("/like", herenow.LikeHotspot)

			// POST /hotspot/{id}/clone
			r.Post("/clone", herenow.CloneHotspotHandler)

			// POST/DELETE /hotspot/{id}/subscription
			r.Post("/subscription", herenow.SubscribeUnsubscribeHandler)
			r.Delete("/subscription", herenow.SubscribeUnsubscribeHandler)
		})
	})

	r.Get("/categories", herenow.GetCategoriesHandler)
	r.Get("/mysubscriptions", herenow.GetMySubscriptions)

	addr := fmt.Sprintf(":%d", conf.Port)

	log.Printf("Service ready\n")

	err := http.ListenAndServe(addr, r)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

	return 0
}

func main() {

	Init()

	if os.Getenv("API_DB_PASSWORD") != "" {
		conf.DB.Password = os.Getenv("API_DB_PASSWORD")
	}

	//globals.Home, _ = filepath.Abs(path.Dir(os.Args[0]))

	mapCommands := make(map[string]Command)
	mapCommands["start"] = Command{Start, "", "Start service"}

	flag.BoolVar(&flagVersion, "v", false, "Show version")
	flag.IntVar(&conf.Port, "P", 9876, "Set server port")

	flag.Usage = func() {
		fmt.Printf("%s %s\n", path.Base(os.Args[0]), conf.Package.Version)
		fmt.Printf("Usage: %s [options] command [arguments]\n", path.Base(os.Args[0]))

		fmt.Println("\nCommands:")

		for key, element := range mapCommands {
			fmt.Printf("  %-8s %-10s %s\n", key, element.args, element.help)
		}

		fmt.Println("\nOptions:")

		flag.VisitAll(func(f *flag.Flag) {
			a, d := "", ""

			if f.Value.String() != "false" {
				d = "(default: " + f.Value.String() + ")"
				a = "<value>"
			}
			fmt.Printf("  -%s %-10s %s %s\n", f.Name, a, f.Usage, d) // f.Name, f.Value
		})
	}

	flag.Parse()

	if flagVersion {
		fmt.Println(conf.Package.Version)
	}

	args := flag.Args()

	if len(args) == 0 {
		os.Exit(0)
	}

	var exitValue int = 0

	if c, found := mapCommands[args[0]]; found {
		exitValue = c.f(args)
	} else {
		fmt.Fprintln(os.Stderr, "Unknown command: ", args[0])
		exitValue = 1
	}

	os.Exit(exitValue)
}
