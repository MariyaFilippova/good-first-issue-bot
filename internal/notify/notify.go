package notify

import (
	"awesomeProject/internal/github"
	"awesomeProject/internal/store"
	"bytes"
	"context"
	"fmt"
	"log"
)

func buildMessage(issues []github.Issue, repo store.Repo) string {
	var buffer bytes.Buffer
	for _, is := range issues {
		buffer.WriteString(fmt.Sprintf("#%-6d %s\n         %s\n", is.Number, is.Title, is.HTMLURL))
	}
	buffer.WriteString(fmt.Sprintf("\n%d new \"good first issue\" issues in %s/%s\n", len(issues), repo.Owner, repo.Name))
	return buffer.String()
}

func (m *Mailer) Notify(ctx context.Context, st *store.Store, repo store.Repo, issues []github.Issue) error {
	subs, err := st.ListSubscribersForRepo(ctx, repo.ID)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		var fresh []github.Issue
		for _, issue := range issues {
			if issue.PullRequest != nil {
				continue
			}
			already, err := st.AlreadyNotified(ctx, sub.ID, issue.Id)
			if err != nil {
				return err
			}
			if !already {
				fresh = append(fresh, issue)
			}
		}

		if len(fresh) == 0 {
			continue
		}

		subject := fmt.Sprintf("New good first issues in %s/%s", repo.Owner, repo.Name)
		if err := m.Send(ctx, sub.Email, subject, buildMessage(fresh, repo)); err != nil {
			log.Printf("notify %s about %s/%s: %v", sub.Email, repo.Owner, repo.Name, err)
			continue
		}

		for _, issue := range fresh {
			if _, err := st.MarkNotified(ctx, sub.ID, repo.ID, issue.Id); err != nil {
				return err
			}
		}
	}
	return nil
}
