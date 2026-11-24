package report

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func frozenPrev() *Dataset {
	nov := at("2026-11-05 10:00")
	return build(
		[]Project{
			{ID: "done-proj", Name: "Done", State: "completed", CreatedAt: nov},
			{ID: "open-proj", Name: "Open", State: "started", CreatedAt: nov, InitiativeIDs: []string{"done-ini", "sprint"}},
			{ID: "other-proj", Name: "Other", State: "started", CreatedAt: nov},
		},
		[]Initiative{
			{ID: "done-ini", Name: "[W] Shipped", Status: "Completed"},
			{ID: "live-ini", Name: "[W] Running", Status: "Active"},
			{ID: "sprint", Name: "46.Cloud Sprint", Status: "Completed"},
		},
		[]Task{
			{ID: "done-task", TeamID: "est", StateType: "completed", ProjectID: "other-proj", Estimate: ptr(3.0), CreatedAt: nov},
			{ID: "open-in-done-proj", TeamID: "est", StateType: "started", ProjectID: "done-proj", Estimate: ptr(2.0), CreatedAt: nov},
			{ID: "open-task", TeamID: "est", StateType: "started", ProjectID: "other-proj", Estimate: ptr(1.0), CreatedAt: nov},
			{ID: "meeting", Title: "Weekly meeting", TeamID: "est", StateType: "completed", Estimate: ptr(1.0), CreatedAt: nov},
		},
	)
}

