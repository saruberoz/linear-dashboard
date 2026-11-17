package report

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// NormalizeLabel lower-cases a label and drops everything except a–z and 0–9,
// so "B2B Non Group", "b2b-ng" and " B2B NG " compare equal to their aliases.
func NormalizeLabel(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// SegmentOf returns the segment key a label name means, or "" if none.
func SegmentOf(labelName string) string {
	n := NormalizeLabel(labelName)
	for _, s := range Segments {
		for _, a := range s.aliases {
			if n == a {
				return s.Key
			}
		}
	}
	return ""
}

// SegmentsOf returns the distinct segment keys among label names, in canonical order.
func SegmentsOf(labelNames []string) []string {
	found := map[string]bool{}
	for _, n := range labelNames {
		if k := SegmentOf(n); k != "" {
			found[k] = true
		}
	}
	out := []string{}
	for _, s := range Segments {
		if found[s.Key] {
			out = append(out, s.Key)
		}
	}
	return out
}

// SegmentLabel returns the display label of a segment key.
func SegmentLabel(key string) string {
	for _, s := range Segments {
		if s.Key == key {
			return s.Label
		}
	}
	return key
}

var visibleInitiative = regexp.MustCompile(`(?i)^\s*\[\s*(w|ws)\s*\]`)

// IsVisibleInitiative reports whether an initiative is shown: its name starts
// with [W] or [WS], ignoring case and spaces. Sprints and buckets are hidden.
func IsVisibleInitiative(name string) bool { return visibleInitiative.MatchString(name) }

// Bucket maps a Linear state type to a status bucket. "duplicate" counts as canceled.
func Bucket(stateType string) string {
	switch stateType {
	case "completed":
		return StatusDone
	case "started":
		return StatusInProgress
	case "canceled", "duplicate":
		return StatusCanceled
	default: // backlog, unstarted, triage
		return StatusNotStarted
	}
}

// Derive computes the tree-dependent fields of every task. It must run over the
// complete set of tasks fetched in one sync, never a date-filtered subset, so a
// parent whose estimated sub-tasks fall outside a date range still counts as 0.
//
// A task is "deferred" when an estimated, non-canceled sub-task exists anywhere
// below it; a deferred or canceled task counts 0 points. A task is "missing an
// estimate" when it has none, is not canceled, its team uses estimates, and it
// is not deferred.
func Derive(tasks []Task, teams map[string]Team, labels map[string]Label) {
	children := map[string][]int{}
	for i := range tasks {
		if p := tasks[i].ParentID; p != "" {
			children[p] = append(children[p], i)
		}
	}
	memo := make([]int8, len(tasks)) // 0 unknown, 1 false, 2 true, 3 visiting
	var hasEstimated func(i int) bool
	hasEstimated = func(i int) bool {
		switch memo[i] {
		case 1:
			return false
		case 2:
			return true
		case 3: // cycle: treat as no descendants
			return false
		}
		memo[i] = 3
		r := false
		for _, c := range children[tasks[i].ID] {
			ct := &tasks[c]
			if ct.Status() == StatusCanceled {
				continue
			}
			if ct.Estimate != nil || hasEstimated(c) {
				r = true
				break
			}
		}
		if r {
			memo[i] = 2
		} else {
			memo[i] = 1
		}
		return r
	}
	for i := range tasks {
		t := &tasks[i]
		t.HasEstimatedDescendant = hasEstimated(i)
		canceled := t.Status() == StatusCanceled
		switch {
		case canceled || t.HasEstimatedDescendant || t.Estimate == nil:
			t.EffectivePts = 0
		default:
			t.EffectivePts = *t.Estimate
		}
		t.MissingEstimate = t.Estimate == nil && !canceled &&
			teams[t.TeamID].EstimationType != "notUsed" && !t.HasEstimatedDescendant
		names := make([]string, 0, len(t.LabelIDs))
		for _, id := range t.LabelIDs {
			names = append(names, labels[id].Name)
		}
		t.TaskSegments = SegmentsOf(names)
	}
}

// Range is an inclusive date range in a time zone, stored as [start, end).
type Range struct {
	From, To   string
	Start, End time.Time
}

// ParseRange parses "YYYY-MM-DD" bounds in loc.
func ParseRange(from, to string, loc *time.Location) (Range, error) {
	f, err := time.ParseInLocation("2006-01-02", from, loc)
	if err != nil {
		return Range{}, fmt.Errorf("invalid from date %q: %w", from, err)
	}
	t, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		return Range{}, fmt.Errorf("invalid to date %q: %w", to, err)
	}
	if t.Before(f) {
		return Range{}, fmt.Errorf("to date %s is before from date %s", to, from)
	}
	return Range{From: from, To: to, Start: f, End: t.AddDate(0, 0, 1)}, nil
}

func (r Range) contains(ts *time.Time) bool {
	return ts != nil && !ts.Before(r.Start) && ts.Before(r.End)
}

// InRange reports whether a task counts in the range: it was created in it,
// completed in it, or in progress during it (started before the range ends and
// not completed or canceled before the range starts; an open task is unfinished).
func (r Range) InRange(t *Task) bool {
	if r.contains(&t.CreatedAt) || r.contains(t.CompletedAt) {
		return true
	}
	if t.StartedAt == nil || !t.StartedAt.Before(r.End) {
		return false
	}
	finished := t.CompletedAt
	if finished == nil {
		finished = t.CanceledAt
	}
	return finished == nil || !finished.Before(r.Start)
}
