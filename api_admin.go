package main

import (
	"encoding/json"
	"net/http"

	"github.com/dgrijalva/jwt-go"
)

type Question struct {
	Category         string `json:"category"`
	QuestionType     string `json:"type"`
	Difficulty       string `json:"difficulty"`
	QuestionBody     string `json:"question"`
	CorrectAnswer    string `json:"correct_answer"`
	IncorrectAnswer1 string `json:"incorrect_answer1"`
	IncorrectAnswer2 string `json:"incorrect_answer2"`
	IncorrectAnswer3 string `json:"incorrect_answer3"`
}

type AdminAttempt struct {
	Action string `json:"action"`
	Data   struct {
		Username string `json:"username"`
		ValueStr string `json:"value_str,omitempty"`
		ValueInt int    `json:"value_int,omitempty"`
	} `json:"data"`
}

type AdminUser struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Score    int64  `json:"score"`
}

type AdminResponseData struct {
	Users []AdminUser `json:"users,omitempty"`
}

type AdminResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
	Data    *AdminResponseData `json:"data,omitempty"`
}

func (context *Context) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Ensure the user's an admin
	var isAdmin bool

	isAdminStmt := `
		SELECT is_admin
		FROM users
		WHERE username = ?;`
	err := context.db.QueryRow(isAdminStmt, auth["iss"].(string)).Scan(&isAdmin)
	if err != nil {
		panic(err)
	}

	if isAdmin {
		var payload AdminResponseData

		allUsersStmt := `
			SELECT *
			FROM users
			WHERE is_admin = false;`
		rows, err := context.db.Query(allUsersStmt)
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		for rows.Next() {
			var sqlUser SQLUser
			var adminUser AdminUser

			err := rows.Scan(
				&adminUser.Username,
				&adminUser.Email,
				&sqlUser.PasswordHash,
				&adminUser.Score,
				&sqlUser.GamesPlayed,
				&sqlUser.IsAdmin,
			)
			if err != nil {
				panic(err)
			}

			payload.Users = append(payload.Users, adminUser)
		}

		err = rows.Err()
		if err != nil {
			panic(err)
		}

		response, err := json.Marshal(AdminResponse{
			Success: true,
			Data:    &payload,
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	}

	// Return a failure payload
	response, err := json.Marshal(AdminResponse{
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

func (context *Context) AdminEndpoint(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Ensure the user's an admin
	var isAdmin bool

	isAdminStmt := `
		SELECT is_admin
		FROM users
		WHERE username = ?;`
	err := context.db.QueryRow(isAdminStmt, auth["iss"].(string)).Scan(&isAdmin)
	if err != nil {
		panic(err)
	}

	if isAdmin {
		// Decode the received JSON body
		var adminAttempt AdminAttempt

		err = json.NewDecoder(r.Body).Decode(&adminAttempt)
		if err != nil {
			panic(err)
		}

		// TODO: Ensure the target username exists before going further

		switch adminAttempt.Action {
		// Scheme for action strings: category.action.target
		case "users.reset.score":
			resetUsersScoreStmt := `
				UPDATE users
				SET score = 0;`
			_, err = context.db.Exec(resetUsersScoreStmt)
			if err != nil {
				panic(err)
			}

			// Return a success payload
			response, err := json.Marshal(AdminResponse{
				Success: true,
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		case "user.modify.username":
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
			response, err := json.Marshal(AdminResponse{
				Success: true,
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		case "user.modify.score":
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
			response, err := json.Marshal(AdminResponse{
				Success: true,
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		case "user.delete":
			deleteUserStmt := `
				DELETE FROM users
				WHERE username = ?;`
			_, err = context.db.Exec(deleteUserStmt, adminAttempt.Data.Username)
			if err != nil {
				panic(err)
			}

			// Return a success payload
			response, err := json.Marshal(AdminResponse{
				Success: true,
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		default:
			// Return a failure payload
			response, err := json.Marshal(AdminResponse{
				Success: false,
				Message: "Invalid action given",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}
	}

	// Return a failure payload
	response, err := json.Marshal(AdminResponse{
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

func (context *Context) AddNewQuestion(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Ensure the user's an admin
	var isAdmin bool

	isAdminStmt := `
		SELECT is_admin
		FROM users
		WHERE username = ?;`
	err := context.db.QueryRow(isAdminStmt, auth["iss"].(string)).Scan(&isAdmin)
	if err != nil {
		panic(err)
	}

	if isAdmin {
		var newQuestion Question

		err := json.NewDecoder(r.Body).Decode(&newQuestion)
		if err != nil {
			panic(err)
		}
		_, err = context.db.Exec(
			`INSERT INTO questions (question_body, category, difficulty, type, correct_answer, incorrect_answer_1,incorrect_answer_2,incorrect_answer_3) VALUES (?,?,?,?,?,?,?,?);`,
			newQuestion.QuestionBody,
			newQuestion.Category,
			newQuestion.Difficulty,
			newQuestion.QuestionType,
			newQuestion.CorrectAnswer,
			newQuestion.IncorrectAnswer1,
			newQuestion.IncorrectAnswer2,
			newQuestion.IncorrectAnswer3,
		)

		// Return a success payload
		response, err := json.Marshal(AdminResponse{
			Success: true,
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	}

	// Return a failure payload
	response, err := json.Marshal(AdminResponse{
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
