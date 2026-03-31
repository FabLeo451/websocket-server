package server

import (
	//"encoding/json"
	"errors"
	//"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	//"path"
	"ekhoes-server/auth"
	"ekhoes-server/config"
	"ekhoes-server/db"
	"ekhoes-server/herenow"
	"ekhoes-server/system"
	"ekhoes-server/terminal"
	"ekhoes-server/websocket"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"context"
	"os/signal"
	"syscall"
)

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
func Start() int {

	log.Printf("Starting %s %s\n\tpid=%d\n\tlocal=%v\n\tpostgres=%v\n\tredis=%v\n",
		config.Name(),
		config.Version(),
		os.Getpid(),
		config.Local(),
		config.PosgresEnabled(),
		config.RedisEnabled())

	err := db.OpenStaff("")

	if err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(DynamicCORSMiddleware)

	r.Get("/", GetRoot)
	r.Post("/login", auth.Login)
	r.Post("/logout", auth.Logout)

	// Websocket endpoint
	r.Method("GET", "/ws", http.HandlerFunc(websocket.HandleConnection))

	// Ctl routes
	r.Route("/ctl", func(r chi.Router) {
		r.Get("/sessions", auth.GetSessionsHandler)
		r.Delete("/session/{id}", auth.DeleteSessionHandler)
		r.Delete("/sessions", auth.DeleteAllSessionsHandler)

		r.Get("/connections", websocket.GetConnectionsHandler)

		r.Get("/system", system.GetSystemInfo)
		r.Get("/top", system.TopCpuProcesses)
	})

	r.Get("/metrics", GetMetrics)

	r.Get("/terminal", terminal.OpenTerminal)

	// Init applications

	if !herenow.Init(r) {
		log.Println("Error initializing HereNow")
		return 1
	}

	// Create a context that will be removed on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf(":%d", config.Port())

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Listening on port %d...\n", config.Port())

		err := srv.ListenAndServe()

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			//log.Println("Error starting server:", err)
			errCh <- err
			return
		}

		errCh <- nil
	}()

	// Wait for events
	select {
	case err := <-errCh:
		if err != nil {
			log.Println("Error starting server:", err)
			return 1
		}
	case <-ctx.Done():
		log.Println("Termination signal received")
	}

	// Wait for signals
	//<-ctx.Done()
	//log.Println("Termination signal received")

	// Create a context for timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	log.Println("Shutting down server...")

	// Close server connections, no more accepted requests.
	// Wait pending connections to end within the timeout.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Can't shut down gracefully: %v. Forcing Close()", err)
		if cerr := srv.Close(); cerr != nil {
			log.Printf("Errore Close(): %v", cerr)
		}
	}

	db.CloseStuff()

	log.Println("Server stopped")

	return 0
}
