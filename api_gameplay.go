package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// Gamemodes
const (
	GamemodeMarathon = iota // 0
	GamemodeSprint          // 1
)

type SessionAnswerAttempt struct {
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

type SessionLeaveAttempt struct {
	SessionID string `json:"session_id"`
}

type SessionJoinAttempt struct {
	SessionID string `json:"session_id"`
	Password  string `json:"password"`
}

type SessionStartAttempt struct {
	Gamemode     int    `json:"gamemode"`
	Category     string `json:"category"`
	Difficulty   string `json:"difficulty"`
	SinglePlayer bool   `json:"single_player"`
	Password     string `json:"password"`
}

type SessionPlayerAnswer struct {
	Correct bool   `json:"correct"`
	Answer  string `json:"answer"`
}

type SessionPlayer struct {
	Score int `json:"score"`
	// TODO: I think it would be preferable if this kept track of the entire
	// question, the given answer, and the correctness.
	Answers []*SessionPlayerAnswer `json:"answers"`
}

type Session struct {
	// TODO: Should category be locked?
	Gamemode           int
	Category           string
	Difficulty         string
	StartedAt          time.Time
	SinglePlayer       bool
	Password           string
	Players            map[string]*SessionPlayer // map username to player's data
	CurrentQuestion    *SQLQuestion
	QuestionExpiration time.Time
	QuestionHistory    []*SQLQuestion // Index corresponds with Players.Player.Answered
}

type SQLQuestion struct {
	ID               string `json:"id"`
	Body             string `json:"body"`
	CorrectAnswer    string `json:"correct_answer"`
	IncorrectAnswer1 string `json:"incorrect_answer_1"`
	IncorrectAnswer2 string `json:"incorrect_answer_2"`
	IncorrectAnswer3 string `json:"incorrect_answer_3"`
}

type SessionResponseQuestion struct {
	ID      string   `json:"id"`
	Body    string   `json:"body"`
	Answers []string `json:"answers"`
}

type SessionResponseData struct {
	SessionID    string                    `json:"session_id,omitempty"`
	SinglePlayer bool                      `json:"single_player,omitempty"`
	StartedAt    int64                     `json:"started_at,omitempty"`
	Players      map[string]*SessionPlayer `json:"players,omitempty"`
	Questions    []SessionResponseQuestion `json:"questions,omitempty"`
	Correct      bool                      `json:"correct,omitempty"`
}

// Generic struct for responding to authentication requests
type SessionResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	Data    *SessionResponseData `json:"data,omitempty"`
}

func contains(source *[]string, find *string) bool {
	for idx := range *source {
		if (*source)[idx] == *find {
			return true
		}
	}

	return false
}

