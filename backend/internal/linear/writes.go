package linear

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotDone is returned when Linear accepts a write but reports it did not succeed.
var ErrNotDone = errors.New("linear: the change was not made")

type mutationResult struct {
	Success bool `json:"success"`
}

// mutate runs a single-field mutation and checks its success flag.
func (c *Client) mutate(ctx context.Context, query string, vars map[string]any) error {
	var data map[string]mutationResult
	if err := c.do(ctx, query, vars, &data); err != nil {
		return err
	}
	for _, r := range data {
		if !r.Success {
			return ErrNotDone
		}
	}
	return nil
}

// SetEstimate sets an issue's estimate; nil removes it.
func (c *Client) SetEstimate(ctx context.Context, issueID string, estimate *float64) error {
	return c.mutate(ctx, `mutation($id: String!, $estimate: Int) {
  issueUpdate(id: $id, input: { estimate: $estimate }) { success } }`,
		map[string]any{"id": issueID, "estimate": intEstimate(estimate)})
}

// SetProject moves an issue to a project; "" takes it out of any project.
func (c *Client) SetProject(ctx context.Context, issueID, projectID string) error {
	var project any
	if projectID != "" {
		project = projectID
	}
	return c.mutate(ctx, `mutation($id: String!, $project: String) {
  issueUpdate(id: $id, input: { projectId: $project }) { success } }`,
		map[string]any{"id": issueID, "project": project})
}

// LinkProject adds a project to an initiative.
func (c *Client) LinkProject(ctx context.Context, projectID, initiativeID string) error {
	return c.mutate(ctx, `mutation($project: String!, $initiative: String!) {
  initiativeToProjectCreate(input: { projectId: $project, initiativeId: $initiative }) { success } }`,
		map[string]any{"project": projectID, "initiative": initiativeID})
}

// UnlinkProject removes a project from an initiative. It is a no-op when the
// link is already gone.
func (c *Client) UnlinkProject(ctx context.Context, projectID, initiativeID string) error {
	var data struct {
		Project struct {
			InitiativeToProjects struct {
				Nodes []struct {
					ID         string `json:"id"`
					Initiative idRef  `json:"initiative"`
				} `json:"nodes"`
			} `json:"initiativeToProjects"`
		} `json:"project"`
	}
	err := c.do(ctx, `query($id: String!) { project(id: $id) {
  initiativeToProjects(first: 50) { nodes { id initiative { id } } } } }`, map[string]any{"id": projectID}, &data)
	if err != nil {
		return err
	}
	for _, n := range data.Project.InitiativeToProjects.Nodes {
		if n.Initiative.ID == initiativeID {
			return c.mutate(ctx, `mutation($id: String!) { initiativeToProjectDelete(id: $id) { success } }`,
				map[string]any{"id": n.ID})
		}
	}
	return nil
}

// Comment target: exactly one of IssueID and ProjectID is set.
type CommentTarget struct {
	IssueID   string
	ProjectID string
}

// Comment writes a Markdown comment on an issue or a project.
func (c *Client) Comment(ctx context.Context, target CommentTarget, body string) error {
	input := map[string]any{"body": body}
	switch {
	case target.IssueID != "":
		input["issueId"] = target.IssueID
	case target.ProjectID != "":
		input["projectId"] = target.ProjectID
	default:
		return fmt.Errorf("linear: comment needs an issue or a project")
	}
	return c.mutate(ctx, `mutation($input: CommentCreateInput!) { commentCreate(input: $input) { success } }`,
		map[string]any{"input": input})
}

// intEstimate converts an estimate for Linear's Int estimate field.
func intEstimate(e *float64) any {
	if e == nil {
		return nil
	}
	return int(*e)
}
