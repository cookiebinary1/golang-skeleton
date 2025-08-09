// Package natsx wraps NATS connection and JetStream setup shared by services.
package natsx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamName is the application-wide JetStream stream. All topics live under "events.>".
const (
	StreamName   = "APP"
	StreamFilter = "events.>"
)

// Connect establishes a NATS connection with retries (services race the
// NATS container at startup) and returns the core connection plus JetStream handle.
func Connect(ctx context.Context, log *slog.Logger, url, name string) (*nats.Conn, jetstream.JetStream, error) {
	var (
		nc  *nats.Conn
		err error
	)
	const attempts = 30
	for i := 1; i <= attempts; i++ {
		nc, err = nats.Connect(url,
			nats.Name(name),
			nats.Timeout(3*time.Second),
			nats.MaxReconnects(-1),
			nats.ReconnectWait(2*time.Second),
			nats.ConnectHandler(func(c *nats.Conn) { log.Info("connected to nats", "url", c.ConnectedUrl()) }),
			nats.DisconnectErrHandler(func(_ *nats.Conn, e error) { log.Warn("nats disconnected", "err", e) }),
		)
		if err == nil {
			break
		}
		if i == attempts {
			return nil, nil, fmt.Errorf("nats not reachable after %d attempts: %w", attempts, err)
		}
		log.Warn("nats not ready, retrying", "attempt", i, "err", err)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("jetstream: %w", err)
	}
	return nc, js, nil
}

// EnsureStream creates the application stream if missing, or updates its config.
func EnsureStream(ctx context.Context, js jetstream.JetStream, log *slog.Logger) (jetstream.Stream, error) {
	s, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{StreamFilter},
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    7 * 24 * time.Hour,
		MaxBytes:  64 << 20,
	})
	if err != nil {
		return nil, fmt.Errorf("ensure stream %s: %w", StreamName, err)
	}
	log.Info("jetstream stream ready", "stream", StreamName)
	return s, nil
}

// PublishJSON publishes v as JSON payload on subject.
func PublishJSON(ctx context.Context, js jetstream.JetStream, subject string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if _, err := js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}
