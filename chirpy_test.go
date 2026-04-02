package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"example.com/chirpy/internal/auth"
	"example.com/chirpy/internal/database"
	_ "github.com/lib/pq"
)

func TestHandlerReadiness(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/healthz", nil)
	w := httptest.NewRecorder()

	handlerReadiness(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expected := "OK"
	if w.Body.String() != expected {
		t.Errorf("Expected body %q, got %q", expected, w.Body.String())
	}
}

func TestHandlerMetrics(t *testing.T) {
	cfg := &apiConfig{
		fileserverHits: atomic.Int32{},
	}

	// Simulate some hits
	cfg.fileserverHits.Add(5)

	req := httptest.NewRequest("GET", "/admin/metrics", nil)
	w := httptest.NewRecorder()

	cfg.handlerMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Chirpy has been visited 5 times!") {
		t.Errorf("Expected metrics to contain hit count, got %q", body)
	}
}

func TestHandlerChirpsCreate(t *testing.T) {
	// Setup test database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Reset database
	queries := database.New(db)
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Skip("Could not reset database, skipping test")
	}
	err = queries.ResetChirps(context.Background())
	if err != nil {
		t.Skip("Could not reset chirps, skipping test")
	}

	// Create test user
	user, err := queries.CreateUser(context.Background(), database.CreateUserParams{
		Email:          "test@example.com",
		HashedPassword: "hashed",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	cfg := &apiConfig{
		database: queries,
		platform: "test",
	}

	tests := []struct {
		name           string
		method         string
		body           map[string]interface{}
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response)
	}{
		{
			name:   "Valid chirp creation",
			method: "POST",
			body: map[string]interface{}{
				"body":    "This is a valid chirp",
				"user_id": user.ID.String(),
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var chirp Chirp
				if err := json.NewDecoder(resp.Body).Decode(&chirp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
					return
				}
				if chirp.Body != "This is a valid chirp" {
					t.Errorf("Expected body 'This is a valid chirp', got %s", chirp.Body)
				}
			},
		},
		{
			name:   "Chirp with profanity",
			method: "POST",
			body: map[string]interface{}{
				"body":    "This has kerfuffle in it",
				"user_id": user.ID.String(),
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *http.Response) {
				var chirp Chirp
				if err := json.NewDecoder(resp.Body).Decode(&chirp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
					return
				}
				if chirp.Body != "This has **** in it" {
					t.Errorf("Expected body 'This has **** in it', got %s", chirp.Body)
				}
			},
		},
		{
			name:   "Chirp too long",
			method: "POST",
			body: map[string]interface{}{
				"body":    strings.Repeat("a", 141),
				"user_id": user.ID.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Missing body",
			method: "POST",
			body: map[string]interface{}{
				"user_id": user.ID.String(),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Wrong method",
			method: "GET",
			body: map[string]interface{}{
				"body":    "test",
				"user_id": user.ID.String(),
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(test.body)
			req := httptest.NewRequest(test.method, "/api/chirps", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			cfg.handlerChirpsCreate(w, req)

			if w.Code != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, w.Code)
			}

			if test.checkResponse != nil {
				test.checkResponse(t, w.Result())
			}
		})
	}
}

func TestHandlerChirpsGet(t *testing.T) {
	// Setup test database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Reset database
	queries := database.New(db)
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Skip("Could not reset database, skipping test")
	}
	err = queries.ResetChirps(context.Background())
	if err != nil {
		t.Skip("Could not reset chirps, skipping test")
	}

	// Create test user
	user, err := queries.CreateUser(context.Background(), database.CreateUserParams{
		Email:          "test@example.com",
		HashedPassword: "hashed",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test chirps
	chirp1, err := queries.CreateChirp(context.Background(), database.CreateChirpParams{
		Body:   "First test chirp",
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create chirp: %v", err)
	}

	chirp2, err := queries.CreateChirp(context.Background(), database.CreateChirpParams{
		Body:   "Second test chirp",
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create chirp: %v", err)
	}

	cfg := &apiConfig{
		database: queries,
	}

	req := httptest.NewRequest("GET", "/api/chirps", nil)
	w := httptest.NewRecorder()

	cfg.handlerChirpsGet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var chirps []Chirp
	if err := json.NewDecoder(w.Body).Decode(&chirps); err != nil {
		t.Errorf("Failed to decode response: %v", err)
		return
	}

	if len(chirps) != 2 {
		t.Errorf("Expected 2 chirps, got %d", len(chirps))
	}

	// Check that chirps are returned
	found1 := false
	found2 := false
	for _, c := range chirps {
		if c.ID == chirp1.ID {
			found1 = true
		}
		if c.ID == chirp2.ID {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Error("Not all expected chirps found in response")
	}
}

func TestHandlerChirpGet(t *testing.T) {
	// Setup test database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Reset database
	queries := database.New(db)
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Skip("Could not reset database, skipping test")
	}
	err = queries.ResetChirps(context.Background())
	if err != nil {
		t.Skip("Could not reset chirps, skipping test")
	}

	// Create test user
	user, err := queries.CreateUser(context.Background(), database.CreateUserParams{
		Email:          "test@example.com",
		HashedPassword: "hashed",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test chirp
	chirp, err := queries.CreateChirp(context.Background(), database.CreateChirpParams{
		Body:   "Test chirp for get",
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create chirp: %v", err)
	}

	cfg := &apiConfig{
		database: queries,
	}

	req := httptest.NewRequest("GET", "/api/chirps/"+chirp.ID.String(), nil)
	w := httptest.NewRecorder()

	cfg.handlerChirpGet(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var retrievedChirp Chirp
	if err := json.NewDecoder(w.Body).Decode(&retrievedChirp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
		return
	}

	if retrievedChirp.ID != chirp.ID {
		t.Errorf("Expected chirp ID %s, got %s", chirp.ID, retrievedChirp.ID)
	}
	if retrievedChirp.Body != chirp.Body {
		t.Errorf("Expected body %s, got %s", chirp.Body, retrievedChirp.Body)
	}
}

func TestHandlerCreateUsers(t *testing.T) {
	// Setup test database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Reset database
	queries := database.New(db)
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Skip("Could not reset database, skipping test")
	}

	cfg := &apiConfig{
		database: queries,
		platform: "test",
	}

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid user creation",
			body: map[string]interface{}{
				"email":    "newuser@example.com",
				"password": "password123",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Missing email",
			body: map[string]interface{}{
				"password": "password123",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing password",
			body: map[string]interface{}{
				"email": "newuser@example.com",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(test.body)
			req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			cfg.handlerCreateUsers(w, req)

			if w.Code != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandlerLogin(t *testing.T) {
	// Setup test database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Reset database
	queries := database.New(db)
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Skip("Could not reset database, skipping test")
	}

	// Create test user
	hashedPassword, err := auth.HashPassword("testpassword")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = queries.CreateUser(context.Background(), database.CreateUserParams{
		Email:          "login@example.com",
		HashedPassword: hashedPassword,
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	cfg := &apiConfig{
		database: queries,
		platform: "test",
	}

	tests := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Valid login",
			body: map[string]interface{}{
				"email":    "login@example.com",
				"password": "testpassword",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid password",
			body: map[string]interface{}{
				"email":    "login@example.com",
				"password": "wrongpassword",
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid email",
			body: map[string]interface{}{
				"email":    "nonexistent@example.com",
				"password": "testpassword",
			},
			expectedStatus: http.StatusInternalServerError, // Since GetUserByEmail will fail
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(test.body)
			req := httptest.NewRequest("POST", "/api/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			cfg.handlerLogin(w, req)

			if w.Code != test.expectedStatus {
				t.Errorf("Expected status %d, got %d", test.expectedStatus, w.Code)
			}
		})
	}
}
