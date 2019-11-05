package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Context makes variables across scope boundaries
type Context struct {
	db       *sql.DB
	sessions map[string]*Session // map session ID to session
}

func main() {
	// Context provides global variables in a safe capacity
	context := Context{}
	context.sessions = make(map[string]*Session, 0)

	var err error

	// Generate a connection string
	dbPath := fmt.Sprintf(
		"%s:%s@tcp(%s)/trivia?charset=utf8mb4",
		"trivia",         // username
		"supersecret123", // password
		"0.0.0.0",        // address
	)

	context.db, err = sql.Open("mysql", dbPath)
	if err != nil {
		panic(err)
	}
	defer context.db.Close()

	log.Println("Connected to the database")

	// TODO: It's probably preferrable if these don't live here
	createUsersStmt := `
		CREATE TABLE IF NOT EXISTS users
		(
			username        VARCHAR(255) NOT NULL UNIQUE,
			email           VARCHAR(255) NOT NULL UNIQUE,
			password_hash   TEXT NOT NULL,
			score           INT NOT NULL,
			games_played    INT NOT NULL
		);`
	_, err = context.db.Exec(createUsersStmt)
	if err != nil {
		panic(err)
	}

	createLeaderboardStmt := `
		CREATE OR REPLACE VIEW leaderboard
		AS SELECT username, score
		FROM users
		ORDER BY score DESC;`
	_, err = context.db.Exec(createLeaderboardStmt)
	if err != nil {
		panic(err)
	}

	// HTTP router
	router := mux.NewRouter()

	headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	methods := handlers.AllowedMethods([]string{"GET", "POST"})
	origins := handlers.AllowedOrigins([]string{"*"})

	// Route registrations
	// User endpoints
	router.HandleFunc("/api/user/create", context.UserCreateEndpoint).Methods("POST")
	router.HandleFunc("/api/user/auth", context.UserAuthEndpoint).Methods("POST")
	router.HandleFunc("/api/user/info", context.UserInfoEndpoint).Methods("POST")
	// router.HandleFunc("/api/user/delete", context.UserAuthenticationEndpoint).Methods("POST")

	// Gameplay endpoints
	router.HandleFunc("/api/game/start", ValidateJWTMiddleware(context.GameStart)).Methods("GET")
	router.HandleFunc("/api/game/join", ValidateJWTMiddleware(context.GameJoin)).Methods("POST")
	router.HandleFunc("/api/game/leave", ValidateJWTMiddleware(context.GameLeave)).Methods("POST")
	router.HandleFunc("/api/game/info", ValidateJWTMiddleware(context.GameGetInfo)).Methods("GET")
	router.HandleFunc("/api/game/modify", ValidateJWTMiddleware(context.GameModify)).Methods("GET")
	router.HandleFunc("/api/game/question", ValidateJWTMiddleware(context.GameGetQuestion)).Methods("GET")
	router.HandleFunc("/api/game/answer", ValidateJWTMiddleware(context.GamePostAnswer)).Methods("POST")

	router.HandleFunc("/api/meta", context.GameMeta).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(headers, methods, origins)(router))) // Start the server
}
