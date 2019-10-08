package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// TODO: This isn't acceptable
var signingKey = []byte("SuperDuperSecretSigningKey")

// Struct received when a user attempts to create an account
type UserCreateAttempt struct {
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
	var createAttempt UserCreateAttempt

	err := json.NewDecoder(r.Body).Decode(&createAttempt)
	if err != nil {
		panic(err)
	}

	// Ensure the given username is unique
	var count int

	validUsernameStmt := `SELECT COUNT(*) FROM users WHERE username=?;`
	err = context.db.QueryRow(validUsernameStmt, createAttempt.Username).Scan(&count)
	if err != nil {
		panic(err)
	}

	if count > 0 {
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

	count = 0 // Reset count

	// Ensure the given email is unique
	validEmailStmt := `SELECT COUNT(*) FROM users WHERE email=?`
	err = context.db.QueryRow(validEmailStmt, createAttempt.Email).Scan(&count)
	if err != nil {
		panic(err)
	}

	if count > 0 {
		// Return a failure payload
		response, err := json.Marshal(AuthResponse{
			Success: false,
			Message: "Email already exists",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(response)

		return
	}

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
	w.WriteHeader(http.StatusOK)
	w.Write(response)

	return
}

// TODO: It might be preferrable to not run this as a function
func ValidatePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (context *Context) UserAuthEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		log.Println("Auth CORS request")

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type")
			w.WriteHeader(http.StatusOK)

			return
		}
	}

	log.Println("Auth request")

	// Decode the received JSON body

	var authAttempt AuthAttempt

	err := json.NewDecoder(r.Body).Decode(&authAttempt)
	if err != nil {
		panic(err)
	}

	// Verify the given information
	var sqlUser SQLUser

	stmt := `SELECT * FROM users WHERE username=?;`
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
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(response)

			return
		} else {
			panic(err)
		}
	}

	if ValidatePassword(authAttempt.Password, sqlUser.PasswordHash) {
		// Generate a JSON web token (JWT)
		// TODO: It's most likely preferrable if tokens expire.
		// https://godoc.org/github.com/dgrijalva/jwt-go#MapClaims
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iss": authAttempt.Username,    // Issuer
			"iat": time.Now().UTC().Unix(), // IssuedAt
		})

		// Sign the token using our secret key
		tokenString, err := token.SignedString(signingKey)
		if err != nil {
			panic(err)
		}

		// Return a success payload
		response, err := json.Marshal(AuthResponse{

			Data: &AuthResponseData{
				Username: authAttempt.Username,
				Token:    tokenString,
			},
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	} else {
		log.Println("Invalid password")

		// Return a failure payload
		response, err := json.Marshal(AuthResponse{
			Success: false,
			Message: "Invalid password",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(response)

		return
	}
}

func ValidateJWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader != "" {
			bearerToken := strings.Split(authorizationHeader, " ")

			if len(bearerToken) == 2 && bearerToken[0] == "Bearer" {
				token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("Error when parsing bearer token")
					}

					return []byte(signingKey), nil
				})
				if err != nil {
					response, err := json.Marshal(AuthResponse{
						Success: false,
						Message: err.Error(),
					})
					if err != nil {
						panic(err)
					}

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write(response)

					return
				}

				if token.Valid {
					r = r.WithContext(context.WithValue(r.Context(), "decoded", token.Claims))
					next(w, r)

					return
				} else {
					response, err := json.Marshal(AuthResponse{
						Success: false,
						Message: "Invalid authorization header",
					})
					if err != nil {
						panic(err)
					}

					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write(response)

					return
				}
			} else {
				response, err := json.Marshal(AuthResponse{
					Success: false,
					Message: "Invalid authorization header format",
				})
				if err != nil {
					panic(err)
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write(response)

				return
			}
		} else {
			response, err := json.Marshal(AuthResponse{
				Success: false,
				Message: "An authorization header is required",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(response)

			return
		}
	})
}
