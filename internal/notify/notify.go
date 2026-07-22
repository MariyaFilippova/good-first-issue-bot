package notify

import (
	"awesomeProject/internal/github"
	"awesomeProject/internal/store"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
)

type job struct {
	email, subject, body string
	issueIDs             []int64
	repoID, userID       int64
}

type Notifier struct {
	queue  chan job
	mailer *Mailer
}

func NewNotifier() *Notifier {
	return &Notifier{
		queue:  make(chan job, 100),
		mailer: NewMailer(os.Getenv("RESEND_API_KEY"), os.Getenv("RESEND_FROM")),
	}
}

func (n *Notifier) schedule(email, subject, body string, userID, repoID int64, issuesIDs []int64) {
	n.queue <- job{
		email, subject, body,
		issuesIDs, repoID, userID}
}

func (n *Notifier) startSending(ctx context.Context, st *store.Store) {
	for {
		select {
		case job := <-n.queue:
			log.Printf("sending email to %s", job.email)
			if err := n.mailer.Send(ctx, job.email, job.subject, job.body); err != nil {
				log.Printf("send to %s failed: %v", job.email, err)
				continue
			}
			for _, issue := range job.issueIDs {
				if _, err := st.MarkNotified(ctx, job.userID, job.repoID, issue); err != nil {
					log.Printf("mark notified %d failed: %v", issue, err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (n *Notifier) Start(ctx context.Context, st *store.Store, workers int) {
	for i := 0; i < workers; i++ {
		go n.startSending(ctx, st)
	}
}

func (n *Notifier) buildMessage(issues []github.Issue, repo store.Repo) string {
	var buffer bytes.Buffer
	for _, is := range issues {
		buffer.WriteString(fmt.Sprintf("#%-6d %s\n         %s\n", is.Number, is.Title, is.HTMLURL))
	}
	buffer.WriteString(fmt.Sprintf("\n%d new \"good first issue\" issues in %s/%s\n", len(issues), repo.Owner, repo.Name))
	return buffer.String()
}

func (n *Notifier) Notify(ctx context.Context, st *store.Store, repo store.Repo, issues []github.Issue) error {
	subs, err := st.ListSubscribersForRepo(ctx, repo.ID)
	if err != nil {
		return err
	}

	var candidates []github.Issue
	issueIDs := make([]int64, 0, len(issues))
	for _, issue := range issues {
		if issue.PullRequest != nil {
			continue
		}
		candidates = append(candidates, issue)
		issueIDs = append(issueIDs, issue.Id)
	}

	userIDs := make([]int64, len(subs))
	for i, sub := range subs {
		userIDs[i] = sub.ID
	}

	notified, err := st.NotifiedIssueIDs(ctx, userIDs, issueIDs)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		seen := notified[sub.ID]
		var fresh []github.Issue
		for _, issue := range candidates {
			if !seen[issue.Id] {
				fresh = append(fresh, issue)
			}
		}

		if len(fresh) == 0 {
			continue
		}

		message := n.buildMessage(fresh, repo)
		subject := fmt.Sprintf("New good first issues in %s/%s", repo.Owner, repo.Name)

		issueIDs := make([]int64, len(fresh))
		for i, issue := range fresh {
			issueIDs[i] = issue.Id
		}

		n.schedule(sub.Email, subject, message, sub.ID, repo.ID, issueIDs)
	}
	return nil
}
