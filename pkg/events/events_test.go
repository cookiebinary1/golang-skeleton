package events

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserCreatedJSONRoundTrip(t *testing.T) {
	in := UserCreated{ID: 42, Name: "Ada", Email: "ada@example.com", Timestamp: time.Unix(1700000000, 0).UTC()}

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out UserCreated
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out != in {
		t.Fatalf("round trip mismatch: %+v != %+v", out, in)
	}
}
