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

func (s *Store) NotifiedIssueIDs(ctx context.Context, userIDs, githubIssueIDs []int64) (map[int64]map[int64]bool, error) {
	seen := make(map[int64]map[int64]bool, len(userIDs))
	if len(userIDs) == 0 || len(githubIssueIDs) == 0 {
		return seen, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT user_id, github_issue_id FROM notified
		 WHERE user_id = ANY($1) AND github_issue_id = ANY($2)`,
		userIDs, githubIssueIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID, issueID int64
		if err := rows.Scan(&userID, &issueID); err != nil {
			return nil, err
		}
		if seen[userID] == nil {
			seen[userID] = make(map[int64]bool)
		}
		seen[userID][issueID] = true
	}
	return seen, rows.Err()
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
