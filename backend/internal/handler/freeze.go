package handler

import (
	"context"
	"errors"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/jackc/pgx/v5"

	pb "linear-dashboard/backend/api/dashboard/v1"
	"linear-dashboard/backend/internal/report"
	"linear-dashboard/backend/internal/snapshot"
	"linear-dashboard/backend/internal/store"
)

// Error reasons of the freeze endpoints.
const (
	ReasonChangeDecided     = "CHANGE_DECIDED"
	ReasonChangeStale       = "CHANGE_STALE"
	ReasonLinearWriteFailed = "LINEAR_WRITE_FAILED"
	ReasonNoteFailed        = "NOTE_FAILED"
)

var _ pb.FreezeServiceHTTPServer = (*Dashboard)(nil)

// decidedShown is how many decided changes the list returns.
const decidedShown = 50

func (h *Dashboard) ListFrozenChanges(ctx context.Context, _ *pb.ListFrozenChangesRequest) (*pb.ListFrozenChangesResponse, error) {
	return h.frozenLists(ctx)
}

func (h *Dashboard) ApproveFrozenChange(ctx context.Context, req *pb.ApproveFrozenChangeRequest) (*pb.ApproveFrozenChangeResponse, error) {
	if err := h.runner.Approve(ctx, req.GetId()); err != nil {
		return nil, decisionError(err)
	}
	lists, err := h.frozenLists(ctx)
	return &pb.ApproveFrozenChangeResponse{Changes: lists}, err
}

func (h *Dashboard) RejectFrozenChange(ctx context.Context, req *pb.RejectFrozenChangeRequest) (*pb.RejectFrozenChangeResponse, error) {
	if err := h.runner.Reject(ctx, req.GetId(), req.GetReason()); err != nil {
		return nil, decisionError(err)
	}
	lists, err := h.frozenLists(ctx)
	return &pb.RejectFrozenChangeResponse{Changes: lists}, err
}

func decisionError(err error) error {
	switch {
	case errors.Is(err, snapshot.ErrRunning):
		return kerrors.Conflict(ReasonSyncRunning, "a sync is running; try again when it has finished")
	case errors.Is(err, store.ErrNotPending):
		return kerrors.Conflict(ReasonChangeDecided, "this change was already decided or withdrawn")
	case errors.Is(err, pgx.ErrNoRows):
		return kerrors.NotFound(ReasonNotFound, "no such change")
	case errors.Is(err, report.ErrChangeTarget):
		return kerrors.Conflict(ReasonChangeStale, "the task or project of this change is no longer in the dashboard")
	case errors.Is(err, snapshot.ErrNoteFailed):
		return kerrors.New(502, ReasonNoteFailed, err.Error())
	case errors.Is(err, snapshot.ErrLinearWrite):
		return kerrors.New(502, ReasonLinearWriteFailed, err.Error())
	default:
		return kerrors.InternalServer(ReasonInternal, "could not record the decision")
	}
}

func (h *Dashboard) frozenLists(ctx context.Context) (*pb.ListFrozenChangesResponse, error) {
	ds := h.runner.Dataset()
	if ds == nil {
		return nil, kerrors.ServiceUnavailable(ReasonNoData, "no data yet: the first Linear sync has not finished")
	}
	pending, err := h.store.FrozenChanges(ctx, true, 0)
	if err != nil {
		return nil, kerrors.InternalServer(ReasonInternal, "could not read pending changes")
	}
	decided, err := h.store.FrozenChanges(ctx, false, decidedShown)
	if err != nil {
		return nil, kerrors.InternalServer(ReasonInternal, "could not read decided changes")
	}
	res := &pb.ListFrozenChangesResponse{}
	for _, c := range pending {
		res.Pending = append(res.Pending, frozenRow(ds, c))
	}
	for _, c := range decided {
		res.Decided = append(res.Decided, frozenRow(ds, c))
	}
	return res, nil
}

func frozenRow(ds *report.Dataset, c store.FrozenChangeRow) *pb.FrozenChange {
	t := ds.Describe(c.FrozenChange)
	row := &pb.FrozenChange{
		Id: c.ID, Kind: c.Kind, Status: c.Status, What: t.What, OldValue: t.Old, NewValue: t.New,
		FrozenBy: t.FrozenBy, DetectedAt: c.DetectedAt.Format(time.RFC3339), Reason: c.Reason,
	}
	if c.DecidedAt != nil {
		row.DecidedAt = c.DecidedAt.Format(time.RFC3339)
	}
	if c.Kind == report.ChangeInitiative {
		if p := ds.Projects[c.ProjectID]; p != nil {
			row.ItemTitle, row.ItemUrl = p.Name, p.URL
		}
	} else if task := ds.Task(c.TaskID); task != nil {
		row.ItemIdentifier, row.ItemTitle, row.ItemUrl = task.Identifier, task.Title, task.URL
	}
	return row
}
