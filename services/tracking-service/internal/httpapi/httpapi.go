// Package httpapi wires the tracking-service HTTP surface.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/parcelpigeon/tracking-service/internal/metrics"
	"github.com/parcelpigeon/tracking-service/internal/projection"
	"github.com/parcelpigeon/tracking-service/internal/store"
	"github.com/parcelpigeon/tracking-service/internal/upstream"
)

// Fallback fetches an authoritative record when the read model has no entry.
type Fallback interface {
	Track(ctx context.Context, trackingNumber string) (projection.State, error)
}

type API struct {
	Store    store.Store
	Fallback Fallback
	Log      *slog.Logger
}

func (a *API) Router() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", metrics.Middleware("/healthz", a.healthz))
	r.Get("/readyz", metrics.Middleware("/readyz", a.readyz))
	r.Get("/track/{tn}", metrics.Middleware("/track/{tn}", a.getTrack))
	r.Get("/track/{tn}/events", metrics.Middleware("/track/{tn}/events", a.getEvents))
	r.Get("/notifications", metrics.Middleware("/notifications", a.getNotifications))
	r.Handle("/metrics", promhttp.Handler())
	return r
}

func (a *API) resolve(ctx context.Context, tn string) (projection.State, error) {
	st, err := a.Store.Get(ctx, tn)
	if err == nil {
		metrics.CacheHits.Inc()
		return st, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return projection.State{}, err
	}

	metrics.CacheMisses.Inc()
	st, ferr := a.Fallback.Track(ctx, tn)
	if ferr != nil {
		return projection.State{}, ferr
	}
	// Warm the cache so the next lookup is a hit.
	if serr := a.Store.Save(ctx, st); serr != nil {
		a.Log.Warn("cache warm failed", "tracking_number", tn, "err", serr)
	}
	return st, nil
}

func (a *API) getTrack(w http.ResponseWriter, r *http.Request) {
	tn := chi.URLParam(r, "tn")
	st, err := a.resolve(r.Context(), tn)
	if err != nil {
		a.writeResolveError(w, tn, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (a *API) getEvents(w http.ResponseWriter, r *http.Request) {
	tn := chi.URLParam(r, "tn")
	st, err := a.resolve(r.Context(), tn)
	if err != nil {
		a.writeResolveError(w, tn, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trackingNumber": st.TrackingNumber,
		"status":         st.Status,
		"events":         st.Events,
	})
}

func (a *API) getNotifications(w http.ResponseWriter, r *http.Request) {
	items, err := a.Store.Notifications(r.Context(), 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *API) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	checks := map[string]bool{"redis": a.Store.Ping(r.Context()) == nil}
	ready := true
	for _, ok := range checks {
		ready = ready && ok
	}
	code := http.StatusOK
	if !ready {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"ready": ready, "checks": checks})
}

func (a *API) writeResolveError(w http.ResponseWriter, tn string, err error) {
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, upstream.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown tracking number: " + tn})
		return
	}
	a.Log.Error("resolve failed", "tracking_number", tn, "err", err)
	writeJSON(w, http.StatusBadGateway, map[string]string{"error": "tracking lookup failed"})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
