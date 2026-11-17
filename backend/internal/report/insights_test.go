package report

import (
	"errors"
	"strings"
	"testing"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

func spec(measure, slice, segment string, f *pb.InsightFilters) *pb.InsightSpec {
	return &pb.InsightSpec{Measure: measure, Slice: slice, Segment: segment, View: "bar", Filters: f}
}

func groupValues(res *pb.InsightResult) map[string]float64 {
	out := map[string]float64{}
	for _, g := range res.Groups {
		out[g.Key] = g.Value
	}
	return out
}

func TestInsightSegmentSplitAndCount(t *testing.T) {
	d, r := fixture(t)
	tests := []struct {
		name      string
		spec      *pb.InsightSpec
		dash      *pb.InsightFilters
		want      map[string]float64
		wantTotal float64
	}{
		{
			name: "effort by segment splits the shared project",
			spec: spec(MeasureEffort, "segment", "", nil),
			want: map[string]float64{"B2B": 6, "B2C": 8}, wantTotal: 14,
		},
		{
			name: "count by segment counts a shared task in both",
			spec: spec(MeasureCount, "segment", "", nil),
			want: map[string]float64{"B2B": 2, "B2C": 4}, wantTotal: 4,
		},
		{
			name: "segment filter counts the shared project at its share",
			spec: spec(MeasureEffort, "assignee", "", &pb.InsightFilters{Segments: []string{"B2B"}}),
			want: map[string]float64{"alice": 4, "bob": 2}, wantTotal: 6,
		},
		{
			name: "dashboard and insight filters intersect",
			spec: spec(MeasureEffort, "assignee", "", &pb.InsightFilters{Statuses: []string{StatusDone, StatusInProgress}}),
			dash: &pb.InsightFilters{Statuses: []string{StatusDone}},
			want: map[string]float64{"alice": 8}, wantTotal: 8,
		},
		{
			name: "untagged work needs both levels to include it",
			spec: spec(MeasureEffort, "segment", "", &pb.InsightFilters{IncludeUntagged: true}),
			dash: &pb.InsightFilters{IncludeUntagged: true},
			want: map[string]float64{"B2B": 6, "B2C": 8, noneKey: 1}, wantTotal: 15,
		},
		{
			name: "disjoint filters match nothing",
			spec: spec(MeasureCount, "status", "", &pb.InsightFilters{Segments: []string{"B2G"}}),
			dash: &pb.InsightFilters{Segments: []string{"B2B"}},
			want: map[string]float64{}, wantTotal: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := d.RunInsight(tt.dash, tt.spec, r)
			if err != nil {
				t.Fatal(err)
			}
			got := groupValues(res)
			if len(got) != len(tt.want) {
				t.Fatalf("groups = %v, want %v", got, tt.want)
			}
			for k, v := range tt.want {
				if !near(got[k], v) {
					t.Errorf("group %s = %v, want %v", k, got[k], v)
				}
			}
			if !near(res.Total, tt.wantTotal) {
				t.Errorf("total = %v, want %v", res.Total, tt.wantTotal)
			}
		})
	}
}

// Every bar, every stacked segment and the whole metric must equal the total of
// the tasks a click opens.
func TestInsightDrillDownAddsUp(t *testing.T) {
	d, r := fixture(t)
	all := &pb.InsightFilters{IncludeUntagged: true}
	specs := []*pb.InsightSpec{
		spec(MeasureEffort, "assignee", "segment", all),
		spec(MeasureEffort, "segment", "label", all),
		spec(MeasureCount, "label", "status", all),
		spec(MeasureCount, "status", "label", all),
		spec(MeasureEffort, "status", "label", all),
		spec(MeasureCount, "segment", "priority", all),
		spec(MeasureEffort, "initiative", "", all),
		spec(MeasureEffort, "created_month", "status", nil),
		spec(MeasureEffort, "estimate", "work_type", all),
		spec(MeasureCount, "work_type", "label", all),
	}
	for _, s := range specs {
		t.Run(s.Measure+" by "+s.Slice+"/"+s.Segment, func(t *testing.T) {
			res, err := d.RunInsight(all, s, r)
			if err != nil {
				t.Fatal(err)
			}
			total, err := d.InsightTasks(all, s, "", "", r)
			if err != nil {
				t.Fatal(err)
			}
			if !near(total.Total, res.Total) {
				t.Errorf("metric %v, drill-down %v", res.Total, total.Total)
			}
			for _, g := range res.Groups {
				bar, _ := d.InsightTasks(all, s, g.Key, "", r)
				if !near(bar.Total, g.Value) {
					t.Errorf("bar %s = %v, drill-down %v", g.Key, g.Value, bar.Total)
				}
				for _, c := range g.Segments {
					cell, _ := d.InsightTasks(all, s, g.Key, c.Key, r)
					if !near(cell.Total, c.Value) {
						t.Errorf("cell %s/%s = %v, drill-down %v", g.Key, c.Key, c.Value, cell.Total)
					}
				}
			}
		})
	}
}

