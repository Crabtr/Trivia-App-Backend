package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// Context makes variables across scope boundaries
type Context struct {
	db *sql.DB
}

func main() {
	// Context provides global variables in a safe capacity
	context := Context{}

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

	// Route registrations
	router.HandleFunc("/api/user/create", context.UserCreateEndpoint).Methods("POST")
	router.HandleFunc("/api/user/auth", context.UserAuthEndpoint).Methods("POST")
	// router.HandleFunc("/api/user/delete", context.UserAuthenticationEndpoint).Methods("POST")

	log.Fatal(http.ListenAndServe(":8080", router)) // Start the server
}
