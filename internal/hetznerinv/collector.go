package hetznerinv

import "context"

type Collector interface {
	Collect(ctx context.Context, token string) (Inventory, error)
}
