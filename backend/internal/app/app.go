// Package app wires configuration, storage, sync, scheduling and the HTTP server.
package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	nethttp "net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/encoding"
	"github.com/go-kratos/kratos/v3/encoding/protojson"
	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/robfig/cron/v3"
	gprotojson "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "linear-dashboard/backend/api/dashboard/v1"
	"linear-dashboard/backend/internal/handler"
	"linear-dashboard/backend/internal/linear"
	"linear-dashboard/backend/internal/snapshot"
	"linear-dashboard/backend/internal/store"
)

// Config is read from the environment.
type Config struct {
	LinearAPIKey    string
	DatabaseURL     string
	HTTPAddr        string
	SyncCron        string // one or more cron specs separated by ";"
	TimeZone        *time.Location
	SyncCooldown    time.Duration // between two full manual syncs (Resync)
	RefreshCooldown time.Duration // between two changed-only syncs (Refresh)
}

// ConfigFromEnv reads and checks the configuration.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		LinearAPIKey: os.Getenv("LINEAR_API_KEY"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		HTTPAddr:     envOr("HTTP_ADDR", ":8000"),
		SyncCron:     envOr("SYNC_CRON", DefaultSyncCron),
	}
	if cfg.LinearAPIKey == "" {
		return Config{}, fmt.Errorf("LINEAR_API_KEY is not set")
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is not set")
	}
	loc, err := time.LoadLocation(envOr("SYNC_TZ", "Asia/Jakarta"))
	if err != nil {
		return Config{}, fmt.Errorf("loading SYNC_TZ: %w", err)
	}
	cfg.TimeZone = loc
	cfg.SyncCooldown, err = time.ParseDuration(envOr("SYNC_COOLDOWN", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("parsing SYNC_COOLDOWN: %w", err)
	}
	cfg.RefreshCooldown, err = time.ParseDuration(envOr("REFRESH_COOLDOWN", "2m"))
	if err != nil {
		return Config{}, fmt.Errorf("parsing REFRESH_COOLDOWN: %w", err)
	}
	return cfg, nil
}

// DefaultSyncCron syncs every night at 01:00 and every weekday at 17:00, in SYNC_TZ.
const DefaultSyncCron = "0 1 * * *; 0 17 * * 1-5"

// addSchedules registers fn once for each ";"-separated cron spec.
func addSchedules(c *cron.Cron, specs string, fn func()) error {
	for _, spec := range strings.Split(specs, ";") {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}
		if _, err := c.AddFunc(spec, fn); err != nil {
			return fmt.Errorf("parsing SYNC_CRON %q: %w", spec, err)
		}
	}
	return nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// Run starts the backend and blocks until it receives SIGINT or SIGTERM.
func Run(ctx context.Context, cfg Config, logger *slog.Logger) error {
	base, cancel := context.WithCancel(ctx)
	defer cancel()

	st, err := store.Open(base, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	if n, err := st.FailAbandonedRuns(base); err != nil {
		return err
	} else if n > 0 {
		logger.Warn("marked abandoned syncs as failed", "count", n)
	}

	runner := snapshot.NewRunner(linear.New(cfg.LinearAPIKey), st, map[string]time.Duration{
		snapshot.TriggerManual:  cfg.SyncCooldown,
		snapshot.TriggerRefresh: cfg.RefreshCooldown,
	}, logger)
	ds, err := st.Load(base)
	if err != nil {
		return err
	}
	if ds != nil {
		runner.SetDataset(ds)
		logger.Info("loaded stored workspace", "synced_at", ds.SyncedAt, "tasks", len(ds.Tasks))
	} else if _, err := runner.Start(base, base, snapshot.TriggerStartup); err != nil {
		return fmt.Errorf("starting first sync: %w", err)
	}

	sched := cron.New(cron.WithLocation(cfg.TimeZone))
	if err := addSchedules(sched, cfg.SyncCron, func() {
		if _, err := runner.Start(base, base, snapshot.TriggerSchedule); err != nil {
			logger.Warn("scheduled sync skipped", "err", err)
		}
	}); err != nil {
		return err
	}
	sched.Start()
	defer func() { <-sched.Stop().Done() }()

	srv := http.NewServer(
		http.Address(cfg.HTTPAddr),
		http.Timeout(30*time.Second),
		http.Middleware(recovery.Recovery()),
		http.RequestDecoder(decodeProtoJSON),
		http.ResponseEncoder(encodeProtoJSON),
		http.ErrorEncoder(encodeError),
	)
	h := handler.New(base, runner, st, cfg.TimeZone)
	pb.RegisterDashboardServiceHTTPServer(srv, h)
	pb.RegisterSyncServiceHTTPServer(srv, h)
	pb.RegisterInsightServiceHTTPServer(srv, h)
	srv.HandleFunc("/healthz", func(w nethttp.ResponseWriter, _ *nethttp.Request) { w.WriteHeader(nethttp.StatusOK) })

	app := kratos.New(
		kratos.Name("effort-dashboard"),
		kratos.Server(srv),
		kratos.Signal(syscall.SIGINT, syscall.SIGTERM),
		kratos.StopTimeout(10*time.Second),
	)
	err = app.Run()
	cancel()      // stop a running sync; it records itself as failed
	runner.Wait() // let it record the outcome before the store closes
	return err
}

// decodeProtoJSON reads request bodies with protojson, so camelCase and
// snake_case field names are both accepted, mirroring what the API sends back.
// (Kratos v3's default JSON codec only accepts snake_case.)
func decodeProtoJSON(r *nethttp.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return kerrors.BadRequest("CODEC", err.Error())
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	msg, ok := v.(proto.Message)
	if !ok {
		return kerrors.BadRequest("CODEC", "request is not a protobuf message")
	}
	if err := (gprotojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, msg); err != nil {
		return kerrors.BadRequest("CODEC", "invalid JSON body: "+err.Error())
	}
	return nil
}

// encodeProtoJSON always answers with protojson: camelCase field names and zero
// values included, so "0 pts" is present rather than missing.
func encodeProtoJSON(w nethttp.ResponseWriter, _ *nethttp.Request, v any) error {
	if v == nil {
		return nil
	}
	body, err := encoding.GetCodec(protojson.Name).Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(body)
	return err
}

func encodeError(w nethttp.ResponseWriter, _ *nethttp.Request, err error) {
	se := kerrors.FromError(err)
	body, merr := encoding.GetCodec("json").Marshal(map[string]any{
		"code": se.Code, "reason": se.Reason, "message": se.Message,
	})
	if merr != nil {
		w.WriteHeader(nethttp.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(se.Code))
	_, _ = w.Write(body)
}
