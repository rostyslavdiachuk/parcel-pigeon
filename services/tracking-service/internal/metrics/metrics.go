package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	CacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tracking_cache_hits_total", Help: "Tracking lookups served from Redis",
	})
	CacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tracking_cache_misses_total", Help: "Tracking lookups that fell back to shipments-service",
	})
	EventsConsumed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tracking_events_consumed_total", Help: "Domain events consumed from RabbitMQ",
	}, []string{"outcome"})
	NotificationsSent = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tracking_notifications_sent_total", Help: "Delivery notification emails sent",
	})
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total", Help: "HTTP requests handled",
	}, []string{"method", "path", "status"})
	latency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration_seconds", Help: "HTTP request latency",
	}, []string{"method", "path"})
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware records RED metrics. Pass a stable route label (e.g. "/track/{tn}")
// rather than the raw path to keep cardinality bounded.
func Middleware(route string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		start := time.Now()
		next(rec, r)
		latency.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		requests.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
	}
}
