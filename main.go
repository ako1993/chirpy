package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
)

type apiConfig struct {
	fileserverhits atomic.Int32
}

var apiCfg apiConfig

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverhits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func ServeHttp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) WriteHits(w http.ResponseWriter, r *http.Request) {
	strVal := strconv.FormatInt(int64(cfg.fileserverhits.Load()), 10)
	w.Write([]byte(fmt.Sprintf("Hits: %v", strVal)))
}

func (cfg *apiConfig) resetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverhits.Store(0)
}

func main() {
	const filepathroot = "."
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(filepathroot)))))
	mux.HandleFunc("/healthz", ServeHttp)
	mux.HandleFunc("/metrics", apiCfg.WriteHits)
	mux.HandleFunc("/reset", apiCfg.resetHits)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
	}
}
