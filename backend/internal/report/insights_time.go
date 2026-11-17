package report

import (
	"math"
	"sort"
	"strings"
	"time"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

// Time measures, in days.
const (
	MeasureCycleTime  = "cycle_time"  // started → completed, tasks completed in the range
	MeasureLeadTime   = "lead_time"   // created → completed, tasks completed in the range
	MeasureIssueAge   = "issue_age"   // created → last sync, tasks still open and created by the range end
	MeasureTriageTime = "triage_time" // entered triage → triaged, tasks triaged in the range
)

const maxPoints = 3000

// IsTimeMeasure reports whether a measure is a duration in days.
func IsTimeMeasure(m string) bool {
	switch m {
	case MeasureCycleTime, MeasureLeadTime, MeasureIssueAge, MeasureTriageTime:
		return true
	}
	return false
}

// inTimePopulation reports whether a task counts for a time measure in r.
func (d *Dataset) inTimePopulation(t *Task, measure string, r Range) bool {
	done := t.Status() == StatusDone && t.CompletedAt != nil
	switch measure {
	case MeasureCycleTime:
		return done && t.StartedAt != nil && r.contains(t.CompletedAt)
	case MeasureLeadTime:
		return done && r.contains(t.CompletedAt)
	case MeasureTriageTime:
		return t.StartedTriageAt != nil && t.TriagedAt != nil && r.contains(t.TriagedAt)
	case MeasureIssueAge:
		open := t.Status() == StatusInProgress || t.Status() == StatusNotStarted
		return open && t.CreatedAt.Before(r.End)
	}
	return false
}

// durationDays is a task's value for a time measure, and the date it is plotted at.
func (d *Dataset) durationDays(t *Task, measure string) (float64, time.Time) {
	var from, to time.Time
	switch measure {
	case MeasureCycleTime:
		from, to = *t.StartedAt, *t.CompletedAt
	case MeasureLeadTime:
		from, to = t.CreatedAt, *t.CompletedAt
	case MeasureTriageTime:
		from, to = *t.StartedTriageAt, *t.TriagedAt
	case MeasureIssueAge:
		from, to = t.CreatedAt, d.SyncedAt
		return math.Max(0, to.Sub(from).Hours()/24), from
	}
	return math.Max(0, to.Sub(from).Hours()/24), to
}

// labelsMatch applies the label filters: one of the chosen label names, and a
// label containing every "label contains" text.
func (f mergedFilters) labelsMatch(d *Dataset, t *Task) bool {
	if f.labels == nil && len(f.labelMatch) == 0 {
		return true
	}
	names := make([]string, 0, len(t.LabelIDs))
	for _, id := range t.LabelIDs {
		if l, ok := d.Labels[id]; ok {
			names = append(names, strings.ToLower(l.Name))
		}
	}
	if f.labels != nil {
		hit := false
		for _, n := range names {
			if f.labels[n] {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	for _, m := range f.labelMatch {
		hit := false
		for _, n := range names {
			if strings.Contains(n, m) {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

func lowerAll(xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x = strings.ToLower(strings.TrimSpace(x)); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// timeStats summarizes durations with linear-interpolated percentiles.
func timeStats(days []float64) *pb.TimeStats {
	if len(days) == 0 {
		return &pb.TimeStats{}
	}
	xs := append([]float64(nil), days...)
	sort.Float64s(xs)
	q := func(p float64) float64 {
		pos := p * float64(len(xs)-1)
		lo := int(math.Floor(pos))
		hi := int(math.Ceil(pos))
		return xs[lo] + (xs[hi]-xs[lo])*(pos-float64(lo))
	}
	return &pb.TimeStats{N: int32(len(xs)), P25: q(0.25), P50: q(0.5), P75: q(0.75), P95: q(0.95), Max: xs[len(xs)-1]}
}

// runTimeInsight computes a time-measure insight: the median per group and per
// segment cell, the overall distribution and one scatter point per task.
// Multi-valued dimensions put a task in every group it belongs to.
func (d *Dataset) runTimeInsight(dash *pb.InsightFilters, ins *pb.InsightSpec, r Range) *pb.InsightResult {
	f := mergeFilters(dash, ins.GetFilters())
	sliceDim, segDim, measure := ins.GetSlice(), ins.GetSegment(), ins.GetMeasure()
	res := &pb.InsightResult{Measure: measure, Slice: sliceDim, Segment: segDim, TimeMeasure: true,
		MultiValued: insightDims[sliceDim].multi || (segDim != "" && insightDims[segDim].multi)}
	type acc struct {
		label  string
		shared int32
		days   []float64
		cells  map[string]*struct {
			label string
			days  []float64
		}
	}
	groups := map[string]*acc{}
	legend := map[string]*struct {
		label string
		days  []float64
	}{}
	var all []float64
	for _, x := range d.insightTasks(f, r, measure) {
		t := x.t
		days, at := d.durationDays(t, measure)
		all = append(all, days)
		sks := d.dimKeys(t, sliceDim, f, r)
		var gks []dimKey
		if segDim != "" {
			gks = d.dimKeys(t, segDim, f, r)
		}
		for _, sk := range sks {
			g := groups[sk.key]
			if g == nil {
				g = &acc{label: sk.label, cells: map[string]*struct {
					label string
					days  []float64
				}{}}
				groups[sk.key] = g
			}
			g.days = append(g.days, days)
			if len(sks) > 1 {
				g.shared++
			}
			for _, gk := range gks {
				c := g.cells[gk.key]
				if c == nil {
					c = &struct {
						label string
						days  []float64
					}{label: gk.label}
					g.cells[gk.key] = c
				}
				c.days = append(c.days, days)
				lv := legend[gk.key]
				if lv == nil {
					lv = &struct {
						label string
						days  []float64
					}{label: gk.label}
					legend[gk.key] = lv
				}
				lv.days = append(lv.days, days)
			}
			if len(res.Points) < maxPoints {
				p := &pb.InsightPoint{TaskId: t.ID, Identifier: t.Identifier, Title: t.Title, Url: t.URL,
					Date: at.In(r.Start.Location()).Format("2006-01-02"), Days: days, GroupKey: sk.key, GroupLabel: sk.label}
				if len(gks) > 0 {
					p.SegmentKey = gks[0].key
				}
				res.Points = append(res.Points, p)
			} else {
				res.PointsOmitted++
			}
		}
	}
	res.Stats = timeStats(all)
	res.Total = res.Stats.P50
	res.Tasks = res.Stats.N
	for key, g := range groups {
		st := timeStats(g.days)
		grp := &pb.InsightGroup{Key: key, Label: g.label, Value: st.P50, Tasks: st.N, Stats: st, SharedTasks: g.shared}
		for ck, c := range g.cells {
			cs := timeStats(c.days)
			grp.Segments = append(grp.Segments, &pb.InsightValue{Key: ck, Label: c.label, Value: cs.P50, Tasks: cs.N, Stats: cs})
		}
		sortInsightValues(grp.Segments, segDim)
		res.Groups = append(res.Groups, grp)
	}
	sortGroups(res.Groups, sliceDim)
	for k, lv := range legend {
		ls := timeStats(lv.days)
		res.SegmentValues = append(res.SegmentValues, &pb.InsightValue{Key: k, Label: lv.label, Value: ls.P50, Tasks: ls.N, Stats: ls})
	}
	sortInsightValues(res.SegmentValues, segDim)
	return res
}

// timeInsightTasks lists the tasks behind a time-measure value with their
// durations, longest first; total is their median, which equals the clicked value.
func (d *Dataset) timeInsightTasks(dash *pb.InsightFilters, ins *pb.InsightSpec, sliceKey, segKey string, r Range) *pb.ListInsightTasksResponse {
	f := mergeFilters(dash, ins.GetFilters())
	sliceDim, segDim, measure := ins.GetSlice(), ins.GetSegment(), ins.GetMeasure()
	res := &pb.ListInsightTasksResponse{}
	var days []float64
	for _, x := range d.insightTasks(f, r, measure) {
		if sliceKey != "" && !hasKey(d.dimKeys(x.t, sliceDim, f, r), sliceKey) {
			continue
		}
		if segKey != "" && segDim != "" && !hasKey(d.dimKeys(x.t, segDim, f, r), segKey) {
			continue
		}
		v, _ := d.durationDays(x.t, measure)
		row := d.taskRow(x.t, x.pts(), 0)
		row.HasDays, row.Days = true, v
		if sliceKey != "" {
			row.SplitCount = int32(splitCount(sliceDim, len(d.dimKeys(x.t, sliceDim, f, r)), "", 1))
		}
		res.Tasks = append(res.Tasks, row)
		days = append(days, v)
	}
	sort.SliceStable(res.Tasks, func(i, j int) bool { return res.Tasks[i].Days > res.Tasks[j].Days })
	res.Total = timeStats(days).P50
	return res
}
