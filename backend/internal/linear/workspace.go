package linear

import (
	"context"
	"fmt"
	"time"

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
	Trashed     int // deleted tasks skipped at fetch time
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
  pageInfo { hasNextPage endCursor } nodes { id name status url createdAt } } }`
	projectsQuery = `query($after: String) { projects(first: 50, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor }
  nodes { id name state url createdAt labels(first: 20) { nodes { id } } initiatives(first: 20) { nodes { id } } } } }`
	issuesQuery = `query($after: String) { issues(first: 100, after: $after, includeArchived: true) {
  pageInfo { hasNextPage endCursor }
  nodes { id identifier title url estimate priority createdAt startedAt completedAt canceledAt archivedAt startedTriageAt triagedAt trashed
    team { id } state { name type } assignee { id } project { id } parent { id } labels(first: 20) { nodes { id } } } } }`
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

// FetchWorkspace downloads the whole workspace: archived items included, deleted
// (trashed) tasks excluded, disabled users included.
func (c *Client) FetchWorkspace(ctx context.Context) (*Workspace, error) {
	ws := &Workspace{}
	steps := []struct {
		name string
		run  func() error
	}{
		{"teams", func() error {
			return paginate(ctx, c, teamsQuery, func(ns []struct {
				ID, Key, Name       string
				IssueEstimationType string `json:"issueEstimationType"`
			}) {
				for _, n := range ns {
					ws.Teams = append(ws.Teams, report.Team{ID: n.ID, Key: n.Key, Name: n.Name, EstimationType: n.IssueEstimationType})
				}
			})
		}},
		{"users", func() error {
			return paginate(ctx, c, usersQuery, func(ns []struct {
				ID, Name    string
				DisplayName string `json:"displayName"`
				Active      bool
			}) {
				for _, n := range ns {
					ws.Users = append(ws.Users, report.User{ID: n.ID, Name: n.Name, DisplayName: n.DisplayName, Active: n.Active})
				}
			})
		}},
		{"issue labels", func() error { return c.fetchLabels(ctx, issueLabelsQuery, "issue", ws) }},
		{"project labels", func() error { return c.fetchLabels(ctx, projectLabelsQuery, "project", ws) }},
		{"initiatives", func() error {
			return paginate(ctx, c, initiativesQuery, func(ns []struct {
				ID, Name, Status, URL string
				CreatedAt             time.Time `json:"createdAt"`
			}) {
				for _, n := range ns {
					ws.Initiatives = append(ws.Initiatives, report.Initiative{ID: n.ID, Name: n.Name, Status: n.Status, URL: n.URL, CreatedAt: n.CreatedAt})
				}
			})
		}},
		{"projects", func() error {
			return paginate(ctx, c, projectsQuery, func(ns []struct {
				ID, Name, State, URL string
				CreatedAt            time.Time `json:"createdAt"`
				Labels               struct{ Nodes []idRef }
				Initiatives          struct{ Nodes []idRef }
			}) {
				for _, n := range ns {
					ws.Projects = append(ws.Projects, report.Project{ID: n.ID, Name: n.Name, State: n.State, URL: n.URL,
						CreatedAt: n.CreatedAt, LabelIDs: ids(n.Labels.Nodes), InitiativeIDs: ids(n.Initiatives.Nodes)})
				}
			})
		}},
		{"tasks", func() error {
			return paginate(ctx, c, issuesQuery, func(ns []issueNode) {
				for _, n := range ns {
					if n.Trashed != nil && *n.Trashed {
						ws.Trashed++
						continue
					}
					ws.Tasks = append(ws.Tasks, toTask(n))
				}
			})
		}},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			return nil, fmt.Errorf("fetching %s: %w", s.name, err)
		}
	}
	return ws, nil
}

func (c *Client) fetchLabels(ctx context.Context, query, kind string, ws *Workspace) error {
	return paginate(ctx, c, query, func(ns []labelNode) {
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

func ids(refs []idRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.ID)
	}
	return out
}
