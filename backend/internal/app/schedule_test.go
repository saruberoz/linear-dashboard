package app

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"
)

func TestAddSchedulesRunsEverySpec(t *testing.T) {
	jkt, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	c := cron.New(cron.WithLocation(jkt))
	if err := addSchedules(c, "0 1 * * *; 0 17 * * 1-5", func() {}); err != nil {
		t.Fatal(err)
	}
	// Friday 13:00 WIB: next runs are Fri 17:00 and Sat 01:00. Saturday has no 17:00 run.
	from := time.Date(2026, 11, 20, 13, 0, 0, 0, jkt)
	var got []time.Time
	for _, e := range c.Entries() {
		next := e.Schedule.Next(from)
		got = append(got, next, e.Schedule.Next(next))
	}
	want := map[string]bool{
		"2026-11-20 17:00": true, "2026-11-23 17:00": true, // weekday 17:00, skipping the weekend
		"2026-11-21 01:00": true, "2026-11-22 01:00": true, // nightly 01:00
	}
	if len(got) != len(want) {
		t.Fatalf("got %d run times %v, want %d", len(got), got, len(want))
	}
	for _, g := range got {
		if k := g.In(jkt).Format("2006-01-02 15:04"); !want[k] {
			t.Errorf("unexpected run at %s", k)
		}
	}
}

func TestAddSchedulesRejectsBadSpec(t *testing.T) {
	if err := addSchedules(cron.New(), "0 1 * * *; nonsense", func() {}); err == nil {
		t.Error("want an error for an invalid schedule")
	}
}
