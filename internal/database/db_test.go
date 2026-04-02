package database

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestCreateUser(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Reset the database before test
	queries := New(db)
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Fatalf("Failed to reset users: %v", err)
	}

	// Test creating a user
	user, err := queries.CreateUser(context.Background(), CreateUserParams{
		Email:          "test@example.com",
		HashedPassword: "hashedpassword",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email %s, got %s", "test@example.com", user.Email)
	}

	if user.HashedPassword != "hashedpassword" {
		t.Errorf("Expected hashed password %s, got %s", "hashedpassword", user.HashedPassword)
	}

	// Test getting user by email
	retrievedUser, err := queries.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("Failed to get user by email: %v", err)
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
	}

	// Test creating duplicate user (should fail due to unique email)
	_, err = queries.CreateUser(context.Background(), CreateUserParams{
		Email:          "test@example.com",
		HashedPassword: "anotherpassword",
	})
	if err == nil {
		t.Error("Expected error when creating user with duplicate email")
	}
}

func TestCreateChirp(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	queries := New(db)

	// Reset databases
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Fatalf("Failed to reset users: %v", err)
	}
	err = queries.ResetChirps(context.Background())
	if err != nil {
		t.Fatalf("Failed to reset chirps: %v", err)
	}

	// Create a test user first
	user, err := queries.CreateUser(context.Background(), CreateUserParams{
		Email:          "chirpuser@example.com",
		HashedPassword: "hashedpassword",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test creating a chirp
	chirp, err := queries.CreateChirp(context.Background(), CreateChirpParams{
		Body:   "This is a test chirp",
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create chirp: %v", err)
	}

	if chirp.Body != "This is a test chirp" {
		t.Errorf("Expected body %s, got %s", "This is a test chirp", chirp.Body)
	}

	if chirp.UserID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, chirp.UserID)
	}

	// Test getting the chirp
	retrievedChirp, err := queries.GetChirp(context.Background(), chirp.ID)
	if err != nil {
		t.Fatalf("Failed to get chirp: %v", err)
	}

	if retrievedChirp.ID != chirp.ID {
		t.Errorf("Expected chirp ID %s, got %s", chirp.ID, retrievedChirp.ID)
	}
}

func TestGetChirps(t *testing.T) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:84308@localhost:5432/chirpy_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	queries := New(db)

	// Reset databases
	err = queries.ResetUsers(context.Background())
	if err != nil {
		t.Fatalf("Failed to reset users: %v", err)
	}
	err = queries.ResetChirps(context.Background())
	if err != nil {
		t.Fatalf("Failed to reset chirps: %v", err)
	}

	// Create a test user
	user, err := queries.CreateUser(context.Background(), CreateUserParams{
		Email:          "getchirps@example.com",
		HashedPassword: "hashedpassword",
	})
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create multiple chirps
	chirp1, err := queries.CreateChirp(context.Background(), CreateChirpParams{
		Body:   "First chirp",
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create first chirp: %v", err)
	}

	chirp2, err := queries.CreateChirp(context.Background(), CreateChirpParams{
		Body:   "Second chirp",
		UserID: user.ID,
	})
	if err != nil {
		t.Fatalf("Failed to create second chirp: %v", err)
	}

	// Test getting all chirps
	chirps, err := queries.GetChirps(context.Background())
	if err != nil {
		t.Fatalf("Failed to get chirps: %v", err)
	}

	if len(chirps) != 2 {
		t.Errorf("Expected 2 chirps, got %d", len(chirps))
	}

	// Check that our chirps are in the results
	foundChirp1 := false
	foundChirp2 := false
	for _, c := range chirps {
		if c.ID == chirp1.ID {
			foundChirp1 = true
		}
		if c.ID == chirp2.ID {
			foundChirp2 = true
		}
	}

	if !foundChirp1 {
		t.Error("First chirp not found in results")
	}
	if !foundChirp2 {
		t.Error("Second chirp not found in results")
	}
}
