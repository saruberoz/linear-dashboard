package report

import (
	"sort"
	"strconv"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

// Contributor-table dimensions.
const (
	DimPriority = "priority"
	DimStatus   = "status"
	DimSegment  = "segment"
)

// Reserved column and assignee keys for drill-downs.
const (
	ColumnTotal   = "_total"
	ColumnMissing = "_missing"
	Unassigned    = "_unassigned"
)

var priorityOrder = []struct {
	key   int
	label string
}{{1, "Urgent"}, {2, "High"}, {3, "Medium"}, {4, "Low"}, {0, "No priority"}}

var statusOrder = []struct{ key, label string }{
	{StatusDone, "Done"}, {StatusInProgress, "In progress"}, {StatusNotStarted, "Not started"},
}

// weighted is a task with the fraction of its effort that the current scope counts.
// On a segment scope, a task in a project with N segments has weight 1/N.
type weighted struct {
	t *Task
	w float64
}

func (x weighted) pts() float64 { return x.t.EffectivePts * x.w }

// summarize builds the summary block. Canceled tasks count toward neither effort
// nor tasks_total. segmentScope hides the Segment contributor table, because on a
// segment view every task already belongs to that one segment.
func (d *Dataset) summarize(ws []weighted, segmentScope bool) *pb.Summary {
	s := &pb.Summary{ByStatus: &pb.StatusSplit{}}
	prio := map[int]float64{}
	label := map[string]float64{}
	for _, x := range ws {
		t := x.t
		if t.MissingEstimate {
			s.TasksMissing++
		}
		if t.Status() == StatusCanceled {
			continue
		}
		s.TasksTotal++
		e := x.pts()
		s.Total += e
		addStatus(s.ByStatus, t.Status(), e)
		prio[t.Priority] += e
		names := d.LabelNames(t)
		if len(names) == 0 {
			s.Unlabeled += e
			continue
		}
		for _, n := range names {
			label[n] += e / float64(len(names))
		}
	}
	for _, p := range priorityOrder {
		s.ByPriority = append(s.ByPriority, &pb.Breakdown{
			Key: strconv.Itoa(p.key), Label: p.label, Pts: prio[p.key], Pct: pct(prio[p.key], s.Total),
		})
	}
	labeled := s.Total - s.Unlabeled
	for n, v := range label {
		if v > 0 {
			s.ByLabel = append(s.ByLabel, &pb.Breakdown{Key: n, Label: n, Pts: v, Pct: pct(v, labeled)})
		}
	}
	sort.Slice(s.ByLabel, func(i, j int) bool {
		if s.ByLabel[i].Pts != s.ByLabel[j].Pts {
			return s.ByLabel[i].Pts > s.ByLabel[j].Pts
		}
		return s.ByLabel[i].Key < s.ByLabel[j].Key
	})
	dims := []string{DimPriority, DimStatus}
	if !segmentScope {
		dims = append(dims, DimSegment)
	}
	for _, dim := range dims {
		s.Contributors = append(s.Contributors, d.contributors(ws, dim))
	}
	return s
}

func addStatus(s *pb.StatusSplit, status string, e float64) {
	switch status {
	case StatusDone:
		s.Done += e
	case StatusInProgress:
		s.InProgress += e
	case StatusNotStarted:
		s.NotStarted += e
	}
}

func pct(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b * 100
}

// columnsOf returns the column keys of a dimension, in display order.
func columnsOf(dim string) []*pb.Breakdown {
	var out []*pb.Breakdown
	switch dim {
	case DimPriority:
		for _, p := range priorityOrder {
			out = append(out, &pb.Breakdown{Key: strconv.Itoa(p.key), Label: p.label})
		}
	case DimStatus:
		for _, s := range statusOrder {
			out = append(out, &pb.Breakdown{Key: s.key, Label: s.label})
		}
	case DimSegment:
		for _, s := range Segments {
			out = append(out, &pb.Breakdown{Key: s.Key, Label: s.Label})
		}
	}
	return out
}

// keysOf returns the column keys a task falls in for a dimension. Under the
// segment dimension a task belongs to every segment of its project, and its
// effort is split evenly between them.
func (d *Dataset) keysOf(t *Task, dim string) []string {
	switch dim {
	case DimPriority:
		return []string{strconv.Itoa(t.Priority)}
	case DimStatus:
		return []string{t.Status()}
	case DimSegment:
		if p := d.Projects[t.ProjectID]; p != nil {
			return p.Segments
		}
	}
	return nil
}

func assigneeKey(t *Task) string {
	if t.AssigneeID == "" {
		return Unassigned
	}
	return t.AssigneeID
}

// teamOf returns the row's entry for a team, adding it when new.
func teamOf(row *pb.ContributorRow, t Team) *pb.ContributorTeam {
	for _, ct := range row.Teams {
		if ct.Key == t.Key {
			return ct
		}
	}
	ct := &pb.ContributorTeam{Key: t.Key, Name: t.Name}
	row.Teams = append(row.Teams, ct)
	return ct
}

// contributors builds one contributor table: assignee rows by the dimension's columns.
func (d *Dataset) contributors(ws []weighted, dim string) *pb.ContributorTable {
	tbl := &pb.ContributorTable{Dim: dim, Columns: columnsOf(dim)}
	colTotal := map[string]float64{}
	rows := map[string]*pb.ContributorRow{}
	for _, x := range ws {
		t := x.t
		if t.Status() == StatusCanceled {
			continue
		}
		id := assigneeKey(t)
		row := rows[id]
		if row == nil {
			name := d.AssigneeName(t.AssigneeID)
			if id == Unassigned {
				name = "Unassigned"
			}
			row = &pb.ContributorRow{AssigneeId: id, Assignee: name, Cells: map[string]*pb.ContributorCell{}}
			rows[id] = row
		}
		e := x.pts()
		row.Total += e
		row.Count++
		team := teamOf(row, d.Teams[t.TeamID])
		team.Pts += e
		team.Count++
		tbl.Total += e
		if t.MissingEstimate {
			row.Missing++
			tbl.Missing++
		}
		keys := d.keysOf(t, dim)
		for _, k := range keys {
			c := row.Cells[k]
			if c == nil {
				c = &pb.ContributorCell{}
				row.Cells[k] = c
			}
			share := e / float64(len(keys))
			c.Pts += share
			c.Count++
			colTotal[k] += share
		}
	}
	for _, c := range tbl.Columns {
		c.Pts = colTotal[c.Key]
		c.Pct = pct(c.Pts, tbl.Total)
	}
	for _, r := range rows {
		sort.Slice(r.Teams, func(i, j int) bool {
			a, b := r.Teams[i], r.Teams[j]
			if a.Pts != b.Pts {
				return a.Pts > b.Pts
			}
			if a.Count != b.Count {
				return a.Count > b.Count
			}
			return a.Key < b.Key
		})
		tbl.Rows = append(tbl.Rows, r)
	}
	sort.Slice(tbl.Rows, func(i, j int) bool {
		a, b := tbl.Rows[i], tbl.Rows[j]
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.Assignee < b.Assignee
	})
	return tbl
}

// cellTasks returns exactly the tasks behind one contributor-table number, each
// with the share it contributed, so the shares add up to the number shown.
func (d *Dataset) cellTasks(ws []weighted, dim, column, assigneeID string) ([]*pb.TaskRow, float64) {
	var rows []*pb.TaskRow
	var total float64
	for _, x := range ws {
		t := x.t
		if t.Status() == StatusCanceled || assigneeKey(t) != assigneeID {
			continue
		}
		share := x.pts()
		switch column {
		case ColumnTotal:
		case ColumnMissing:
			if !t.MissingEstimate {
				continue
			}
		default:
			keys := d.keysOf(t, dim)
			if !contains(keys, column) {
				continue
			}
			share /= float64(len(keys))
		}
		rows = append(rows, d.taskRow(t, share, 0))
		total += share
	}
	sortTaskRows(rows)
	return rows, total
}

// taskRow converts a task for the API.
func (d *Dataset) taskRow(t *Task, share float64, depth int) *pb.TaskRow {
	r := &pb.TaskRow{
		Id: t.ID, Identifier: t.Identifier, Title: t.Title, Url: t.URL,
		StateName: t.StateName, Status: t.Status(), Priority: int32(t.Priority),
		Assignee: d.AssigneeName(t.AssigneeID), Labels: d.LabelNames(t),
		CountedPts: t.EffectivePts, SharePts: share, Deferred: t.Deferred(),
		Missing: t.MissingEstimate, ProjectId: t.ProjectID, Depth: int32(depth),
		TaskSegments: t.TaskSegments, CreatedAt: t.CreatedAt.Format("2006-01-02"),
	}
	if t.Estimate != nil {
		r.HasEstimate = true
		r.Estimate = *t.Estimate
	}
	if p := d.Projects[t.ProjectID]; p != nil {
		r.ProjectName = p.Name
	}
	return r
}

// sortTaskRows orders tasks by identifier, numerically within a team prefix.
func sortTaskRows(rows []*pb.TaskRow) {
	sort.SliceStable(rows, func(i, j int) bool { return identifierLess(rows[i].Identifier, rows[j].Identifier) })
}

func identifierLess(a, b string) bool {
	pa, na := splitIdentifier(a)
	pb, nb := splitIdentifier(b)
	if pa != pb {
		return pa < pb
	}
	return na < nb
}

func splitIdentifier(s string) (string, int) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '-' {
			n, err := strconv.Atoi(s[i+1:])
			if err != nil {
				return s, 0
			}
			return s[:i], n
		}
	}
	return s, 0
}
