package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type SessionLeaveAttempt struct {
	SessionID string `json:"session_id"`
}

type SessionJoinAttempt struct {
	SessionID string `json:"session_id"`
	Password  string `json:"password"`
}

type SessionStartAttempt struct {
	Public   bool   `json:"public"`
	Password string `json:"password"`
}

type SessionPlayer struct {
	Score int `json:"score"`
	// TODO: I think it would be preferrable if this kept track of the entire
	// question, the given answer, and the correctness.
	Answered []int `json:"answered_questions"`
}

type Session struct {
	// TODO: Should category be locked?
	Type      int
	StartedAt time.Time
	Public    bool
	Password  string
	Players   map[string]*SessionPlayer // map username to player's data
}

// TODO
type SQLQuestion struct {
}

type SessionResponseData struct {
	SessionID string                    `json:"session_id,omitempty"`
	Public    bool                      `json:"public,omitempty"`
	StartedAt int64                     `json:"started_at,omitempty"`
	Players   map[string]*SessionPlayer `json:"players,omitempty"`
	Questions []SQLQuestion             `json:"questions,omitempty"`
}

// Generic struct for responding to authentication requests
type SessionResponse struct {
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	Data    *SessionResponseData `json:"data,omitempty"`
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
	for _, session := range context.sessions {
		for username := range session.Players {
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

	// Create a new session
	// TODO: Random strings is terrible practice, but it works for now
	source := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	rand.Seed(time.Now().UnixNano())

	runes := make([]rune, 16)
	for i := range runes {
		runes[i] = source[rand.Intn(62)]
	}

	sessionID := string(runes)

	// TODO: Set the session's password
	context.sessions[sessionID] = &Session{
		Type:      1,
		StartedAt: time.Now().UTC(),
		Public:    startAttempt.Public,
		Password:  startAttempt.Password,
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
// TODO
func (context *Context) GameGetQuestion(w http.ResponseWriter, r *http.Request) {
}

// API endpoint for clients to send answers
// TODO
func (context *Context) GamePostAnswer(w http.ResponseWriter, r *http.Request) {
}

// API endpoint for clients to get session meta information
func (context *Context) GameGetMeta(w http.ResponseWriter, r *http.Request) {
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
					Public:    session.Public,
					StartedAt: session.StartedAt.Unix(),
					Players:   session.Players,
				},
			}

			// If nil slices aren't converted to empty ones, then the JSON
			// output contains NULL references and that's gross
			for _, player := range payload.Data.Players {
				if player.Answered == nil {
					player.Answered = make([]int, 0)
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
