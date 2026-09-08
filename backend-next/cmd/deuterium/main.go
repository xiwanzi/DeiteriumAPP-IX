package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/httpapi"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(os.Args[1:]); err != nil {
		slog.Error("command failed", "reason", err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: deuterium serve|migrate|export-legacy|import-legacy|grant-permission|core-key")
	}
	if args[0] == "core-key" {
		secret := identity.Secret()
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"token": secret, "tokenSha256": store.Digest([]byte(secret))})
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	configPath := flags.String("config", "config.json", "service configuration file")
	input := flags.String("file", "", "legacy JSONL export")
	apply := flags.Bool("apply", false, "apply validated legacy import; default is read-only")
	user := flags.String("user", "", "stable userId")
	permission := flags.String("permission", "", "permission to grant")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	switch args[0] {
	case "serve", "migrate", "export-legacy", "import-legacy", "grant-permission", "asset-gc-preview":
	default:
		return errors.New("unknown command")
	}
	dsn := os.Getenv("DEUTERIUM_DSN")
	if args[0] == "export-legacy" {
		dsn = os.Getenv("DEUTERIUM_LEGACY_DSN")
	}
	if dsn == "" {
		return errors.New("database environment variable is required (export uses DEUTERIUM_LEGACY_DSN; other commands use DEUTERIUM_DSN)")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	openCtx, openCancel := context.WithTimeout(ctx, 10*time.Second)
	db, err := store.Open(openCtx, dsn)
	openCancel()
	if err != nil {
		return err
	}
	defer db.DB.Close()
	switch args[0] {
	case "asset-gc-preview":
		assets, err := db.PreviewAssetLifecycleV2(ctx, 1000)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(assets)
	case "export-legacy":
		if *input == "" {
			return errors.New("--file is required")
		}
		file, err := os.OpenFile(*input, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return errors.New("export file already exists or cannot be created")
		}
		count, err := identity.ExportLegacy(ctx, db, file)
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			_ = os.Remove(*input)
			return errors.New("legacy export failed; incomplete file removed")
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]int{"exported": count})
	case "migrate":
		if err = db.Migrate(ctx); err != nil {
			return errors.New("migration failed; check schema and migration checksum")
		}
		slog.Info("schema migrated")
		return nil
	case "import-legacy":
		if *input == "" {
			return errors.New("--file is required")
		}
		file, err := os.Open(*input)
		if err != nil {
			return errors.New("legacy export could not be opened")
		}
		defer file.Close()
		users, err := identity.ReadLegacy(file)
		if err != nil {
			return err
		}
		result, err := identity.Import(ctx, db, users, *apply)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	case "grant-permission":
		if *user == "" || !store.ValidPermission(*permission) {
			return errors.New("--user and a supported explicit --permission are required")
		}
		if err = db.Grant(ctx, *user, *permission); err != nil {
			return errors.New("permission grant failed")
		}
		slog.Info("permission granted")
		return nil
	case "serve":
		c, err := config.Load(*configPath)
		if err != nil {
			return err
		}
		if db.Ready(ctx) != nil {
			return errors.New("database schema is not ready; run migrate first")
		}
		// The first release is intentionally one application instance per database.
		lease, err := db.DB.Conn(ctx)
		if err != nil {
			return errors.New("runtime lock unavailable")
		}
		defer lease.Close()
		var locked int
		if err = lease.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT(DATABASE(), ':deuterium-runtime'),0)").Scan(&locked); err != nil || locked != 1 {
			return errors.New("another backend instance is active or runtime lock unavailable")
		}
		defer lease.ExecContext(context.Background(), "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':deuterium-runtime'))")
		if err = db.RecoverCore(ctx); err != nil {
			return errors.New("Core operation recovery failed")
		}
		app := httpapi.New(db, c)
		defer app.Close()
		app.StartCommerceV2()
		app.StartAssetLifecycleV2()
		server := &http.Server{Addr: c.Listen, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
		result := make(chan error, 1)
		go func() { result <- server.ListenAndServe() }()
		slog.Info("backend starting", "listen", c.Listen, "publicOrigin", c.PublicOrigin)
		maintenance := time.NewTicker(time.Minute)
		defer maintenance.Stop()
		leaseTick := time.NewTicker(5 * time.Second)
		defer leaseTick.Stop()
		var reason error
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case err := <-result:
				if !errors.Is(err, http.ErrServerClosed) {
					reason = errors.New("HTTP listener failed")
				}
				break loop
			case <-leaseTick.C:
				checkCtx, checkCancel := context.WithTimeout(ctx, 3*time.Second)
				err := lease.PingContext(checkCtx)
				checkCancel()
				if err != nil {
					reason = errors.New("runtime lock connection lost; restart after database recovery")
					break loop
				}
			case <-maintenance.C:
				checkCtx, checkCancel := context.WithTimeout(ctx, 5*time.Second)
				err := db.Sweep(checkCtx)
				if err == nil {
					err = db.ReconcilePublicSocialNotificationsV2(checkCtx)
				}
				checkCancel()
				if err != nil {
					slog.Warn("maintenance deferred")
				}
			}
		}
		app.Close()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown: %w", err)
		}
		return reason
	}
	return nil
}
