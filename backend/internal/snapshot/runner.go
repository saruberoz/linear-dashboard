// Package snapshot copies the Linear workspace into the store: fetch it (all
// tasks, or only tasks changed since the last sync), derive the counting fields,
// then replace the stored copy in one transaction.
package snapshot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"linear-dashboard/backend/internal/linear"
	"linear-dashboard/backend/internal/report"
	"linear-dashboard/backend/internal/store"
)

// Sync triggers. A refresh fetches only tasks changed since the last
// successful sync; every other trigger fetches the whole workspace.
const (
	TriggerStartup  = "startup"
	TriggerSchedule = "schedule"
	TriggerManual   = "manual"
	TriggerRefresh  = "refresh"
)

// refreshOverlap is how far before the last successful sync a refresh looks,
// so tasks edited while that sync was fetching are not missed.
const refreshOverlap = 5 * time.Minute

// Errors returned by Start before any work begins.
var (
	ErrRunning  = errors.New("snapshot: a sync is already running")
	ErrCooldown = errors.New("snapshot: sync cooldown")
)

// Fetcher downloads the workspace.
type Fetcher interface {
	FetchWorkspace(ctx context.Context, since time.Time, progress func(int)) (*linear.Workspace, error)
}

// Writer undoes a rejected change in Linear and leaves a note there.
type Writer interface {
	SetEstimate(ctx context.Context, issueID string, estimate *float64) error
	SetProject(ctx context.Context, issueID, projectID string) error
	LinkProject(ctx context.Context, projectID, initiativeID string) error
	UnlinkProject(ctx context.Context, projectID, initiativeID string) error
	Comment(ctx context.Context, target linear.CommentTarget, body string) error
}

// ErrLinearWrite is returned when Linear refused to undo a rejected change;
// nothing was changed then.
var ErrLinearWrite = errors.New("snapshot: Linear refused to undo the change")

// ErrNoteFailed is returned when a rejected change was undone in Linear but
// the note explaining it could not be written.
var ErrNoteFailed = errors.New("snapshot: the change was undone in Linear, but the note could not be written")

// Runner runs one sync at a time. Only one backend process runs syncs, so an
// in-process lock is enough; abandoned "running" rows are cleared at startup.
type Runner struct {
	fetch     Fetcher
	write     Writer
	store     *store.Store
	cooldowns map[string]time.Duration // per trigger; triggers not listed have none
	timeout   time.Duration
	logger    *slog.Logger
	current   atomic.Pointer[report.Dataset]

	mu      sync.Mutex // held for the whole run
	running atomic.Bool
	wg      sync.WaitGroup

	// Progress of the running sync.
	fetched  atomic.Int64 // records downloaded so far
	expected atomic.Int64 // records in the current copy; 0 before the first sync
	saving   atomic.Bool  // download finished, writing to the store
}

// savingPercent is shown once the download is done; writing has no finer progress.
const savingPercent = 95

// NewRunner returns a runner. cooldowns holds the minimum time between two
// starts of the same trigger, for the triggers people press (manual, refresh).
func NewRunner(f Fetcher, w Writer, s *store.Store, cooldowns map[string]time.Duration, logger *slog.Logger) *Runner {
	return &Runner{fetch: f, write: w, store: s, cooldowns: cooldowns, timeout: 30 * time.Minute, logger: logger}
}

// Dataset returns the current workspace copy, or nil before the first successful sync.
func (r *Runner) Dataset() *report.Dataset { return r.current.Load() }

// SetDataset installs a dataset loaded from the store at startup.
func (r *Runner) SetDataset(d *report.Dataset) { r.current.Store(d) }

// Running reports whether a sync is in progress.
func (r *Runner) Running() bool { return r.running.Load() }

// Progress reports a running sync's phase ("fetching" or "saving") and percent
// done. While fetching, it compares records downloaded so far with the size of
// the current copy, so it is -1 on the first sync, when that size is unknown.
func (r *Runner) Progress() (phase string, percent int32) {
	if r.saving.Load() {
		return "saving", savingPercent
	}
	return "fetching", fetchPercent(r.fetched.Load(), r.expected.Load())
}

func fetchPercent(fetched, expected int64) int32 {
	if expected <= 0 {
		return -1
	}
	return int32(min(fetched*savingPercent/expected, savingPercent))
}

func size(d *report.Dataset) int64 {
	if d == nil {
		return 0
	}
	return int64(len(d.Teams) + len(d.Users) + len(d.Labels) + len(d.Initiatives) + len(d.Projects) + len(d.Tasks))
}

