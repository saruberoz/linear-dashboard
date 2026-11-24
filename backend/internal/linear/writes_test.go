package linear

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// recorder answers every query with reply and keeps the requests it saw.
type recorder struct {
	mu   sync.Mutex
	reqs []gqlRequest
}

func (rec *recorder) server(t *testing.T, reply func(gqlRequest) string) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req gqlRequest
		_ = json.Unmarshal(body, &req)
		rec.mu.Lock()
		rec.reqs = append(rec.reqs, req)
		rec.mu.Unlock()
		_, _ = io.WriteString(w, reply(req))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func ok(field string) string { return `{"data":{"` + field + `":{"success":true}}}` }

func TestSetEstimateAndProjectSendNullsToClear(t *testing.T) {
	rec := &recorder{}
	c := newTestClient(rec.server(t, func(gqlRequest) string { return ok("issueUpdate") }))
	ctx := context.Background()
	three := 3.0
	for _, err := range []error{
		c.SetEstimate(ctx, "i1", &three), c.SetEstimate(ctx, "i1", nil),
		c.SetProject(ctx, "i1", "p1"), c.SetProject(ctx, "i1", ""),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	got := []any{rec.reqs[0].Variables["estimate"], rec.reqs[1].Variables["estimate"], rec.reqs[2].Variables["project"], rec.reqs[3].Variables["project"]}
	want := []any{float64(3), nil, "p1", nil}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("request %d sent %v, want %v", i, got[i], want[i])
		}
	}
	if _, present := rec.reqs[1].Variables["estimate"]; !present {
		t.Errorf("clearing the estimate must send estimate: null, not leave it out")
	}
}

func TestUnlinkProjectDeletesOnlyThatLink(t *testing.T) {
	rec := &recorder{}
	c := newTestClient(rec.server(t, func(r gqlRequest) string {
		if strings.Contains(r.Query, "initiativeToProjects") {
			return `{"data":{"project":{"initiativeToProjects":{"nodes":[
			  {"id":"link-a","initiative":{"id":"ini-a"}},{"id":"link-b","initiative":{"id":"ini-b"}}]}}}}`
		}
		return ok("initiativeToProjectDelete")
	}))
	if err := c.UnlinkProject(context.Background(), "p1", "ini-b"); err != nil {
		t.Fatal(err)
	}
	if len(rec.reqs) != 2 || rec.reqs[1].Variables["id"] != "link-b" {
		t.Errorf("requests %+v, want a lookup then a delete of link-b", rec.reqs)
	}
}

func TestCommentTargetsAndFailure(t *testing.T) {
	rec := &recorder{}
	c := newTestClient(rec.server(t, func(gqlRequest) string { return `{"data":{"commentCreate":{"success":false}}}` }))
	err := c.Comment(context.Background(), CommentTarget{ProjectID: "p1"}, "note")
	if !errors.Is(err, ErrNotDone) {
		t.Errorf("err = %v, want ErrNotDone when Linear reports no success", err)
	}
	input, _ := rec.reqs[0].Variables["input"].(map[string]any)
	if input["projectId"] != "p1" || input["body"] != "note" || input["issueId"] != nil {
		t.Errorf("comment input = %v, want the project and body only", input)
	}
	if err := c.Comment(context.Background(), CommentTarget{}, "note"); err == nil {
		t.Errorf("a comment with no target must fail")
	}
}
