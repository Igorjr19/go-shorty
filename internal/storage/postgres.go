package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"github.com/Igorjr19/go-shorty/internal/entity"
)

const pgUniqueViolation = "23505"

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{
		db: db,
	}
}

func (p *PostgresStorage) Save(ctx context.Context, link entity.Link) error {
	q := `INSERT INTO links (code, original_url, created_at) VALUES ($1, $2, $3)`
	_, err := p.db.ExecContext(ctx, q, link.Code, link.OriginalURL, link.CreatedAt)

	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
		return ErrCodeExists
	}
	return err
}

func (p *PostgresStorage) Load(ctx context.Context, code string) (entity.Link, error) {
	q := `SELECT code, original_url, created_at, visits FROM links WHERE code = $1`
	return scanLink(p.db.QueryRowContext(ctx, q, code))
}

func (p *PostgresStorage) Visit(ctx context.Context, code string) (entity.Link, error) {
	q := `UPDATE links SET visits = visits + 1 WHERE code = $1
		RETURNING code, original_url, created_at, visits`
	return scanLink(p.db.QueryRowContext(ctx, q, code))
}

func scanLink(row *sql.Row) (entity.Link, error) {
	var link entity.Link
	err := row.Scan(&link.Code, &link.OriginalURL, &link.CreatedAt, &link.Visits)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Link{}, ErrNotFound
		}
		return entity.Link{}, err
	}
	return link, nil
}
