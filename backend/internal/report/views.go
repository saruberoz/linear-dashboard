package report

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	pb "linear-dashboard/backend/api/dashboard/v1"
)

// Errors returned by views; the handler maps them to HTTP errors.
var (
	ErrNotFound     = errors.New("report: not found")
	ErrInvalidScope = errors.New("report: invalid scope")
	ErrInvalidDim   = errors.New("report: invalid dimension")
)

// Scope is a parsed scope string.
type Scope struct {
	Kind string // "all" | "segment" | "initiative" | "project"
	ID   string
}

// ParseScope parses "all" | "segment:KEY" | "initiative:ID" | "project:ID". Empty means "all".
func ParseScope(s string) (Scope, error) {
	if s == "" || s == "all" {
		return Scope{Kind: "all"}, nil
	}
	kind, id, ok := strings.Cut(s, ":")
	if !ok || id == "" {
		return Scope{}, fmt.Errorf("%w: %q", ErrInvalidScope, s)
	}
	switch kind {
	case "segment":
		if !isSegmentKey(id) {
			return Scope{}, fmt.Errorf("%w: unknown segment %q", ErrInvalidScope, id)
		}
	case "initiative", "project":
	default:
		return Scope{}, fmt.Errorf("%w: %q", ErrInvalidScope, s)
	}
	return Scope{Kind: kind, ID: id}, nil
}

func isSegmentKey(k string) bool {
	for _, s := range Segments {
		if s.Key == k {
			return true
		}
	}
	return false
}

// scopeTasks returns the in-range tasks of a scope with their weights.
//   - all: every visible tagged project, weight 1
//   - segment: visible projects with that segment, weight 1/N for N segments
//   - initiative: the initiative's visible tagged projects, weight 1
//   - project: that project, tagged or not, weight 1
func (d *Dataset) scopeTasks(sc Scope, r Range) ([]weighted, error) {
	projects, err := d.scopeProjects(sc, r)
	if err != nil {
		return nil, err
	}
	var out []weighted
	for _, p := range projects {
		w := 1.0
		if sc.Kind == "segment" {
			w = 1 / float64(len(p.Segments))
		}
		for _, t := range d.TasksInRange(p.ID, r) {
			out = append(out, weighted{t, w})
		}
	}
	return out, nil
}

func (d *Dataset) scopeProjects(sc Scope, r Range) ([]*Project, error) {
	switch sc.Kind {
	case "all":
		return d.TaggedProjects(r, ""), nil
	case "segment":
		return d.TaggedProjects(r, sc.ID), nil
	case "initiative":
		ini := d.Initiatives[sc.ID]
		if ini == nil || !ini.Visible {
			return nil, fmt.Errorf("%w: initiative %s", ErrNotFound, sc.ID)
		}
		var out []*Project
		for _, pid := range ini.ProjectIDs {
			if p := d.Projects[pid]; p != nil && len(p.Segments) > 0 && d.ProjectVisible(p, r) {
				out = append(out, p)
			}
		}
		sortProjects(out)
		return out, nil
	case "project":
		p := d.Projects[sc.ID]
		if p == nil {
			return nil, fmt.Errorf("%w: project %s", ErrNotFound, sc.ID)
		}
		return []*Project{p}, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrInvalidScope, sc.Kind)
}

// projectRow summarizes one project. segmentKey, when set, fills share_pts with
// the project's 1/N share for that segment.
func (d *Dataset) projectRow(p *Project, r Range, segmentKey string) *pb.ProjectRow {
	row := &pb.ProjectRow{Id: p.ID, Name: p.Name, Url: p.URL, State: p.State, StartDate: p.StartDate, TargetDate: p.TargetDate,
		Segments: nonNil(p.Segments), Status: &pb.StatusSplit{}}
	for _, t := range d.TasksInRange(p.ID, r) {
		if t.MissingEstimate {
			row.Missing++
		}
		if t.Status() == StatusCanceled {
			continue
		}
		row.Tasks++
		row.Pts += t.EffectivePts
		addStatus(row.Status, t.Status(), t.EffectivePts)
	}
	row.SharePts = row.Pts
	if segmentKey != "" && len(p.Segments) > 1 {
		row.SplitOf = int32(len(p.Segments))
		row.SharePts = row.Pts / float64(len(p.Segments))
	}
	return row
}

