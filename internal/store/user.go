package store

import "context"

func (s *Store) UpsertGitHubUser(ctx context.Context, githubID int64, email string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (github_id, email) VALUES ($1, $2)
		 ON CONFLICT (github_id) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
		githubID, email).Scan(&id)
	return id, err
}

func (s *Store) AlreadyNotified(ctx context.Context, userID, githubIssueID int64) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM notified WHERE user_id = $1 AND github_issue_id = $2)`,
		userID, githubIssueID).Scan(&exists)
	return exists, err
}

func (s *Store) MarkNotified(ctx context.Context, userID, repoID, githubIssueID int64) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO notified (user_id, repo_id, github_issue_id) VALUES ($1, $2, $3)
         ON CONFLICT DO NOTHING`,
		userID, repoID, githubIssueID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