func TestFreezeHoldsChangesToCompletedWork(t *testing.T) {
	nov := at("2026-11-05 10:00")
	tasks := []Task{
		{ID: "done-task", TeamID: "est", StateType: "completed", ProjectID: "other-proj", Estimate: ptr(5.0), CreatedAt: nov},
		{ID: "open-in-done-proj", TeamID: "est", StateType: "started", ProjectID: "other-proj", Estimate: nil, CreatedAt: nov},
		{ID: "open-task", TeamID: "est", StateType: "started", ProjectID: "other-proj", Estimate: ptr(8.0), CreatedAt: nov},
		{ID: "new-in-done-proj", TeamID: "est", StateType: "backlog", ProjectID: "done-proj", CreatedAt: nov},
		{ID: "meeting", Title: "Weekly meeting", TeamID: "est", StateType: "completed", Estimate: ptr(4.0), CreatedAt: nov},
	}
	projects := []Project{
		{ID: "done-proj", State: "completed"},
		{ID: "open-proj", State: "started", InitiativeIDs: []string{"live-ini"}},
		{ID: "other-proj", State: "started", InitiativeIDs: []string{"done-ini"}},
	}
	got := Freeze(frozenPrev(), tasks, projects)

	want := []FrozenChange{
		{Kind: ChangeEstimate, TaskID: "done-task", Old: "3", New: "5", FrozenBy: "task"},
		{Kind: ChangeEstimate, TaskID: "open-in-done-proj", Old: "2", New: "", FrozenBy: "project:done-proj"},
		{Kind: ChangeInitiative, ProjectID: "open-proj", InitiativeID: "done-ini", Old: Linked, New: "", FrozenBy: "initiative:done-ini"},
		{Kind: ChangeInitiative, ProjectID: "other-proj", InitiativeID: "done-ini", Old: "", New: Linked, FrozenBy: "initiative:done-ini"},
		{Kind: ChangeProject, TaskID: "new-in-done-proj", Old: "", New: "done-proj", FrozenBy: "project:done-proj"},
		{Kind: ChangeProject, TaskID: "open-in-done-proj", Old: "done-proj", New: "other-proj", FrozenBy: "project:done-proj"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("changes:\n got %+v\nwant %+v", got, want)
	}

	byID := map[string]Task{}
	for _, tk := range tasks {
		byID[tk.ID] = tk
	}
	if e := byID["done-task"].Estimate; e == nil || *e != 3 {
		t.Errorf("completed task estimate = %v, want the frozen 3", e)
	}
	if tk := byID["open-in-done-proj"]; tk.ProjectID != "done-proj" || tk.Estimate == nil || *tk.Estimate != 2 {
		t.Errorf("task in completed project = %s / %v, want done-proj / 2", tk.ProjectID, tk.Estimate)
	}
	if tk := byID["new-in-done-proj"]; tk.ProjectID != "" {
		t.Errorf("task created in completed project kept project %q, want none until approved", tk.ProjectID)
	}
	if e := byID["open-task"].Estimate; e == nil || *e != 8 {
		t.Errorf("open task estimate = %v, want the new 8 (not frozen)", e)
	}
	if e := byID["meeting"].Estimate; e == nil || *e != 4 {
		t.Errorf("ignored task estimate = %v, want the new 4 (never frozen)", e)
	}
	if ids := projects[1].InitiativeIDs; !slices.Equal(ids, []string{"live-ini", "done-ini"}) {
		t.Errorf("open-proj initiatives = %v, want live-ini kept and done-ini restored; the hidden sprint unlink goes through", ids)
	}
	if ids := projects[2].InitiativeIDs; len(ids) != 0 {
		t.Errorf("other-proj initiatives = %v, want the new link to the completed initiative held back", ids)
	}
}

func TestFreezeWithoutChangesOrPrevious(t *testing.T) {
	prev := frozenPrev()
	tasks := slices.Clone(prev.Tasks)
	projects := []Project{*prev.Projects["done-proj"], *prev.Projects["open-proj"], *prev.Projects["other-proj"]}
	if got := Freeze(prev, tasks, projects); len(got) != 0 {
		t.Errorf("unchanged data gave changes %+v", got)
	}
	if got := Freeze(nil, tasks, projects); got != nil {
		t.Errorf("first sync gave changes %+v", got)
	}
}

func TestFreezeLetsAFirstEstimateThrough(t *testing.T) {
	nov := at("2026-11-05 10:00")
	prev := build(
		[]Project{{ID: "done-proj", State: "completed", CreatedAt: nov}},
		nil,
		[]Task{
			{ID: "done-unestimated", TeamID: "est", StateType: "completed", CreatedAt: nov},
			{ID: "unestimated-in-done-proj", TeamID: "est", StateType: "started", ProjectID: "done-proj", CreatedAt: nov},
			{ID: "done-estimated", TeamID: "est", StateType: "completed", Estimate: ptr(1.0), CreatedAt: nov},
		},
	)
	tasks := []Task{
		{ID: "done-unestimated", TeamID: "est", StateType: "completed", Estimate: ptr(7.0), CreatedAt: nov},
		{ID: "unestimated-in-done-proj", TeamID: "est", StateType: "started", ProjectID: "done-proj", Estimate: ptr(2.0), CreatedAt: nov},
		{ID: "done-estimated", TeamID: "est", StateType: "completed", CreatedAt: nov},
	}
	got := Freeze(prev, tasks, []Project{{ID: "done-proj", State: "completed"}})

	want := []FrozenChange{{Kind: ChangeEstimate, TaskID: "done-estimated", Old: "1", New: "", FrozenBy: "task"}}
	if !slices.Equal(got, want) {
		t.Errorf("changes:\n got %+v\nwant %+v", got, want)
	}
	if e := tasks[0].Estimate; e == nil || *e != 7 {
		t.Errorf("first estimate on a completed task = %v, want 7 kept", e)
	}
	if e := tasks[1].Estimate; e == nil || *e != 2 {
		t.Errorf("first estimate in a completed project = %v, want 2 kept", e)
	}
	if e := tasks[2].Estimate; e == nil || *e != 1 {
		t.Errorf("removed estimate = %v, want 1 restored", e)
	}
}

func TestWithChangeAppliesAnApprovedChange(t *testing.T) {
	prev := frozenPrev()
	got, err := prev.WithChange(FrozenChange{Kind: ChangeEstimate, TaskID: "done-task", Old: "3", New: "5"})
	if err != nil {
		t.Fatal(err)
	}
	if tk := got.taskByID["done-task"]; *tk.Estimate != 5 || tk.EffectivePts != 5 {
		t.Errorf("approved estimate = %v (counted %v), want 5 counted", *tk.Estimate, tk.EffectivePts)
	}
	if tk := prev.taskByID["done-task"]; *tk.Estimate != 3 {
		t.Errorf("approving changed the previous copy to %v", *tk.Estimate)
	}

	got, err = prev.WithChange(FrozenChange{Kind: ChangeProject, TaskID: "open-in-done-proj", Old: "done-proj", New: "other-proj"})
	if err != nil {
		t.Fatal(err)
	}
	if tk := got.taskByID["open-in-done-proj"]; tk.ProjectID != "other-proj" {
		t.Errorf("approved project = %q, want other-proj", tk.ProjectID)
	}

	got, err = prev.WithChange(FrozenChange{Kind: ChangeInitiative, ProjectID: "open-proj", InitiativeID: "done-ini", Old: Linked})
	if err != nil {
		t.Fatal(err)
	}
	if ids := got.Projects["open-proj"].InitiativeIDs; slices.Contains(ids, "done-ini") {
		t.Errorf("approved unlink left initiatives %v", ids)
	}
	if ids := got.Initiatives["done-ini"].ProjectIDs; slices.Contains(ids, "open-proj") {
		t.Errorf("initiative still lists the unlinked project: %v", ids)
	}
	if ids := prev.Projects["open-proj"].InitiativeIDs; !slices.Contains(ids, "done-ini") {
		t.Errorf("approving changed the previous copy's links to %v", ids)
	}

	if _, err := prev.WithChange(FrozenChange{Kind: ChangeEstimate, TaskID: "gone", New: "1"}); !errors.Is(err, ErrChangeTarget) {
		t.Errorf("missing task err = %v, want ErrChangeTarget", err)
	}
}

func TestRevertNoteExplainsTheUndo(t *testing.T) {
	d := frozenPrev()
	tests := []struct {
		c    FrozenChange
		want string
	}{
		{FrozenChange{Kind: ChangeEstimate, TaskID: "done-task", Old: "3", New: "5", FrozenBy: "task"},
			"The estimate was changed from 3 pts to 5 pts after the task was completed. The estimate is set back to 3 pts."},
		{FrozenChange{Kind: ChangeProject, TaskID: "x", Old: "", New: "done-proj", FrozenBy: "project:done-proj"},
			"This task was moved from no project to Done after project Done was completed. It is moved back to no project."},
		{FrozenChange{Kind: ChangeInitiative, ProjectID: "other-proj", InitiativeID: "done-ini", New: Linked, FrozenBy: "initiative:done-ini"},
			"This project was linked to initiative [W] Shipped after initiative [W] Shipped was completed. The link is removed."},
	}
	for _, tt := range tests {
		note := d.RevertNote(tt.c, "  ")
		if !strings.Contains(note, tt.want) || strings.Contains(note, "Reason:") {
			t.Errorf("note = %q, want it to contain %q and no empty reason", note, tt.want)
		}
	}
	if note := d.RevertNote(tests[0].c, "Scope was agreed at sprint start"); !strings.HasSuffix(note, "Reason: Scope was agreed at sprint start") {
		t.Errorf("note = %q, want it to end with the reason", note)
	}
}
