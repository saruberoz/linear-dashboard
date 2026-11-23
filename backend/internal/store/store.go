// Package store persists the workspace copy and sync history in CockroachDB.
package store

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/cockroachdb" // registers cockroachdb://
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"linear-dashboard/backend/internal/report"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Store reads and writes the workspace copy.
type Store struct {
	pool *pgxpool.Pool
}

// Open connects to the database, retrying while it starts up, and applies migrations.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}
	cfg.MaxConns = 8
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 10 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}
	for attempt := 1; ; attempt++ {
		err = pool.Ping(ctx)
		if err == nil {
			break
		}
		if attempt == 30 {
			pool.Close()
			return nil, fmt.Errorf("database not reachable after %d attempts: %w", attempt, err)
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	if err := migrateUp(databaseURL); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func migrateUp(databaseURL string) error {
	src, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("loading migrations: %w", err)
	}
	url := "cockroachdb://" + strings.TrimPrefix(strings.TrimPrefix(databaseURL, "postgresql://"), "postgres://")
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return fmt.Errorf("opening migrator: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

// Replace swaps the stored workspace for ds in one transaction: readers see
// either the previous copy or the new one, never a mix.
func (s *Store) Replace(ctx context.Context, ds *report.Dataset) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for _, table := range []string{"issue_labels", "issues", "initiative_projects", "project_labels", "project_teams",
			"projects", "initiatives", "labels", "users", "teams"} {
			if _, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE true"); err != nil {
				return fmt.Errorf("clearing %s: %w", table, err)
			}
		}
		for _, c := range copies(ds) {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{c.table}, c.columns, pgx.CopyFromRows(c.rows)); err != nil {
				return fmt.Errorf("writing %s: %w", c.table, err)
			}
		}
		return nil
	})
}

type tableCopy struct {
	table   string
	columns []string
	rows    [][]any
}

func copies(ds *report.Dataset) []tableCopy {
	teams := tableCopy{table: "teams", columns: []string{"id", "key", "name", "estimation_type"}}
	for _, t := range ds.Teams {
		teams.rows = append(teams.rows, []any{t.ID, t.Key, t.Name, t.EstimationType})
	}
	users := tableCopy{table: "users", columns: []string{"id", "name", "display_name", "active"}}
	for _, u := range ds.Users {
		users.rows = append(users.rows, []any{u.ID, u.Name, u.DisplayName, u.Active})
	}
	labels := tableCopy{table: "labels", columns: []string{"id", "kind", "name", "group_name"}}
	for _, l := range ds.Labels {
		labels.rows = append(labels.rows, []any{l.ID, l.Kind, l.Name, nullString(l.Group)})
	}
	inis := tableCopy{table: "initiatives", columns: []string{"id", "name", "status", "url", "created_at", "is_visible", "target_date"}}
	for _, i := range ds.Initiatives {
		inis.rows = append(inis.rows, []any{i.ID, i.Name, i.Status, i.URL, i.CreatedAt, i.Visible, nullString(i.TargetDate)})
	}
	projects := tableCopy{table: "projects", columns: []string{"id", "name", "state", "url", "created_at", "segments", "start_date", "target_date"}}
	plabels := tableCopy{table: "project_labels", columns: []string{"project_id", "label_id"}}
	links := tableCopy{table: "initiative_projects", columns: []string{"initiative_id", "project_id"}}
	pteams := tableCopy{table: "project_teams", columns: []string{"project_id", "team_id"}}
	for _, p := range ds.Projects {
		projects.rows = append(projects.rows, []any{p.ID, p.Name, p.State, p.URL, p.CreatedAt, nonNil(p.Segments),
			nullString(p.StartDate), nullString(p.TargetDate)})
		for _, l := range unique(p.LabelIDs) {
			plabels.rows = append(plabels.rows, []any{p.ID, l})
		}
		for _, i := range unique(p.InitiativeIDs) {
			links.rows = append(links.rows, []any{i, p.ID})
		}
		for _, t := range unique(p.TeamIDs) {
			pteams.rows = append(pteams.rows, []any{p.ID, t})
		}
	}
	issues := tableCopy{table: "issues", columns: []string{"id", "identifier", "title", "url", "team_id", "project_id",
		"parent_id", "assignee_id", "estimate", "priority", "state_name", "state_type", "created_at", "started_at",
		"completed_at", "canceled_at", "archived_at", "effective_pts", "has_estimated_descendant",
		"is_missing_estimate", "task_segments", "started_triage_at", "triaged_at"}}
	ilabels := tableCopy{table: "issue_labels", columns: []string{"issue_id", "label_id"}}
	for i := range ds.Tasks {
		t := &ds.Tasks[i]
		issues.rows = append(issues.rows, []any{t.ID, t.Identifier, t.Title, t.URL, t.TeamID,
			nullString(t.ProjectID), nullString(t.ParentID), nullString(t.AssigneeID), t.Estimate, int16(t.Priority),
			t.StateName, t.StateType, t.CreatedAt, t.StartedAt, t.CompletedAt, t.CanceledAt, t.ArchivedAt,
			t.EffectivePts, t.HasEstimatedDescendant, t.MissingEstimate, nonNil(t.TaskSegments),
			t.StartedTriageAt, t.TriagedAt})
		for _, l := range unique(t.LabelIDs) {
			ilabels.rows = append(ilabels.rows, []any{t.ID, l})
		}
	}
	return []tableCopy{teams, users, labels, inis, projects, plabels, links, pteams, issues, ilabels}
}

