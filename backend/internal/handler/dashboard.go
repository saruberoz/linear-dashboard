// Package handler implements the HTTP API on top of the report views.
package handler

import (
	"context"
	"errors"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"

	pb "linear-dashboard/backend/api/dashboard/v1"
	"linear-dashboard/backend/internal/report"
	"linear-dashboard/backend/internal/snapshot"
	"linear-dashboard/backend/internal/store"
)

// Error reasons returned by the API.
const (
	ReasonInvalidRange = "INVALID_RANGE"
	ReasonInvalidScope = "INVALID_SCOPE"
	ReasonInvalidDim   = "INVALID_DIM"
	ReasonNotFound     = "NOT_FOUND"
	ReasonNoData       = "NO_DATA"
	ReasonSyncRunning  = "SYNC_RUNNING"
	ReasonSyncCooldown = "SYNC_COOLDOWN"
	ReasonInternal     = "INTERNAL"
)

// Dashboard serves the read endpoints and the sync endpoints.
type Dashboard struct {
	runner *snapshot.Runner
	store  *store.Store
	loc    *time.Location
	base   context.Context // canceled on shutdown; background syncs run under it
}

// New returns the API handler.
func New(base context.Context, r *snapshot.Runner, s *store.Store, loc *time.Location) *Dashboard {
	return &Dashboard{runner: r, store: s, loc: loc, base: base}
}

var (
	_ pb.DashboardServiceHTTPServer = (*Dashboard)(nil)
	_ pb.SyncServiceHTTPServer      = (*Dashboard)(nil)
)

// prepare returns the current dataset and the parsed range. Empty dates default
// to the current month in the dashboard's time zone.
func (h *Dashboard) prepare(from, to string) (*report.Dataset, report.Range, error) {
	ds := h.runner.Dataset()
	if ds == nil {
		return nil, report.Range{}, kerrors.ServiceUnavailable(ReasonNoData, "no data yet: the first Linear sync has not finished")
	}
	if from == "" || to == "" {
		now := time.Now().In(h.loc)
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, h.loc)
		if from == "" {
			from = first.Format("2006-01-02")
		}
		if to == "" {
			to = first.AddDate(0, 1, -1).Format("2006-01-02")
		}
	}
	r, err := report.ParseRange(from, to, h.loc)
	if err != nil {
		return nil, report.Range{}, kerrors.BadRequest(ReasonInvalidRange, err.Error())
	}
	return ds, r, nil
}

// mapErr converts report errors to API errors.
func mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, report.ErrNotFound):
		return kerrors.NotFound(ReasonNotFound, err.Error())
	case errors.Is(err, report.ErrInvalidScope):
		return kerrors.BadRequest(ReasonInvalidScope, err.Error())
	case errors.Is(err, report.ErrInvalidDim):
		return kerrors.BadRequest(ReasonInvalidDim, err.Error())
	default:
		return kerrors.InternalServer(ReasonInternal, "internal error")
	}
}

func (h *Dashboard) GetMenu(ctx context.Context, req *pb.GetMenuRequest) (*pb.GetMenuResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		if kerrors.Reason(err) == ReasonNoData {
			// The menu still answers before the first sync, so the page can show sync progress.
			return &pb.GetMenuResponse{Sync: h.syncInfo(ctx)}, nil
		}
		return nil, err
	}
	res := ds.Menu(r)
	res.Sync = h.syncInfo(ctx)
	return res, nil
}

func (h *Dashboard) GetOverview(_ context.Context, req *pb.GetOverviewRequest) (*pb.GetOverviewResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	return ds.Overview(r), nil
}

func (h *Dashboard) GetSegment(_ context.Context, req *pb.GetSegmentRequest) (*pb.GetSegmentResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.Segment(req.GetSegment(), r)
	return res, mapErr(err)
}

func (h *Dashboard) GetInitiative(_ context.Context, req *pb.GetInitiativeRequest) (*pb.GetInitiativeResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.Initiative(req.GetId(), r)
	return res, mapErr(err)
}

func (h *Dashboard) GetProject(_ context.Context, req *pb.GetProjectRequest) (*pb.GetProjectResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.Project(req.GetId(), r)
	return res, mapErr(err)
}

func (h *Dashboard) ListTasks(_ context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.CellTasks(req.GetScope(), req.GetDim(), req.GetColumn(), req.GetAssigneeId(), r)
	return res, mapErr(err)
}

func (h *Dashboard) ListMissingEstimates(_ context.Context, req *pb.ListMissingEstimatesRequest) (*pb.ListMissingEstimatesResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.MissingEstimates(req.GetScope(), req.GetAll(), r)
	return res, mapErr(err)
}

func (h *Dashboard) ListSegmentLabelWarnings(_ context.Context, req *pb.ListSegmentLabelWarningsRequest) (*pb.ListSegmentLabelWarningsResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	return ds.SegmentLabelWarnings(r), nil
}

func (h *Dashboard) ListUntaggedProjects(_ context.Context, req *pb.ListUntaggedProjectsRequest) (*pb.ListUntaggedProjectsResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	return ds.UntaggedProjects(r), nil
}

func (h *Dashboard) ListProjectsWithoutInitiative(_ context.Context, req *pb.ListProjectsWithoutInitiativeRequest) (*pb.ListProjectsWithoutInitiativeResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	return ds.ProjectsWithoutInitiative(r), nil
}

func (h *Dashboard) ListInitiativesWithoutProjects(_ context.Context, _ *pb.ListInitiativesWithoutProjectsRequest) (*pb.ListInitiativesWithoutProjectsResponse, error) {
	ds, _, err := h.prepare("", "")
	if err != nil {
		return nil, err
	}
	return ds.InitiativesWithoutProjects(), nil
}

// StartSync replies 409 or 429 before any work starts; otherwise the sync runs
// in the background and the reply reports it as running.
func (h *Dashboard) StartSync(ctx context.Context, _ *pb.StartSyncRequest) (*pb.StartSyncResponse, error) {
	if _, err := h.runner.Start(ctx, h.base, snapshot.TriggerManual); err != nil {
		switch {
		case errors.Is(err, snapshot.ErrRunning):
			return nil, kerrors.Conflict(ReasonSyncRunning, "a sync is already running")
		case errors.Is(err, snapshot.ErrCooldown):
			return nil, kerrors.TooManyRequests(ReasonSyncCooldown, err.Error())
		default:
			return nil, kerrors.InternalServer(ReasonInternal, "could not start sync")
		}
	}
	return &pb.StartSyncResponse{Sync: h.syncInfo(ctx)}, nil
}

func (h *Dashboard) GetSyncStatus(ctx context.Context, _ *pb.GetSyncStatusRequest) (*pb.GetSyncStatusResponse, error) {
	return &pb.GetSyncStatusResponse{Sync: h.syncInfo(ctx)}, nil
}

// syncInfo reports sync state; store errors degrade to "unknown" fields rather than failing the request.
func (h *Dashboard) syncInfo(ctx context.Context) *pb.SyncInfo {
	info := &pb.SyncInfo{Running: h.runner.Running()}
	if t, ok, err := h.store.LastSuccess(ctx); err == nil && ok {
		info.LastSuccessAt = t.Format(time.RFC3339)
	}
	if run, err := h.store.LastRun(ctx); err == nil && run != nil {
		info.LastAttemptAt = run.StartedAt.Format(time.RFC3339)
		info.LastStatus = run.Status
		info.LastError = run.Error
	}
	return info
}
