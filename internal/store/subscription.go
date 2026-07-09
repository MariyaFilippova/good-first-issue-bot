package store

import "context"

func (s Store) Subscribe(ctx context.Context, email, owner, name string) error {
	user_id, err := s.GetUser(ctx, email)
	if err != nil {
		return err
	}

	repo_id, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `INSERT INTO subscriptions (user_id, repo_id) VALUES ($1, $2)`, user_id, repo_id)
	return err
}
