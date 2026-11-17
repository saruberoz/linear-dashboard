package handler

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "linear-dashboard/backend/api/dashboard/v1"
	"linear-dashboard/backend/internal/report"
	"linear-dashboard/backend/internal/store"
)

// Error reasons for dashboards and insights.
const (
	ReasonInvalidInsight = "INVALID_INSIGHT"
	ReasonInvalidName    = "INVALID_NAME"
)

var _ pb.InsightServiceHTTPServer = (*Dashboard)(nil)

func (h *Dashboard) ListDashboards(ctx context.Context, _ *pb.ListDashboardsRequest) (*pb.ListDashboardsResponse, error) {
	rows, err := h.store.ListDashboards(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	res := &pb.ListDashboardsResponse{}
	for _, r := range rows {
		res.Dashboards = append(res.Dashboards, &pb.DashboardSummary{Id: r.ID, Name: r.Name, Insights: r.Insights})
	}
	return res, nil
}

func (h *Dashboard) CreateDashboard(ctx context.Context, req *pb.CreateDashboardRequest) (*pb.CreateDashboardResponse, error) {
	name, err := cleanName(req.GetName(), "Untitled dashboard")
	if err != nil {
		return nil, err
	}
	row, err := h.store.CreateDashboard(ctx, name)
	if err != nil {
		return nil, storeErr(err)
	}
	d, err := toDashboard(row)
	return &pb.CreateDashboardResponse{Dashboard: d}, err
}

func (h *Dashboard) GetDashboard(ctx context.Context, req *pb.GetDashboardRequest) (*pb.GetDashboardResponse, error) {
	row, err := h.store.GetDashboard(ctx, req.GetId())
	if err != nil {
		return nil, storeErr(err)
	}
	d, err := toDashboard(row)
	return &pb.GetDashboardResponse{Dashboard: d}, err
}

func (h *Dashboard) UpdateDashboard(ctx context.Context, req *pb.UpdateDashboardRequest) (*pb.UpdateDashboardResponse, error) {
	name, err := cleanName(req.GetName(), "")
	if err != nil {
		return nil, err
	}
	filters, err := marshal(req.GetFilters())
	if err != nil {
		return nil, err
	}
	row, err := h.store.UpdateDashboard(ctx, req.GetId(), name, filters)
	if err != nil {
		return nil, storeErr(err)
	}
	d, err := toDashboard(row)
	return &pb.UpdateDashboardResponse{Dashboard: d}, err
}

func (h *Dashboard) DeleteDashboard(ctx context.Context, req *pb.DeleteDashboardRequest) (*pb.DeleteDashboardResponse, error) {
	return &pb.DeleteDashboardResponse{}, storeErr(h.store.DeleteDashboard(ctx, req.GetId()))
}

func (h *Dashboard) CreateInsight(ctx context.Context, req *pb.CreateInsightRequest) (*pb.CreateInsightResponse, error) {
	title, spec, err := checkInsight(req.GetTitle(), req.GetSpec())
	if err != nil {
		return nil, err
	}
	row, err := h.store.CreateInsight(ctx, req.GetDashboardId(), title, spec)
	if err != nil {
		return nil, storeErr(err)
	}
	in, err := toInsight(*row)
	return &pb.CreateInsightResponse{Insight: in}, err
}

func (h *Dashboard) UpdateInsight(ctx context.Context, req *pb.UpdateInsightRequest) (*pb.UpdateInsightResponse, error) {
	title, spec, err := checkInsight(req.GetTitle(), req.GetSpec())
	if err != nil {
		return nil, err
	}
	row, err := h.store.UpdateInsight(ctx, req.GetDashboardId(), req.GetId(), title, spec, req.GetPosition())
	if err != nil {
		return nil, storeErr(err)
	}
	in, err := toInsight(*row)
	return &pb.UpdateInsightResponse{Insight: in}, err
}

func (h *Dashboard) DeleteInsight(ctx context.Context, req *pb.DeleteInsightRequest) (*pb.DeleteInsightResponse, error) {
	return &pb.DeleteInsightResponse{}, storeErr(h.store.DeleteInsight(ctx, req.GetDashboardId(), req.GetId()))
}

func (h *Dashboard) RunInsight(_ context.Context, req *pb.RunInsightRequest) (*pb.RunInsightResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.RunInsight(req.GetDashboardFilters(), req.GetSpec(), r)
	if err != nil {
		return nil, insightErr(err)
	}
	return &pb.RunInsightResponse{Result: res}, nil
}

func (h *Dashboard) ListInsightTasks(_ context.Context, req *pb.ListInsightTasksRequest) (*pb.ListInsightTasksResponse, error) {
	ds, r, err := h.prepare(req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, err
	}
	res, err := ds.InsightTasks(req.GetDashboardFilters(), req.GetSpec(), req.GetSliceKey(), req.GetSegmentKey(), r)
	if err != nil {
		return nil, insightErr(err)
	}
	return res, nil
}

func (h *Dashboard) GetInsightOptions(_ context.Context, _ *pb.GetInsightOptionsRequest) (*pb.GetInsightOptionsResponse, error) {
	ds, _, err := h.prepare("", "")
	if err != nil {
		return nil, err
	}
	res := &pb.GetInsightOptionsResponse{}
	for _, t := range ds.Teams {
		res.Teams = append(res.Teams, &pb.TeamOption{Id: t.ID, Key: t.Key, Name: t.Name})
	}
	sort.Slice(res.Teams, func(i, j int) bool { return res.Teams[i].Name < res.Teams[j].Name })
	// Distinct task label names (the same name exists in several teams), most used first.
	uses := map[string]int32{}
	names := map[string]string{}
	for i := range ds.Tasks {
		for _, id := range ds.Tasks[i].LabelIDs {
			if l, ok := ds.Labels[id]; ok && l.Kind == "issue" {
				k := strings.ToLower(l.Name)
				uses[k]++
				names[k] = l.Name
			}
		}
	}
	for k, n := range uses {
		res.Labels = append(res.Labels, &pb.LabelOption{Name: names[k], Tasks: n})
	}
	sort.Slice(res.Labels, func(i, j int) bool {
		if res.Labels[i].Tasks != res.Labels[j].Tasks {
			return res.Labels[i].Tasks > res.Labels[j].Tasks
		}
		return res.Labels[i].Name < res.Labels[j].Name
	})
	return res, nil
}

// ---------- helpers ----------

func cleanName(name, fallback string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = fallback
	}
	if name == "" || len(name) > 120 {
		return "", kerrors.BadRequest(ReasonInvalidName, "name must be 1-120 characters")
	}
	return name, nil
}

