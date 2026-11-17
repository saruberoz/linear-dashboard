package report

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

// ErrInvalidInsight reports an insight spec with an unknown measure, dimension or view.
var ErrInvalidInsight = errors.New("report: invalid insight")

// Insight measures and views.
const (
	MeasureEffort = "effort"
	MeasureCount  = "count"
)

const noneKey = "_none"

// Insight dimensions. Multi-valued dimensions (segment, label, initiative) can
// put one task in several groups: effort splits evenly across segments and
// labels (as everywhere else in the dashboard) and counts in full under each
// initiative; the issue count adds 1 to every group the task is in.
var insightDims = map[string]struct {
	multi, split bool
}{
	"assignee": {}, "priority": {}, "status": {}, "team": {}, "project": {},
	"created_month": {}, "completed_month": {}, "estimate": {}, "work_type": {},
	"segment":    {multi: true, split: true},
	"label":      {multi: true, split: true},
	"initiative": {multi: true},
}

type dimKey struct{ key, label string }

// mergedFilters is the intersection of dashboard and insight filters.
type mergedFilters struct {
	segments, statuses, teams map[string]bool
	labels                    map[string]bool // lower-cased label names
	labelMatch                []string        // lower-cased substrings; a task needs a label containing each
	priorities                map[int32]bool
	untagged                  bool
	none                      bool // two non-empty lists had nothing in common
}

func mergeFilters(a, b *pb.InsightFilters) mergedFilters {
	m := mergedFilters{untagged: a.GetIncludeUntagged() && b.GetIncludeUntagged()}
	intersect := func(x, y []string) map[string]bool {
		if len(x) == 0 && len(y) == 0 {
			return nil
		}
		out := map[string]bool{}
		for _, v := range x {
			if len(y) == 0 || contains(y, v) {
				out[v] = true
			}
		}
		if len(x) == 0 {
			for _, v := range y {
				out[v] = true
			}
		}
		if len(out) == 0 {
			m.none = true
		}
		return out
	}
	m.segments = intersect(a.GetSegments(), b.GetSegments())
	m.statuses = intersect(a.GetStatuses(), b.GetStatuses())
	m.teams = intersect(a.GetTeamIds(), b.GetTeamIds())
	m.labels = intersect(lowerAll(a.GetLabels()), lowerAll(b.GetLabels()))
	for _, x := range []string{a.GetLabelMatch(), b.GetLabelMatch()} {
		if x = strings.ToLower(strings.TrimSpace(x)); x != "" {
			m.labelMatch = append(m.labelMatch, x)
		}
	}
	var pa, pbs []string
	for _, p := range a.GetPriorities() {
		pa = append(pa, strconv.Itoa(int(p)))
	}
	for _, p := range b.GetPriorities() {
		pbs = append(pbs, strconv.Itoa(int(p)))
	}
	if ps := intersect(pa, pbs); ps != nil {
		m.priorities = map[int32]bool{}
		for p := range ps {
			n, _ := strconv.Atoi(p)
			m.priorities[int32(n)] = true
		}
	}
	return m
}

// ValidateSpec checks an insight spec's measure, dimensions and view.
func ValidateSpec(spec *pb.InsightSpec) error {
	switch spec.GetMeasure() {
	case MeasureEffort, MeasureCount, MeasureCycleTime, MeasureLeadTime, MeasureIssueAge, MeasureTriageTime:
	default:
		return fmt.Errorf("%w: measure %q", ErrInvalidInsight, spec.GetMeasure())
	}
	if _, ok := insightDims[spec.GetSlice()]; !ok {
		return fmt.Errorf("%w: slice %q", ErrInvalidInsight, spec.GetSlice())
	}
	if s := spec.GetSegment(); s != "" {
		if _, ok := insightDims[s]; !ok || s == spec.GetSlice() {
			return fmt.Errorf("%w: segment %q", ErrInvalidInsight, s)
		}
	}
	switch spec.GetView() {
	case "bar", "table", "metric":
	case "scatter":
		if !IsTimeMeasure(spec.GetMeasure()) {
			return fmt.Errorf("%w: a scatterplot needs a time measure", ErrInvalidInsight)
		}
	default:
		return fmt.Errorf("%w: view %q", ErrInvalidInsight, spec.GetView())
	}
	return nil
}

