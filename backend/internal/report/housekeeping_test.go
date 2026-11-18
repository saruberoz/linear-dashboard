package report

import (
	"sort"
	"testing"
	"time"
)

func housekeepingRange(t *testing.T) Range {
	t.Helper()
	r, err := ParseRange("2026-11-01", "2026-11-30", jakarta)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func build(projects []Project, initiatives []Initiative, tasks []Task) *Dataset {
	teams := []Team{{ID: "est", EstimationType: "linear"}}
	Derive(tasks, map[string]Team{"est": teams[0]}, map[string]Label{})
	return NewDataset(time.Now(), teams, []User{{ID: "alice", DisplayName: "alice"}}, nil, initiatives, projects, tasks)
}

func TestTasksWithoutProjectGroupsInRangeOpenTasks(t *testing.T) {
	nov, sep := at("2026-11-05 10:00"), at("2026-09-05 10:00")
	d := build([]Project{{ID: "p", Name: "P", CreatedAt: nov}}, nil, []Task{
		{ID: "a1", TeamID: "est", AssigneeID: "alice", StateType: "backlog", CreatedAt: nov},
		{ID: "a2", TeamID: "est", StateType: "backlog", CreatedAt: nov},
		{ID: "canceled", TeamID: "est", StateType: "canceled", CreatedAt: nov, CanceledAt: ptr(nov)},
		{ID: "old", TeamID: "est", StateType: "completed", CreatedAt: sep, CompletedAt: ptr(sep)},
		{ID: "inproject", TeamID: "est", ProjectID: "p", StateType: "backlog", CreatedAt: nov},
	})
	r := housekeepingRange(t)
	res := d.TasksWithoutProject(r)
	got := map[string]int{}
	for _, g := range res.Groups {
		got[g.Assignee] = len(g.Tasks)
	}
	if res.Total != 2 || got["alice"] != 1 || got["Unassigned"] != 1 || len(got) != 2 {
		t.Errorf("total %d groups %v, want 2 tasks: alice 1, Unassigned 1", res.Total, got)
	}
	if n := d.Menu(r).NoProjectTaskCount; n != 2 {
		t.Errorf("menu count = %d, want 2", n)
	}
}

func TestMissingTimelineListsOpenWorkWithoutDates(t *testing.T) {
	nov, sep := at("2026-11-05 10:00"), at("2026-09-05 10:00")
	d := build([]Project{
		{ID: "dated", Name: "Dated", State: "started", CreatedAt: nov, StartDate: "2026-11-01", TargetDate: "2026-12-01"},
		{ID: "notarget", Name: "No target", State: "started", CreatedAt: nov, StartDate: "2026-11-01"},
		{ID: "nostart", Name: "No start", State: "planned", CreatedAt: nov, TargetDate: "2026-12-01"},
		{ID: "canceled", Name: "Canceled", State: "canceled", CreatedAt: nov},
		{ID: "outofrange", Name: "Out of range", State: "backlog", CreatedAt: sep},
	}, []Initiative{
		{ID: "open", Name: "[W] Open", Status: "Active", CreatedAt: sep},
		{ID: "planned", Name: "[WS] Planned", Status: "Active", CreatedAt: sep, TargetDate: "2026-12-31"},
		{ID: "sprint", Name: "46.Cloud Sprint", Status: "Active", CreatedAt: sep},
		{ID: "done", Name: "[W] Done", Status: "Completed", CreatedAt: sep},
	}, nil)
	r := housekeepingRange(t)
	res := d.MissingTimeline(r)
	var projects, initiatives []string
	for _, p := range res.Projects {
		projects = append(projects, p.Id)
	}
	for _, i := range res.Initiatives {
		initiatives = append(initiatives, i.Id)
	}
	sort.Strings(projects)
	if want := []string{"canceled", "nostart", "notarget"}; !equal(projects, want) {
		t.Errorf("projects %v, want %v (canceled listed for the state filter)", projects, want)
	}
	if want := []string{"open"}; !equal(initiatives, want) {
		t.Errorf("initiatives %v, want %v (only open [W]/[WS] ones without a target date)", initiatives, want)
	}
	for _, p := range res.Projects {
		if p.Id == "notarget" && (p.StartDate != "2026-11-01" || p.TargetDate != "") {
			t.Errorf("notarget dates start=%q target=%q, want 2026-11-01 and empty", p.StartDate, p.TargetDate)
		}
	}
	if n := d.Menu(r).MissingTimelineCount; n != 3 {
		t.Errorf("menu count = %d, want 3 (2 projects not canceled + 1 initiative)", n)
	}
}

func TestProjectsWithoutTasksIgnoresTheRange(t *testing.T) {
	nov, old := at("2026-11-05 10:00"), at("2025-01-05 10:00")
	d := build([]Project{
		{ID: "empty", Name: "Empty", State: "backlog", CreatedAt: old},
		{ID: "emptycanceled", Name: "Empty canceled", State: "canceled", CreatedAt: old},
		{ID: "busy", Name: "Busy", State: "started", CreatedAt: old},
	}, nil, []Task{{ID: "t", TeamID: "est", ProjectID: "busy", StateType: "backlog", CreatedAt: nov}})
	r := housekeepingRange(t)
	var got []string
	for _, p := range d.ProjectsWithoutTasks(r).Projects {
		got = append(got, p.Id)
	}
	sort.Strings(got)
	if want := []string{"empty", "emptycanceled"}; !equal(got, want) {
		t.Errorf("projects %v, want %v", got, want)
	}
	if n := d.Menu(r).EmptyProjectCount; n != 1 {
		t.Errorf("menu count = %d, want 1 (canceled left out)", n)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
