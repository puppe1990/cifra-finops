package scheduler

import (
	"context"
	"log"
	"time"
)

// Monthly runs job on the last day of every month at hour UTC, until ctx is
// canceled. Sleep is injected so the loop stays testable.
type Monthly struct {
	hour  int
	job   func(context.Context) error
	now   func() time.Time
	after func(time.Duration) <-chan time.Time
	logf  func(string, ...any)
}

func NewMonthly(hour int, job func(context.Context) error) *Monthly {
	return &Monthly{
		hour:  hour,
		job:   job,
		now:   time.Now,
		after: time.After,
		logf:  log.Printf,
	}
}

func (m *Monthly) Run(ctx context.Context) {
	for {
		now := m.now()
		next := NextRun(now, m.hour)
		m.logf("monthly AWS sync scheduled for %s", next.Format(time.RFC3339))
		select {
		case <-ctx.Done():
			return
		case <-m.after(next.Sub(now)):
			if err := m.job(ctx); err != nil {
				m.logf("monthly AWS sync failed: %v", err)
			}
		}
	}
}
