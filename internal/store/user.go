package store

import "context"

func (s *Store) AddUser(ctx context.Context, email string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email) VALUES ($1)
		 ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
		email).Scan(&id)
	return id, err
}

func (s *Store) GetUser(ctx context.Context, email string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1`,
		email).Scan(&id)
	return id, err
}
