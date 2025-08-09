// Package events defines the shared event schema (subjects and payloads).
// Producers and consumers import this package so contracts stay in one place.
package events

import "time"

// Subjects used on the APP JetStream stream.
const (
	SubjectUserCreated = "events.user.created"
)

// UserCreated is published by the api service after a user row is committed.
type UserCreated struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}
