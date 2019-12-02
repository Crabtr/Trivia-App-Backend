package main

import (
	"encoding/json"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

type AdminAttempt struct {
	Action string `json:"action"`
	Data   struct {
		// Go doesn't have variable overloading, so we have to specify the data
		// type for values
		Username string `json:"username"`
		ValueStr string `json:"value_str,omitempty"`
		ValueInt int    `json:"value_int,omitempty"`
	} `json:"data"`
}

func (context *Context) AdminEndpoint(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Decode the received JSON body
	var adminAttempt AdminAttempt

	err := json.NewDecoder(r.Body).Decode(&adminAttempt)
	if err != nil {
		panic(err)
	}

	// Ensure the user's an admin
	var isAdmin bool

	isAdminStmt := `
		SELECT is_admin
		FROM users
		WHERE username = ?;`
	err = context.db.QueryRow(isAdminStmt, auth["iss"].(string)).Scan(&isAdmin)
	if err != nil {
		panic(err)
	}

	if isAdmin {
		// TODO: Ensure the target username exists before going further

		// Scheme for action strings: category.action.target
		if adminAttempt.Action == "user.modify.username" {
			updateUserStmt := `
				UPDATE users
				SET username = ?
				WHERE username = ?;`
			_, err = context.db.Exec(
				updateUserStmt,
				adminAttempt.Data.ValueStr,
				adminAttempt.Data.Username,
			)
			if err != nil {
				panic(err)
			}

			// Return a success payload
			response, err := json.Marshal(AuthResponse{
				Success: true,
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(response)

			return
		} else if adminAttempt.Action == "user.modify.score" {
			// TODO: This will need to be modified when separate leaderboards
			// are implemented
			updateUserStmt := `
				UPDATE users
				SET score = ?
				WHERE username = ?;`
			_, err = context.db.Exec(
				updateUserStmt,
				adminAttempt.Data.ValueInt,
				adminAttempt.Data.Username,
			)
			if err != nil {
				panic(err)
			}

			// Return a success payload
			response, err := json.Marshal(AuthResponse{
				Success: true,
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(response)

			return
		} else if adminAttempt.Action == "user.delete" {
			deleteUserStmt := `
				DELETE FROM users
				WHERE username = ?;`
			_, err = context.db.Exec(deleteUserStmt, adminAttempt.Data.Username)
			if err != nil {
				panic(err)
			}

			// Return a success payload
			response, err := json.Marshal(AuthResponse{
				Success: true,
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
				Message: "Invalid action given",
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

	// Return a failure payload
	response, err := json.Marshal(AuthResponse{
		Success: false,
		Message: "User isn't an admin",
	})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(response)

	return
}