// insightTasks returns the tasks a measure looks at that pass the filters, each
// with its effort weight (a segment filter counts a shared project at its share).
// Effort and count use the dashboard's range rule and skip canceled tasks; each
// time measure has its own population (see inTimePopulation).
func (d *Dataset) insightTasks(f mergedFilters, r Range, measure string) []weighted {
	if f.none {
		return nil
	}
	var out []weighted
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if IsTimeMeasure(measure) {
			if !d.inTimePopulation(t, measure, r) {
				continue
			}
		} else if t.Status() == StatusCanceled || !r.InRange(t) {
			continue
		}
		if !f.labelsMatch(d, t) {
			continue
		}
		p := d.Projects[t.ProjectID]
		tagged := p != nil && len(p.Segments) > 0
		if !tagged && !f.untagged {
			continue
		}
		if f.statuses != nil && !f.statuses[t.Status()] {
			continue
		}
		if f.priorities != nil && !f.priorities[int32(t.Priority)] {
			continue
		}
		if f.teams != nil && !f.teams[t.TeamID] {
			continue
		}
		w := 1.0
		if f.segments != nil {
			if !tagged {
				continue
			}
			n := 0
			for _, s := range p.Segments {
				if f.segments[s] {
					n++
				}
			}
			if n == 0 {
				continue
			}
			w = float64(n) / float64(len(p.Segments))
		}
		out = append(out, weighted{t, w})
	}
	return out
}

// dimKeys returns the groups a task falls in for a dimension.
func (d *Dataset) dimKeys(t *Task, dim string, f mergedFilters, r Range) []dimKey {
	loc := r.Start.Location()
	switch dim {
	case "assignee":
		if t.AssigneeID == "" {
			return []dimKey{{Unassigned, "Unassigned"}}
		}
		return []dimKey{{t.AssigneeID, d.AssigneeName(t.AssigneeID)}}
	case "priority":
		return []dimKey{{strconv.Itoa(t.Priority), priorityLabel(t.Priority)}}
	case "status":
		return []dimKey{{t.Status(), statusLabel(t.Status())}}
	case "team":
		tm := d.Teams[t.TeamID]
		return []dimKey{{t.TeamID, firstNonEmpty(tm.Name, tm.Key, "Unknown team")}}
	case "project":
		if p := d.Projects[t.ProjectID]; p != nil {
			return []dimKey{{p.ID, p.Name}}
		}
		return []dimKey{{noneKey, "No project"}}
	case "segment":
		var out []dimKey
		if p := d.Projects[t.ProjectID]; p != nil {
			for _, s := range p.Segments {
				if f.segments == nil || f.segments[s] {
					out = append(out, dimKey{s, SegmentLabel(s)})
				}
			}
		}
		if len(out) == 0 {
			return []dimKey{{noneKey, "Untagged"}}
		}
		return out
	case "label":
		// With label filters set, group only by the labels they pick, so a feature
		// label is not split with process labels such as "Release This Sprint".
		var all, picked []dimKey
		for _, id := range t.LabelIDs {
			if l, ok := d.Labels[id]; ok {
				k := dimKey{id, l.FullName()}
				all = append(all, k)
				if f.pickLabel(l.Name) {
					picked = append(picked, k)
				}
			}
		}
		if len(picked) > 0 {
			return picked
		}
		if len(all) == 0 {
			return []dimKey{{noneKey, "No label"}}
		}
		return all
	case "initiative":
		var out []dimKey
		if p := d.Projects[t.ProjectID]; p != nil {
			for _, ini := range d.VisibleInitiativesOf(p) {
				out = append(out, dimKey{ini.ID, ini.Name})
			}
		}
		if len(out) == 0 {
			return []dimKey{{noneKey, "No initiative"}}
		}
		return out
	case "estimate":
		if t.Estimate == nil {
			return []dimKey{{noneKey, "No estimate"}}
		}
		v := strconv.FormatFloat(*t.Estimate, 'f', -1, 64)
		return []dimKey{{v, v + " pts"}}
	case "work_type":
		if d.isBug(t) {
			return []dimKey{{"bug", "Bugs & hotfixes"}}
		}
		return []dimKey{{"other", "Other work"}}
	case "created_month":
		m := t.CreatedAt.In(loc)
		return []dimKey{{m.Format("2006-01"), m.Format("Jan 2006")}}
	case "completed_month":
		if t.CompletedAt == nil {
			return []dimKey{{noneKey, "Not completed"}}
		}
		m := t.CompletedAt.In(loc)
		return []dimKey{{m.Format("2006-01"), m.Format("Jan 2006")}}
	}
	return nil
}

