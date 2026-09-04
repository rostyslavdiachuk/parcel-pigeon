package projection

import "testing"

func msg(status, eventType, occurredAt string) Message {
	return Message{
		TrackingNumber: "PP-TEST0001",
		Status:         status,
		EventType:      eventType,
		Location:       "Hub",
		OccurredAt:     occurredAt,
		Origin:         "London",
		Destination:    "Paris",
	}
}

func TestReduceAppendsEventsInOrder(t *testing.T) {
	var s State
	s, changed := Reduce(s, msg("IN_TRANSIT", "PICKED_UP", "2026-01-01T10:00:00Z"), "now")
	if !changed {
		t.Fatal("first event should change state")
	}
	s, _ = Reduce(s, msg("DELIVERED", "DELIVERED", "2026-01-03T10:00:00Z"), "now")
	// deliver an out-of-order earlier scan
	s, _ = Reduce(s, msg("OUT_FOR_DELIVERY", "OUT_FOR_DELIVERY", "2026-01-02T10:00:00Z"), "now")

	if len(s.Events) != 3 {
		t.Fatalf("want 3 events, got %d", len(s.Events))
	}
	want := []string{
		"2026-01-01T10:00:00Z",
		"2026-01-02T10:00:00Z",
		"2026-01-03T10:00:00Z",
	}
	for i, w := range want {
		if s.Events[i].OccurredAt != w {
			t.Errorf("event %d: want %s, got %s", i, w, s.Events[i].OccurredAt)
		}
	}
	if s.Status != "DELIVERED" {
		t.Errorf("status: want DELIVERED, got %s", s.Status)
	}
}

func TestReduceIsIdempotent(t *testing.T) {
	var s State
	m := msg("IN_TRANSIT", "PICKED_UP", "2026-01-01T10:00:00Z")
	s, _ = Reduce(s, m, "now")
	s, changed := Reduce(s, m, "now")
	if changed {
		t.Fatal("replaying the same message must not change state")
	}
	if len(s.Events) != 1 {
		t.Fatalf("want 1 event after replay, got %d", len(s.Events))
	}
}

func TestReduceTracksPreviousStatus(t *testing.T) {
	var s State
	s, _ = Reduce(s, msg("IN_TRANSIT", "PICKED_UP", "2026-01-01T10:00:00Z"), "now")
	s, _ = Reduce(s, msg("OUT_FOR_DELIVERY", "OUT_FOR_DELIVERY", "2026-01-02T10:00:00Z"), "now")
	if s.PreviousStatus != "IN_TRANSIT" {
		t.Errorf("want previous IN_TRANSIT, got %s", s.PreviousStatus)
	}
}
