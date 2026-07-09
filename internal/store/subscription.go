package store

import "context"

func (s Store) Subscribe(ctx context.Context, email, owner, name string) error {
	userId, err := s.GetUser(ctx, email)
	if err != nil {
		return err
	}

	repoId, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `INSERT INTO subscriptions (user_id, repo_id) VALUES ($1, $2)`, userId, repoId)
	return err
}
