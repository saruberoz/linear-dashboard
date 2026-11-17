package report

import (
	"testing"
	"time"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

// timeFixture: bugs and features with known durations (days) in November 2026.
func timeFixture(t *testing.T) (*Dataset, Range) {
	t.Helper()
	day := func(d int) *time.Time { v := at("2026-11-01 00:00").AddDate(0, 0, d); return &v }
	labels := []Label{
		{ID: "lb2b", Kind: "project", Name: "B2B"},
		{ID: "bug", Kind: "issue", Name: "Bug"},
		{ID: "repeat", Kind: "issue", Name: "Repeat Bug - Agent"},
		{ID: "feat", Kind: "issue", Name: "Feature"},
	}
	tasks := []Task{
		// done bugs: cycle 2, 4, 10 days; lead 3, 6, 12 days
		{ID: "b1", Identifier: "X-1", ProjectID: "p", Priority: 2, StateType: "completed", CreatedAt: *day(0), StartedAt: day(1), CompletedAt: day(3), LabelIDs: []string{"bug"}},
		{ID: "b2", Identifier: "X-2", ProjectID: "p", Priority: 2, StateType: "completed", CreatedAt: *day(0), StartedAt: day(2), CompletedAt: day(6), LabelIDs: []string{"repeat"}},
		{ID: "b3", Identifier: "X-3", ProjectID: "p", Priority: 1, StateType: "completed", CreatedAt: *day(0), StartedAt: day(2), CompletedAt: day(12), LabelIDs: []string{"bug"}},
		// done feature: cycle 1, triaged after 5 days
		{ID: "f1", Identifier: "X-4", ProjectID: "p", Priority: 3, StateType: "completed", CreatedAt: *day(0), StartedAt: day(5), CompletedAt: day(6),
			StartedTriageAt: day(0), TriagedAt: day(5), LabelIDs: []string{"feat"}},
		// open bug, created on day 2; never started
		{ID: "b4", Identifier: "X-5", ProjectID: "p", Priority: 1, StateType: "unstarted", CreatedAt: *day(2), LabelIDs: []string{"bug"}},
		// done without ever being started: no cycle time, has lead time 8
		{ID: "b5", Identifier: "X-6", ProjectID: "p", Priority: 4, StateType: "completed", CreatedAt: *day(0), CompletedAt: day(8), LabelIDs: []string{"bug"}},
		// canceled bug: in no time population
		{ID: "b6", Identifier: "X-7", ProjectID: "p", StateType: "canceled", CreatedAt: *day(0), StartedAt: day(1), CanceledAt: day(2), LabelIDs: []string{"bug"}},
	}
	d := NewDataset(at("2026-11-22 00:00"), nil, nil, labels, nil,
		[]Project{{ID: "p", Name: "P", CreatedAt: *day(0), LabelIDs: []string{"lb2b"}}}, tasks)
	r, err := ParseRange("2026-11-01", "2026-11-30", jakarta)
	if err != nil {
		t.Fatal(err)
	}
	return d, r
}

func TestTimeMeasures(t *testing.T) {
	d, r := timeFixture(t)
	bugs := &pb.InsightFilters{LabelMatch: "bug"}
	tests := []struct {
		name   string
		spec   *pb.InsightSpec
		median float64
		n      int32
		groups map[string]float64
	}{
		{
			name:   "bug cycle time skips never-started and canceled bugs",
			spec:   &pb.InsightSpec{Measure: MeasureCycleTime, Slice: "priority", View: "bar", Filters: bugs},
			median: 4, n: 3, groups: map[string]float64{"2": 3, "1": 10}, // High: median(2,4)=3, Urgent: 10
		},
		{
			name:   "bug lead time includes never-started bugs",
			spec:   &pb.InsightSpec{Measure: MeasureLeadTime, Slice: "status", View: "metric", Filters: bugs},
			median: 7, n: 4, // 3, 6, 8, 12
		},
		{
			name:   "label contains matches every bug label spelling",
			spec:   &pb.InsightSpec{Measure: MeasureLeadTime, Slice: "label", View: "table", Filters: bugs},
			median: 7, n: 4, groups: map[string]float64{"bug": 8, "repeat": 6}, // Bug: 3, 8, 12 → 8; Repeat: 6
		},
		{
			name:   "issue age counts open tasks to the last sync",
			spec:   &pb.InsightSpec{Measure: MeasureIssueAge, Slice: "priority", View: "bar", Filters: bugs},
			median: 19, n: 1, // created Nov 3, synced Nov 22
		},
		{
			name:   "triage time uses the triage window",
			spec:   &pb.InsightSpec{Measure: MeasureTriageTime, Slice: "assignee", View: "metric"},
			median: 5, n: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := d.RunInsight(nil, tt.spec, r)
			if err != nil {
				t.Fatal(err)
			}
			if !res.TimeMeasure || !near(res.Total, tt.median) || res.Tasks != tt.n {
				t.Errorf("median %v of %d tasks, want %v of %d", res.Total, res.Tasks, tt.median, tt.n)
			}
			got := groupValues(res)
			for k, v := range tt.groups {
				if !near(got[k], v) {
					t.Errorf("group %s median = %v, want %v (all: %v)", k, got[k], v, got)
				}
			}
			// The drill-down's median equals every clicked value.
			all, _ := d.InsightTasks(nil, tt.spec, "", "", r)
			if !near(all.Total, res.Total) || int32(len(all.Tasks)) != res.Tasks {
				t.Errorf("metric drill-down median %v over %d tasks, want %v over %d", all.Total, len(all.Tasks), res.Total, res.Tasks)
			}
			for _, g := range res.Groups {
				bar, _ := d.InsightTasks(nil, tt.spec, g.Key, "", r)
				if !near(bar.Total, g.Value) {
					t.Errorf("group %s = %v, drill-down median %v", g.Key, g.Value, bar.Total)
				}
				for _, row := range bar.Tasks {
					if !row.HasDays {
						t.Errorf("drill-down row %s has no duration", row.Identifier)
					}
				}
			}
		})
	}
}

func TestTimeStatsPercentiles(t *testing.T) {
	st := timeStats([]float64{10, 1, 4, 2, 8})
	if st.N != 5 || !near(st.P50, 4) || !near(st.P25, 2) || !near(st.P75, 8) || !near(st.Max, 10) {
		t.Errorf("stats = %+v, want n=5 p25=2 p50=4 p75=8 max=10", st)
	}
	if empty := timeStats(nil); empty.N != 0 {
		t.Errorf("empty stats = %+v", empty)
	}
}

func TestScatterPointsPerTask(t *testing.T) {
	d, r := timeFixture(t)
	res, err := d.RunInsight(nil, &pb.InsightSpec{Measure: MeasureCycleTime, Slice: "priority", View: "scatter"}, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Points) != 4 { // b1, b2, b3, f1
		t.Fatalf("points = %d, want 4", len(res.Points))
	}
	for _, p := range res.Points {
		if p.Identifier == "X-3" && (!near(p.Days, 10) || p.Date != "2026-11-13") {
			t.Errorf("X-3 point = %+v, want 10 days at 2026-11-13", p)
		}
	}
}
