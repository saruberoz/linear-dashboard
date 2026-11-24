package snapshot

import (
	"testing"
	"time"

	"linear-dashboard/backend/internal/report"
)

func TestMergeTasks(t *testing.T) {
	est := 3.0
	stored := []report.Task{
		{ID: "same", Title: "unchanged", EffectivePts: 2},
		{ID: "edited", Title: "old title"},
		{ID: "trashed", Title: "deleted in Linear"},
	}
	changed := []report.Task{
		{ID: "edited", Title: "new title", Estimate: &est},
		{ID: "new", Title: "created since the last sync"},
	}
	got := mergeTasks(stored, changed, []string{"trashed", "never-stored"})

	titles := map[string]string{}
	for _, tk := range got {
		if _, dup := titles[tk.ID]; dup {
			t.Errorf("task %s appears twice", tk.ID)
		}
		titles[tk.ID] = tk.Title
	}
	want := map[string]string{"same": "unchanged", "edited": "new title", "new": "created since the last sync"}
	if len(titles) != len(want) {
		t.Errorf("merged tasks = %v, want %v", titles, want)
	}
	for id, title := range want {
		if titles[id] != title {
			t.Errorf("task %s title = %q, want %q", id, titles[id], title)
		}
	}

	got[0].EffectivePts = 99
	if stored[0].EffectivePts != 2 {
		t.Errorf("changing a merged task changed the stored copy")
	}
}

func TestRefreshCountsLikeAFullSync(t *testing.T) {
	est := func(v float64) *float64 { return &v }
	teams := map[string]report.Team{"t": {ID: "t", EstimationType: "linear"}}
	full := []report.Task{
		{ID: "epic", TeamID: "t", StateType: "started", Estimate: est(8)},
		{ID: "sync", Title: "Weekly meeting", TeamID: "t", StateType: "started", ParentID: "epic"},
		{ID: "child", Title: "Diskusi middleware", TeamID: "t", StateType: "started", ParentID: "sync", Estimate: est(1)},
	}
	afterFull := countable(full)
	report.Derive(afterFull, teams, nil)
	stored := report.NewDataset(time.Now(), nil, nil, nil, nil, nil, afterFull).Tasks

	edited := countable([]report.Task{{ID: "child", Title: "Diskusi middleware v2", TeamID: "t", StateType: "started", ParentID: "sync", Estimate: est(1)}})
	afterRefresh := mergeTasks(stored, edited, nil)
	report.Derive(afterRefresh, teams, nil)

	pts := func(ts []report.Task) float64 {
		var sum float64
		for _, t := range ts {
			sum += t.EffectivePts
		}
		return sum
	}
	if a, b := pts(afterFull), pts(afterRefresh); a != b {
		t.Errorf("full sync counts %v pts, refresh %v pts; want the same", a, b)
	}
}