// Menu returns the left-menu tree: segment → visible initiatives → projects.
func (d *Dataset) Menu(r Range) *pb.GetMenuResponse {
	res := &pb.GetMenuResponse{}
	for _, s := range Segments {
		ps := d.TaggedProjects(r, s.Key)
		ms := &pb.MenuSegment{Key: s.Key, Label: s.Label, ProjectCount: int32(len(ps))}
		for _, g := range d.groupByInitiative(ps) {
			mi := &pb.MenuInitiative{Id: g.id, Name: g.name}
			for _, p := range g.projects {
				mi.Projects = append(mi.Projects, &pb.MenuProject{Id: p.ID, Name: p.Name})
			}
			ms.Initiatives = append(ms.Initiatives, mi)
		}
		res.Segments = append(res.Segments, ms)
	}
	missing, _ := d.missingTasks(Scope{Kind: "all"}, r, false)
	res.MissingCount = int32(len(missing))
	res.WarningCount = int32(len(d.warningTasks(r)))
	res.UntaggedCount = notCanceled(d.untaggedProjects(r))
	res.NoInitiativeCount = int32(len(d.projectsWithoutInitiative(r)))
	res.EmptyInitiativeCount = int32(len(d.initiativesWithoutProjects()))
	res.NoProjectTaskCount = int32(len(d.tasksWithoutProject(r)))
	res.MissingTimelineCount = int32(len(d.initiativesWithoutTarget())) + notCanceled(d.projectsWithoutTimeline(r))
	res.EmptyProjectCount = notCanceled(d.projectsWithoutTasks())
	return res
}

type initiativeGroup struct {
	id, name, url string
	projects      []*Project
}

// groupByInitiative groups projects under their visible initiatives (a project
// in two initiatives appears under both), with "No initiative" last.
func (d *Dataset) groupByInitiative(ps []*Project) []*initiativeGroup {
	byID := map[string]*initiativeGroup{}
	for _, p := range ps {
		inis := d.VisibleInitiativesOf(p)
		if len(inis) == 0 {
			g := byID[""]
			if g == nil {
				g = &initiativeGroup{name: "No initiative"}
				byID[""] = g
			}
			g.projects = append(g.projects, p)
			continue
		}
		for _, ini := range inis {
			g := byID[ini.ID]
			if g == nil {
				g = &initiativeGroup{id: ini.ID, name: ini.Name, url: ini.URL}
				byID[ini.ID] = g
			}
			g.projects = append(g.projects, p)
		}
	}
	out := make([]*initiativeGroup, 0, len(byID))
	for _, g := range byID {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		if (out[i].id == "") != (out[j].id == "") {
			return out[j].id == ""
		}
		return out[i].name < out[j].name
	})
	return out
}

// Overview returns the segment shares and the all-tagged-work summary.
func (d *Dataset) Overview(r Range) *pb.GetOverviewResponse {
	res := &pb.GetOverviewResponse{}
	shares := map[string]*pb.SegmentShare{}
	for _, s := range Segments {
		sh := &pb.SegmentShare{Key: s.Key, Label: s.Label, Status: &pb.StatusSplit{}}
		shares[s.Key] = sh
		res.Segments = append(res.Segments, sh)
	}
	for _, p := range d.TaggedProjects(r, "") {
		row := d.projectRow(p, r, "")
		n := float64(len(p.Segments))
		for _, k := range p.Segments {
			sh := shares[k]
			sh.Projects++
			if len(p.Segments) > 1 {
				sh.Shared++
			}
			sh.Pts += row.Pts / n
			sh.Status.Done += row.Status.Done / n
			sh.Status.InProgress += row.Status.InProgress / n
			sh.Status.NotStarted += row.Status.NotStarted / n
		}
	}
	for _, sh := range res.Segments {
		res.TaggedTotal += sh.Pts
	}
	for _, sh := range res.Segments {
		sh.Pct = pct(sh.Pts, res.TaggedTotal)
	}
	ws, _ := d.scopeTasks(Scope{Kind: "all"}, r)
	res.Summary = d.summarize(ws, false)
	warn := d.warningTasks(r)
	res.Warnings = &pb.WarningSummary{Tasks: int32(len(warn))}
	for _, t := range warn {
		if t.Status() != StatusCanceled && t.Estimate != nil {
			res.Warnings.Pts += *t.Estimate
		}
		if p := d.Projects[t.ProjectID]; p == nil || len(p.Segments) == 0 {
			res.Warnings.Uncounted++
		}
	}
	res.UntaggedProjects = notCanceled(d.untaggedProjects(r))
	return res
}

