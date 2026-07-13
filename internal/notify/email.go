package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Mailer sends email via the Resend API (https://resend.com). It holds the API
// key and the verified "from" address.
type Mailer struct {
	apiKey string
	from   string
	http   *http.Client
}

func NewMailer(apiKey, from string) *Mailer {
	return &Mailer{
		apiKey: apiKey,
		from:   from,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	if m.apiKey == "" {
		fmt.Printf("=== EMAIL TO %s ===\nSubject: %s\n%s\n", to, subject, body)
		return nil
	}

	payload := struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Text    string   `json:"text"`
	}{From: m.from, To: []string{to}, Subject: subject, Text: body}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend %s: %s", resp.Status, msg)
	}
	return nil
}
