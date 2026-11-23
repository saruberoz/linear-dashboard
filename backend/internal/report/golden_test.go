package report

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// TestGoldenNovember2026 pins the Go rules to an independent implementation:
// testdata/fixture.json is an anonymized export (cmd/fixture) of the 62
// segment-tagged projects synced on 2026-11-17, and the expected values below
// are what the reviewed prototype's JavaScript rules (mockup/lib/model.js,
// since removed) computed on that same fixture. On 2026-11-23 the effort rule
// changed to count the highest estimate in a task tree instead of sub-tasks;
// the new totals (786, was 1134) match an independent re-computation.
func TestGoldenNovember2026(t *testing.T) {
	raw, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Teams       []Team
		Users       []User
		Labels      []Label
		Initiatives []Initiative
		Projects    []Project
		Tasks       []Task
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	teams := map[string]Team{}
	for _, tm := range f.Teams {
		teams[tm.ID] = tm
	}
	labels := map[string]Label{}
	for _, l := range f.Labels {
		labels[l.ID] = l
	}
	Derive(f.Tasks, teams, labels)
	d := NewDataset(time.Now(), f.Teams, f.Users, f.Labels, f.Initiatives, f.Projects, f.Tasks)
	r, err := ParseRange("2026-11-01", "2026-11-30", jakarta)
	if err != nil {
		t.Fatal(err)
	}
	o := d.Overview(r)

	want := map[string]struct {
		pts      float64
		projects int32
	}{
		"B2B": {227.33, 19}, "B2G": {16.33, 5}, "B2BNG": {0, 0}, "B2C": {542.33, 27},
	}
	for _, s := range o.Segments {
		w := want[s.Key]
		if math.Abs(s.Pts-w.pts) > 0.01 || s.Projects != w.projects {
			t.Errorf("%s: %.2f pts, %d projects; want %.2f pts, %d projects", s.Key, s.Pts, s.Projects, w.pts, w.projects)
		}
	}
	if math.Abs(o.TaggedTotal-786) > 0.01 || math.Abs(o.Summary.Total-786) > 0.01 {
		t.Errorf("tagged total %.2f, summary total %.2f; want 786 for both", o.TaggedTotal, o.Summary.Total)
	}
	if o.Summary.TasksMissing != 44 || o.Summary.TasksTotal != 585 {
		t.Errorf("unestimated %d of %d; want 44 of 585", o.Summary.TasksMissing, o.Summary.TasksTotal)
	}
}
