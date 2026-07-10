package store

import "context"

type Subscription struct {
	User int64
	Repo int64
}

func (s *Store) Subscribe(ctx context.Context, email, owner, name string) error {
	userId, err := s.GetUser(ctx, email)
	if err != nil {
		return err
	}

	repoId, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `INSERT INTO subscriptions (user_id, repo_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userId, repoId)
	return err
}

func (s *Store) Unsubscribe(ctx context.Context, email, owner, name string) error {
	userId, err := s.GetUser(ctx, email)
	if err != nil {
		return err
	}

	repoId, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `DELETE FROM subscriptions WHERE user_id = $1 AND repo_id = $2`, userId, repoId)
	return err
}

func (s *Store) ListSubscriptions(ctx context.Context, userId int64) ([]Subscription, error) {
	rows, err := s.pool.Query(ctx, `SELECT user_id, repo_id FROM subscriptions WHERE user_id = $1`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.User, &sub.Repo); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}

	return subs, rows.Err()
}
