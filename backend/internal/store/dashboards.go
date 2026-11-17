package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound is returned when a dashboard or insight does not exist.
var ErrNotFound = errors.New("store: not found")

// DashboardRow is a saved dashboard; Filters is the JSON of InsightFilters.
type DashboardRow struct {
	ID        string
	Name      string
	Filters   []byte
	CreatedAt time.Time
	UpdatedAt time.Time
	Insights  []InsightRow
}

// InsightRow is a saved insight; Spec is the JSON of InsightSpec.
type InsightRow struct {
	ID       string
	Title    string
	Spec     []byte
	Position int32
}

// DashboardSummaryRow is one entry of the dashboard list.
type DashboardSummaryRow struct {
	ID       string
	Name     string
	Insights int32
}

// ListDashboards returns all dashboards by name.
func (s *Store) ListDashboards(ctx context.Context) ([]DashboardSummaryRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT d.id, d.name, (SELECT count(*) FROM insights i WHERE i.dashboard_id = d.id)::INT4
	  FROM dashboards d ORDER BY d.name, d.created_at`)
	if err != nil {
		return nil, fmt.Errorf("listing dashboards: %w", err)
	}
	defer rows.Close()
	var out []DashboardSummaryRow
	for rows.Next() {
		var r DashboardSummaryRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Insights); err != nil {
			return nil, fmt.Errorf("scanning dashboard: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateDashboard adds an empty dashboard.
func (s *Store) CreateDashboard(ctx context.Context, name string) (*DashboardRow, error) {
	var id string
	if err := s.pool.QueryRow(ctx, `INSERT INTO dashboards (name) VALUES ($1) RETURNING id`, name).Scan(&id); err != nil {
		return nil, fmt.Errorf("creating dashboard: %w", err)
	}
	return s.GetDashboard(ctx, id)
}

// GetDashboard returns a dashboard with its insights in position order.
func (s *Store) GetDashboard(ctx context.Context, id string) (*DashboardRow, error) {
	d := &DashboardRow{}
	err := s.pool.QueryRow(ctx, `SELECT id, name, filters, created_at, updated_at FROM dashboards WHERE id = $1`, id).
		Scan(&d.ID, &d.Name, &d.Filters, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading dashboard: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id, title, spec, position FROM insights WHERE dashboard_id = $1
	  ORDER BY position, created_at`, id)
	if err != nil {
		return nil, fmt.Errorf("reading insights: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var in InsightRow
		if err := rows.Scan(&in.ID, &in.Title, &in.Spec, &in.Position); err != nil {
			return nil, fmt.Errorf("scanning insight: %w", err)
		}
		d.Insights = append(d.Insights, in)
	}
	return d, rows.Err()
}

// UpdateDashboard renames a dashboard and replaces its filters.
func (s *Store) UpdateDashboard(ctx context.Context, id, name string, filters []byte) (*DashboardRow, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE dashboards SET name = $2, filters = $3, updated_at = now() WHERE id = $1`, id, name, filters)
	if err != nil {
		return nil, fmt.Errorf("updating dashboard: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetDashboard(ctx, id)
}

// DeleteDashboard removes a dashboard and its insights.
func (s *Store) DeleteDashboard(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM dashboards WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting dashboard: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateInsight appends an insight to a dashboard.
func (s *Store) CreateInsight(ctx context.Context, dashboardID, title string, spec []byte) (*InsightRow, error) {
	in := &InsightRow{Title: title, Spec: spec}
	err := s.pool.QueryRow(ctx, `INSERT INTO insights (dashboard_id, title, spec, position)
	  SELECT $1, $2, $3, COALESCE(max(position) + 1, 0)::INT4 FROM insights WHERE dashboard_id = $1
	  RETURNING id, position`, dashboardID, title, spec).Scan(&in.ID, &in.Position)
	if err != nil {
		if isForeignKeyViolation(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("creating insight: %w", err)
	}
	if err := s.touch(ctx, dashboardID); err != nil {
		return nil, err
	}
	return in, nil
}

// UpdateInsight replaces an insight's title, spec and position.
func (s *Store) UpdateInsight(ctx context.Context, dashboardID, id, title string, spec []byte, position int32) (*InsightRow, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE insights SET title = $3, spec = $4, position = $5 WHERE id = $2 AND dashboard_id = $1`,
		dashboardID, id, title, spec, position)
	if err != nil {
		return nil, fmt.Errorf("updating insight: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := s.touch(ctx, dashboardID); err != nil {
		return nil, err
	}
	return &InsightRow{ID: id, Title: title, Spec: spec, Position: position}, nil
}

// DeleteInsight removes one insight.
func (s *Store) DeleteInsight(ctx context.Context, dashboardID, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM insights WHERE id = $2 AND dashboard_id = $1`, dashboardID, id)
	if err != nil {
		return fmt.Errorf("deleting insight: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return s.touch(ctx, dashboardID)
}

func (s *Store) touch(ctx context.Context, dashboardID string) error {
	if _, err := s.pool.Exec(ctx, `UPDATE dashboards SET updated_at = now() WHERE id = $1`, dashboardID); err != nil {
		return fmt.Errorf("touching dashboard: %w", err)
	}
	return nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	return errors.As(err, &pgErr) && pgErr.SQLState() == "23503"
}
