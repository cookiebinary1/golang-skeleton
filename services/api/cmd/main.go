// api is the demo HTTP service: Postgres-backed users API that publishes
// domain events to NATS JetStream.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cookiebinary1/golang-skeleton/pkg/config"
	"github.com/cookiebinary1/golang-skeleton/pkg/httpx"
	"github.com/cookiebinary1/golang-skeleton/pkg/logger"
	"github.com/cookiebinary1/golang-skeleton/pkg/natsx"
	"github.com/cookiebinary1/golang-skeleton/pkg/postgres"
	"github.com/cookiebinary1/golang-skeleton/services/api/internal/handler"
	"github.com/cookiebinary1/golang-skeleton/services/api/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	log := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err := run(ctx, cfg, log); err != nil {
		log.Error("service failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	log.Info("starting service", "service", cfg.ServiceName, "env", cfg.Env)

	dsn := cfg.DatabaseDSN()

	pool, err := postgres.Connect(ctx, log, dsn)
	if err != nil {
		log.Error("postgres connect failed", "err", err)
		return err
	}
	defer pool.Close()
	if err := store.Migrate(dsn, log); err != nil {
		return err
	}

	nc, js, err := natsx.Connect(ctx, log, cfg.NATSURL, cfg.ServiceName)
	if err != nil {
		log.Error("nats connect failed", "err", err)
		return err
	}
	defer nc.Close()

	if _, err := natsx.EnsureStream(ctx, js, log); err != nil {
		log.Error("stream setup failed", "err", err)
		return err
	}

	repo := store.New(pool)
	router := httpx.NewRouter(log, func(ctx context.Context) error {
		if !nc.IsConnected() {
			return fmt.Errorf("nats disconnected")
		}
		return postgres.Ping(ctx, pool)
	})
	handler.New(log, repo).Routes(router)

	relayDone := make(chan struct{})
	go func() { defer close(relayDone); repo.Relay(ctx, js, log) }()
	defer func() { cancel(); <-relayDone }()
	if err := httpx.Run(ctx, log, httpx.NewServer(cfg.HTTPAddr, router)); err != nil {
		return err
	}
	return nil
}