// Segment returns one segment's view, with project effort split evenly across segments.
func (d *Dataset) Segment(key string, r Range) (*pb.GetSegmentResponse, error) {
	if !isSegmentKey(key) {
		return nil, fmt.Errorf("%w: segment %s", ErrNotFound, key)
	}
	sc := Scope{Kind: "segment", ID: key}
	ws, err := d.scopeTasks(sc, r)
	if err != nil {
		return nil, err
	}
	ps := d.TaggedProjects(r, key)
	res := &pb.GetSegmentResponse{Key: key, Label: SegmentLabel(key), ProjectCount: int32(len(ps)),
		Summary: d.summarize(ws, true)}
	for _, g := range d.groupByInitiative(ps) {
		ig := &pb.InitiativeGroup{Id: g.id, Name: g.name, Url: g.url}
		for _, p := range g.projects {
			row := d.projectRow(p, r, key)
			ig.SharePts += row.SharePts
			ig.Projects = append(ig.Projects, row)
		}
		res.Groups = append(res.Groups, ig)
	}
	return res, nil
}

// Initiative returns an initiative's view over its tagged projects, at real totals.
func (d *Dataset) Initiative(id string, r Range) (*pb.GetInitiativeResponse, error) {
	sc := Scope{Kind: "initiative", ID: id}
	ps, err := d.scopeProjects(sc, r)
	if err != nil {
		return nil, err
	}
	ws, _ := d.scopeTasks(sc, r)
	ini := d.Initiatives[id]
	res := &pb.GetInitiativeResponse{
		Initiative: &pb.InitiativeRef{Id: ini.ID, Name: ini.Name, Url: ini.URL},
		Summary:    d.summarize(ws, false),
	}
	segs := map[string]bool{}
	for _, p := range ps {
		res.Projects = append(res.Projects, d.projectRow(p, r, ""))
		for _, s := range p.Segments {
			segs[s] = true
		}
	}
	for _, s := range Segments {
		if segs[s.Key] {
			res.Segments = append(res.Segments, s.Key)
		}
	}
	return res, nil
}

// Project returns a project's view at its real total, with in-range tasks nested
// under in-range parents. A sub-task whose parent is out of range is shown at depth 0.
func (d *Dataset) Project(id string, r Range) (*pb.GetProjectResponse, error) {
	p := d.Projects[id]
	if p == nil {
		return nil, fmt.Errorf("%w: project %s", ErrNotFound, id)
	}
	ws, _ := d.scopeTasks(Scope{Kind: "project", ID: id}, r)
	res := &pb.GetProjectResponse{Project: d.projectRow(p, r, ""), Summary: d.summarize(ws, false)}
	for _, ini := range d.VisibleInitiativesOf(p) {
		res.Initiatives = append(res.Initiatives, &pb.InitiativeRef{Id: ini.ID, Name: ini.Name, Url: ini.URL})
	}
	shown := map[string]bool{}
	for _, x := range ws {
		shown[x.t.ID] = true
	}
	var walk func(t *Task, depth int)
	walk = func(t *Task, depth int) {
		res.Tasks = append(res.Tasks, d.taskRow(t, t.EffectivePts, depth))
		var kids []*Task
		for _, c := range d.children[t.ID] {
			if shown[c.ID] && c.ProjectID == id {
				kids = append(kids, c)
			}
		}
		sort.Slice(kids, func(i, j int) bool { return identifierLess(kids[i].Identifier, kids[j].Identifier) })
		for _, c := range kids {
			walk(c, depth+1)
		}
	}
	var roots []*Task
	for _, x := range ws {
		if x.t.ParentID == "" || !shown[x.t.ParentID] {
			roots = append(roots, x.t)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return identifierLess(roots[i].Identifier, roots[j].Identifier) })
	for _, t := range roots {
		walk(t, 0)
	}
	return res, nil
}

