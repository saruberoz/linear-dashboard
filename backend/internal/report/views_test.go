package report

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"
)

// fixture: two projects in November.
//
//	shared: B2B + B2C, initiative "[W] Protection"; alice 8 pts done (High), bob 4 pts started (Medium)
//	solo:   B2C only, initiative "46.Cloud Sprint" (hidden); alice 2 pts (No priority), one unestimated task
//	loose:  untagged project; a task carrying the task label "b2b"
func fixture(t *testing.T) (*Dataset, Range) {
	t.Helper()
	nov := at("2026-11-05 10:00")
	labels := []Label{
		{ID: "lb2b", Kind: "project", Name: "B2B", Group: "WS"},
		{ID: "lb2c", Kind: "project", Name: "B2C", Group: "WS"},
		{ID: "ltask", Kind: "issue", Name: "b2b"},
		{ID: "lbug", Kind: "issue", Name: "Bug"},
		{ID: "lfeat", Kind: "issue", Name: "Feature"},
	}
	tasks := []Task{
		{ID: "t1", Identifier: "X-1", TeamID: "est", ProjectID: "shared", AssigneeID: "alice", Priority: 2, StateType: "completed", Estimate: ptr(8.0), CreatedAt: nov, CompletedAt: ptr(nov), LabelIDs: []string{"lbug", "lfeat"}},
		{ID: "t2", Identifier: "X-2", TeamID: "est", ProjectID: "shared", AssigneeID: "bob", Priority: 3, StateType: "started", Estimate: ptr(4.0), CreatedAt: nov, StartedAt: ptr(nov)},
		{ID: "t3", Identifier: "X-3", TeamID: "est", ProjectID: "solo", AssigneeID: "alice", Priority: 0, StateType: "backlog", Estimate: ptr(2.0), CreatedAt: nov},
		{ID: "t4", Identifier: "X-4", TeamID: "est", ProjectID: "solo", StateType: "backlog", CreatedAt: nov},
		{ID: "t5", Identifier: "X-5", TeamID: "est", ProjectID: "loose", StateType: "backlog", Estimate: ptr(1.0), CreatedAt: nov, LabelIDs: []string{"ltask"}},
	}
	teams := []Team{{ID: "est", EstimationType: "linear"}}
	lm := map[string]Label{}
	for _, l := range labels {
		lm[l.ID] = l
	}
	tm := map[string]Team{"est": teams[0]}
	Derive(tasks, tm, lm)
	d := NewDataset(time.Now(), teams,
		[]User{{ID: "alice", DisplayName: "alice"}, {ID: "bob", DisplayName: "bob"}},
		labels,
		[]Initiative{{ID: "iw", Name: "[W] Protection", Status: "Active"}, {ID: "isprint", Name: "46.Cloud Sprint"}},
		[]Project{
			{ID: "shared", Name: "Shared", CreatedAt: nov, LabelIDs: []string{"lb2b", "lb2c"}, InitiativeIDs: []string{"iw"}},
			{ID: "solo", Name: "Solo", CreatedAt: nov, LabelIDs: []string{"lb2c"}, InitiativeIDs: []string{"isprint"}},
			{ID: "loose", Name: "Loose", CreatedAt: nov},
		}, tasks)
	r, err := ParseRange("2026-11-01", "2026-11-30", jakarta)
	if err != nil {
		t.Fatal(err)
	}
	return d, r
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestOverviewSharesAddUpToTaggedTotal(t *testing.T) {
	d, r := fixture(t)
	o := d.Overview(r)
	want := map[string]float64{"B2B": 6, "B2G": 0, "B2BNG": 0, "B2C": 8} // shared 12 split 6/6, solo 2
	var pctSum float64
	for _, s := range o.Segments {
		if !near(s.Pts, want[s.Key]) {
			t.Errorf("%s pts = %v, want %v", s.Key, s.Pts, want[s.Key])
		}
		pctSum += s.Pct
	}
	if !near(o.TaggedTotal, o.Summary.Total) || !near(o.TaggedTotal, 14) {
		t.Errorf("tagged total %v, summary total %v, want both 14", o.TaggedTotal, o.Summary.Total)
	}
	if !near(pctSum, 100) {
		t.Errorf("segment percentages sum to %v, want 100", pctSum)
	}
	if o.Warnings.Tasks != 1 || o.Warnings.Uncounted != 1 || o.UntaggedProjects != 1 {
		t.Errorf("warnings %+v untagged %d, want 1 task, 1 uncounted, 1 untagged project", o.Warnings, o.UntaggedProjects)
	}
}

func TestOverviewEffortPerContributor(t *testing.T) {
	d, r := fixture(t)
	o := d.Overview(r)
	// B2B: shared (alice, bob), 6 pts. B2C: shared + solo (alice, bob; t4 is unassigned), 8 pts.
	want := map[string]struct {
		n   int32
		avg float64
	}{"B2B": {2, 3}, "B2G": {0, 0}, "B2BNG": {0, 0}, "B2C": {2, 4}, "INTERNAL": {0, 0}}
	for _, s := range o.Segments {
		w := want[s.Key]
		if s.Contributors != w.n || !near(s.PtsPerContributor, w.avg) {
			t.Errorf("%s contributors=%d avg=%v, want %d and %v", s.Key, s.Contributors, s.PtsPerContributor, w.n, w.avg)
		}
	}
}

func TestEffortPerContributorLeavesOutUnassignedEffort(t *testing.T) {
	nov := at("2026-11-05 10:00")
	labels := []Label{{ID: "lb2b", Kind: "project", Name: "B2B"}}
	tasks := []Task{
		{ID: "a", TeamID: "est", ProjectID: "p", AssigneeID: "alice", StateType: "backlog", Estimate: ptr(4.0), CreatedAt: nov},
		{ID: "b", TeamID: "est", ProjectID: "p", AssigneeID: "bob", StateType: "backlog", Estimate: ptr(2.0), CreatedAt: nov},
		{ID: "u", TeamID: "est", ProjectID: "p", StateType: "backlog", Estimate: ptr(10.0), CreatedAt: nov},
	}
	teams := []Team{{ID: "est", EstimationType: "linear"}}
	Derive(tasks, map[string]Team{"est": teams[0]}, map[string]Label{"lb2b": labels[0]})
	d := NewDataset(time.Now(), teams, nil, labels, nil,
		[]Project{{ID: "p", Name: "P", CreatedAt: nov, LabelIDs: []string{"lb2b"}}}, tasks)
	r, _ := ParseRange("2026-11-01", "2026-11-30", jakarta)
	for _, s := range d.Overview(r).Segments {
		if s.Key != "B2B" {
			continue
		}
		// Effort keeps the unassigned 10; the average is (4+2) / 2 people.
		if !near(s.Pts, 16) || s.Contributors != 2 || !near(s.PtsPerContributor, 3) {
			t.Errorf("B2B pts=%v people=%d avg=%v, want 16, 2, 3", s.Pts, s.Contributors, s.PtsPerContributor)
		}
	}
}

func TestSegmentViewSplitsSharedProjects(t *testing.T) {
	d, r := fixture(t)
	s, err := d.Segment("B2B", r)
	if err != nil {
		t.Fatal(err)
	}
	if !near(s.Summary.Total, 6) {
		t.Errorf("B2B total = %v, want 6 (half of the shared project)", s.Summary.Total)
	}
	for _, c := range s.Summary.Contributors {
		if c.Dim == DimSegment {
			t.Errorf("segment view returned a Segment contributor table")
		}
	}
	row := s.Groups[0].Projects[0]
	if !near(row.Pts, 12) || !near(row.SharePts, 6) || row.SplitOf != 2 {
		t.Errorf("project row pts=%v share=%v split=%d, want 12/6/2", row.Pts, row.SharePts, row.SplitOf)
	}
}

func TestSegmentViewInitiativeSubtotalsAddUpToSegmentTotal(t *testing.T) {
	d, r := fixture(t)
	d.Initiatives["iw2"] = &Initiative{ID: "iw2", Name: "[W] Second", Visible: true, ProjectIDs: []string{"shared"}}
	d.Projects["shared"].InitiativeIDs = append(d.Projects["shared"].InitiativeIDs, "iw2")

	s, err := d.Segment("B2C", r)
	if err != nil {
		t.Fatal(err)
	}
	// B2C: shared 12 pts split across 2 segments = 6, then across 2 initiatives = 3 each; solo 2.
	want := map[string]float64{"[W] Protection": 3, "[W] Second": 3, "No initiative": 2}
	var sum float64
	for _, g := range s.Groups {
		if !near(g.SharePts, want[g.Name]) {
			t.Errorf("group %q share = %v, want %v", g.Name, g.SharePts, want[g.Name])
		}
		var rows float64
		for _, p := range g.Projects {
			rows += p.SharePts
		}
		if !near(rows, g.SharePts) {
			t.Errorf("group %q rows add up to %v, subtotal is %v", g.Name, rows, g.SharePts)
		}
		sum += g.SharePts
	}
	if !near(sum, s.Summary.Total) {
		t.Errorf("initiative subtotals sum to %v, segment total is %v", sum, s.Summary.Total)
	}
	row := s.Groups[0].Projects[0]
	if !near(row.Pts, 12) || !near(row.SharePts, 3) || row.SplitOf != 2 || row.InitiativeSplitOf != 2 {
		t.Errorf("row pts=%v share=%v split=%d initiativeSplit=%d, want 12/3/2/2",
			row.Pts, row.SharePts, row.SplitOf, row.InitiativeSplitOf)
	}
}

func TestDrillDownAddsUpToTheCell(t *testing.T) {
	d, r := fixture(t)
	tests := []struct {
		name, scope, dim, column, assignee string
		want                               float64
	}{
		{name: "segment view, shared project", scope: "segment:B2B", dim: DimPriority, column: "2", assignee: "alice", want: 4},
		{name: "overview, segment column splits", scope: "all", dim: DimSegment, column: "B2C", assignee: "alice", want: 6},
		{name: "overview, total column", scope: "all", dim: DimStatus, column: ColumnTotal, assignee: "alice", want: 10},
		{name: "unassigned row", scope: "all", dim: DimStatus, column: StatusNotStarted, assignee: Unassigned, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := d.scopeTasks(mustScope(t, tt.scope), r)
			if err != nil {
				t.Fatal(err)
			}
			tbl := d.contributors(ws, tt.dim)
			var cell float64
			for _, row := range tbl.Rows {
				if row.AssigneeId != tt.assignee {
					continue
				}
				if tt.column == ColumnTotal {
					cell = row.Total
				} else if c := row.Cells[tt.column]; c != nil {
					cell = c.Pts
				}
			}
			res, err := d.CellTasks(tt.scope, tt.dim, tt.column, tt.assignee, r)
			if err != nil {
				t.Fatal(err)
			}
			if !near(cell, tt.want) || !near(res.Total, cell) {
				t.Errorf("cell=%v drill-down=%v, want both %v", cell, res.Total, tt.want)
			}
		})
	}
}

