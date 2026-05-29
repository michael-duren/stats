package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	graphqlEndpoint = "https://api.github.com/graphql"
	restBase        = "https://api.github.com"
	// userAgent is mandatory: GitHub rejects API requests without one with 403.
	userAgent = "github.com/michael-duren/stats"
)

// Client talks to the GitHub GraphQL + REST APIs. It rotates across the
// supplied personal access tokens round-robin to spread rate-limit budget
// (each token gets 5000 req/hr).
type Client struct {
	tokens []string
	next   atomic.Uint64
	http   *http.Client
}

// NewClient builds a client from one or more PATs. Passing zero tokens is
// allowed (unauthenticated, 60 req/hr) but strongly discouraged.
func NewClient(tokens []string) *Client {
	return &Client{
		tokens: tokens,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) token() string {
	if len(c.tokens) == 0 {
		return ""
	}
	i := c.next.Add(1) - 1
	return c.tokens[int(i%uint64(len(c.tokens)))]
}

// APIError carries the HTTP status so handlers can map upstream failures
// (e.g. 404 user-not-found) to the right response.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("github api: %d: %s", e.Status, e.Message) }

func (c *Client) graphql(ctx context.Context, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if t := c.token(); t != "" {
		req.Header.Set("Authorization", "bearer "+t)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		status := http.StatusBadGateway
		// GitHub returns NOT_FOUND in the errors array for missing users.
		if strings.EqualFold(envelope.Errors[0].Type, "NOT_FOUND") {
			status = http.StatusNotFound
		}
		return &APIError{Status: status, Message: envelope.Errors[0].Message}
	}
	return json.Unmarshal(envelope.Data, out)
}

func (c *Client) restGET(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, restBase+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	if t := c.token(); t != "" {
		req.Header.Set("Authorization", "bearer "+t)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}
	return json.Unmarshal(raw, out)
}
