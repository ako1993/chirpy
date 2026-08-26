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

	"chirpy/internal/auth"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverhits atomic.Int32
	dbQueries      *database.Queries
	platform       string `env:"PLATFORM"`
	token_secret   string `env:"TOKEN_SECRET"`
}

type User struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"-"`
	Token          string    `json:"token"`
	Refresh_token  string    `json:"refresh_token"`
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
	mux.HandleFunc("GET /api/chirps", get_all_chirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", get_single_chirp)
	mux.HandleFunc("POST /api/login", login_user)
	mux.HandleFunc("POST /api/refresh", check_refresh_token)
	mux.HandleFunc("POST /api/revoke", revoke_refresh_token)
	mux.HandleFunc("PUT /api/users", authorize_user)

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
	request_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	apiCfg.token_secret = os.Getenv("TOKEN_SECRET")
	validation_id, err := auth.ValidateJWT(request_token, apiCfg.token_secret)
	if err != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}
	user_id := uuid.NullUUID{UUID: validation_id, Valid: true}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp too long!")
	} else {
		chirp_params := database.CreateChirpParams{
			Body:   params.Body,
			UserID: user_id,
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

func get_all_chirps(w http.ResponseWriter, r *http.Request) {
	return_chirps := []Chirp{}
	chirps, err := apiCfg.dbQueries.GetAllChirps(r.Context())
	if err != nil {
		fmt.Println(err)
	}
	for _, chirp := range chirps {
		new_chirp := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		return_chirps = append(return_chirps, new_chirp)
	}
	respondWithJSON(w, 200, return_chirps)
}

func get_single_chirp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	new_id, err := uuid.Parse(id)
	if err != nil {
		fmt.Println(err)
	}
	chirp, err := apiCfg.dbQueries.GetChirp(r.Context(), new_id)
	if err != nil {
		fmt.Println(err)
	}
	if chirp.ID == uuid.Nil {
		respondWithError(w, 404, "No chirp found")
	} else {
		return_chirp := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}
		respondWithJSON(w, 200, return_chirp)
	}
}

func create_user(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	hashed_pw, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Println(err)
	}
	user_params := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashed_pw,
	}
	new_user, err := apiCfg.dbQueries.CreateUser(r.Context(), user_params)
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

func login_user(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	user_to_match, err := apiCfg.dbQueries.Lookup_user_by_email(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		fmt.Printf("LOOKUP FAILED:%v", err)
	}
	confirm_match, err := auth.CheckPasswordHash(params.Password, user_to_match.HashedPassword)
	if err != nil {
		respondWithError(w, 401, "Incorrect email or password")
		fmt.Printf("MATCH FAILED:%v", err)
		fmt.Printf("USER PW: %v; HASH: %v", params.Password, user_to_match.HashedPassword)
	}
	if confirm_match == true {
		apiCfg.token_secret = os.Getenv("TOKEN_SECRET")
		Token, err := auth.MakeJWT(user_to_match.ID, apiCfg.token_secret, 60*time.Minute)
		if err != nil {
			fmt.Printf("Token generation failed: %v", err)
		}
		refresh_token := auth.MakeRefreshToken()
		refresh_token_params := database.CreateRefreshTokenParams{
			Token:  refresh_token,
			UserID: uuid.NullUUID{UUID: user_to_match.ID, Valid: true},
		}
		_, err = apiCfg.dbQueries.CreateRefreshToken(r.Context(), refresh_token_params)
		if err != nil {
			fmt.Printf("Refresh Token generation failed: %v", err)
		}
		confirmed_user := User{
			ID:            user_to_match.ID,
			CreatedAt:     user_to_match.CreatedAt,
			UpdatedAt:     user_to_match.UpdatedAt,
			Email:         user_to_match.Email,
			Token:         Token,
			Refresh_token: refresh_token,
		}
		respondWithJSON(w, 200, confirmed_user)
	} else {
		respondWithError(w, 401, "Incorrect email or password")
	}

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

func check_refresh_token(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}
	user_id, err := apiCfg.dbQueries.GetUserFromRefreshToken(r.Context(), refresh_token)
	if err != nil {
		respondWithError(w, 401, err.Error())
		return
	}

	if user_id.UUID == uuid.Nil {
		respondWithError(w, 401, "Token Expired")
		return
	} else {
		apiCfg.token_secret = os.Getenv("TOKEN_SECRET")
		new_access_token, err := auth.MakeJWT(user_id.UUID, apiCfg.token_secret, time.Hour)
		if err != nil {
			respondWithError(w, 401, err.Error())
			return
		}
		type token struct {
			Token string `json:"token"`
		}
		token_ := token{
			Token: new_access_token,
		}
		respondWithJSON(w, 200, token_)
	}
}

func revoke_refresh_token(w http.ResponseWriter, r *http.Request) {
	refresh_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		fmt.Printf("Error getting refresh token from header:%v", err)
		return
	}
	err = apiCfg.dbQueries.RevokeRefreshRoken(r.Context(), refresh_token)
	if err != nil {
		fmt.Printf("Error revoking refresh token:%v", err)
		return
	}
	respondWithJSON(w, 204, "Refresh Token Revoked")
}

func authorize_user(w http.ResponseWriter, r *http.Request) {
	access_token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, 401, "Token Malformed or missing")
		return
	}
	apiCfg.token_secret = os.Getenv("TOKEN_SECRET")
	user, err := auth.ValidateJWT(access_token, apiCfg.token_secret)
	if err != nil {
		respondWithError(w, 401, "Invalid access token")
		return
	}

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	hashed_pw, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Printf("Failed to hash:%v", err)
		return
	}
	update_params := database.UpdateEmailandPasswordParams{
		ID:             user,
		Email:          params.Email,
		HashedPassword: hashed_pw,
	}
	err = apiCfg.dbQueries.UpdateEmailandPassword(r.Context(), update_params)
	if err != nil {
		respondWithError(w, 401, "Failed to update email and password")
		return
	}
	user_info, err := apiCfg.dbQueries.Lookup_user_by_email(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, 404, "User not found")
		return
	}
	returned_user := User{
		ID:        user_info.ID,
		CreatedAt: user_info.CreatedAt,
		UpdatedAt: user_info.UpdatedAt,
		Email:     user_info.Email,
	}
	respondWithJSON(w, 200, returned_user)

}
