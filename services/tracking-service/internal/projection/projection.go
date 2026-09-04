// Package projection holds the pure read-model reducer for shipment tracking.
//
// It has no I/O: given the current State and an incoming Message it returns the
// next State plus whether anything actually changed. This is the main unit under
// test and the reference for "an event-sourced read model" in the lectures.
package projection

import "sort"

// Message is the JSON payload published by shipments-service on the
// "parcelpigeon" topic exchange.
type Message struct {
	TrackingNumber string `json:"trackingNumber"`
	Status         string `json:"status"`
	PreviousStatus string `json:"previousStatus"`
	EventType      string `json:"eventType"`
	Location       string `json:"location"`
	OccurredAt     string `json:"occurredAt"`
	RecipientEmail string `json:"recipientEmail"`
	RecipientName  string `json:"recipientName"`
	Origin         string `json:"origin"`
	Destination    string `json:"destination"`
	ETA            string `json:"eta"`
}

// Event is one entry in the public tracking timeline.
type Event struct {
	Status     string `json:"status"`
	EventType  string `json:"eventType"`
	Location   string `json:"location"`
	OccurredAt string `json:"occurredAt"`
}

// State is the denormalized tracking record cached in Redis.
type State struct {
	TrackingNumber string  `json:"trackingNumber"`
	Status         string  `json:"status"`
	PreviousStatus string  `json:"previousStatus"`
	Origin         string  `json:"origin"`
	Destination    string  `json:"destination"`
	RecipientName  string  `json:"recipientName"`
	RecipientEmail string  `json:"recipientEmail"`
	ETA            string  `json:"eta"`
	Events         []Event `json:"events"`
	UpdatedAt      string  `json:"updatedAt"`
}

// Reduce applies msg to state and returns the new state and whether it changed.
// It is idempotent: replaying a message that was already folded in is a no-op.
func Reduce(state State, msg Message, now string) (State, bool) {
	ev := Event{
		Status:     msg.Status,
		EventType:  msg.EventType,
		Location:   msg.Location,
		OccurredAt: msg.OccurredAt,
	}
	latestSeen := ""
	for _, e := range state.Events {
		if e.OccurredAt == ev.OccurredAt && e.EventType == ev.EventType {
			return state, false // already applied
		}
		if e.OccurredAt > latestSeen {
			latestSeen = e.OccurredAt
		}
	}

	state.TrackingNumber = msg.TrackingNumber
	if msg.Origin != "" {
		state.Origin = msg.Origin
	}
	if msg.Destination != "" {
		state.Destination = msg.Destination
	}
	if msg.RecipientName != "" {
		state.RecipientName = msg.RecipientName
	}
	if msg.RecipientEmail != "" {
		state.RecipientEmail = msg.RecipientEmail
	}

	// Only the chronologically newest event drives the headline status, so a
	// late-arriving older scan is recorded without rewinding the shipment.
	if msg.OccurredAt >= latestSeen {
		state.PreviousStatus = state.Status
		state.Status = msg.Status
		state.ETA = msg.ETA
	}

	state.Events = append(state.Events, ev)
	sort.SliceStable(state.Events, func(i, j int) bool {
		return state.Events[i].OccurredAt < state.Events[j].OccurredAt
	})
	state.UpdatedAt = now
	return state, true
}
