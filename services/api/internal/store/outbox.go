package store

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nats-io/nats.go/jetstream"
)

// Relay publishes committed events. A row is removed only after JetStream ACKs it.
// Delivery is at least once: consumers must make their side effects idempotent.
func (s *Store) Relay(ctx context.Context, js jetstream.JetStream, log *slog.Logger) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for i := 0; i < 100; i++ {
				attempt, cancel := context.WithTimeout(ctx, 5*time.Second)
				found, err := s.relayOne(attempt, js)
				cancel()
				if err != nil {
					log.Error("outbox publish failed; will retry", "err", err)
					break
				}
				if !found {
					break
				}
			}
		}
	}
}

func (s *Store) relayOne(ctx context.Context, js jetstream.JetStream) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var id int64
	var subject string
	var payload []byte
	err = tx.QueryRow(ctx, "SELECT id, subject, payload FROM outbox ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1").Scan(&id, &subject, &payload)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Stable message IDs deduplicate retries within the stream's duplicate window.
	if _, err := js.Publish(ctx, subject, payload, jetstream.WithMsgID(fmt.Sprintf("outbox-%d", id))); err != nil {
		return true, err
	}
	if _, err := tx.Exec(ctx, "DELETE FROM outbox WHERE id = $1", id); err != nil {
		return true, err
	}
	return true, tx.Commit(ctx)
}