// share is what one task contributes to one (slice, segment) cell.
func share(measure string, x weighted, sliceDim string, nSlice int, segDim string, nSeg int) float64 {
	if measure == MeasureCount {
		return 1
	}
	v := x.pts()
	if insightDims[sliceDim].split {
		v /= float64(nSlice)
	}
	if segDim != "" && insightDims[segDim].split {
		v /= float64(nSeg)
	}
	return v
}

// RunInsight computes one insight.
func (d *Dataset) RunInsight(dash *pb.InsightFilters, ins *pb.InsightSpec, r Range) (*pb.InsightResult, error) {
	if err := ValidateSpec(ins); err != nil {
		return nil, err
	}
	if IsTimeMeasure(ins.GetMeasure()) {
		return d.runTimeInsight(dash, ins, r), nil
	}
	f := mergeFilters(dash, ins.GetFilters())
	sliceDim, segDim, measure := ins.GetSlice(), ins.GetSegment(), ins.GetMeasure()
	ws := d.insightTasks(f, r, measure)
	res := &pb.InsightResult{Measure: measure, Slice: sliceDim, Segment: segDim,
		MultiValued: insightDims[sliceDim].multi || (segDim != "" && insightDims[segDim].multi)}

	groups := map[string]*pb.InsightGroup{}
	segIndex := map[string]map[string]*pb.InsightValue{}
	legend := map[string]*pb.InsightValue{}
	for _, x := range ws {
		if measure == MeasureCount {
			res.Total++
		} else {
			res.Total += x.pts()
		}
		res.Tasks++
		sks := d.dimKeys(x.t, sliceDim, f, r)
		var gks []dimKey
		if segDim != "" {
			gks = d.dimKeys(x.t, segDim, f, r)
		}
		for _, sk := range sks {
			g := groups[sk.key]
			if g == nil {
				g = &pb.InsightGroup{Key: sk.key, Label: sk.label}
				groups[sk.key] = g
				segIndex[sk.key] = map[string]*pb.InsightValue{}
			}
			g.Tasks++
			if len(sks) > 1 {
				g.SharedTasks++
			}
			if segDim == "" {
				g.Value += share(measure, x, sliceDim, len(sks), "", 1)
				continue
			}
			for _, gk := range gks {
				v := share(measure, x, sliceDim, len(sks), segDim, len(gks))
				g.Value += v
				cell := segIndex[sk.key][gk.key]
				if cell == nil {
					cell = &pb.InsightValue{Key: gk.key, Label: gk.label}
					segIndex[sk.key][gk.key] = cell
					g.Segments = append(g.Segments, cell)
				}
				cell.Value += v
				cell.Tasks++
				if len(gks) > 1 {
					cell.SharedTasks++
				}
				lv := legend[gk.key]
				if lv == nil {
					lv = &pb.InsightValue{Key: gk.key, Label: gk.label}
					legend[gk.key] = lv
				}
				lv.Value += v
				lv.Tasks++
			}
		}
	}
	for _, g := range groups {
		if measure == MeasureCount {
			g.Value = float64(g.Tasks) // distinct tasks, even if a task sits in two of its segments
		}
		res.Groups = append(res.Groups, g)
	}
	sortGroups(res.Groups, sliceDim)
	for _, lv := range legend {
		res.SegmentValues = append(res.SegmentValues, lv)
	}
	sortInsightValues(res.SegmentValues, segDim)
	for _, g := range res.Groups {
		sortInsightValues(g.Segments, segDim)
	}
	return res, nil
}

// InsightTasks returns the tasks behind one value of an insight, each with the
// effort it contributed; total equals the clicked value (effort or count).
func (d *Dataset) InsightTasks(dash *pb.InsightFilters, ins *pb.InsightSpec, sliceKey, segKey string, r Range) (*pb.ListInsightTasksResponse, error) {
	if err := ValidateSpec(ins); err != nil {
		return nil, err
	}
	if IsTimeMeasure(ins.GetMeasure()) {
		return d.timeInsightTasks(dash, ins, sliceKey, segKey, r), nil
	}
	f := mergeFilters(dash, ins.GetFilters())
	sliceDim, segDim, measure := ins.GetSlice(), ins.GetSegment(), ins.GetMeasure()
	res := &pb.ListInsightTasksResponse{}
	for _, x := range d.insightTasks(f, r, measure) {
		sks := d.dimKeys(x.t, sliceDim, f, r)
		if sliceKey != "" && !hasKey(sks, sliceKey) {
			continue
		}
		nSeg := 1
		if segDim != "" {
			gks := d.dimKeys(x.t, segDim, f, r)
			if segKey != "" && !hasKey(gks, segKey) {
				continue
			}
			nSeg = len(gks)
		}
		nSlice, sDim := len(sks), sliceDim
		if sliceKey == "" {
			nSlice, sDim = 1, "" // no slice clicked: the whole task counts once
		}
		seg := segDim
		if segKey == "" {
			seg = ""
		}
		effort := share(MeasureEffort, x, sDim, nSlice, seg, nSeg)
		row := d.taskRow(x.t, effort, 0)
		row.SplitCount = int32(splitCount(sDim, len(sks), seg, nSeg))
		res.Tasks = append(res.Tasks, row)
		if measure == MeasureCount {
			res.Total++
		} else {
			res.Total += effort
		}
	}
	sortTaskRows(res.Tasks)
	return res, nil
}

