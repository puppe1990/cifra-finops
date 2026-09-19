package scheduler

import (
	"context"
	"testing"
	"time"
)

func TestNextRun_landsOnLastDayOfMonthAtHourUTC(t *testing.T) {
	cases := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "mid month waits for the last day",
			now:  time.Date(2026, time.January, 15, 8, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.January, 31, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "last day before hour runs same day",
			now:  time.Date(2026, time.January, 31, 10, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.January, 31, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "last day after hour rolls to next month",
			now:  time.Date(2026, time.January, 31, 23, 30, 0, 0, time.UTC),
			want: time.Date(2026, time.February, 28, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "leap year february",
			now:  time.Date(2028, time.February, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2028, time.February, 29, 23, 0, 0, 0, time.UTC),
		},
		{
			name: "december rolls into next year",
			now:  time.Date(2026, time.December, 30, 12, 0, 0, 0, time.UTC),
			want: time.Date(2026, time.December, 31, 23, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NextRun(tc.now, 23); !got.Equal(tc.want) {
				t.Fatalf("NextRun(%s) = %s, want %s", tc.now, got, tc.want)
			}
		})
	}
}

func TestNextRun_isAlwaysInTheFuture(t *testing.T) {
	now := time.Date(2026, time.March, 31, 23, 0, 0, 0, time.UTC)
	if got := NextRun(now, 23); !got.After(now) {
		t.Fatalf("NextRun(%s) = %s, want strictly after now", now, got)
	}
}

func TestMonthlyRun_invokesJobOnTickAndStopsOnCancel(t *testing.T) {
	ticks := make(chan time.Time, 1)
	jobRan := make(chan struct{}, 1)
	m := NewMonthly(23, func(_ context.Context) error {
		jobRan <- struct{}{}
		return nil
	})
	m.now = func() time.Time { return time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC) }
	m.after = func(time.Duration) <-chan time.Time { return ticks }
	m.logf = func(string, ...any) {}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		m.Run(ctx)
		close(done)
	}()

	ticks <- time.Now()
	select {
	case <-jobRan:
	case <-time.After(time.Second):
		t.Fatal("monthly job was not invoked")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}