// Load reads the stored workspace. It returns nil when nothing has been synced yet.
func (s *Store) Load(ctx context.Context) (*report.Dataset, error) {
	syncedAt, ok, err := s.lastSuccess(ctx)
	if err != nil || !ok {
		return nil, err
	}
	var (
		teams    []report.Team
		users    []report.User
		labels   []report.Label
		inis     []report.Initiative
		projects []report.Project
		tasks    []report.Task
	)
	if err := collect(ctx, s.pool, "SELECT id, key, name, estimation_type FROM teams", func(r pgx.Rows) error {
		var t report.Team
		err := r.Scan(&t.ID, &t.Key, &t.Name, &t.EstimationType)
		teams = append(teams, t)
		return err
	}); err != nil {
		return nil, err
	}
	if err := collect(ctx, s.pool, "SELECT id, name, display_name, active FROM users", func(r pgx.Rows) error {
		var u report.User
		err := r.Scan(&u.ID, &u.Name, &u.DisplayName, &u.Active)
		users = append(users, u)
		return err
	}); err != nil {
		return nil, err
	}
	if err := collect(ctx, s.pool, "SELECT id, kind, name, COALESCE(group_name, '') FROM labels", func(r pgx.Rows) error {
		var l report.Label
		err := r.Scan(&l.ID, &l.Kind, &l.Name, &l.Group)
		labels = append(labels, l)
		return err
	}); err != nil {
		return nil, err
	}
	if err := collect(ctx, s.pool, "SELECT id, name, status, url, created_at, COALESCE(target_date, '') FROM initiatives", func(r pgx.Rows) error {
		var i report.Initiative
		err := r.Scan(&i.ID, &i.Name, &i.Status, &i.URL, &i.CreatedAt, &i.TargetDate)
		inis = append(inis, i)
		return err
	}); err != nil {
		return nil, err
	}
	const projectsSQL = `SELECT p.id, p.name, p.state, p.url, p.created_at, COALESCE(p.start_date, ''), COALESCE(p.target_date, ''),
	  COALESCE((SELECT array_agg(label_id) FROM project_labels WHERE project_id = p.id), ARRAY[]::STRING[]),
	  COALESCE((SELECT array_agg(initiative_id) FROM initiative_projects WHERE project_id = p.id), ARRAY[]::STRING[]),
	  COALESCE((SELECT array_agg(team_id) FROM project_teams WHERE project_id = p.id), ARRAY[]::STRING[])
	FROM projects p`
	if err := collect(ctx, s.pool, projectsSQL, func(r pgx.Rows) error {
		var p report.Project
		err := r.Scan(&p.ID, &p.Name, &p.State, &p.URL, &p.CreatedAt, &p.StartDate, &p.TargetDate, &p.LabelIDs, &p.InitiativeIDs, &p.TeamIDs)
		projects = append(projects, p)
		return err
	}); err != nil {
		return nil, err
	}
	issueLabels := map[string][]string{}
	if err := collect(ctx, s.pool, "SELECT issue_id, label_id FROM issue_labels", func(r pgx.Rows) error {
		var issue, label string
		err := r.Scan(&issue, &label)
		issueLabels[issue] = append(issueLabels[issue], label)
		return err
	}); err != nil {
		return nil, err
	}
	const issuesSQL = `SELECT id, identifier, title, url, team_id, COALESCE(project_id, ''), COALESCE(parent_id, ''),
	  COALESCE(assignee_id, ''), estimate, priority, state_name, state_type, created_at, started_at, completed_at,
	  canceled_at, archived_at, effective_pts, has_estimated_descendant, is_missing_estimate, task_segments,
	  started_triage_at, triaged_at FROM issues`
	if err := collect(ctx, s.pool, issuesSQL, func(r pgx.Rows) error {
		var t report.Task
		var prio int16
		err := r.Scan(&t.ID, &t.Identifier, &t.Title, &t.URL, &t.TeamID, &t.ProjectID, &t.ParentID, &t.AssigneeID,
			&t.Estimate, &prio, &t.StateName, &t.StateType, &t.CreatedAt, &t.StartedAt, &t.CompletedAt,
			&t.CanceledAt, &t.ArchivedAt, &t.EffectivePts, &t.HasEstimatedDescendant, &t.MissingEstimate, &t.TaskSegments,
			&t.StartedTriageAt, &t.TriagedAt)
		t.Priority = int(prio)
		t.LabelIDs = issueLabels[t.ID]
		tasks = append(tasks, t)
		return err
	}); err != nil {
		return nil, err
	}
	return report.NewDataset(syncedAt, teams, users, labels, inis, projects, tasks), nil
}

