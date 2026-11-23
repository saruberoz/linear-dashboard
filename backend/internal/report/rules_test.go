package report

import (
	"testing"
	"time"
)

var jakarta = mustLoc("Asia/Jakarta")

func mustLoc(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func ptr[T any](v T) *T { return &v }

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, jakarta)
	if err != nil {
		panic(err)
	}
	return t
}

func TestSegmentOf(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  string
	}{
		{name: "exact", label: "B2B", want: "B2B"},
		{name: "lower case with spaces", label: " b2b ", want: "B2B"},
		{name: "government", label: "b2g", want: "B2G"},
		{name: "non group long form", label: "B2B Non Group", want: "B2BNG"},
		{name: "non group short form", label: "B2B NG", want: "B2BNG"},
		{name: "non group hyphen", label: "b2b-ng", want: "B2BNG"},
		{name: "non group underscore", label: "b2b_ng", want: "B2BNG"},
		{name: "consumer", label: "B2C", want: "B2C"},
		{name: "internal", label: "Internal_PARKEE", want: "INTERNAL"},
		{name: "internal with space", label: "internal parkee", want: "INTERNAL"},
		{name: "internal alone", label: "Internal", want: ""},
		{name: "near miss", label: "B2BX", want: ""},
		{name: "unrelated", label: "Bug", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SegmentOf(tt.label); got != tt.want {
				t.Errorf("SegmentOf(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}

func TestSegmentsOfCanonicalOrder(t *testing.T) {
	got := SegmentsOf([]string{"B2C", "b2b non group", "Data", "B2B", "b2c"})
	want := []string{"B2B", "B2BNG", "B2C"}
	if len(got) != len(want) {
		t.Fatalf("SegmentsOf = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SegmentsOf = %v, want %v", got, want)
		}
	}
}

func TestIsVisibleInitiative(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "W prefix", in: "[W] CMS Improvement", want: true},
		{name: "WS prefix", in: "[WS] ERP Centrepark", want: true},
		{name: "W draft no space", in: "[W][DRAFT] PPOB", want: false},
		{name: "draft with suffix", in: "[W] [DRAFT-AO] Release Safety Net", want: false},
		{name: "draft mixed case", in: "[W] [Draft] SNAP API Compliance", want: false},
		{name: "drafting is not draft", in: "[W] Drafting tool", want: true},
		{name: "lower case with spaces", in: " [ w ]  EV Report", want: true},
		{name: "sprint", in: "46.Cloud Sprint 11 Nov - 24 Nov", want: false},
		{name: "bucket", in: "Cloud Kanban", want: false},
		{name: "team initiative", in: "[OS] [QA] Automation", want: false},
		{name: "W not at start", in: "Plan [W] later", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVisibleInitiative(tt.in); got != tt.want {
				t.Errorf("IsVisibleInitiative(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsIgnoredTask(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"[MEETING] Daily standup", true},
		{"sync meetings", true},
		{"Meet 19 Oct, bahas parkee protection", true},
		{"[DISCUSS] Update Seeder Staging DB", true},
		{"[DISCUSSION] discussion with engineer", true},
		{"3 Nov BSH Discussion", true},
		{"Weekly RECAP", true},
		{"Check Responsive", true},
		{"[Check] Calculate Visa Promo", true},
		{"re-check payment", true},
		{"Asses Issue settlement", true},
		{"[ASSESS] PRD & Engineering Assessment", true},
		{"Pre-Assessment", true},
		{"OS_Data cleanup", true},
		{"[OS_DATA_] import", true},
		{"Fix checkout flow", false},
		{"Create Mock Endpoint Of TITO Checkin", false},
		{"Show Checkbox Dropdown", false},
		{"QA checklist", false},
		{"Build login", false},
	}
	for _, tt := range tests {
		if got := IsIgnoredTask(tt.in); got != tt.want {
			t.Errorf("IsIgnoredTask(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestBucket(t *testing.T) {
	tests := []struct{ in, want string }{
		{"completed", StatusDone}, {"started", StatusInProgress}, {"backlog", StatusNotStarted},
		{"unstarted", StatusNotStarted}, {"triage", StatusNotStarted},
		{"canceled", StatusCanceled}, {"duplicate", StatusCanceled},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := Bucket(tt.in); got != tt.want {
				t.Errorf("Bucket(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDerive(t *testing.T) {
	teams := map[string]Team{"est": {ID: "est", EstimationType: "linear"}, "off": {ID: "off", EstimationType: "notUsed"}}
	labels := map[string]Label{"b2b": {ID: "b2b", Name: "b2b"}, "bug": {ID: "bug", Name: "Bug"}}
	tasks := []Task{
		{ID: "plain", TeamID: "est", StateType: "started", Estimate: ptr(3.0)},
		{ID: "parent", TeamID: "est", StateType: "started", Estimate: ptr(8.0)},
		{ID: "child", TeamID: "est", StateType: "started", ParentID: "parent", Estimate: ptr(5.0)},
		{ID: "grandparent", TeamID: "est", StateType: "started", Estimate: ptr(13.0)},
		{ID: "middle", TeamID: "est", StateType: "started", ParentID: "grandparent"},
		{ID: "grandchild", TeamID: "est", StateType: "started", ParentID: "middle", Estimate: ptr(2.0)},
		{ID: "p-canceled-child", TeamID: "est", StateType: "started", Estimate: ptr(4.0)},
		{ID: "canceled-child", TeamID: "est", StateType: "canceled", ParentID: "p-canceled-child", Estimate: ptr(4.0)},
		{ID: "canceled", TeamID: "est", StateType: "canceled", Estimate: ptr(7.0)},
		{ID: "duplicate", TeamID: "est", StateType: "duplicate", Estimate: ptr(7.0)},
		{ID: "no-estimate", TeamID: "est", StateType: "backlog"},
		{ID: "no-estimate-team-off", TeamID: "off", StateType: "backlog"},
		{ID: "no-estimate-canceled", TeamID: "est", StateType: "canceled"},
		{ID: "labeled", TeamID: "est", StateType: "backlog", Estimate: ptr(1.0), LabelIDs: []string{"b2b", "bug"}},
		{ID: "unest-parent", TeamID: "est", StateType: "started"},
		{ID: "unest-parent-child", TeamID: "est", StateType: "started", ParentID: "unest-parent", Estimate: ptr(5.0)},
		{ID: "unest-parent-sibling", TeamID: "est", StateType: "started", ParentID: "unest-parent"},
		{ID: "canceled-parent", TeamID: "est", StateType: "canceled", Estimate: ptr(6.0)},
		{ID: "canceled-parent-child", TeamID: "est", StateType: "started", ParentID: "canceled-parent", Estimate: ptr(2.0)},
		{ID: "meeting", Title: "[MEETING] Daily standup", TeamID: "est", StateType: "completed", Estimate: ptr(2.0)},
		{ID: "meeting-child", TeamID: "est", StateType: "completed", ParentID: "meeting", Estimate: ptr(3.0)},
		{ID: "meeting-unestimated", Title: "Meeting", TeamID: "est", StateType: "backlog"},
		{ID: "meeting-under-unest", Title: "sync meetings", TeamID: "est", StateType: "started", ParentID: "unest-parent-2", Estimate: ptr(1.0)},
		{ID: "unest-parent-2", TeamID: "est", StateType: "started"},
		{ID: "cycle-a", TeamID: "est", StateType: "started", ParentID: "cycle-b", Estimate: ptr(1.0)},
		{ID: "cycle-b", TeamID: "est", StateType: "started", ParentID: "cycle-a", Estimate: ptr(1.0)},
	}
	Derive(tasks, teams, labels)
	byID := map[string]Task{}
	for _, task := range tasks {
		byID[task.ID] = task
	}
	tests := []struct {
		id       string
		wantPts  float64
		deferred bool
		missing  bool
	}{
		{id: "plain", wantPts: 3},
		{id: "parent", wantPts: 8},
		{id: "child", wantPts: 0, deferred: true},
		{id: "grandparent", wantPts: 13},
		{id: "middle", wantPts: 0},
		{id: "grandchild", wantPts: 0, deferred: true},
		{id: "p-canceled-child", wantPts: 4},
		{id: "canceled-child", wantPts: 0},
		{id: "unest-parent", wantPts: 0},
		{id: "unest-parent-child", wantPts: 5},
		{id: "unest-parent-sibling", wantPts: 0, missing: true},
		{id: "canceled-parent", wantPts: 0},
		{id: "canceled-parent-child", wantPts: 2},
		{id: "meeting", wantPts: 0},
		{id: "meeting-child", wantPts: 3},
		{id: "meeting-unestimated", wantPts: 0},
		{id: "unest-parent-2", wantPts: 0, missing: true},
		{id: "canceled", wantPts: 0},
		{id: "duplicate", wantPts: 0},
		{id: "no-estimate", wantPts: 0, missing: true},
		{id: "no-estimate-team-off", wantPts: 0},
		{id: "no-estimate-canceled", wantPts: 0},
		{id: "labeled", wantPts: 1},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := byID[tt.id]
			if got.EffectivePts != tt.wantPts || got.Deferred() != tt.deferred || got.MissingEstimate != tt.missing {
				t.Errorf("%s: pts=%v deferred=%v missing=%v, want pts=%v deferred=%v missing=%v",
					tt.id, got.EffectivePts, got.Deferred(), got.MissingEstimate, tt.wantPts, tt.deferred, tt.missing)
			}
		})
	}
	t.Run("task segment labels", func(t *testing.T) {
		if got := byID["labeled"].TaskSegments; len(got) != 1 || got[0] != "B2B" {
			t.Errorf("TaskSegments = %v, want [B2B]", got)
		}
	})
	t.Run("cycle terminates", func(t *testing.T) {
		if byID["cycle-a"].EffectivePts+byID["cycle-b"].EffectivePts == 2 {
			t.Errorf("a two-task cycle counted both tasks in full")
		}
	})
}

func TestInRange(t *testing.T) {
	r, err := ParseRange("2026-11-01", "2026-11-30", jakarta)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		task Task
		want bool
	}{
		{name: "created on first day 00:00", task: Task{CreatedAt: at("2026-11-01 00:00")}, want: true},
		{name: "created on last day 23:59", task: Task{CreatedAt: at("2026-11-30 23:59")}, want: true},
		{name: "created the day after", task: Task{CreatedAt: at("2026-12-01 00:00")}, want: false},
		{name: "created before, no activity", task: Task{CreatedAt: at("2026-10-01 10:00")}, want: false},
		{name: "completed in range", task: Task{CreatedAt: at("2026-09-01 10:00"), CompletedAt: ptr(at("2026-11-15 10:00"))}, want: true},
		{name: "completed the day before", task: Task{CreatedAt: at("2026-09-01 10:00"), StartedAt: ptr(at("2026-10-01 10:00")), CompletedAt: ptr(at("2026-10-31 23:59"))}, want: false},
		{name: "started before and still open", task: Task{CreatedAt: at("2026-09-01 10:00"), StartedAt: ptr(at("2026-09-02 10:00"))}, want: true},
		{name: "started before, canceled during", task: Task{CreatedAt: at("2026-09-01 10:00"), StartedAt: ptr(at("2026-09-02 10:00")), CanceledAt: ptr(at("2026-11-02 10:00"))}, want: true},
		{name: "started after the range", task: Task{CreatedAt: at("2026-09-01 10:00"), StartedAt: ptr(at("2026-12-02 10:00"))}, want: false},
		{name: "spans Sep to Dec counts in Nov", task: Task{CreatedAt: at("2026-09-01 10:00"), StartedAt: ptr(at("2026-09-02 10:00")), CompletedAt: ptr(at("2026-12-10 10:00"))}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.InRange(&tt.task); got != tt.want {
				t.Errorf("InRange = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRangeRejectsBadInput(t *testing.T) {
	tests := []struct{ name, from, to string }{
		{name: "bad from", from: "2026-13-01", to: "2026-11-30"},
		{name: "bad to", from: "2026-11-01", to: "nope"},
		{name: "to before from", from: "2026-11-30", to: "2026-11-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseRange(tt.from, tt.to, jakarta); err == nil {
				t.Errorf("ParseRange(%q, %q) succeeded, want error", tt.from, tt.to)
			}
		})
	}
}
