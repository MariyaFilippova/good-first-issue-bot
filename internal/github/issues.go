package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Issue struct {
	Id          int64     `json:"id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	HTMLURL     string    `json:"html_url"`
	Labels      []Label   `json:"labels"`
	PullRequest *struct{} `json:"pull_request"`
}

type Label struct {
	Name string `json:"name"`
}

type Client struct {
	token string
	http  *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		token: token,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

type FetchResult struct {
	Issues      []Issue
	ETag        string // ETag header from a 200 response ("" on 304)
	NotModified bool   // true when GitHub returned 304 Not Modified
}

func (c *Client) FetchIssues(ctx context.Context, owner, name, label string, etag *string) (*FetchResult, error) {
	endpoint := "https://api.github.com/repos/" + owner + "/" + name + "/issues"

	q := url.Values{}
	q.Set("labels", label)
	q.Set("state", "open")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	q.Set("per_page", "100")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	if etag != nil {
		req.Header.Set("If-None-Match", *etag)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// 304: nothing changed since our ETag — cheap, doesn't count against the
	// rate limit. No body to read.
	if resp.StatusCode == http.StatusNotModified {
		return &FetchResult{NotModified: true}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github returned %s", resp.Status)
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}
	return &FetchResult{Issues: issues, ETag: resp.Header.Get("ETag")}, nil
}