// CellTasks returns the drill-down list behind one contributor-table number.
func (d *Dataset) CellTasks(scope, dim, column, assigneeID string, r Range) (*pb.ListTasksResponse, error) {
	sc, err := ParseScope(scope)
	if err != nil {
		return nil, err
	}
	switch dim {
	case DimPriority, DimStatus:
	case DimSegment:
		if sc.Kind == "segment" {
			return nil, fmt.Errorf("%w: segment columns are not shown on segment views", ErrInvalidDim)
		}
	default:
		if column != ColumnTotal && column != ColumnMissing {
			return nil, fmt.Errorf("%w: %q", ErrInvalidDim, dim)
		}
	}
	ws, err := d.scopeTasks(sc, r)
	if err != nil {
		return nil, err
	}
	rows, total := d.cellTasks(ws, dim, column, assigneeID)
	return &pb.ListTasksResponse{Tasks: rows, Total: total}, nil
}

// missingTasks returns the in-range tasks missing an estimate. With all=true the
// "all" scope also covers untagged projects and tasks without a project.
func (d *Dataset) missingTasks(sc Scope, r Range, all bool) ([]*Task, error) {
	var out []*Task
	if sc.Kind == "all" && all {
		for i := range d.Tasks {
			t := &d.Tasks[i]
			if t.MissingEstimate && r.InRange(t) {
				out = append(out, t)
			}
		}
		return out, nil
	}
	ws, err := d.scopeTasks(sc, r)
	if err != nil {
		return nil, err
	}
	for _, x := range ws {
		if x.t.MissingEstimate {
			out = append(out, x.t)
		}
	}
	return out, nil
}

// MissingEstimates groups unestimated in-range tasks by assignee, largest group first.
func (d *Dataset) MissingEstimates(scope string, all bool, r Range) (*pb.ListMissingEstimatesResponse, error) {
	sc, err := ParseScope(scope)
	if err != nil {
		return nil, err
	}
	tasks, err := d.missingTasks(sc, r, all)
	if err != nil {
		return nil, err
	}
	return &pb.ListMissingEstimatesResponse{ScopeTitle: d.scopeTitle(sc, all), Total: int32(len(tasks)),
		Groups: d.byAssignee(tasks)}, nil
}

// byAssignee groups tasks by assignee, oldest task first within a group and the
// largest group first.
func (d *Dataset) byAssignee(tasks []*Task) []*pb.AssigneeGroup {
	groups := map[string]*pb.AssigneeGroup{}
	for _, t := range tasks {
		id := assigneeKey(t)
		g := groups[id]
		if g == nil {
			name := d.AssigneeName(t.AssigneeID)
			if id == Unassigned {
				name = "Unassigned"
			}
			g = &pb.AssigneeGroup{AssigneeId: id, Assignee: name}
			groups[id] = g
		}
		g.Tasks = append(g.Tasks, d.taskRow(t, 0, 0))
	}
	var out []*pb.AssigneeGroup
	for _, g := range groups {
		sort.Slice(g.Tasks, func(i, j int) bool { return g.Tasks[i].CreatedAt < g.Tasks[j].CreatedAt })
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Tasks) != len(out[j].Tasks) {
			return len(out[i].Tasks) > len(out[j].Tasks)
		}
		return out[i].Assignee < out[j].Assignee
	})
	return out
}

func (d *Dataset) scopeTitle(sc Scope, all bool) string {
	switch sc.Kind {
	case "segment":
		return SegmentLabel(sc.ID)
	case "initiative":
		if ini := d.Initiatives[sc.ID]; ini != nil {
			return ini.Name
		}
	case "project":
		if p := d.Projects[sc.ID]; p != nil {
			return p.Name
		}
	}
	if all {
		return "All projects"
	}
	return "All tagged projects"
}

// warningTasks returns in-range tasks that carry a segment label on the task itself.
func (d *Dataset) warningTasks(r Range) []*Task {
	var out []*Task
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if len(t.TaskSegments) > 0 && r.InRange(t) {
			out = append(out, t)
		}
	}
	return out
}

