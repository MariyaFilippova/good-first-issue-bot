package store

import "context"

func (s *Store) Subscribe(ctx context.Context, userID int64, owner, name string) error {
	if err := s.AddRepo(ctx, owner, name); err != nil {
		return err
	}
	repoID, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id, repo_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		userID, repoID)
	return err
}

func (s *Store) IsSubscribed(ctx context.Context, userID int64, owner, name string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM subscriptions sub
			JOIN repos r ON r.id = sub.repo_id
			WHERE sub.user_id = $1 AND r.owner = $2 AND r.name = $3)`,
		userID, owner, name).Scan(&exists)
	return exists, err
}

func (s *Store) Unsubscribe(ctx context.Context, userID int64, owner, name string) error {
	repoID, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`DELETE FROM subscriptions WHERE user_id = $1 AND repo_id = $2`,
		userID, repoID)
	return err
}

func (s *Store) ListSubscribedRepos(ctx context.Context, userID int64) ([]Repo, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.owner, r.name
		FROM subscriptions sub
		JOIN repos r ON r.id = sub.repo_id
		WHERE sub.user_id = $1
		ORDER BY r.owner, r.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repo
	for rows.Next() {
		var rp Repo
		if err := rows.Scan(&rp.Owner, &rp.Name); err != nil {
			return nil, err
		}
		repos = append(repos, rp)
	}
	return repos, rows.Err()
}

type Subscriber struct {
	ID    int64
	Email string
}

func (s *Store) ListSubscribersForRepo(ctx context.Context, repoID int64) ([]Subscriber, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.email
		FROM subscriptions s
		JOIN users u ON u.id = s.user_id
		WHERE s.repo_id = $1`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var sub Subscriber
		if err := rows.Scan(&sub.ID, &sub.Email); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}
