package link

import (
	"context"
	"errors"
	"math/rand"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func generateRandomCode() string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz"
	const n = 6

	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(b)
}

// constructor function === in Go, we use New
func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db: db,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func (s *Store) Create(ctx context.Context, url string) (LinkResponse, error) {

	query := `
	INSERT INTO links (code, url)
	VALUES ($1, $2)
	RETURNING code, url
	`

	for range 5 {
		code := generateRandomCode()
		var link LinkResponse
		err := s.db.QueryRow(ctx, query, code, url).Scan(&link.Code, &link.URL)
		if err == nil {
			return link, nil
		}

		if isUniqueViolation(err) {
			continue
		}
		return LinkResponse{}, err
	}

	return LinkResponse{}, errors.New("could not generate unique code")
}

func (s *Store) Get(ctx context.Context, code string) (LinkResponse, bool, error) {
	query := `
	 SELECT code, url 
	 FROM links
	 WHERE code = $1
	`
	var link LinkResponse
	err := s.db.QueryRow(ctx, query, code).Scan(&link.Code, &link.URL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LinkResponse{}, false, nil
		}
		return LinkResponse{}, false, err
	}
	return link, true, nil
}

func (s *Store) GetStats(ctx context.Context, code string) (LinkStatResponse, bool, error) {
	const query = `
		SELECT code, url, click_count, created_at, last_accessed_at
		FROM links
		WHERE code = $1
	`

	var stats LinkStatResponse
	err := s.db.QueryRow(ctx, query, code).Scan(
		&stats.Code,
		&stats.URL,
		&stats.ClickCount,
		&stats.CreatedAt,
		&stats.LastAccessedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LinkStatResponse{}, false, nil
		}

		return LinkStatResponse{}, false, err
	}

	return stats, true, nil
}
