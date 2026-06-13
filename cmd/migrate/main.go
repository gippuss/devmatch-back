package main

import (
	"fmt"
	"os"

	"github.com/gippuss/devmatch-back/internal/config"
	"github.com/gippuss/devmatch-back/internal/platform/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if len(os.Args) < 2 {
		panic("usage: go run ./cmd/migrate [up|down|reset]")
	}

	command := os.Args[1]
	if err := postgres.RunMigrationCommand(cfg.DatabaseURL, "migrations", command); err != nil {
		panic(err)
	}

	fmt.Printf("migration command %s completed\n", command)
}
