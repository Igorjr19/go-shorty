package storage

import (
	"database/sql"

	"github.com/lib/pq"

	"github.com/Igorjr19/go-shorty/internal/config"
	"github.com/Igorjr19/go-shorty/internal/entity"
)

type PostgresStorage struct {
	db *sql.DB
	pq.Config
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	defer db.Close()
	return &PostgresStorage{
		db: config.ConnectDB(),
	}

}

func (p *PostgresStorage) Save(link entity.Link) error {
	q := `INSERT INTO links (code, original_url, created_at) VALUES ($1, $2, $3)`
	_, err := p.db.Exec(q, link.Code, link.OriginalURL, link.CreatedAt)
	return err
}

func (p *PostgresStorage) Load(code string) (entity.Link, error) {
	q := `SELECT code, original_url, created_at FROM links WHERE code = $1`
	row := p.db.QueryRow(q, code)

	var link entity.Link
	err := row.Scan(&link.Code, &link.OriginalURL, &link.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Link{}, ErrNotFound
		}
		return entity.Link{}, err
	}
	return link, nil
}
