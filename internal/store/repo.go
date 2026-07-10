package store

import (
	"context"
	"time"
)

type Repo struct {
	ID        int64
	Owner     string
	Name      string
	ETag      *string    // nil = never fetched (SQL NULL)
	HighWater *time.Time // nil = never polled (SQL NULL)
}

func (s *Store) AddRepo(ctx context.Context, owner, name string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO repos (owner, name) VALUES ($1, $2)
		 ON CONFLICT (owner, name) DO NOTHING`,
		owner, name)
	return err
}

func (s *Store) GetRepo(ctx context.Context, owner, name string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM repos WHERE owner = $1 AND name = $2`, owner, name).Scan(&id)
	return id, err
}

func (s *Store) ListRepos(ctx context.Context) ([]Repo, error) {
	rows, err := s.pool.Query(ctx, `SELECT owner, name FROM repos`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.Owner, &r.Name); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *Store) DueRepos(ctx context.Context, limit int) ([]Repo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, owner, name, etag, high_water
		FROM repos
		WHERE next_poll_at <= now()
		ORDER BY next_poll_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.Owner, &r.Name, &r.ETag, &r.HighWater); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}
