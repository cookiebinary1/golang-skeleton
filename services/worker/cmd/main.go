// worker is the demo background service: consumes domain events from NATS
// JetStream (durable consumer) and processes them.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cookiebinary1/golang-skeleton/pkg/config"
	"github.com/cookiebinary1/golang-skeleton/pkg/events"
	"github.com/cookiebinary1/golang-skeleton/pkg/httpx"
	"github.com/cookiebinary1/golang-skeleton/pkg/logger"
	"github.com/cookiebinary1/golang-skeleton/pkg/natsx"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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

	consumer, err := js.CreateOrUpdateConsumer(ctx, natsx.StreamName, jetstream.ConsumerConfig{
		Durable:       "worker",
		FilterSubject: events.SubjectUserCreated,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
	})
	if err != nil {
		log.Error("consumer setup failed", "err", err)
		return err
	}

	// Shared router gives worker its own liveness/readiness endpoints,
	// which Kubernetes probes target.
	router := httpx.NewRouter(log, func(ctx context.Context) error {
		if nc.Status() != nats.CONNECTED {
			return context.Canceled
		}
		return nil
	})

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		log.Info("event received",
			"subject", msg.Subject(),
			"payload", string(msg.Data()),
		)
		// Demo processing ends here; real services would do work, then Ack.
		if err := msg.Ack(); err != nil {
			log.Warn("ack failed", "err", err)
		}
	})
	if err != nil {
		log.Error("consume failed", "err", err)
		return err
	}
	defer cc.Stop()

	log.Info("worker consuming events", "stream", natsx.StreamName, "durable", "worker")

	return httpx.Run(ctx, log, httpx.NewServer(cfg.HTTPAddr, router))
}
