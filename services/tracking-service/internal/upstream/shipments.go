// Package upstream is the fallback client to shipments-service, used when the
// Redis read model has no entry for a tracking number yet.
package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/parcelpigeon/tracking-service/internal/projection"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: 5 * time.Second}}
}

type shipmentDTO struct {
	TrackingNumber string `json:"tracking_number"`
	RecipientName  string `json:"recipient_name"`
	RecipientEmail string `json:"recipient_email"`
	Origin         string `json:"origin"`
	Destination    string `json:"destination"`
	Status         string `json:"status"`
	ETA            string `json:"eta"`
	UpdatedAt      string `json:"updated_at"`
	Scans          []struct {
		EventType  string `json:"event_type"`
		Location   string `json:"location"`
		OccurredAt string `json:"occurred_at"`
	} `json:"scans"`
}

// ErrNotFound signals a 404 from shipments-service.
var ErrNotFound = fmt.Errorf("shipment not found upstream")

// Track fetches the authoritative record and shapes it into a read-model State.
func (c *Client) Track(ctx context.Context, trackingNumber string) (projection.State, error) {
	url := fmt.Sprintf("%s/internal/track/%s", c.baseURL, trackingNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return projection.State{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return projection.State{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return projection.State{}, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return projection.State{}, fmt.Errorf("shipments-service returned %d", resp.StatusCode)
	}

	var dto shipmentDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return projection.State{}, err
	}

	st := projection.State{
		TrackingNumber: dto.TrackingNumber,
		Status:         dto.Status,
		Origin:         dto.Origin,
		Destination:    dto.Destination,
		RecipientName:  dto.RecipientName,
		RecipientEmail: dto.RecipientEmail,
		ETA:            dto.ETA,
		UpdatedAt:      dto.UpdatedAt,
	}
	for _, s := range dto.Scans {
		st.Events = append(st.Events, projection.Event{
			EventType:  s.EventType,
			Location:   s.Location,
			OccurredAt: s.OccurredAt,
		})
	}
	return st, nil
}
