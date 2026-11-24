// Package linear fetches the workspace from Linear's GraphQL API.
package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

const defaultEndpoint = "https://api.linear.app/graphql"

// ErrRateLimited is returned when Linear keeps rate-limiting after all retries.
var ErrRateLimited = errors.New("linear: rate limited")

// Client calls Linear's GraphQL API with an API key.
type Client struct {
	endpoint string
	apiKey   string
	http     *http.Client
	retries  int
	backoff  time.Duration
}

// New returns a client for the given API key.
func New(apiKey string) *Client {
	return &Client{
		endpoint: defaultEndpoint,
		apiKey:   apiKey,
		http:     &http.Client{Timeout: 60 * time.Second},
		retries:  5,
		backoff:  time.Second,
	}
}

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type gqlError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors"`
}

// failure says whether a failed request may be sent again.
type failure int

const (
	final       failure = iota
	rateLimited         // Linear asked us to slow down: always safe to retry
	dropped             // the connection failed: Linear may or may not have run it
)

// do runs one query and decodes its data into out. It retries rate-limit
// responses, and dropped connections for reads (not mutations, which might
// have run already), with increasing delays. Errors never include the API key.
func (c *Client) do(ctx context.Context, query string, vars map[string]any, out any) error {
	body, err := json.Marshal(gqlRequest{Query: query, Variables: vars})
	if err != nil {
		return fmt.Errorf("encoding query: %w", err)
	}
	read := !strings.HasPrefix(strings.TrimSpace(query), "mutation")
	delay := c.backoff
	for attempt := 0; ; attempt++ {
		kind, err := c.post(ctx, body, out)
		if err == nil {
			return nil
		}
		retry := kind == rateLimited || (kind == dropped && read && ctx.Err() == nil)
		if !retry || attempt >= c.retries {
			return err
		}
		wait := delay + time.Duration(rand.Int64N(int64(delay)/2+1))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		delay *= 2
	}
}

// post sends one request and says how it failed.
func (c *Client) post(ctx context.Context, body []byte, out any) (failure, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return final, fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return dropped, fmt.Errorf("calling linear: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return dropped, fmt.Errorf("reading linear response: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return rateLimited, ErrRateLimited
	}
	var gr gqlResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		return final, fmt.Errorf("linear returned HTTP %d with an unreadable body", resp.StatusCode)
	}
	if len(gr.Errors) > 0 {
		e := gr.Errors[0]
		if e.Extensions.Code == "RATELIMITED" {
			return rateLimited, ErrRateLimited
		}
		return final, fmt.Errorf("linear error (%s): %s", e.Extensions.Code, e.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return final, fmt.Errorf("linear returned HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(gr.Data, out); err != nil {
		return final, fmt.Errorf("decoding linear data: %w", err)
	}
	return final, nil
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type connection[T any] struct {
	PageInfo pageInfo `json:"pageInfo"`
	Nodes    []T      `json:"nodes"`
}

// paginate runs a query whose single root field is a connection taking $after,
// calling fn for every page and progress, when set, with the page's record count.
func paginate[T any](ctx context.Context, c *Client, query string, progress func(int), fn func([]T)) error {
	return paginateVars(ctx, c, query, nil, progress, fn)
}

// paginateVars is paginate with extra query variables besides the cursor.
func paginateVars[T any](ctx context.Context, c *Client, query string, vars map[string]any, progress func(int), fn func([]T)) error {
	var after any
	for {
		var data map[string]connection[T]
		page := map[string]any{"after": after}
		for k, v := range vars {
			page[k] = v
		}
		if err := c.do(ctx, query, page, &data); err != nil {
			return err
		}
		if len(data) != 1 {
			return fmt.Errorf("expected one connection in response, got %d", len(data))
		}
		for _, conn := range data {
			fn(conn.Nodes)
			if progress != nil {
				progress(len(conn.Nodes))
			}
			if !conn.PageInfo.HasNextPage {
				return nil
			}
			after = conn.PageInfo.EndCursor
		}
	}
}