func collect(ctx context.Context, pool *pgxpool.Pool, sql string, scan func(pgx.Rows) error) error {
	rows, err := pool.Query(ctx, sql)
	if err != nil {
		return fmt.Errorf("querying %q: %w", firstWords(sql), err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return fmt.Errorf("scanning %q: %w", firstWords(sql), err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reading %q: %w", firstWords(sql), err)
	}
	return nil
}

// ---------- sync runs ----------

// Run is one row of sync history.
type Run struct {
	ID         int64
	Trigger    string
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
	Error      string
}

// StartRun records a new running sync.
func (s *Store) StartRun(ctx context.Context, trigger string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO sync_runs (trigger, status) VALUES ($1, 'running') RETURNING id`, trigger).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("recording sync start: %w", err)
	}
	return id, nil
}

// FinishRun records the outcome of a sync.
func (s *Store) FinishRun(ctx context.Context, id int64, status string, stats map[string]int, errText string) error {
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("encoding sync stats: %w", err)
	}
	_, err = s.pool.Exec(ctx, `UPDATE sync_runs SET status = $2, finished_at = now(), stats = $3, error = $4 WHERE id = $1`,
		id, status, statsJSON, nullString(errText))
	if err != nil {
		return fmt.Errorf("recording sync finish: %w", err)
	}
	return nil
}

// FailAbandonedRuns marks runs left "running" by a previous process as failed.
// Only one backend process runs syncs, so at startup none can really be running.
func (s *Store) FailAbandonedRuns(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE sync_runs SET status = 'failed', finished_at = now(),
	  error = 'abandoned: the backend stopped during this sync' WHERE status = 'running'`)
	if err != nil {
		return 0, fmt.Errorf("clearing abandoned syncs: %w", err)
	}
	return tag.RowsAffected(), nil
}

// LastManualStart returns when the last manual sync started, if any.
func (s *Store) LastManualStart(ctx context.Context) (time.Time, bool, error) {
	var t *time.Time
	if err := s.pool.QueryRow(ctx, `SELECT max(started_at) FROM sync_runs WHERE trigger = 'manual'`).Scan(&t); err != nil {
		return time.Time{}, false, fmt.Errorf("reading last manual sync: %w", err)
	}
	if t == nil {
		return time.Time{}, false, nil
	}
	return *t, true, nil
}

// LastRun returns the most recent sync, if any.
func (s *Store) LastRun(ctx context.Context) (*Run, error) {
	var r Run
	var errText *string
	err := s.pool.QueryRow(ctx, `SELECT id, trigger, status, started_at, finished_at, error FROM sync_runs
	  ORDER BY started_at DESC LIMIT 1`).Scan(&r.ID, &r.Trigger, &r.Status, &r.StartedAt, &r.FinishedAt, &errText)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading last sync: %w", err)
	}
	if errText != nil {
		r.Error = *errText
	}
	return &r, nil
}

// LastSuccess returns when the last successful sync finished.
func (s *Store) LastSuccess(ctx context.Context) (time.Time, bool, error) {
	return s.lastSuccess(ctx)
}

func (s *Store) lastSuccess(ctx context.Context) (time.Time, bool, error) {
	var t *time.Time
	if err := s.pool.QueryRow(ctx, `SELECT max(finished_at) FROM sync_runs WHERE status = 'succeeded'`).Scan(&t); err != nil {
		return time.Time{}, false, fmt.Errorf("reading last successful sync: %w", err)
	}
	if t == nil {
		return time.Time{}, false, nil
	}
	return *t, true, nil
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

func unique(xs []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func firstWords(sql string) string {
	f := strings.Fields(sql)
	if len(f) > 6 {
		f = f[:6]
	}
	return strings.Join(f, " ")
}
