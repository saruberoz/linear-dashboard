package linear

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"linear-dashboard/backend/internal/report"
)

// Workspace is everything the dashboard needs from Linear, as fetched.
type Workspace struct {
	Teams       []report.Team
	Users       []report.User
	Labels      []report.Label
	Initiatives []report.Initiative
	Projects    []report.Project
	Tasks       []report.Task
	TrashedIDs  []string // deleted tasks skipped at fetch time
}

const (
	teamsQuery = `query($after: String) { teams(first: 100, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor } nodes { id key name issueEstimationType } } }`
	usersQuery = `query($after: String) { users(first: 100, after: $after, includeDisabled: true) {
  pageInfo { hasNextPage endCursor } nodes { id name displayName active } } }`
	issueLabelsQuery = `query($after: String) { issueLabels(first: 250, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor } nodes { id name parent { name } } } }`
	projectLabelsQuery = `query($after: String) { projectLabels(first: 250, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor } nodes { id name parent { name } } } }`
	initiativesQuery = `query($after: String) { initiatives(first: 50, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor } nodes { id name status url createdAt targetDate } } }`
	projectsQuery = `query($after: String) { projects(first: 250, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor }
  nodes { id name state url createdAt startDate targetDate labels(first: 20) { nodes { id } } initiatives(first: 20) { nodes { id } } teams(first: 20) { nodes { id } } } } }`
	issueFields = `pageInfo { hasNextPage endCursor }
  nodes { id identifier title url estimate priority createdAt startedAt completedAt canceledAt archivedAt startedTriageAt triagedAt trashed
    team { id } state { name type } assignee { id } project { id } parent { id } labels(first: 20) { nodes { id } } } } }`
	issuesQuery        = `query($after: String) { issues(first: 100, after: $after, includeArchived: true) {` + issueFields
	changedIssuesQuery = `query($after: String, $since: DateTimeOrDuration) {
  issues(first: 100, after: $after, includeArchived: true, filter: { updatedAt: { gt: $since } }) {` + issueFields
)

type idRef struct {
	ID string `json:"id"`
}

type labelNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Parent *struct {
		Name string `json:"name"`
	} `json:"parent"`
}

