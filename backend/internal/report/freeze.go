package report

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Kinds of change Linear can make to frozen (completed) work.
const (
	ChangeEstimate   = "estimate"   // a task's estimate
	ChangeProject    = "project"    // the project a task belongs to
	ChangeInitiative = "initiative" // a project's link to an initiative
)

// Linked is the value of an initiative change side where the project is linked.
const Linked = "linked"

// FrozenChange is a change Linear made to completed work. The dashboard keeps
// the old value until someone approves the change.
type FrozenChange struct {
	Kind         string
	TaskID       string // estimate and project changes
	ProjectID    string // initiative changes
	InitiativeID string // initiative changes
	// Old and New are an estimate ("" for none), a project ID ("" for none), or
	// for an initiative change Linked or "".
	Old, New string
	// FrozenBy names the completed work that froze it: "task", "project:<id>"
	// or "initiative:<id>".
	FrozenBy string
}

// Key identifies what changed, so a later sync updates the same pending change
// instead of adding another.
func (c FrozenChange) Key() string {
	return c.Kind + ":" + c.TaskID + ":" + c.ProjectID + ":" + c.InitiativeID
}

// FormatEstimate writes an estimate the way FrozenChange stores it.
func FormatEstimate(e *float64) string {
	if e == nil {
		return ""
	}
	return strconv.FormatFloat(*e, 'f', -1, 64)
}

