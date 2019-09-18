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

// Struct received when a user attempts to create an account
type CreateAttempt struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// Struct received when a user attempts to authenticate
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

// Generic struct for a row in the 'users' table
type SQLUser struct {
	Username     string
	Email        string
	PasswordHash string
	Score        int
	GamesPlayed  int
}

func (context *Context) UserCreateEndpoint(w http.ResponseWriter, r *http.Request) {
	// Decode the received JSON body
	var createAttempt CreateAttempt

	err := json.NewDecoder(r.Body).Decode(&createAttempt)
	if err != nil {
		panic(err)
	}

	// If there doesn't exist a user with the given username, then continue
	// with the user creation routine
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

		// Add the user's information to the database
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

		// Return a success payload
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
		// Return a failure payload
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

// TODO: It might be preferrable to not run this as a function
func ValidatePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (context *Context) UserAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	// Decode the received JSON body
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
			// This error only catches if the user doesn't exist
			// Send a failure response
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
		// Generate a JSON web token (JWT)
		// TODO: It's most lkely preferrable to use jwt.StandardClaims instead
		// of jwt.MapClaims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": authAttempt.Username,
			"password": authAttempt.Password,
		})

		// Sign the token using our secret key
		tokenString, err := token.SignedString(signingKey)
		if err != nil {
			panic(err)
		}

		// Return a success payload
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
		// Return a failure payload
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
