// Package snapshot copies the Linear workspace into the store: fetch everything,
// derive the counting fields, then replace the stored copy in one transaction.
package snapshot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"linear-dashboard/backend/internal/linear"
	"linear-dashboard/backend/internal/report"
	"linear-dashboard/backend/internal/store"
)

// Sync triggers.
const (
	TriggerStartup  = "startup"
	TriggerSchedule = "schedule"
	TriggerManual   = "manual"
)

// Errors returned by Start before any work begins.
var (
	ErrRunning  = errors.New("snapshot: a sync is already running")
	ErrCooldown = errors.New("snapshot: manual sync cooldown")
)

// Fetcher downloads the workspace.
type Fetcher interface {
	FetchWorkspace(ctx context.Context, progress func(int)) (*linear.Workspace, error)
}

// Runner runs one sync at a time. Only one backend process runs syncs, so an
// in-process lock is enough; abandoned "running" rows are cleared at startup.
type Runner struct {
	fetch    Fetcher
	store    *store.Store
	cooldown time.Duration
	timeout  time.Duration
	logger   *slog.Logger
	current  atomic.Pointer[report.Dataset]

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

// NewRunner returns a runner. cooldown applies to manual syncs only.
func NewRunner(f Fetcher, s *store.Store, cooldown time.Duration, logger *slog.Logger) *Runner {
	return &Runner{fetch: f, store: s, cooldown: cooldown, timeout: 30 * time.Minute, logger: logger}
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

// Start checks the lock and, for manual syncs, the cooldown, records the run and
// returns its id; the sync itself continues in the background under base, which
// is canceled only when the backend shuts down. Failed runs count toward the cooldown.
func (r *Runner) Start(ctx, base context.Context, trigger string) (int64, error) {
	if !r.mu.TryLock() {
		return 0, ErrRunning
	}
	if trigger == TriggerManual {
		last, ok, err := r.store.LastManualStart(ctx)
		if err != nil {
			r.mu.Unlock()
			return 0, err
		}
		if ok && time.Since(last) < r.cooldown {
			r.mu.Unlock()
			return 0, fmt.Errorf("%w: next manual sync allowed at %s", ErrCooldown, last.Add(r.cooldown).Format(time.RFC3339))
		}
	}
	id, err := r.store.StartRun(ctx, trigger)
	if err != nil {
		r.mu.Unlock()
		return 0, err
	}
	r.fetched.Store(0)
	r.expected.Store(size(r.Dataset()))
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
	stats, err := r.run(ctx)
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

// run fetches, derives and writes. Nothing is written unless the whole fetch succeeded.
func (r *Runner) run(ctx context.Context) (map[string]int, error) {
	ws, err := r.fetch.FetchWorkspace(ctx, func(n int) { r.fetched.Add(int64(n)) })
	if err != nil {
		return nil, err
	}
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
		"trashed_skipped": ws.Trashed,
	}
	if err := r.store.Replace(ctx, ds); err != nil {
		return stats, err
	}
	r.current.Store(ds)
	return stats, nil
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
