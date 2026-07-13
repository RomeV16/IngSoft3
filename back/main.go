package main

import (
	"log"
	"net/http"
	"os"
)

// resolveDSN determina la cadena de conexión: DATABASE_URL > DB_DSN > SQLite local.
func resolveDSN() string {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}
	if dsn == "" {
		dsn = "./employees.db"
	}
	return dsn
}

func main() {
	store, err := NewStore(resolveDSN())
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()
	if err := store.Init(); err != nil {
		log.Fatalf("failed to init db: %v", err)
	}

	mux := http.NewServeMux()
	api := NewAPI(store)
	api.RegisterRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// withCORS disables CORS entirely by returning permissive headers for every request.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const defaultMethods = "GET, POST, PUT, DELETE, OPTIONS, PATCH"

		// Always allow every origin.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Mirror the requested headers/methods when present so browsers never block.
		if reqHeaders := r.Header.Get("Access-Control-Request-Headers"); reqHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "*")
		}

		if reqMethod := r.Header.Get("Access-Control-Request-Method"); reqMethod != "" {
			w.Header().Set("Access-Control-Allow-Methods", reqMethod)
		} else {
			w.Header().Set("Access-Control-Allow-Methods", defaultMethods)
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
