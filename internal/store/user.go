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

func (s *Store) SubscribersOf(ctx context.Context, owner, name string) ([]string, error) {
	id, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT u.email
  FROM subscriptions s
  JOIN users u ON u.id = s.user_id
  WHERE s.repo_id = $1`, id)
	if err != nil {
		return nil, err
	}
	var emails []string
	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	return emails, rows.Err()
}
