package main
import _ "github.com/lib/pq"
import (
	"log"
	"net/http"
	"sync/atomic"
)

godotenv.Load()
dbURL := os.Getenv("DB_URL")
db, err := sql.Open("postgres", dbURL)
if err != nil {
	log.Fatalf("Failed to connect to database: %v", err)
}
defer db.Close()	

type apiConfig struct {
	fileserverHits atomic.Int32
	database *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	dbQueries := database.New(db)

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}

	mux := http.NewServeMux()

	// NOTE: method-based patterns, no trailing slash, and using the existing handlers

	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.Handle("/app/", fsHandler)
	// method-specific routing, using existing handlers
	mux.HandleFunc("GET /api/healthz", handlerReadiness)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("POST /api/validate_chirp", handlerValidate)

	//mux.HandleFunc("GET /api/metrics", apiCfg.handlerMetrics)

	log.Print("Listening...")

	http.ListenAndServe(":8080", mux)
}
