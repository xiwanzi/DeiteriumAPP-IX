package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/httpapi"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/imagereview"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

func main() {
	if e := run(); e != nil {
		slog.Error("image review service stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	migrate := flag.Bool("migrate-only", false, "create review schema in its independent database, then exit")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	audit, e := store.Open(ctx, os.Getenv("DEUTERIUM_IMAGE_REVIEW_DSN"))
	if e != nil {
		return e
	}
	defer audit.DB.Close()
	audit.DB.SetMaxOpenConns(6)
	var reviewDatabaseName string
	if e = audit.DB.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&reviewDatabaseName); e != nil {
		return e
	}
	if !strings.HasPrefix(reviewDatabaseName, "deuterium_image_review") {
		return errors.New("review database name must start with deuterium_image_review")
	}
	repo := &imagereview.Repository{DB: audit.DB}
	if *migrate {
		return repo.Migrate(ctx)
	}
	source, e := store.Open(ctx, os.Getenv("DEUTERIUM_IMAGE_REVIEW_SOURCE_DSN"))
	if e != nil {
		return e
	}
	defer source.DB.Close()
	source.DB.SetMaxOpenConns(3)
	var sourceName, auditName string
	if e = source.DB.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&sourceName); e != nil {
		return e
	}
	if e = audit.DB.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&auditName); e != nil {
		return e
	}
	if sourceName == auditName {
		return errors.New("image review requires a separate database")
	}
	if _, e = repo.Summary(ctx); e != nil {
		return errors.New("review schema missing; run migrate-only with review database")
	}
	objects, e := objectstorage.FromEnvironment()
	if e != nil {
		return e
	}
	reviewer, e := imagereview.NewDeepSeek(os.Getenv("DEUTERIUM_IMAGE_REVIEW_API_KEY"))
	if e != nil {
		return e
	}
	origins := strings.Split(os.Getenv("DEUTERIUM_IMAGE_REVIEW_ORIGINS"), ",")
	if len(origins) == 0 || origins[0] == "" {
		return errors.New("review public origins required")
	}
	listen := os.Getenv("DEUTERIUM_IMAGE_REVIEW_LISTEN")
	if listen == "" {
		listen = "127.0.0.1:8086"
	}
	host, _, e := net.SplitHostPort(listen)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("review listener must be loopback")
	}
	c := config.Config{Listen: listen, PublicOrigin: origins[0], AdditionalPublicOrigins: origins[1:], Development: os.Getenv("DEUTERIUM_IMAGE_REVIEW_DEVELOPMENT") == "true"}
	if e = c.Validate(); e != nil {
		return e
	}
	if _, e = audit.DB.ExecContext(ctx, "UPDATE image_review_runtime SET started_at=COALESCE(started_at,UTC_TIMESTAMP(6)) WHERE id=1"); e != nil {
		return e
	}
	var startedAt time.Time
	if e = audit.DB.QueryRowContext(ctx, "SELECT started_at FROM image_review_runtime WHERE id=1").Scan(&startedAt); e != nil {
		return e
	}
	origin := &imagereview.Source{DB: source.DB, Prefix: objects.Config.Prefix, StartAt: startedAt}
	if e = origin.Detect(ctx); e != nil {
		return errors.New("source schema unavailable")
	}
	// Dedicated lease connection, checked before each job; a disconnected owner exits.
	lease, e := audit.DB.Conn(ctx)
	if e != nil {
		return e
	}
	defer lease.Close()
	var locked int
	if e = lease.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT(DATABASE(), ':image-review-worker'),0)").Scan(&locked); e != nil || locked != 1 {
		return errors.New("image review worker already running")
	}
	defer lease.ExecContext(context.Background(), "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':image-review-worker'))")
	check := func(ctx context.Context) error {
		var valid bool
		e := lease.QueryRowContext(ctx, "SELECT IS_USED_LOCK(CONCAT(DATABASE(), ':image-review-worker'))=CONNECTION_ID()").Scan(&valid)
		if e != nil || !valid {
			return errors.New("image review worker lease lost")
		}
		return nil
	}
	service := &imagereview.Service{Repo: repo, Source: origin, Objects: &imagereview.S3Objects{Client: objects}, Reviewer: reviewer}
	server := &http.Server{Addr: listen, Handler: httpapi.ImageReviewHandler(source, c, service), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 45 * time.Second, MaxHeaderBytes: 16384}
	faults := make(chan error, 2)
	go service.RunScanner(ctx)
	go func() { faults <- service.RunWorker(ctx, check) }()
	go func() { faults <- server.ListenAndServe() }()
	slog.Info("image review started", "listen", listen, "model", imagereview.Model, "threshold", imagereview.Threshold)
	var result error
	select {
	case <-ctx.Done():
	case e = <-faults:
		if !errors.Is(e, http.ErrServerClosed) && !errors.Is(e, context.Canceled) {
			result = fmt.Errorf("review runtime failed: %w", e)
		}
	}
	stop()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
	return result
}