// API endpoint for clients to start games
func (context *Context) GameStart(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Decode the received JSON body
	var startAttempt SessionStartAttempt

	err := json.NewDecoder(r.Body).Decode(&startAttempt)
	if err != nil {
		panic(err)
	}

	// Ensure a session doesn't already exist with the given user
	for _, session := range context.sessions { // no key, just value
		for username := range session.Players { // key
			if username == auth["iss"] {
				response, err := json.Marshal(SessionResponse{
					Success: false,
					Message: "User is already in a session",
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
	}

	// Ensure the gamemode ID is valid
	if startAttempt.Gamemode < GamemodeMarathon || startAttempt.Gamemode > GamemodeSprint {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Invalid gamemode ID",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	}

	// TODO: Ensure the category is valid

	// Ensure the difficulty is valid
	if !contains(&[]string{"easy", "medium", "hard"}, &startAttempt.Difficulty) {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Invalid difficulty",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	}

	// If a session is multi-player, then a password is optional. If it's
	// single-player, then it can't have a password.
	if startAttempt.SinglePlayer == false && startAttempt.Password != "" {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Private sessions can't have passwords",
		})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		return
	}

	// Create a new session
	// TODO: Random strings is terrible practice, but it works for now
	source := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	rand.Seed(time.Now().UnixNano())

	runes := make([]rune, 16)
	for i := range runes {
		runes[i] = source[rand.Intn(62)]
	}

	sessionID := string(runes)

	// Create a session and add it to the global state
	context.sessions[sessionID] = &Session{
		Gamemode:     startAttempt.Gamemode,
		Category:     startAttempt.Category,
		Difficulty:   startAttempt.Difficulty,
		StartedAt:    time.Now().UTC(),
		SinglePlayer: startAttempt.SinglePlayer,
		Password:     startAttempt.Password,
	}
	context.sessions[sessionID].Players = make(map[string]*SessionPlayer, 1)
	context.sessions[sessionID].Players[auth["iss"].(string)] = &SessionPlayer{}

	response, err := json.Marshal(SessionResponse{
		Success: true,
		Data:    &SessionResponseData{SessionID: sessionID},
	})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(response)

	return
}

// API endpoint for clients to join games
func (context *Context) GameJoin(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Decode the received JSON body
	var joinAttempt SessionJoinAttempt

	err := json.NewDecoder(r.Body).Decode(&joinAttempt)
	if err != nil {
		panic(err)
	}

	// Ensure the session exists
	if session, ok := context.sessions[joinAttempt.SessionID]; ok {
		// Ensure the session is either public

		// Ensure the given password is correct
		if joinAttempt.Password != session.Password {
			response, err := json.Marshal(SessionResponse{
				Success: false,
				Message: "Invalid password",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			// TODO: Should this return an http.StatusUnauthorized (401) instead?
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}

		// Insert the player into the session if they aren't already in it
		if _, ok := session.Players[auth["iss"].(string)]; !ok {
			session.Players[auth["iss"].(string)] = &SessionPlayer{}

			response, err := json.Marshal(SessionResponse{Success: true})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		} else {
			response, err := json.Marshal(SessionResponse{
				Success: false,
				Message: "You're already in the session",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}

	} else {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Session doesn't exist",
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

// API endpoint for clients to leave games
func (context *Context) GameLeave(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Decode the received JSON body
	var leaveAttempt SessionLeaveAttempt

	err := json.NewDecoder(r.Body).Decode(&leaveAttempt)
	if err != nil {
		panic(err)
	}

	// Ensure the session exists
	if session, ok := context.sessions[leaveAttempt.SessionID]; ok {
		// Ensure the player's in the session
		if _, ok := session.Players[auth["iss"].(string)]; ok {
			delete(session.Players, auth["iss"].(string))

			// Delete the session if there's 0 players
			if len(session.Players) == 0 {
				delete(context.sessions, leaveAttempt.SessionID)
			}

			response, err := json.Marshal(SessionResponse{Success: true})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		} else {
			response, err := json.Marshal(SessionResponse{
				Success: false,
				Message: "You aren't in the session",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}

	} else {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Session doesn't exist",
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

// API endpoint for clients to get questions
func (context *Context) GameGetQuestion(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	r.ParseForm()

	// /api/game/question?session=session_id
	sessionID := r.FormValue("session")

	// Ensure the session exists
	// TODO: Should this become middleware?
	if session, ok := context.sessions[sessionID]; ok {
		// Ensure the player's in the given session
		isInSession := false
		for username := range session.Players {
			if username == auth["iss"] {
				isInSession = true
			}
		}

		if isInSession {
			// Ensure a question is set
			if session.CurrentQuestion == nil {
				// TODO: Set a question given the category ID that hasn't been
				// given yet
				stmt := `
					SELECT (id, question_body, correct_answer, incorrect_answer_1, incorrect_answer_2, incorrect_answer_3)
					FROM questions
					WHERE category=?
					AND difficulty=?;`
				err := context.db.QueryRow(stmt, session.Category, session.Difficulty).Scan(

					&session.CurrentQuestion.ID,
					&session.CurrentQuestion.Body,
					&session.CurrentQuestion.CorrectAnswer,
					&session.CurrentQuestion.IncorrectAnswer1,
					&session.CurrentQuestion.IncorrectAnswer2,
					&session.CurrentQuestion.IncorrectAnswer3,
				)
				if err != nil {
					panic(err)

				}
				// session.QuestionHistory
				session.QuestionHistory = append(session.QuestionHistory, session.CurrentQuestion)

				// Expiration date time.Now().Add(time.Second * 60)
				session.QuestionExpiration = time.Now().Add(time.Second * 60)
			}

			payload := SessionResponse{
				Success: true,
				Data:    &SessionResponseData{},
			}

			var responseQuestion SessionResponseQuestion

			responseQuestion.ID = session.CurrentQuestion.ID
			responseQuestion.Body = session.CurrentQuestion.Body

			responseQuestion.Answers = append(responseQuestion.Answers, session.CurrentQuestion.CorrectAnswer)
			responseQuestion.Answers = append(responseQuestion.Answers, session.CurrentQuestion.IncorrectAnswer1)
			responseQuestion.Answers = append(responseQuestion.Answers, session.CurrentQuestion.IncorrectAnswer2)
			responseQuestion.Answers = append(responseQuestion.Answers, session.CurrentQuestion.IncorrectAnswer3)

			response, err := json.Marshal(payload)
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		} else {
			response, err := json.Marshal(SessionResponse{
				Success: false,
				Message: "You aren't in the session",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}
	} else {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Session doesn't exist",
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

// API endpoint for clients to send answers
func (context *Context) GamePostAnswer(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	// Decode the received JSON body
	var answerAttempt SessionAnswerAttempt

	err := json.NewDecoder(r.Body).Decode(&answerAttempt)
	if err != nil {
		panic(err)
	}

	// Ensure the session exists
	if session, ok := context.sessions[answerAttempt.SessionID]; ok {
		// Ensure the player's in the given session
		isInSession := false
		for username := range session.Players {
			if username == auth["iss"] {
				isInSession = true
			}
		}

		if isInSession {
			if answerAttempt.Answer == session.CurrentQuestion.CorrectAnswer {
				// TODO: Record answer and points

				response, err := json.Marshal(SessionResponse{
					Success: true,
					Data:    &SessionResponseData{Correct: true},
				})
				if err != nil {
					panic(err)
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write(response)

				return
			} else {
				// TODO: Record answer and points

				response, err := json.Marshal(SessionResponse{
					Success: true,
					Data:    &SessionResponseData{Correct: false},
				})
				if err != nil {
					panic(err)
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write(response)

				return
			}
		} else {
			response, err := json.Marshal(SessionResponse{
				Success: false,
				Message: "You aren't in the session",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}
	} else {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Session doesn't exist",
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

// API endpoint for clients to get session meta information
func (context *Context) GameGetInfo(w http.ResponseWriter, r *http.Request) {
	// Pull the user's decoded authentication information from their parsed token
	decoded := r.Context().Value("decoded")
	auth := decoded.(jwt.MapClaims)

	r.ParseForm()

	sessionID := r.FormValue("session")

	// Ensure the session exists
	if session, ok := context.sessions[sessionID]; ok {
		// Ensure the player's in the given session
		isInSession := false
		for username := range session.Players {
			if username == auth["iss"] {
				isInSession = true
			}
		}

		if isInSession {
			payload := SessionResponse{
				Success: true,
				Data: &SessionResponseData{
					SinglePlayer: session.SinglePlayer,
					StartedAt:    session.StartedAt.UTC().Unix(),
					Players:      session.Players,
				},
			}

			// If nil slices aren't converted to empty ones, then the JSON
			// output contains NULL references and that's gross
			for _, player := range payload.Data.Players {
				if player.Answers == nil {
					player.Answers = make([]*SessionPlayerAnswer, 0)
				}
			}

			response, err := json.Marshal(payload)
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		} else {
			response, err := json.Marshal(SessionResponse{
				Success: false,
				Message: "You aren't in the session",
			})
			if err != nil {
				panic(err)
			}

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			w.Write(response)

			return
		}
	} else {
		response, err := json.Marshal(SessionResponse{
			Success: false,
			Message: "Session doesn't exist",
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

func (context *Context) GameModify(w http.ResponseWriter, r *http.Request) {
}
