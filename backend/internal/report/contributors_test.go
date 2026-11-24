package report

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func TestContributorRowsListTheirTeams(t *testing.T) {
	teams := []Team{{ID: "be", Key: "BE", Name: "Backend"}, {ID: "fe", Key: "FE", Name: "Frontend"}}
	tasks := []Task{
		{ID: "1", TeamID: "fe", AssigneeID: "alice", StateType: "completed", Estimate: ptr(2.0)},
		{ID: "2", TeamID: "be", AssigneeID: "alice", StateType: "completed", Estimate: ptr(5.0)},
		{ID: "3", TeamID: "be", AssigneeID: "alice", StateType: "started", Estimate: ptr(1.0)},
		{ID: "4", TeamID: "fe", AssigneeID: "alice", StateType: "canceled", Estimate: ptr(8.0)},
		{ID: "5", TeamID: "fe", StateType: "backlog", Estimate: ptr(3.0)},
	}
	Derive(tasks, map[string]Team{"be": teams[0], "fe": teams[1]}, nil)
	d := NewDataset(time.Now(), teams, []User{{ID: "alice", DisplayName: "alice"}}, nil, nil, nil, tasks)
	var ws []weighted
	for i := range d.Tasks {
		ws = append(ws, weighted{t: &d.Tasks[i], w: 1})
	}

	got := map[string][]string{}
	for _, row := range d.contributors(ws, DimStatus).Rows {
		for _, tm := range row.Teams {
			got[row.AssigneeId] = append(got[row.AssigneeId], fmt.Sprintf("%s %s %v pts %d tasks", tm.Key, tm.Name, tm.Pts, tm.Count))
		}
	}
	want := []string{"BE Backend 6 pts 2 tasks", "FE Frontend 2 pts 1 tasks"}
	if !slices.Equal(got["alice"], want) {
		t.Errorf("alice's teams = %q, want %q (most effort first, canceled work left out)", got["alice"], want)
	}
	if !slices.Equal(got[Unassigned], []string{"FE Frontend 3 pts 1 tasks"}) {
		t.Errorf("unassigned teams = %q, want FE only", got[Unassigned])
	}
}
