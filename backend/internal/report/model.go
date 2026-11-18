// Package report holds the dashboard's counting rules. It is pure: no database,
// no network. A Dataset is an in-memory copy of the Linear workspace; views are
// computed from it on every request.
package report

import "time"

// Status buckets for a task, from Linear's state type.
const (
	StatusDone       = "done"
	StatusInProgress = "in_progress"
	StatusNotStarted = "not_started"
	StatusCanceled   = "canceled"
)

// Segment is one of the business segments, in display order.
type Segment struct {
	Key     string
	Label   string
	aliases []string // normalized label spellings that mean this segment
}

// Segments lists the segments in canonical display order.
var Segments = []Segment{
	{Key: "B2B", Label: "B2B", aliases: []string{"b2b"}},
	{Key: "B2G", Label: "B2G", aliases: []string{"b2g"}},
	{Key: "B2BNG", Label: "B2B NG", aliases: []string{"b2bng", "b2bnongroup"}},
	{Key: "B2C", Label: "B2C", aliases: []string{"b2c"}},
	{Key: "INTERNAL", Label: "Internal PARKEE", aliases: []string{"internalparkee"}},
}

// Team is a Linear team; only its estimate setting matters here.
type Team struct {
	ID             string
	Key            string
	Name           string
	EstimationType string // "notUsed" means estimates are turned off
}

// User is a Linear user; DisplayName is shown in contributor tables.
type User struct {
	ID          string
	Name        string
	DisplayName string
	Active      bool
}

// Label is a task or project label; Group is its label group name, if any.
type Label struct {
	ID    string
	Kind  string // "issue" or "project"
	Name  string
	Group string
}

// FullName is "Group / Name", or just Name when the label has no group.
func (l Label) FullName() string {
	if l.Group == "" {
		return l.Name
	}
	return l.Group + " / " + l.Name
}

// Initiative is a Linear initiative.
type Initiative struct {
	ID         string
	Name       string
	Status     string
	URL        string
	CreatedAt  time.Time
	Visible    bool     // name starts with [W] or [WS]
	ProjectIDs []string // all linked projects, tagged or not
}

// Project is a Linear project.
type Project struct {
	ID            string
	Name          string
	State         string
	URL           string
	CreatedAt     time.Time
	LabelIDs      []string
	Segments      []string // derived: segment keys in canonical order
	InitiativeIDs []string // all linked initiatives
}

// Task is a Linear issue.
type Task struct {
	ID          string
	Identifier  string
	Title       string
	URL         string
	TeamID      string
	ProjectID   string // "" when the task has no project
	ParentID    string
	AssigneeID  string
	Estimate    *float64
	Priority    int
	StateName   string
	StateType   string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	CanceledAt  *time.Time
	ArchivedAt  *time.Time
	// Triage window: when the task entered triage and when it left it.
	StartedTriageAt *time.Time
	TriagedAt       *time.Time
	LabelIDs        []string

	// Derived over the full task tree at sync time (see Derive).
	HasEstimatedDescendant bool
	EffectivePts           float64
	MissingEstimate        bool
	TaskSegments           []string
}

// Status returns the task's status bucket.
func (t *Task) Status() string { return Bucket(t.StateType) }

// Deferred reports whether the task's own estimate is ignored because an
// estimated, non-canceled sub-task exists below it.
func (t *Task) Deferred() bool { return t.HasEstimatedDescendant }
