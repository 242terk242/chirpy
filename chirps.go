package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"example.com/chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    string    `json:"user_id"`
}

func (cfg *apiConfig) handlerChirpsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type parameters struct {
		Body   string `json:"body"`
		UserId string `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	if params.Body == "" || params.UserId == "" {
		respondWithError(w, http.StatusBadRequest, "Body and User ID are required", nil)
		return
	}

	if len(params.Body) > 140 {
		w.WriteHeader(400)
		w.Write([]byte(`{"error": "Chirp is too long"}`))
		return
	}

	var result string
	var s []string
	var append_s []string
	s = make([]string, 3)
	append_s = make([]string, 0)
	s[0] = "kerfuffle"
	s[1] = "sharbert"
	s[2] = "fornax"
	words := strings.Fields(params.Body)
	for _, word := range words {
		var banned bool = false
		for _, bannedWord := range s {
			if strings.ToLower(word) == bannedWord {
				banned = true
				break
			}
		}
		if !banned {
			append_s = append(append_s, word)
		} else {
			append_s = append(append_s, "****")
		}

	}
	result = strings.Join(append_s, " ")

	userID, err := uuid.Parse(params.UserId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Invalid user ID", err)
		return
	}

	chirp, err := cfg.database.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   result,
		UserID: userID,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create chirp", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID.String(),
	})
}
