package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"

	"github.com/parcelpigeon/tracking-service/internal/projection"
	"github.com/parcelpigeon/tracking-service/internal/store"
	"github.com/parcelpigeon/tracking-service/internal/upstream"
)

type stubFallback struct {
	state projection.State
	err   error
	calls int
}

func (s *stubFallback) Track(context.Context, string) (projection.State, error) {
	s.calls++
	return s.state, s.err
}

func newAPI(t *testing.T, fb Fallback) (*API, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	return &API{
		Store:    store.NewRedisStore(mr.Addr()),
		Fallback: fb,
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, mr
}

func do(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestGetTrackServesFromCache(t *testing.T) {
	fb := &stubFallback{err: upstream.ErrNotFound}
	api, _ := newAPI(t, fb)
	_ = api.Store.Save(context.Background(), projection.State{
		TrackingNumber: "PP-CACHE001", Status: "IN_TRANSIT",
	})

	rec := do(t, api.Router(), "/track/PP-CACHE001")
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var body projection.State
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Status != "IN_TRANSIT" {
		t.Errorf("want IN_TRANSIT, got %s", body.Status)
	}
	if fb.calls != 0 {
		t.Errorf("fallback should not be called on a cache hit, got %d calls", fb.calls)
	}
}

func TestGetTrackFallsBackAndWarmsCache(t *testing.T) {
	fb := &stubFallback{state: projection.State{TrackingNumber: "PP-MISS0001", Status: "CREATED"}}
	api, _ := newAPI(t, fb)

	if rec := do(t, api.Router(), "/track/PP-MISS0001"); rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if fb.calls != 1 {
		t.Fatalf("want 1 fallback call, got %d", fb.calls)
	}
	// second call should be a cache hit now
	if rec := do(t, api.Router(), "/track/PP-MISS0001"); rec.Code != http.StatusOK {
		t.Fatalf("want 200 on warm cache, got %d", rec.Code)
	}
	if fb.calls != 1 {
		t.Errorf("cache not warmed: fallback called %d times", fb.calls)
	}
}

func TestGetTrackUnknownReturns404(t *testing.T) {
	fb := &stubFallback{err: upstream.ErrNotFound}
	api, _ := newAPI(t, fb)
	if rec := do(t, api.Router(), "/track/PP-NOPE0001"); rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestHealthz(t *testing.T) {
	api, _ := newAPI(t, &stubFallback{})
	if rec := do(t, api.Router(), "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestReadyzFailsWhenRedisDown(t *testing.T) {
	api, mr := newAPI(t, &stubFallback{})
	mr.Close()
	if rec := do(t, api.Router(), "/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
}