// Start checks the lock and the trigger's cooldown, records the run and returns
// its id; the sync itself continues in the background under base, which is
// canceled only when the backend shuts down. Failed runs count toward the cooldown.
func (r *Runner) Start(ctx, base context.Context, trigger string) (int64, error) {
	if !r.mu.TryLock() {
		return 0, ErrRunning
	}
	if cooldown := r.cooldowns[trigger]; cooldown > 0 {
		last, ok, err := r.store.LastStart(ctx, trigger)
		if err != nil {
			r.mu.Unlock()
			return 0, err
		}
		if ok && time.Since(last) < cooldown {
			r.mu.Unlock()
			return 0, fmt.Errorf("%w: next %s allowed at %s", ErrCooldown, trigger, last.Add(cooldown).Format(time.RFC3339))
		}
	}
	id, err := r.store.StartRun(ctx, trigger)
	if err != nil {
		r.mu.Unlock()
		return 0, err
	}
	r.fetched.Store(0)
	r.expected.Store(size(r.Dataset()))
	if trigger == TriggerRefresh && r.Dataset() != nil {
		r.expected.Store(size(r.Dataset()) - int64(len(r.Dataset().Tasks)))
	}
	r.saving.Store(false)
	r.running.Store(true)
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.mu.Unlock()
		defer r.running.Store(false)
		r.execute(base, id, trigger)
	}()
	return id, nil
}

// Wait blocks until a running sync has finished.
func (r *Runner) Wait() { r.wg.Wait() }

func (r *Runner) execute(base context.Context, id int64, trigger string) {
	ctx, cancel := context.WithTimeout(base, r.timeout)
	defer cancel()
	started := time.Now()
	stats, err := r.run(ctx, trigger)
	// Record the outcome even if base was canceled by shutdown.
	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(base), 10*time.Second)
	defer cancelFinish()
	if err != nil {
		msg := sanitize(err.Error())
		if ferr := r.store.FinishRun(finishCtx, id, "failed", stats, msg); ferr != nil {
			r.logger.Error("recording failed sync", "run", id, "err", ferr)
		}
		r.logger.Error("sync failed", "run", id, "trigger", trigger, "after", time.Since(started).Round(time.Second).String(), "err", msg)
		return
	}
	if ferr := r.store.FinishRun(finishCtx, id, "succeeded", stats, ""); ferr != nil {
		r.logger.Error("recording successful sync", "run", id, "err", ferr)
	}
	r.logger.Info("sync succeeded", "run", id, "trigger", trigger, "after", time.Since(started).Round(time.Second).String(), "stats", stats)
}

// run fetches, derives and writes. Nothing is written unless the whole fetch
// succeeded. A refresh with a stored copy fetches only the tasks changed since
// the last successful sync started and merges them into that copy.
func (r *Runner) run(ctx context.Context, trigger string) (map[string]int, error) {
	current := r.Dataset()
	var since time.Time
	if trigger == TriggerRefresh && current != nil {
		last, ok, err := r.store.LastSuccessStart(ctx)
		if err != nil {
			return nil, err
		}
		if ok {
			since = last.Add(-refreshOverlap)
		}
	}
	ws, err := r.fetch.FetchWorkspace(ctx, since, func(n int) { r.fetched.Add(int64(n)) })
	if err != nil {
		return nil, err
	}
	ws.Tasks = countable(ws.Tasks)
	changed := len(ws.Tasks)
	rechecked := func(report.FrozenChange) bool { return true }
	if !since.IsZero() {
		refetched := map[string]bool{}
		for _, t := range ws.Tasks {
			refetched[t.ID] = true
		}
		for _, id := range ws.TrashedIDs {
			refetched[id] = true
		}
		// A refresh only sees tasks Linear changed, so a pending task change is
		// rechecked only when its task came back; projects always come back.
		rechecked = func(c report.FrozenChange) bool { return c.Kind == report.ChangeInitiative || refetched[c.TaskID] }
		ws.Tasks = mergeTasks(current.Tasks, ws.Tasks, ws.TrashedIDs)
	}
	held := report.Freeze(current, ws.Tasks, ws.Projects)
	r.saving.Store(true)
	teams := map[string]report.Team{}
	for _, t := range ws.Teams {
		teams[t.ID] = t
	}
	labels := map[string]report.Label{}
	for _, l := range ws.Labels {
		labels[l.ID] = l
	}
	report.Derive(ws.Tasks, teams, labels)
	ds := report.NewDataset(time.Now(), ws.Teams, ws.Users, ws.Labels, ws.Initiatives, ws.Projects, ws.Tasks)
	stats := map[string]int{
		"teams": len(ws.Teams), "users": len(ws.Users), "labels": len(ws.Labels),
		"initiatives": len(ws.Initiatives), "projects": len(ws.Projects), "tasks": len(ws.Tasks),
		"trashed_skipped": len(ws.TrashedIDs),
	}
	if !since.IsZero() {
		stats["changed_tasks"] = changed
	}
	stats["frozen_held"] = len(held)
	if err := r.store.Replace(ctx, ds, store.FrozenUpdate{Held: held, Rechecked: rechecked}); err != nil {
		return stats, err
	}
	r.current.Store(ds)
	return stats, nil
}