func checkInsight(title string, spec *pb.InsightSpec) (string, []byte, error) {
	if spec == nil {
		return "", nil, kerrors.BadRequest(ReasonInvalidInsight, "spec is required")
	}
	if err := report.ValidateSpec(spec); err != nil {
		return "", nil, insightErr(err)
	}
	t, err := cleanName(title, "Untitled insight")
	if err != nil {
		return "", nil, err
	}
	b, err := marshal(spec)
	return t, b, err
}

func marshal(m proto.Message) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		return nil, kerrors.InternalServer(ReasonInternal, "could not encode")
	}
	return b, nil
}

func toInsight(r store.InsightRow) (*pb.Insight, error) {
	spec := &pb.InsightSpec{}
	if err := protojson.Unmarshal(r.Spec, spec); err != nil {
		return nil, kerrors.InternalServer(ReasonInternal, "stored insight is unreadable")
	}
	return &pb.Insight{Id: r.ID, Title: r.Title, Spec: spec, Position: r.Position}, nil
}

func toDashboard(r *store.DashboardRow) (*pb.SavedDashboard, error) {
	f := &pb.InsightFilters{}
	if err := protojson.Unmarshal(r.Filters, f); err != nil {
		return nil, kerrors.InternalServer(ReasonInternal, "stored dashboard is unreadable")
	}
	d := &pb.SavedDashboard{Id: r.ID, Name: r.Name, Filters: f,
		CreatedAt: r.CreatedAt.Format(time.RFC3339), UpdatedAt: r.UpdatedAt.Format(time.RFC3339)}
	for _, row := range r.Insights {
		in, err := toInsight(row)
		if err != nil {
			return nil, err
		}
		d.Insights = append(d.Insights, in)
	}
	return d, nil
}

func storeErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, store.ErrNotFound):
		return kerrors.NotFound(ReasonNotFound, "dashboard or insight not found")
	default:
		return kerrors.InternalServer(ReasonInternal, "internal error")
	}
}

func insightErr(err error) error {
	if errors.Is(err, report.ErrInvalidInsight) {
		return kerrors.BadRequest(ReasonInvalidInsight, err.Error())
	}
	return mapErr(err)
}
