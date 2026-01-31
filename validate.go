package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func handlerValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"cleaned_body": "` + result + `"}`))

}
