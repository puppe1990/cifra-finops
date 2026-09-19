package app

import (
	"context"
	"os"
	"strconv"

	"github.com/puppe1990/cifra-finops/internal/scheduler"
)

const (
	// MonthlySyncHourEnv overrides the UTC hour the monthly AWS sync fires on
	// the last day of the month.
	MonthlySyncHourEnv     = "CIFRA_MONTHLY_SYNC_HOUR"
	defaultMonthlySyncHour = 23
)

// MonthlySyncHourFromEnv parses an hour in [0,23], falling back to the default
// on empty or invalid input.
func MonthlySyncHourFromEnv(raw string) int {
	hour, err := strconv.Atoi(raw)
	if err != nil || hour < 0 || hour > 23 {
		return defaultMonthlySyncHour
	}
	return hour
}

func newMonthlySync(sync func(context.Context) error) *scheduler.Monthly {
	hour := MonthlySyncHourFromEnv(os.Getenv(MonthlySyncHourEnv))
	return scheduler.NewMonthly(hour, func(ctx context.Context) error {
		return sync(ctx)
	})
}
