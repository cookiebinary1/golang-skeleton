// Package store is the api service's data access layer (pgx + embedded migrations).
package store

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cookiebinary1/golang-skeleton/pkg/events"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User is the demo domain entity.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies pending SQL migrations embedded in the binary at startup.
// This keeps deployments simple: no separate migration job or CLI needed.
func Migrate(dsn string, log *slog.Logger) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migration source: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+trimSchemePrefix(dsn))
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	log.Info("database migrations applied")
	return nil
}

func trimSchemePrefix(dsn string) string {
	for _, p := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(p) && dsn[:len(p)] == p {
			return dsn[len(p):]
		}
	}
	return dsn
}

// Store wraps the pool with typed queries.
type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Create(ctx context.Context, u User) (User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	row := tx.QueryRow(ctx,
		`INSERT INTO users (name, email) VALUES ($1, $2)
		 RETURNING id, name, email, created_at`,
		u.Name, u.Email)
	var out User
	err = row.Scan(&out.ID, &out.Name, &out.Email, &out.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return User{}, ErrDuplicateEmail
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	payload, err := json.Marshal(events.UserCreated{ID: out.ID, Name: out.Name, Email: out.Email, Timestamp: out.CreatedAt})
	if err != nil {
		return User{}, err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO outbox (subject, payload) VALUES ($1, $2)", events.SubjectUserCreated, payload); err != nil {
		return User{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}
	return out, nil
}

func (s *Store) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, email, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) GetByEmail(ctx context.Context, email string) (User, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, name, email, created_at FROM users WHERE email = $1`, email)
	var u User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// ErrNotFound / ErrDuplicateEmail are domain errors the handler maps to HTTP codes.
var (
	ErrNotFound       = errors.New("not found")
	ErrDuplicateEmail = errors.New("email already exists")
)

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
