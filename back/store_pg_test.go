package main

import (
	"testing"
)

// Tests del soporte dual SQLite/PostgreSQL.
// No requieren un servidor PostgreSQL real: sql.Open es lazy y no conecta
// hasta la primera consulta, por lo que podemos verificar la detección de
// driver, la conversión de placeholders y los caminos de error.

func TestNewStore_postgresDSN(t *testing.T) {
	s, err := NewStore("postgres://user:pass@localhost:5432/testdb?sslmode=disable")
	if err != nil {
		t.Fatalf("NewStore postgres: %v", err)
	}
	defer s.Close()
	if !s.postgres {
		t.Fatalf("expected postgres=true for postgres:// DSN")
	}
}

func TestNewStore_postgresqlDSN(t *testing.T) {
	s, err := NewStore("postgresql://user:pass@localhost:5432/testdb?sslmode=disable")
	if err != nil {
		t.Fatalf("NewStore postgresql: %v", err)
	}
	defer s.Close()
	if !s.postgres {
		t.Fatalf("expected postgres=true for postgresql:// DSN")
	}
}

func TestNewStore_sqliteDSN(t *testing.T) {
	s, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore sqlite: %v", err)
	}
	defer s.Close()
	if s.postgres {
		t.Fatalf("expected postgres=false for sqlite DSN")
	}
}

func TestPq_sqlitePassthrough(t *testing.T) {
	s, _ := NewStore(":memory:")
	defer s.Close()
	q := "INSERT INTO employees(name) VALUES(?)"
	if got := s.pq(q); got != q {
		t.Fatalf("sqlite should not rewrite placeholders, got %q", got)
	}
}

func TestPq_postgresPlaceholders(t *testing.T) {
	s, _ := NewStore("postgres://u:p@localhost/x")
	defer s.Close()
	got := s.pq("UPDATE employees SET name=? WHERE id=?")
	want := "UPDATE employees SET name=$1 WHERE id=$2"
	if got != want {
		t.Fatalf("pq conversion: got %q want %q", got, want)
	}
}

func TestInit_postgresUnreachable(t *testing.T) {
	// El branch del schema PostgreSQL se ejecuta, pero el Exec falla
	// porque no hay servidor escuchando — Init debe devolver error.
	s, _ := NewStore("postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1")
	defer s.Close()
	if err := s.Init(); err == nil {
		t.Fatalf("expected error initializing against unreachable postgres")
	}
}

func TestClose_nilStore(t *testing.T) {
	var s *Store
	if err := s.Close(); err != nil {
		t.Fatalf("nil store Close should be nil, got %v", err)
	}
	empty := &Store{}
	if err := empty.Close(); err != nil {
		t.Fatalf("empty store Close should be nil, got %v", err)
	}
}

func TestResolveDSN(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_DSN", "")
	if got := resolveDSN(); got != "./employees.db" {
		t.Fatalf("default DSN: got %q", got)
	}

	t.Setenv("DB_DSN", "/data/employees.db")
	if got := resolveDSN(); got != "/data/employees.db" {
		t.Fatalf("DB_DSN: got %q", got)
	}

	t.Setenv("DATABASE_URL", "postgres://u:p@host/db")
	if got := resolveDSN(); got != "postgres://u:p@host/db" {
		t.Fatalf("DATABASE_URL priority: got %q", got)
	}
}
