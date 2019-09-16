package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// TODO: This isn't acceptable
var signingKey = []byte("SuperDuperSecretSigningKey")

type CreateAttempt struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type AuthAttempt struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponseData struct {
	Username string `json:"username,omitempty"`
	Token    string `json:"token,omitempty"`
}

// Generic struct for responding to authentication requests
type AuthResponse struct {
	Success bool `json:"success"`
	// If either of these variables are empty, then the 'omitempty' flag
	// ensures it isn't included in the JSON output
	Message string            `json:"message,omitempty"`
	Data    *AuthResponseData `json:"data,omitempty"`
}

type SQLUser struct {
	Username     string
	Email        string
	PasswordHash string
	Score        int
	GamesPlayed  int
}

func (context *Context) UserCreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var createAttempt CreateAttempt

	err := json.NewDecoder(r.Body).Decode(&createAttempt)
	if err != nil {
		panic(err)
	}

	// Verify the given information
	var count int

	stmt := `
		SELECT COUNT(*)
		FROM users
		WHERE username=?;`
	err = context.db.QueryRow(stmt, createAttempt.Username).Scan(&count)
	if err != nil {
		panic(err)
	}

	if count == 0 {
		// Generate a password hash using bcrypt
		passwordHash, err := bcrypt.GenerateFromPassword(
			[]byte(createAttempt.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			panic(err)
		}

		_, err = context.db.Exec(
			`INSERT INTO users VALUES (?,?,?,?,?);`,
			createAttempt.Username,
			createAttempt.Email,
			passwordHash,
			0,
			0,
		)
		if err != nil {
			panic(err) // TODO: Better here
		}

		response, err := json.Marshal(AuthResponse{
			Success: true,
			Data: &AuthResponseData{
				Username: createAttempt.Username,
			},
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(response)

		return
	} else {
		response, err := json.Marshal(AuthResponse{
			Success: false,
			Message: "Username already exists",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(response)

		return
	}
}

func ValidatePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (context *Context) UserAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	var authAttempt AuthAttempt

	err := json.NewDecoder(r.Body).Decode(&authAttempt)
	if err != nil {
		panic(err)
	}

	// Verify the given information
	var sqlUser SQLUser

	stmt := `
		SELECT *
		FROM users
		WHERE username=?;`
	err = context.db.QueryRow(stmt, authAttempt.Username).Scan(
		&sqlUser.Username,
		&sqlUser.Email,
		&sqlUser.PasswordHash,
		&sqlUser.Score,
		&sqlUser.GamesPlayed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			response, err := json.Marshal(AuthResponse{
				Success: false,
				Message: "User doesn't exist",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(response)

			return
		} else {
			panic(err)
		}
	}

	if ValidatePassword(authAttempt.Password, sqlUser.PasswordHash) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": authAttempt.Username,
			"password": authAttempt.Password,
		})

		tokenString, err := token.SignedString(signingKey)
		if err != nil {
			panic(err)
		}

		response, err := json.Marshal(AuthResponse{
			Success: true,
			Data: &AuthResponseData{
				Username: authAttempt.Username,
				Token:    tokenString,
			},
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	} else {
		response, err := json.Marshal(AuthResponse{
			Success: false,
			Message: "Invalid password",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(response)

		return
	}
}
