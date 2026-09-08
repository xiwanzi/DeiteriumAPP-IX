// Standalone migration utility: intentionally contains no HTTP server/deployment command.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: deuterium-identity export-legacy|import-legacy --file PATH [--apply]")
	}
	mode := os.Args[1]
	if mode != "export-legacy" && mode != "import-legacy" {
		return errors.New("only export-legacy and import-legacy are supported")
	}
	flags := flag.NewFlagSet(mode, flag.ContinueOnError)
	path := flags.String("file", "", "protected JSONL path")
	apply := flags.Bool("apply", false, "apply import; default is read-only preflight")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if *path == "" || flags.NArg() != 0 {
		return errors.New("--file is required; no extra arguments")
	}
	key := "DEUTERIUM_DSN"
	if mode == "export-legacy" {
		key = "DEUTERIUM_LEGACY_DSN"
	}
	dsn := os.Getenv(key)
	if dsn == "" {
		return errors.New("required database environment variable is missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.DB.Close()
	if mode == "export-legacy" {
		if *apply {
			return errors.New("--apply is not an export option")
		}
		f, err := os.OpenFile(*path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return errors.New("export path exists or cannot be created")
		}
		n, exportErr := identity.ExportLegacy(ctx, db, f)
		closeErr := f.Close()
		if exportErr != nil || closeErr != nil {
			_ = os.Remove(*path)
			return errors.New("export failed; incomplete file removed")
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]int{"exported": n})
	}
	f, err := os.Open(*path)
	if err != nil {
		return errors.New("import file cannot be opened")
	}
	defer f.Close()
	users, err := identity.ReadLegacy(f)
	if err != nil {
		return err
	}
	result, err := identity.Import(ctx, db, users, *apply)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(result)
}
