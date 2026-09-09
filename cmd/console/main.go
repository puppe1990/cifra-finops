package main

import (
	"log"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/console"

	"github.com/puppe1990/cifra-finops/internal/store"
)

func openStore(cfg cais.Config) (*store.SQLiteStore, error) {
	return store.NewSQLiteStore(cfg.DBPath, cfg.Env)
}

func bindings(s *store.SQLiteStore) map[string]any {
	return map[string]any{
		"store": s,
		"db":    s.DB(),
	}
}

func main() {
	cfg := cais.Load()
	s, err := openStore(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	active := s
	if err := console.Run(console.Options{
		AppName:  "cifra-finops",
		Config:   cfg,
		Bindings: bindings(active),
		Reload: func() (map[string]any, error) {
			_ = active.Close()
			next, err := openStore(cfg)
			if err != nil {
				return nil, err
			}
			active = next
			return bindings(active), nil
		},
	}); err != nil {
		log.Print(err)
	}
}
