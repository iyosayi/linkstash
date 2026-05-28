package link

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()

	ctx := context.Background()

	databaseURL := "postgres://postgres:postgres@localhost:5432/linkstash?sslmode=disable"

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return NewStore(pool), pool
}

func TestStoreCreateAndGet(t *testing.T) {
	ctx := context.Background()

	store, pool := newTestStore(t)

	created, err := store.Create(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(ctx, `DELETE FROM links WHERE code = $1`, created.Code)
		if err != nil {
			t.Logf("cleanup link: %v", err)
		}
	})

	got, ok, err := store.Get(ctx, created.Code)
	if err != nil {
		t.Fatalf("get link: %v", err)
	}

	if !ok {
		t.Fatal("expected link to exist")
	}

	if got.URL != "https://example.com" {
		t.Fatalf("expected URL %q, got %q", "https://example.com", got.URL)
	}

	if got.Code != created.Code {
		t.Fatalf("expected code %q, got %q", created.Code, got.Code)
	}

}

func TestStoreGetNotFound(t *testing.T) {
	ctx := context.Background()

	store, _ := newTestStore(t)

	_, ok, err := store.Get(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("get link: %v", err)
	}

	if ok {
		t.Fatal("expected link not to exist")
	}
}