type issueNode struct {
	ID              string     `json:"id"`
	Identifier      string     `json:"identifier"`
	Title           string     `json:"title"`
	URL             string     `json:"url"`
	Estimate        *float64   `json:"estimate"`
	Priority        int        `json:"priority"`
	CreatedAt       time.Time  `json:"createdAt"`
	StartedAt       *time.Time `json:"startedAt"`
	CompletedAt     *time.Time `json:"completedAt"`
	CanceledAt      *time.Time `json:"canceledAt"`
	ArchivedAt      *time.Time `json:"archivedAt"`
	StartedTriageAt *time.Time `json:"startedTriageAt"`
	TriagedAt       *time.Time `json:"triagedAt"`
	Trashed         *bool      `json:"trashed"`
	Team            idRef      `json:"team"`
	State           struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"state"`
	Assignee *idRef `json:"assignee"`
	Project  *idRef `json:"project"`
	Parent   *idRef `json:"parent"`
	Labels   struct {
		Nodes []idRef `json:"nodes"`
	} `json:"labels"`
}

// FetchWorkspace downloads the workspace: archived items included, deleted
// (trashed) tasks excluded, disabled users included. With a zero since it
// fetches every task; otherwise only tasks updated in Linear after since, while
// teams, users, labels, initiatives and projects are always fetched in full.
// The lists are fetched concurrently. progress, when set, is called after every
// page with the number of records it held, and must be safe for concurrent use.
func (c *Client) FetchWorkspace(ctx context.Context, since time.Time, progress func(int)) (*Workspace, error) {
	ws := &Workspace{}
	g, ctx := errgroup.WithContext(ctx)
	steps := []struct {
		name string
		run  func() error
	}{
		{"teams", func() error {
			return paginate(ctx, c, teamsQuery, progress, func(ns []struct {
				ID, Key, Name       string
				IssueEstimationType string `json:"issueEstimationType"`
			}) {
				for _, n := range ns {
					ws.Teams = append(ws.Teams, report.Team{ID: n.ID, Key: n.Key, Name: n.Name, EstimationType: n.IssueEstimationType})
				}
			})
		}},
		{"users", func() error {
			return paginate(ctx, c, usersQuery, progress, func(ns []struct {
				ID, Name    string
				DisplayName string `json:"displayName"`
				Active      bool
			}) {
				for _, n := range ns {
					ws.Users = append(ws.Users, report.User{ID: n.ID, Name: n.Name, DisplayName: n.DisplayName, Active: n.Active})
				}
			})
		}},
		{"issue labels", func() error { return c.fetchLabels(ctx, issueLabelsQuery, "issue", ws, progress) }},
		{"project labels", func() error { return c.fetchLabels(ctx, projectLabelsQuery, "project", ws, progress) }},
		{"initiatives", func() error {
			return paginate(ctx, c, initiativesQuery, progress, func(ns []struct {
				ID, Name, Status, URL string
				CreatedAt             time.Time `json:"createdAt"`
				TargetDate            *string   `json:"targetDate"`
			}) {
				for _, n := range ns {
					ws.Initiatives = append(ws.Initiatives, report.Initiative{ID: n.ID, Name: n.Name, Status: n.Status, URL: n.URL, CreatedAt: n.CreatedAt,
						TargetDate: deref(n.TargetDate)})
				}
			})
		}},
		{"projects", func() error {
			return paginate(ctx, c, projectsQuery, progress, func(ns []struct {
				ID, Name, State, URL string
				CreatedAt            time.Time `json:"createdAt"`
				StartDate            *string   `json:"startDate"`
				TargetDate           *string   `json:"targetDate"`
				Labels               struct{ Nodes []idRef }
				Initiatives          struct{ Nodes []idRef }
				Teams                struct{ Nodes []idRef }
			}) {
				for _, n := range ns {
					ws.Projects = append(ws.Projects, report.Project{ID: n.ID, Name: n.Name, State: n.State, URL: n.URL,
						CreatedAt: n.CreatedAt, LabelIDs: ids(n.Labels.Nodes), InitiativeIDs: ids(n.Initiatives.Nodes),
						TeamIDs: ids(n.Teams.Nodes), StartDate: deref(n.StartDate), TargetDate: deref(n.TargetDate)})
				}
			})
		}},
		{"tasks", func() error {
			query, vars := issuesQuery, map[string]any(nil)
			if !since.IsZero() {
				query, vars = changedIssuesQuery, map[string]any{"since": since.UTC().Format(time.RFC3339)}
			}
			return paginateVars(ctx, c, query, vars, progress, func(ns []issueNode) {
				for _, n := range ns {
					if n.Trashed != nil && *n.Trashed {
						ws.TrashedIDs = append(ws.TrashedIDs, n.ID)
						continue
					}
					ws.Tasks = append(ws.Tasks, toTask(n))
				}
			})
		}},
	}
	for _, s := range steps {
		g.Go(func() error {
			if err := s.run(); err != nil {
				return fmt.Errorf("fetching %s: %w", s.name, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return ws, nil
}

func (c *Client) fetchLabels(ctx context.Context, query, kind string, ws *Workspace, progress func(int)) error {
	return paginate(ctx, c, query, progress, func(ns []labelNode) {
		for _, n := range ns {
			l := report.Label{ID: n.ID, Kind: kind, Name: n.Name}
			if n.Parent != nil {
				l.Group = n.Parent.Name
			}
			ws.Labels = append(ws.Labels, l)
		}
	})
}

func toTask(n issueNode) report.Task {
	t := report.Task{
		ID: n.ID, Identifier: n.Identifier, Title: n.Title, URL: n.URL, TeamID: n.Team.ID,
		Estimate: n.Estimate, Priority: n.Priority, StateName: n.State.Name, StateType: n.State.Type,
		CreatedAt: n.CreatedAt, StartedAt: n.StartedAt, CompletedAt: n.CompletedAt,
		CanceledAt: n.CanceledAt, ArchivedAt: n.ArchivedAt, LabelIDs: ids(n.Labels.Nodes),
		StartedTriageAt: n.StartedTriageAt, TriagedAt: n.TriagedAt,
	}
	if n.Assignee != nil {
		t.AssigneeID = n.Assignee.ID
	}
	if n.Project != nil {
		t.ProjectID = n.Project.ID
	}
	if n.Parent != nil {
		t.ParentID = n.Parent.ID
	}
	return t
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ids(refs []idRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.ID)
	}
	return out
}