// SegmentLabelWarnings groups task-label warnings by project, largest group first.
// counted is true when the project's own segment label already counts the tasks.
func (d *Dataset) SegmentLabelWarnings(r Range) *pb.ListSegmentLabelWarningsResponse {
	tasks := d.warningTasks(r)
	res := &pb.ListSegmentLabelWarningsResponse{Total: int32(len(tasks))}
	groups := map[string]*pb.WarningGroup{}
	for _, t := range tasks {
		g := groups[t.ProjectID]
		if g == nil {
			g = &pb.WarningGroup{ProjectId: t.ProjectID, ProjectName: "No project", ProjectSegments: []string{}}
			if p := d.Projects[t.ProjectID]; p != nil {
				g.ProjectName, g.ProjectUrl = p.Name, p.URL
				g.ProjectSegments = nonNil(p.Segments)
				g.Counted = len(p.Segments) > 0
			}
			groups[t.ProjectID] = g
		}
		if t.Status() != StatusCanceled && t.Estimate != nil {
			g.Pts += *t.Estimate
		}
		g.Tasks = append(g.Tasks, d.taskRow(t, 0, 0))
	}
	for _, g := range groups {
		sortTaskRows(g.Tasks)
		res.Groups = append(res.Groups, g)
	}
	sort.Slice(res.Groups, func(i, j int) bool {
		if len(res.Groups[i].Tasks) != len(res.Groups[j].Tasks) {
			return len(res.Groups[i].Tasks) > len(res.Groups[j].Tasks)
		}
		return res.Groups[i].ProjectName < res.Groups[j].ProjectName
	})
	return res
}

func (d *Dataset) untaggedProjects(r Range) []*Project {
	var out []*Project
	for _, p := range d.Projects {
		if len(p.Segments) == 0 && d.ProjectVisible(p, r) {
			out = append(out, p)
		}
	}
	sortProjects(out)
	return out
}

// UntaggedProjects lists visible projects without a segment label, largest effort first.
func (d *Dataset) UntaggedProjects(r Range) *pb.ListUntaggedProjectsResponse {
	res := &pb.ListUntaggedProjectsResponse{}
	for _, p := range d.untaggedProjects(r) {
		res.Projects = append(res.Projects, d.projectRow(p, r, ""))
	}
	sort.SliceStable(res.Projects, func(i, j int) bool { return res.Projects[i].Pts > res.Projects[j].Pts })
	return res
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// hiddenInitiativesOf names the project's initiatives that are not shown (sprints, buckets).
func (d *Dataset) hiddenInitiativesOf(p *Project) []string {
	var out []string
	for _, id := range p.InitiativeIDs {
		if ini := d.Initiatives[id]; ini != nil && !ini.Visible {
			out = append(out, ini.Name)
		}
	}
	sort.Strings(out)
	return out
}

func (d *Dataset) projectsWithoutInitiative(r Range) []*Project {
	var out []*Project
	for _, p := range d.Projects {
		if len(d.VisibleInitiativesOf(p)) == 0 && d.ProjectVisible(p, r) {
			out = append(out, p)
		}
	}
	return out
}

// ProjectsWithoutInitiative lists projects active in r that are in no [W]/[WS]
// initiative, tagged or not: tagged projects first, then by effort.
func (d *Dataset) ProjectsWithoutInitiative(r Range) *pb.ListProjectsWithoutInitiativeResponse {
	res := &pb.ListProjectsWithoutInitiativeResponse{}
	for _, p := range d.projectsWithoutInitiative(r) {
		row := d.projectRow(p, r, "")
		row.HiddenInitiatives = d.hiddenInitiativesOf(p)
		res.Projects = append(res.Projects, row)
	}
	sort.SliceStable(res.Projects, func(i, j int) bool {
		a, b := res.Projects[i], res.Projects[j]
		if (len(a.Segments) > 0) != (len(b.Segments) > 0) {
			return len(a.Segments) > 0
		}
		if a.Pts != b.Pts {
			return a.Pts > b.Pts
		}
		return a.Name < b.Name
	})
	return res
}

func (d *Dataset) initiativesWithoutProjects() []*Initiative {
	var out []*Initiative
	for _, ini := range d.Initiatives {
		if len(ini.ProjectIDs) == 0 {
			out = append(out, ini)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Visible != out[j].Visible {
			return out[i].Visible
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// InitiativesWithoutProjects lists initiatives with no linked project, [W]/[WS] first.
// It ignores the date range: an initiative with no projects has no tasks to date.
func (d *Dataset) InitiativesWithoutProjects() *pb.ListInitiativesWithoutProjectsResponse {
	res := &pb.ListInitiativesWithoutProjectsResponse{}
	for _, ini := range d.initiativesWithoutProjects() {
		res.Initiatives = append(res.Initiatives, &pb.EmptyInitiative{Id: ini.ID, Name: ini.Name, Url: ini.URL,
			Status: ini.Status, Visible: ini.Visible, CreatedAt: ini.CreatedAt.Format("2006-01-02")})
	}
	return res
}
