package main

import (
	"encoding/json"
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
	w.Header().Set("Content-Type", "text/html")
	strVal := strconv.FormatInt(int64(cfg.fileserverhits.Load()), 10)
	result := fmt.Sprintf(`<html>
				<body>
					<h1>Welcome, Chirpy Admin</h1>
					<p>Chirpy has been visited %v times!</p>
				</body>
				</html>`, strVal)
	w.Write([]byte(result))
}

func (cfg *apiConfig) resetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverhits.Store(0)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	// Create helper function to return error for bad JSON
	type returnErr struct {
		ErrMessage string `json:"error"`
	}
	errBody := returnErr{
		ErrMessage: message,
	}
	dat, err := json.Marshal(errBody)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	//Create helper function to respond to a good request with JSON
	dat, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func validate_chirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp too long!")
	} else {
		type returnJson struct {
			Is_valid bool `json:"valid"`
		}
		jsonBody := returnJson{
			Is_valid: true,
		}
		respondWithJSON(w, 200, jsonBody)
	}

}

func main() {
	const filepathroot = "."
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(filepathroot)))))
	mux.HandleFunc("GET /api/healthz", ServeHttp)
	mux.HandleFunc("POST /api/validate_chirp", validate_chirp)
	mux.HandleFunc("GET /admin/metrics", apiCfg.WriteHits)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHits)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
	}
}