func hasKey(ks []dimKey, k string) bool {
	for _, x := range ks {
		if x.key == k {
			return true
		}
	}
	return false
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

func priorityLabel(p int) string {
	for _, x := range priorityOrder {
		if x.key == p {
			return x.label
		}
	}
	return strconv.Itoa(p)
}

func statusLabel(s string) string {
	for _, x := range statusOrder {
		if x.key == s {
			return x.label
		}
	}
	return s
}

// orderOf gives a fixed order for dimensions that have one; -1 means "sort by value".
func orderOf(dim, key string) int {
	switch dim {
	case "priority":
		for i, p := range priorityOrder {
			if strconv.Itoa(p.key) == key {
				return i
			}
		}
	case "status":
		for i, s := range statusOrder {
			if s.key == key {
				return i
			}
		}
	case "segment":
		for i, s := range Segments {
			if s.Key == key {
				return i
			}
		}
		return len(Segments)
	case "work_type":
		if key == "bug" {
			return 0
		}
		return 1
	}
	return -1
}

// isBug classifies a task as a bug when any of its labels mentions "bug" or
// "hotfix" (Bug, HOTFIX, "New Bug - …", "Repeat Bug - …").
func (d *Dataset) isBug(t *Task) bool {
	for _, id := range t.LabelIDs {
		if l, ok := d.Labels[id]; ok {
			n := strings.ToLower(l.Name)
			if strings.Contains(n, "bug") || strings.Contains(n, "hotfix") {
				return true
			}
		}
	}
	return false
}

func sortGroups(gs []*pb.InsightGroup, dim string) {
	sort.SliceStable(gs, func(i, j int) bool {
		return lessKey(dim, gs[i].Key, gs[j].Key, gs[i].Value, gs[j].Value, gs[i].Label, gs[j].Label)
	})
}

func sortInsightValues(vs []*pb.InsightValue, dim string) {
	sort.SliceStable(vs, func(i, j int) bool {
		return lessKey(dim, vs[i].Key, vs[j].Key, vs[i].Value, vs[j].Value, vs[i].Label, vs[j].Label)
	})
}

// lessKey orders months chronologically, fixed dimensions by their order, and
// the rest by value (largest first); "No …" groups always go last.
func lessKey(dim, ka, kb string, va, vb float64, la, lb string) bool {
	if (ka == noneKey) != (kb == noneKey) {
		return kb == noneKey
	}
	if dim == "created_month" || dim == "completed_month" {
		return ka < kb
	}
	if dim == "estimate" {
		a, _ := strconv.ParseFloat(ka, 64)
		b, _ := strconv.ParseFloat(kb, 64)
		return a < b
	}
	if oa, ob := orderOf(dim, ka), orderOf(dim, kb); oa >= 0 || ob >= 0 {
		return oa < ob
	}
	if va != vb {
		return va > vb
	}
	return la < lb
}

// splitCount is how many groups a task's value is shared with in a drill-down:
// the product of the multi-valued slice and segment memberships it was split by.
func splitCount(sliceDim string, nSlice int, segDim string, nSeg int) int {
	n := 1
	if insightDims[sliceDim].multi && nSlice > 1 {
		n *= nSlice
	}
	if segDim != "" && insightDims[segDim].multi && nSeg > 1 {
		n *= nSeg
	}
	return n
}

// pickLabel reports whether one label satisfies the label filters on its own.
// With no label filters every label is picked.
func (f mergedFilters) pickLabel(name string) bool {
	n := strings.ToLower(name)
	if f.labels != nil && !f.labels[n] {
		return false
	}
	for _, m := range f.labelMatch {
		if !strings.Contains(n, m) {
			return false
		}
	}
	return true
}
