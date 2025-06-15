package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
)

//var version = "1.0.0"

type Callback func([]string) int

type Command struct {
	f    Callback
	args string
	help string
}

var flagVersion = false

// Middleware per CORS
func handleCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Imposta gli header CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")                                // Consenti tutte le origini
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS") // Metodi consentiti
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")                    // Intestazioni consentite

		// Se la richiesta è un "preflight" (OPTIONS), rispondi subito
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Passa alla gestione della richiesta
		next.ServeHTTP(w, r)
	})
}

/**
 * Start service
 */
func Start(args []string) int {

	fmt.Printf("\n   *** %s %s ***\n\n", conf.Package.Name, conf.Package.Version)

	conn := DB_ConnectKeepAlive()

	if conn == nil {
		return 1
	}

	RedisConnect()

	LogWrite("Starting service on port %d...\n", conf.Port)

	router := http.NewServeMux()
	router.HandleFunc("GET /", getRoot)
	router.HandleFunc("GET /metrics", getMetrics)
	router.HandleFunc("OPTIONS /login", optionsPreflight)
	router.HandleFunc("POST /login", login)
	router.HandleFunc("OPTIONS /logout", optionsPreflight)
	router.HandleFunc("POST /logout", logout)
	router.HandleFunc("GET /connect", handleConnection)

	router.HandleFunc("OPTIONS /hotspot", optionsPreflight)
	router.HandleFunc("POST /hotspot", postHotspot)
	router.HandleFunc("OPTIONS /hotspots", optionsPreflight)
	router.HandleFunc("GET /hotspots", getHotspots)

	addr := fmt.Sprintf(":%d", conf.Port)

	// Usa il middleware per CORS
	http.Handle("/", handleCORS(router))

	LogWrite("Service ready\n")

	err := http.ListenAndServe(addr, router)

	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}

	return 0
}

/**
 * Check connections
 */
func Check(args []string) int {

	fmt.Printf("Connecting to database %s:%d... ", conf.DB.Host, conf.DB.Port)

	db, err := DB_Open()

	if err == nil {

		fmt.Println("Success")

		fmt.Printf("Pinging database... ")

		err = db.Ping()

		if err == nil {
			fmt.Println("Success")
		} else {
			fmt.Printf("Error: %s\n", err.Error())
		}
	} else {
		fmt.Printf("Error: %s\n", err.Error())
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
	mapCommands["check"] = Command{Check, "", "Check connections"}

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
