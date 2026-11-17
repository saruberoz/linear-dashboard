package linear

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Split so the source never looks like a real key to secret scanners.
const testKey = "lin_api_" + "TESTKEYshouldNeverLeak0000000000000000"

func newTestClient(srv *httptest.Server) *Client {
	c := New(testKey)
	c.endpoint = srv.URL
	c.backoff = time.Millisecond
	return c
}

func TestPaginateFollowsCursors(t *testing.T) {
	pages := []string{
		`{"data":{"teams":{"pageInfo":{"hasNextPage":true,"endCursor":"c1"},"nodes":[{"id":"a"}]}}}`,
		`{"data":{"teams":{"pageInfo":{"hasNextPage":true,"endCursor":"c2"},"nodes":[{"id":"b"}]}}}`,
		`{"data":{"teams":{"pageInfo":{"hasNextPage":false,"endCursor":""},"nodes":[{"id":"c"}]}}}`,
	}
	var calls atomic.Int32
	var cursors []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != testKey {
			t.Errorf("Authorization header = %q", got)
		}
		var req gqlRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		cursors = append(cursors, req.Variables["after"])
		_, _ = io.WriteString(w, pages[calls.Add(1)-1])
	}))
	defer srv.Close()

	var got []string
	err := paginate(context.Background(), newTestClient(srv), teamsQuery, func(ns []idRef) {
		for _, n := range ns {
			got = append(got, n.ID)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "a,b,c" {
		t.Errorf("ids = %v, want a,b,c", got)
	}
	if cursors[0] != nil || cursors[1] != "c1" || cursors[2] != "c2" {
		t.Errorf("cursors = %v, want [nil c1 c2]", cursors)
	}
}

func TestGraphQLErrorSurfacesWithoutKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"errors":[{"message":"Authentication required","extensions":{"code":"AUTHENTICATION_ERROR"}}]}`)
	}))
	defer srv.Close()

	var out any
	err := newTestClient(srv).do(context.Background(), "{ viewer { id } }", nil, &out)
	if err == nil || !strings.Contains(err.Error(), "AUTHENTICATION_ERROR") {
		t.Fatalf("err = %v, want an AUTHENTICATION_ERROR error", err)
	}
	if strings.Contains(err.Error(), testKey) {
		t.Errorf("error message leaks the API key: %v", err)
	}
}

func TestRateLimitIsRetried(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			_, _ = io.WriteString(w, `{"errors":[{"message":"slow down","extensions":{"code":"RATELIMITED"}}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"viewer":{"id":"me"}}}`)
	}))
	defer srv.Close()

	var out struct{ Viewer idRef }
	if err := newTestClient(srv).do(context.Background(), "{ viewer { id } }", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.Viewer.ID != "me" || calls.Load() != 3 {
		t.Errorf("viewer=%q calls=%d, want me after 3 calls", out.Viewer.ID, calls.Load())
	}
}

func TestRateLimitGivesUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	var out any
	if err := newTestClient(srv).do(context.Background(), "{ x }", nil, &out); !errors.Is(err, ErrRateLimited) {
		t.Errorf("err = %v, want ErrRateLimited", err)
	}
}

func TestTrashedTasksAreSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch {
		case strings.Contains(string(body), "issues("):
			_, _ = io.WriteString(w, `{"data":{"issues":{"pageInfo":{"hasNextPage":false},"nodes":[
			  {"id":"keep","identifier":"A-1","createdAt":"2026-11-01T00:00:00Z","team":{"id":"t"},"state":{"name":"Todo","type":"unstarted"},"labels":{"nodes":[]}},
			  {"id":"gone","identifier":"A-2","trashed":true,"createdAt":"2026-11-01T00:00:00Z","team":{"id":"t"},"state":{"name":"Todo","type":"unstarted"},"labels":{"nodes":[]}}]}}}`)
		default:
			// every other connection is empty
			var req gqlRequest
			_ = json.Unmarshal(body, &req)
			field := strings.Fields(strings.SplitN(req.Query, "{", 3)[1])[0]
			field = strings.SplitN(field, "(", 2)[0]
			_, _ = io.WriteString(w, `{"data":{"`+field+`":{"pageInfo":{"hasNextPage":false},"nodes":[]}}}`)
		}
	}))
	defer srv.Close()

	ws, err := newTestClient(srv).FetchWorkspace(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Tasks) != 1 || ws.Tasks[0].ID != "keep" || ws.Trashed != 1 {
		t.Errorf("tasks=%v trashed=%d, want only 'keep' and 1 trashed", ws.Tasks, ws.Trashed)
	}
}