func TestInsightOrdering(t *testing.T) {
	d, r := fixture(t)
	res, err := d.RunInsight(nil, spec(MeasureCount, "priority", "", nil), r)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, g := range res.Groups {
		keys = append(keys, g.Key)
	}
	// High (2), Medium (3), then No priority (0) last.
	if len(keys) != 3 || keys[0] != "2" || keys[1] != "3" || keys[2] != "0" {
		t.Errorf("priority order = %v, want [2 3 0]", keys)
	}
}

func TestInsightRejectsBadSpecs(t *testing.T) {
	d, r := fixture(t)
	bad := []*pb.InsightSpec{
		{Measure: "velocity", Slice: "assignee", View: "bar"},
		{Measure: MeasureEffort, Slice: "assignee", View: "scatter"},
		{Measure: MeasureEffort, Slice: "nope", View: "bar"},
		{Measure: MeasureEffort, Slice: "assignee", Segment: "assignee", View: "bar"},
		{Measure: MeasureEffort, Slice: "assignee", View: "pie"},
	}
	for _, s := range bad {
		if _, err := d.RunInsight(nil, s, r); !errors.Is(err, ErrInvalidInsight) {
			t.Errorf("spec %v: err = %v, want ErrInvalidInsight", s, err)
		}
	}
}

func TestEstimateAndWorkTypeDimensions(t *testing.T) {
	d, r := fixture(t)
	all := &pb.InsightFilters{IncludeUntagged: true}
	est, err := d.RunInsight(all, spec(MeasureCount, "estimate", "", all), r)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, g := range est.Groups {
		keys = append(keys, g.Key)
	}
	// 1, 2, 4, 8 pts by size, then the unestimated task last.
	if strings.Join(keys, ",") != "1,2,4,8,"+noneKey {
		t.Errorf("estimate groups = %v, want 1,2,4,8,_none", keys)
	}
	wt, err := d.RunInsight(all, spec(MeasureCount, "work_type", "", all), r)
	if err != nil {
		t.Fatal(err)
	}
	got := groupValues(wt)
	// t1 has the "Bug" label and t5 the "b2b" label (no bug); everything else is other work.
	if got["bug"] != 1 || got["other"] != 4 || wt.Groups[0].Key != "bug" {
		t.Errorf("work types = %v (first %s), want bug 1 first, other 4", got, wt.Groups[0].Key)
	}
}

func TestLabelEffortMarksTasksWithTwoLabels(t *testing.T) {
	d, r := fixture(t)
	all := &pb.InsightFilters{IncludeUntagged: true}
	s := spec(MeasureEffort, "label", "", all)
	res, err := d.RunInsight(all, s, r)
	if err != nil {
		t.Fatal(err)
	}
	shared := map[string]int32{}
	for _, g := range res.Groups {
		shared[g.Label] = g.SharedTasks
	}
	got := groupValues(res)
	// X-1 (8 pts) carries Bug and Feature: each label gets half and marks one shared task.
	for _, name := range []string{"Bug", "Feature"} {
		var key string
		for _, g := range res.Groups {
			if g.Label == name {
				key = g.Key
			}
		}
		if got[key] != 4 || shared[name] != 1 {
			t.Errorf("%s = %v pts, %d shared; want 4 pts, 1 shared", name, got[key], shared[name])
		}
	}
	tasks, err := d.InsightTasks(all, s, labelKey(res, "Feature"), "", r)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, row := range tasks.Tasks {
		if row.Identifier == "X-1" {
			found = true
			if row.SplitCount != 2 || row.SharePts != 4 {
				t.Errorf("X-1 split %d share %v, want split 2 share 4", row.SplitCount, row.SharePts)
			}
		}
	}
	if !found {
		t.Error("X-1 missing from the Feature drill-down")
	}
}

func labelKey(res *pb.InsightResult, label string) string {
	for _, g := range res.Groups {
		if g.Label == label {
			return g.Key
		}
	}
	return ""
}

func TestLabelSliceKeepsOnlyFilteredLabels(t *testing.T) {
	d, r := fixture(t)
	all := &pb.InsightFilters{IncludeUntagged: true}
	feat := &pb.InsightFilters{IncludeUntagged: true, Labels: []string{"feature"}}
	res, err := d.RunInsight(all, spec(MeasureEffort, "label", "", feat), r)
	if err != nil {
		t.Fatal(err)
	}
	// X-1 has Bug and Feature; filtered to Feature it gives Feature its full 8 pts, unshared.
	if len(res.Groups) != 1 || res.Groups[0].Label != "Feature" || res.Groups[0].Value != 8 || res.Groups[0].SharedTasks != 0 {
		t.Errorf("groups = %+v, want only Feature with 8 pts and 0 shared", res.Groups)
	}
}
