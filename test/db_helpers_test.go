package test

import (
	"database/sql"
	"testing"
)

func rootDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/zxc?sslmode=disable")
	if err != nil {
		t.Fatalf("open root database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tenantMainDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/"+sanitizeTenantDBName(name)+"?sslmode=disable&search_path=main")
	if err != nil {
		t.Fatalf("open tenant main database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tenantDeployDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", deployDSN(name))
	if err != nil {
		t.Fatalf("open tenant deploy database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func tenantAccountDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", accountDSN(name))
	if err != nil {
		t.Fatalf("open tenant account database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
