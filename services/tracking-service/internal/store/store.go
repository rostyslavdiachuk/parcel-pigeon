package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/parcelpigeon/tracking-service/internal/projection"
)

// ErrNotFound is returned by Get when no projection is cached for a tracking number.
var ErrNotFound = errors.New("tracking state not found")

const (
	stateKeyPrefix = "track:"
	notifyKey      = "track:notifications"
	notifyMax      = 50
)

// Notification is one simulated delivery email, kept for the /notifications feed.
type Notification struct {
	TrackingNumber string `json:"trackingNumber"`
	To             string `json:"to"`
	Subject        string `json:"subject"`
	SentAt         string `json:"sentAt"`
}

// Store is the tracking read-model persistence. Backed by Redis in production,
// swappable for a fake in tests.
type Store interface {
	Get(ctx context.Context, trackingNumber string) (projection.State, error)
	Save(ctx context.Context, state projection.State) error
	AddNotification(ctx context.Context, n Notification) error
	Notifications(ctx context.Context, limit int) ([]Notification, error)
	Ping(ctx context.Context) error
}

// RedisStore implements Store on top of go-redis.
type RedisStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisStore(addr string) *RedisStore {
	return &RedisStore{
		rdb: redis.NewClient(&redis.Options{Addr: addr}),
		ttl: 30 * 24 * time.Hour,
	}
}

func (s *RedisStore) Get(ctx context.Context, tn string) (projection.State, error) {
	raw, err := s.rdb.Get(ctx, stateKeyPrefix+tn).Bytes()
	if errors.Is(err, redis.Nil) {
		return projection.State{}, ErrNotFound
	}
	if err != nil {
		return projection.State{}, err
	}
	var st projection.State
	if err := json.Unmarshal(raw, &st); err != nil {
		return projection.State{}, err
	}
	return st, nil
}

func (s *RedisStore) Save(ctx context.Context, st projection.State) error {
	raw, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, stateKeyPrefix+st.TrackingNumber, raw, s.ttl).Err()
}

func (s *RedisStore) AddNotification(ctx context.Context, n Notification) error {
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	pipe.LPush(ctx, notifyKey, raw)
	pipe.LTrim(ctx, notifyKey, 0, notifyMax-1)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisStore) Notifications(ctx context.Context, limit int) ([]Notification, error) {
	if limit <= 0 || limit > notifyMax {
		limit = notifyMax
	}
	raws, err := s.rdb.LRange(ctx, notifyKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	out := make([]Notification, 0, len(raws))
	for _, raw := range raws {
		var n Notification
		if err := json.Unmarshal([]byte(raw), &n); err == nil {
			out = append(out, n)
		}
	}
	return out, nil
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.rdb.Ping(ctx).Err()
}
