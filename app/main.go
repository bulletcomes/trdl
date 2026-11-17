package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// version is set at build time with -ldflags "-X main.version=<version>".
var version = "dev-0"

func main() {
	server := &http.Server{
		Addr:    ":8080",
		Handler: loggingMiddleware(newRouter()),
		// Bounds how long a client may take to send headers (slowloris protection).
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("TRDL %s listening on %s", version, server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Allow in-flight requests a short period to complete during pod termination.
	ctx, cancel := shutdownContext()
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		_ = server.Close()
	}
}

// newRouter maps the HTTP contract. Method-qualified patterns let the standard
// library handle the edge cases: "GET" patterns also match HEAD; other methods
// on a known path get 405 with an Allow header; unknown paths get 404; "{$}"
// makes "/" match only the root, not every path.
func newRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", textHandler("42"))
	mux.HandleFunc("GET /healthz", textHandler("ok"))
	mux.HandleFunc("GET /readyz", textHandler("ready"))
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		textHandler(version)(w, r)
	})
	return mux
}

func textHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%q duration=%s", r.Method, r.URL.Path, time.Since(start))
	})
}
