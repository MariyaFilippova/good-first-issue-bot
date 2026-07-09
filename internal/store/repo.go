package store

import "context"

type Repo struct {
	Owner string
	Name  string
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