// ParseEstimate reads an estimate written by FormatEstimate.
func ParseEstimate(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Freeze compares freshly fetched tasks and projects with the approved copy
// prev. Every change to frozen work is undone in place, so the fetched data
// keeps prev's value, and is returned. Frozen work is:
//
//   - a completed task: its estimate;
//   - a completed project: which tasks are in it, and the estimates of its tasks;
//   - a completed visible initiative: which projects are linked to it.
//
// A first estimate on a task that had none is not held: it fills a gap
// rather than changing what was agreed. Completion is read from prev, so work
// completed in this sync is frozen from the next one. Ignored tasks are never
// frozen. With no prev (the first sync) nothing is frozen.
func Freeze(prev *Dataset, tasks []Task, projects []Project) []FrozenChange {
	if prev == nil {
		return nil
	}
	completedProject := func(id string) bool {
		p := prev.Projects[id]
		return p != nil && p.State == "completed"
	}
	var out []FrozenChange
	for i := range tasks {
		t := &tasks[i]
		if IsIgnoredTask(t.Title) {
			continue
		}
		p := prev.taskByID[t.ID]
		oldProject := ""
		if p != nil {
			oldProject = p.ProjectID
		}
		if t.ProjectID != oldProject {
			by := ""
			if completedProject(oldProject) {
				by = "project:" + oldProject
			} else if completedProject(t.ProjectID) {
				by = "project:" + t.ProjectID
			}
			if by != "" {
				out = append(out, FrozenChange{Kind: ChangeProject, TaskID: t.ID, Old: oldProject, New: t.ProjectID, FrozenBy: by})
				t.ProjectID = oldProject
			}
		}
		if p != nil && p.Estimate != nil && FormatEstimate(p.Estimate) != FormatEstimate(t.Estimate) {
			by := ""
			if p.StateType == "completed" {
				by = "task"
			} else if completedProject(p.ProjectID) {
				by = "project:" + p.ProjectID
			}
			if by != "" {
				out = append(out, FrozenChange{Kind: ChangeEstimate, TaskID: t.ID,
					Old: FormatEstimate(p.Estimate), New: FormatEstimate(t.Estimate), FrozenBy: by})
				t.Estimate = p.Estimate
			}
		}
	}
	for i := range projects {
		pr := &projects[i]
		var old []string
		if pp := prev.Projects[pr.ID]; pp != nil {
			old = pp.InitiativeIDs
		}
		links := slices.Clone(pr.InitiativeIDs)
		for _, id := range symmetricDifference(old, pr.InitiativeIDs) {
			ini := prev.Initiatives[id]
			if ini == nil || !ini.Visible || !strings.EqualFold(ini.Status, "completed") {
				continue
			}
			c := FrozenChange{Kind: ChangeInitiative, ProjectID: pr.ID, InitiativeID: id, FrozenBy: "initiative:" + id}
			if slices.Contains(old, id) {
				c.Old = Linked
				links = append(links, id)
			} else {
				c.New = Linked
				links = slices.DeleteFunc(links, func(x string) bool { return x == id })
			}
			out = append(out, c)
		}
		pr.InitiativeIDs = links
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

func symmetricDifference(a, b []string) []string {
	var out []string
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	for _, x := range b {
		if !slices.Contains(a, x) {
			out = append(out, x)
		}
	}
	return out
}

// ErrChangeTarget is returned when an approved change names a task or project
// that is no longer in the dataset.
var ErrChangeTarget = errors.New("report: the changed task or project is no longer in the dashboard")

// WithChange returns a copy of d with an approved change applied and every
// derived field recomputed. d itself is left untouched.
func (d *Dataset) WithChange(c FrozenChange) (*Dataset, error) {
	tasks := slices.Clone(d.Tasks)
	projects := make([]Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		p := *p
		p.InitiativeIDs = slices.Clone(p.InitiativeIDs)
		projects = append(projects, p)
	}
	switch c.Kind {
	case ChangeEstimate, ChangeProject:
		i := slices.IndexFunc(tasks, func(t Task) bool { return t.ID == c.TaskID })
		if i < 0 {
			return nil, ErrChangeTarget
		}
		if c.Kind == ChangeProject {
			tasks[i].ProjectID = c.New
			break
		}
		e, err := ParseEstimate(c.New)
		if err != nil {
			return nil, err
		}
		tasks[i].Estimate = e
	case ChangeInitiative:
		i := slices.IndexFunc(projects, func(p Project) bool { return p.ID == c.ProjectID })
		if i < 0 {
			return nil, ErrChangeTarget
		}
		ids := slices.DeleteFunc(projects[i].InitiativeIDs, func(id string) bool { return id == c.InitiativeID })
		if c.New == Linked {
			ids = append(ids, c.InitiativeID)
		}
		projects[i].InitiativeIDs = ids
	default:
		return nil, fmt.Errorf("report: unknown change kind %q", c.Kind)
	}
	Derive(tasks, d.Teams, d.Labels)
	return NewDataset(d.SyncedAt, mapValues(d.Teams), mapValues(d.Users), mapValues(d.Labels),
		derefValues(d.Initiatives), projects, tasks), nil
}

func mapValues[T any](m map[string]T) []T {
	out := make([]T, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func derefValues[T any](m map[string]*T) []T {
	out := make([]T, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	return out
}

// ChangeText describes a frozen change for people.
type ChangeText struct {
	What     string // what changed, e.g. "Estimate" or "Linked to [W] Payments"
	Old, New string // the values, e.g. "3 pts" and "5 pts"
	FrozenBy string // why it is frozen, e.g. "project Checkout was completed"
}

// Describe names a change's values and its frozen work, using names from d.
func (d *Dataset) Describe(c FrozenChange) ChangeText {
	out := ChangeText{FrozenBy: d.frozenBy(c.FrozenBy)}
	switch c.Kind {
	case ChangeEstimate:
		out.What = "Estimate"
		out.Old, out.New = estimateText(c.Old), estimateText(c.New)
	case ChangeProject:
		out.What = "Project"
		out.Old, out.New = d.projectText(c.Old), d.projectText(c.New)
	case ChangeInitiative:
		out.What = "Initiative " + d.initiativeName(c.InitiativeID)
		out.Old, out.New = linkText(c.Old), linkText(c.New)
	}
	return out
}

// RevertNote is the comment written in Linear when a change is rejected and
// undone there. reason, when given, is added as the rejecter's words.
func (d *Dataset) RevertNote(c FrozenChange, reason string) string {
	t := d.Describe(c)
	var what string
	switch c.Kind {
	case ChangeEstimate:
		what = fmt.Sprintf("The estimate was changed from %s to %s after %s. The estimate is set back to %s.",
			t.Old, t.New, t.FrozenBy, t.Old)
	case ChangeProject:
		what = fmt.Sprintf("This task was moved from %s to %s after %s. It is moved back to %s.",
			t.Old, t.New, t.FrozenBy, t.Old)
	case ChangeInitiative:
		name := d.initiativeName(c.InitiativeID)
		if c.New == Linked {
			what = fmt.Sprintf("This project was linked to initiative %s after %s. The link is removed.", name, t.FrozenBy)
		} else {
			what = fmt.Sprintf("This project was unlinked from initiative %s after %s. The link is restored.", name, t.FrozenBy)
		}
	}
	note := "**Change reverted by the Linear Dashboard.**\n\n" + what +
		"\n\nCompleted work is frozen so that reported effort does not change afterwards, and this change was rejected."
	if r := strings.TrimSpace(reason); r != "" {
		note += "\n\nReason: " + r
	}
	return note
}

func (d *Dataset) frozenBy(by string) string {
	kind, id, _ := strings.Cut(by, ":")
	switch kind {
	case "project":
		return "project " + d.projectText(id) + " was completed"
	case "initiative":
		return "initiative " + d.initiativeName(id) + " was completed"
	default:
		return "the task was completed"
	}
}

func estimateText(s string) string {
	if s == "" {
		return "no estimate"
	}
	if s == "1" {
		return "1 pt"
	}
	return s + " pts"
}

func (d *Dataset) projectText(id string) string {
	if id == "" {
		return "no project"
	}
	if p := d.Projects[id]; p != nil {
		return p.Name
	}
	return "a project no longer in the dashboard"
}

func (d *Dataset) initiativeName(id string) string {
	if ini := d.Initiatives[id]; ini != nil {
		return ini.Name
	}
	return "(no longer in the dashboard)"
}

func linkText(s string) string {
	if s == Linked {
		return "linked"
	}
	return "not linked"
}

// Task returns the task with this ID, or nil when it is not in the dataset.
func (d *Dataset) Task(id string) *Task { return d.taskByID[id] }
