package main

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerChirpGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var chirp Chirp

	// Parse chirpID from URL path /api/chirps/{chirpID}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 4 || parts[1] != "api" || parts[2] != "chirps" {
		respondWithError(w, http.StatusBadRequest, "Invalid path", nil)
		return
	}
	chirpIDStr := parts[3]
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}
	result, err := cfg.database.GetChirp(r.Context(), chirpID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Chirp not found", err)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Couldn't get chirp", err)
		return
	}

	chirp = Chirp{
		ID:        result.ID,
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
		Body:      result.Body,
		UserId:    result.UserID.String(),
	}

	respondWithJSON(w, http.StatusOK, chirp)
}
