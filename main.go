package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverhits atomic.Int32
	dbQueries      *database.Queries
	platform       string `env:"PLATFORM"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID     `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Body      string        `json:"body"`
	UserID    uuid.NullUUID `json:"user_id"`
}

var apiCfg apiConfig

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
	}

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}

	apiCfg.dbQueries = database.New(db)

	const filepathroot = "."
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(filepathroot)))))
	mux.HandleFunc("GET /api/healthz", ServeHttp)
	mux.HandleFunc("POST /api/validate_chirp", cleanUserRequest)
	mux.HandleFunc("GET /admin/metrics", apiCfg.WriteHits)
	mux.HandleFunc("POST /admin/reset", clear_users)
	mux.HandleFunc("POST /api/users", create_user)
	mux.HandleFunc("POST /api/chirps", save_chirp)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Println(err)
	}
}

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

func cleanUserRequest(w http.ResponseWriter, r *http.Request) {
	type userInput struct {
		Body string `json:"body"`
	}
	type cleanedResponse struct {
		Cleaned_body string `json:"cleaned_body"`
	}
	decoder := json.NewDecoder(r.Body)
	userInput_ := userInput{}
	err := decoder.Decode(&userInput_)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	body := replaceBadWord(userInput_.Body)
	response := cleanedResponse{
		Cleaned_body: body,
	}
	if len(userInput_.Body) < 140 {
		respondWithJSON(w, 200, response)
	} else {
		respondWithError(w, 400, "ERROR: message body too long")
	}
}

func replaceBadWord(message string) string {
	bad_words := []string{"kerfuffle", "sharbert", "fornax"}
	message_parts := strings.Split(message, " ")
	for i := range message_parts {
		if slices.Contains(bad_words, strings.ToLower(message_parts[i])) {
			message_parts[i] = "****"
		}
	}
	message = strings.Join(message_parts, " ")

	return message
}

func save_chirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body    string        `json:"body"`
		User_id uuid.NullUUID `json:"user_id"`
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
		chirp_params := database.CreateChirpParams{
			Body:   params.Body,
			UserID: params.User_id,
		}

		new_chirp, err := apiCfg.dbQueries.CreateChirp(r.Context(), chirp_params)
		if err != nil {
			fmt.Println(err)
		}
		new_chirp_ := Chirp{
			ID:        new_chirp.ID,
			CreatedAt: new_chirp.CreatedAt,
			UpdatedAt: new_chirp.UpdatedAt,
			Body:      new_chirp.Body,
			UserID:    new_chirp.UserID,
		}
		respondWithJSON(w, 201, new_chirp_)
	}

}

func create_user(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	new_user, err := apiCfg.dbQueries.CreateUser(r.Context(), params.Email)
	if err != nil {
		fmt.Println(err)
	}
	new_user_ := User{
		ID:        new_user.ID,
		CreatedAt: new_user.CreatedAt,
		UpdatedAt: new_user.UpdatedAt,
		Email:     new_user.Email,
	}
	respondWithJSON(w, 201, new_user_)
}

func clear_users(w http.ResponseWriter, r *http.Request) {
	apiCfg.platform = os.Getenv("PLATFORM")
	if apiCfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
	} else {
		err := apiCfg.dbQueries.ClearAllUsers(r.Context())
		if err != nil {
			fmt.Println(err)
		}
	}
}
