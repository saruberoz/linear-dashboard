package report

import (
	"sort"
	"time"
)

// Dataset is one complete, derived copy of the workspace. It is immutable after
// NewDataset returns; a sync builds a new one and swaps it in.
type Dataset struct {
	SyncedAt    time.Time
	Teams       map[string]Team
	Users       map[string]User
	Labels      map[string]Label
	Initiatives map[string]*Initiative
	Projects    map[string]*Project
	Tasks       []Task

	tasksByProject map[string][]*Task // "" key holds tasks without a project
	children       map[string][]*Task
	taskByID       map[string]*Task
}

// NewDataset indexes the data and fills derived project fields. Task-derived
// fields must already be set (by Derive during sync, or loaded from the store).
func NewDataset(syncedAt time.Time, teams []Team, users []User, labels []Label,
	initiatives []Initiative, projects []Project, tasks []Task) *Dataset {
	d := &Dataset{
		SyncedAt:    syncedAt,
		Teams:       map[string]Team{},
		Users:       map[string]User{},
		Labels:      map[string]Label{},
		Initiatives: map[string]*Initiative{},
		Projects:    map[string]*Project{},
		Tasks:       tasks,
	}
	for _, t := range teams {
		d.Teams[t.ID] = t
	}
	for _, u := range users {
		d.Users[u.ID] = u
	}
	for _, l := range labels {
		d.Labels[l.ID] = l
	}
	for i := range initiatives {
		ini := initiatives[i]
		ini.Visible = IsVisibleInitiative(ini.Name)
		ini.ProjectIDs = nil
		d.Initiatives[ini.ID] = &ini
	}
	for i := range projects {
		p := projects[i]
		names := make([]string, 0, len(p.LabelIDs))
		for _, id := range p.LabelIDs {
			names = append(names, d.Labels[id].Name)
		}
		p.Segments = SegmentsOf(names)
		d.Projects[p.ID] = &p
		for _, iid := range p.InitiativeIDs {
			if ini := d.Initiatives[iid]; ini != nil {
				ini.ProjectIDs = append(ini.ProjectIDs, p.ID)
			}
		}
	}
	d.indexTasks()
	return d
}

func (d *Dataset) indexTasks() {
	d.tasksByProject = map[string][]*Task{}
	d.children = map[string][]*Task{}
	d.taskByID = map[string]*Task{}
	for i := range d.Tasks {
		t := &d.Tasks[i]
		d.taskByID[t.ID] = t
		d.tasksByProject[t.ProjectID] = append(d.tasksByProject[t.ProjectID], t)
		if t.ParentID != "" {
			d.children[t.ParentID] = append(d.children[t.ParentID], t)
		}
	}
}

// ForTeams narrows the dataset to the given teams: their tasks, and the projects
// that belong to one of the teams or hold one of their tasks. Initiatives keep
// their full project lists, so initiative housekeeping stays workspace-wide.
// Task-derived fields keep their full-tree values: a parent whose sub-tasks sit
// in another team still defers its estimate to them.
func (d *Dataset) ForTeams(teamIDs map[string]bool) *Dataset {
	out := &Dataset{
		SyncedAt:    d.SyncedAt,
		Teams:       d.Teams,
		Users:       d.Users,
		Labels:      d.Labels,
		Initiatives: d.Initiatives,
		Projects:    map[string]*Project{},
	}
	for i := range d.Tasks {
		t := &d.Tasks[i]
		if !teamIDs[t.TeamID] {
			continue
		}
		out.Tasks = append(out.Tasks, *t)
		if p := d.Projects[t.ProjectID]; p != nil {
			out.Projects[p.ID] = p
		}
	}
	for id, p := range d.Projects {
		for _, tid := range p.TeamIDs {
			if teamIDs[tid] {
				out.Projects[id] = p
				break
			}
		}
	}
	out.indexTasks()
	return out
}

// TasksInRange returns the project's tasks that count in r.
func (d *Dataset) TasksInRange(projectID string, r Range) []*Task {
	var out []*Task
	for _, t := range d.tasksByProject[projectID] {
		if r.InRange(t) {
			out = append(out, t)
		}
	}
	return out
}

// ProjectVisible reports whether a project shows up for r: it was created in r
// or has at least one task that counts in r.
func (d *Dataset) ProjectVisible(p *Project, r Range) bool {
	if r.contains(&p.CreatedAt) {
		return true
	}
	for _, t := range d.tasksByProject[p.ID] {
		if r.InRange(t) {
			return true
		}
	}
	return false
}

// TaggedProjects returns the visible projects that carry at least one segment,
// optionally limited to one segment, sorted by name.
func (d *Dataset) TaggedProjects(r Range, segment string) []*Project {
	var out []*Project
	for _, p := range d.Projects {
		if len(p.Segments) == 0 || (segment != "" && !contains(p.Segments, segment)) {
			continue
		}
		if d.ProjectVisible(p, r) {
			out = append(out, p)
		}
	}
	sortProjects(out)
	return out
}

// VisibleInitiativesOf returns the project's initiatives that are shown, sorted by name.
func (d *Dataset) VisibleInitiativesOf(p *Project) []*Initiative {
	var out []*Initiative
	for _, id := range p.InitiativeIDs {
		if ini := d.Initiatives[id]; ini != nil && ini.Visible {
			out = append(out, ini)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// AssigneeName is the display name shown for a task's assignee.
func (d *Dataset) AssigneeName(id string) string {
	if id == "" {
		return ""
	}
	u, ok := d.Users[id]
	if !ok {
		return ""
	}
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Name
}

// LabelNames returns a task's labels as "Group / Name", in a stable order.
func (d *Dataset) LabelNames(t *Task) []string {
	out := make([]string, 0, len(t.LabelIDs))
	for _, id := range t.LabelIDs {
		if l, ok := d.Labels[id]; ok {
			out = append(out, l.FullName())
		}
	}
	sort.Strings(out)
	return out
}

func sortProjects(ps []*Project) {
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