// Approve accepts a pending change: the dashboard takes Linear's new value and
// recounts. It refuses while a sync runs.
func (r *Runner) Approve(ctx context.Context, id int64) error {
	if !r.mu.TryLock() {
		return ErrRunning
	}
	defer r.mu.Unlock()
	c, current, err := r.pending(ctx, id)
	if err != nil {
		return err
	}
	ds, err := current.WithChange(c.FrozenChange)
	if err != nil {
		return err
	}
	if err := r.store.ApproveFrozen(ctx, ds, id); err != nil {
		return err
	}
	r.current.Store(ds)
	return nil
}

// Reject undoes a pending change in Linear, marks it rejected and writes a note
// in Linear saying why. The dashboard already counts the old value. It refuses
// while a sync runs, so the sync cannot pick up the change halfway.
func (r *Runner) Reject(ctx context.Context, id int64, reason string) error {
	if !r.mu.TryLock() {
		return ErrRunning
	}
	defer r.mu.Unlock()
	c, current, err := r.pending(ctx, id)
	if err != nil {
		return err
	}
	target := linear.CommentTarget{IssueID: c.TaskID}
	switch c.Kind {
	case report.ChangeEstimate:
		var old *float64
		if old, err = report.ParseEstimate(c.Old); err == nil {
			err = r.write.SetEstimate(ctx, c.TaskID, old)
		}
	case report.ChangeProject:
		err = r.write.SetProject(ctx, c.TaskID, c.Old)
	case report.ChangeInitiative:
		target = linear.CommentTarget{ProjectID: c.ProjectID}
		if c.New == report.Linked {
			err = r.write.UnlinkProject(ctx, c.ProjectID, c.InitiativeID)
		} else {
			err = r.write.LinkProject(ctx, c.ProjectID, c.InitiativeID)
		}
	}
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLinearWrite, sanitize(err.Error()))
	}
	if err := r.store.RejectFrozen(ctx, id, reason); err != nil {
		return err
	}
	if err := r.write.Comment(ctx, target, current.RevertNote(c.FrozenChange, reason)); err != nil {
		r.logger.Error("writing revert note", "change", id, "err", sanitize(err.Error()))
		return ErrNoteFailed
	}
	return nil
}

// pending loads a change that is still pending, and the current copy.
func (r *Runner) pending(ctx context.Context, id int64) (*store.FrozenChangeRow, *report.Dataset, error) {
	current := r.Dataset()
	if current == nil {
		return nil, nil, errors.New("snapshot: no data yet")
	}
	c, err := r.store.FrozenChange(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if c.Status != "pending" {
		return nil, nil, store.ErrNotPending
	}
	return c, current, nil
}

// countable leaves out ignored tasks before anything is counted, so a full sync
// and a refresh (whose stored copy never holds them) walk the same task trees.
func countable(tasks []report.Task) []report.Task {
	return slices.DeleteFunc(tasks, func(t report.Task) bool { return report.IsIgnoredTask(t.Title) })
}

// mergeTasks returns the stored tasks with the changed ones replacing their
// stored version, new ones added, and trashed ones removed. Stored tasks are
// copied, so the live dataset is left untouched when the result is derived.
func mergeTasks(stored, changed []report.Task, trashedIDs []string) []report.Task {
	drop := make(map[string]bool, len(changed)+len(trashedIDs))
	for _, t := range changed {
		drop[t.ID] = true
	}
	for _, id := range trashedIDs {
		drop[id] = true
	}
	out := make([]report.Task, 0, len(stored)+len(changed))
	for _, t := range stored {
		if !drop[t.ID] {
			out = append(out, t)
		}
	}
	return append(out, changed...)
}

// sanitize keeps sync errors safe to show in the browser: no API keys.
func sanitize(msg string) string {
	for {
		i := strings.Index(msg, "lin_api_")
		if i < 0 {
			return msg
		}
		j := i + len("lin_api_")
		for j < len(msg) && isAlnum(msg[j]) {
			j++
		}
		msg = msg[:i] + "[redacted]" + msg[j:]
	}
}

func isAlnum(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