func TestSegmentDimRejectedOnSegmentScope(t *testing.T) {
	d, r := fixture(t)
	if _, err := d.CellTasks("segment:B2B", DimSegment, "B2C", "alice", r); !errors.Is(err, ErrInvalidDim) {
		t.Errorf("err = %v, want ErrInvalidDim", err)
	}
}

func TestHiddenInitiativesAndNoInitiativeGroup(t *testing.T) {
	d, r := fixture(t)
	s, err := d.Segment("B2C", r)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Groups) != 2 || s.Groups[0].Name != "[W] Protection" || s.Groups[1].Id != "" {
		t.Fatalf("groups = %v, want [W] Protection then No initiative", s.Groups)
	}
	if _, err := d.Initiative("isprint", r); !errors.Is(err, ErrNotFound) {
		t.Errorf("hidden initiative err = %v, want ErrNotFound", err)
	}
}

func TestInitiativeStatusInMenuAndPage(t *testing.T) {
	d, r := fixture(t)
	var got []string
	for _, s := range d.Menu(r).Segments {
		for _, ini := range s.Initiatives {
			got = append(got, ini.Name+"="+ini.Status)
		}
	}
	if !slices.Contains(got, "[W] Protection=Active") || !slices.Contains(got, "No initiative=") {
		t.Errorf("menu initiatives = %v, want [W] Protection=Active and No initiative with no status", got)
	}
	res, err := d.Initiative("iw", r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Initiative.Status != "Active" {
		t.Errorf("initiative page status = %q, want Active", res.Initiative.Status)
	}
}

func TestMissingEstimatesScopes(t *testing.T) {
	d, r := fixture(t)
	tagged, err := d.MissingEstimates("", false, r)
	if err != nil {
		t.Fatal(err)
	}
	proj, err := d.MissingEstimates("project:shared", false, r)
	if err != nil {
		t.Fatal(err)
	}
	if tagged.Total != 1 || proj.Total != 0 {
		t.Errorf("tagged=%d project=%d, want 1 and 0", tagged.Total, proj.Total)
	}
	if _, err := d.MissingEstimates("bogus", false, r); !errors.Is(err, ErrInvalidScope) {
		t.Errorf("bogus scope err = %v, want ErrInvalidScope", err)
	}
}

func TestWarningsMarkCountedProjects(t *testing.T) {
	d, r := fixture(t)
	w := d.SegmentLabelWarnings(r)
	if w.Total != 1 || w.Groups[0].Counted || w.Groups[0].ProjectName != "Loose" {
		t.Errorf("warnings = %+v, want one uncounted group for Loose", w)
	}
}

func mustScope(t *testing.T, s string) Scope {
	t.Helper()
	sc, err := ParseScope(s)
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func TestProjectsWithoutInitiative(t *testing.T) {
	d, r := fixture(t)
	res := d.ProjectsWithoutInitiative(r)
	var names []string
	for _, p := range res.Projects {
		names = append(names, p.Name)
	}
	// Solo is only in a hidden sprint initiative; Loose has none; Shared is in [W] Protection.
	if len(names) != 2 || names[0] != "Solo" || names[1] != "Loose" {
		t.Fatalf("projects = %v, want [Solo Loose] (tagged first)", names)
	}
	if h := res.Projects[0].HiddenInitiatives; len(h) != 1 || h[0] != "46.Cloud Sprint" {
		t.Errorf("Solo hidden initiatives = %v, want [46.Cloud Sprint]", h)
	}
}

func TestInitiativesWithoutProjects(t *testing.T) {
	d, _ := fixture(t)
	empty := NewDataset(time.Now(), nil, nil, nil,
		[]Initiative{{ID: "a", Name: "Roadmap"}, {ID: "b", Name: "[W] Empty"}, {ID: "c", Name: "[W] Used"}},
		[]Project{{ID: "p", Name: "P", InitiativeIDs: []string{"c"}}}, nil)
	res := empty.InitiativesWithoutProjects()
	if len(res.Initiatives) != 2 || res.Initiatives[0].Name != "[W] Empty" || !res.Initiatives[0].Visible || res.Initiatives[1].Name != "Roadmap" {
		t.Errorf("initiatives = %v, want [W] Empty (visible) then Roadmap", res.Initiatives)
	}
	if got := d.InitiativesWithoutProjects(); len(got.Initiatives) != 0 {
		t.Errorf("fixture initiatives without projects = %v, want none", got.Initiatives)
	}
}

func TestUntaggedCountsLeaveOutCanceledProjects(t *testing.T) {
	d, r := fixture(t)
	d.Projects["dead"] = &Project{ID: "dead", Name: "Dead", State: "canceled", CreatedAt: at("2026-11-05 10:00")}
	if n := len(d.UntaggedProjects(r).Projects); n != 2 {
		t.Errorf("untagged list has %d projects, want 2 (canceled included for the page filter)", n)
	}
	if n := d.Menu(r).UntaggedCount; n != 1 {
		t.Errorf("menu untagged count = %d, want 1 (canceled left out)", n)
	}
	if n := d.Overview(r).UntaggedProjects; n != 1 {
		t.Errorf("overview untagged count = %d, want 1 (canceled left out)", n)
	}
}
