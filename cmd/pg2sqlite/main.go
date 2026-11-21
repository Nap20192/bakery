package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"bakery/internal/outbound/db/pgimport"
)

func main() {
	from := flag.String("from", "", "source Postgres URL")
	to := flag.String("to", "", "target SQLite file (must not exist)")
	migrations := flag.String("migrations", "migrations", "migrations directory")
	flag.Parse()

	if *from == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "usage: pg2sqlite -from <postgres-url> -to <sqlite-file>")
		os.Exit(2)
	}
	if err := pgimport.Import(context.Background(), *from, *to, *migrations); err != nil {
		slog.Error("pg2sqlite failed", "error", err)
		os.Exit(1)
	}
}
