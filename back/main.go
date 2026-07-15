// main.go — PUNTO DE ENTRADA del backend.
//
// Arranque del servidor:
//  1. resolveDSN() decide a qué base conectarse según variables de entorno
//  2. NewStore() abre la conexión (detecta SQLite o PostgreSQL por el DSN)
//  3. Init() crea las tablas si no existen (schema idempotente)
//  4. Se registran las rutas y se levanta el servidor HTTP con CORS abierto
//
// Variables de entorno relevantes (configuradas en Railway):
//   DATABASE_URL → conexión PostgreSQL (la usan DEV y PROD, misma BD)
//   DB_DSN       → alternativa (ruta a archivo SQLite, ej: /data/employees.db)
//   PORT         → puerto HTTP (Railway lo inyecta; default 8080)
//
// Pipeline final: build, tests, análisis estático y despliegue verificados.
package main

import (
	"log"
	"net/http"
	"os"
)

// resolveDSN determina la cadena de conexión con este orden de prioridad:
//  1. DATABASE_URL (PostgreSQL en Railway — producción y dev)
//  2. DB_DSN       (ruta a SQLite, ej. volumen /data en Docker)
//  3. ./employees.db (SQLite local — desarrollo en la máquina propia)
// Está extraída de main() para poder testearla unitariamente
// (TestResolveDSN en store_pg_test.go — main() no se puede testear directo).
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

// withCORS es un middleware: envuelve al router y agrega los headers CORS a
// toda respuesta. Es necesario porque el frontend (frontend-dev...railway.app)
// y el backend (backend-dev...railway.app) viven en DOMINIOS DISTINTOS — sin
// estos headers el navegador bloquearía los fetch del frontend.
// También responde los preflight OPTIONS con 204 sin llegar a los handlers.
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
