// Package consumer folds RabbitMQ shipment events into the Redis read model.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/parcelpigeon/tracking-service/internal/metrics"
	"github.com/parcelpigeon/tracking-service/internal/notify"
	"github.com/parcelpigeon/tracking-service/internal/projection"
	"github.com/parcelpigeon/tracking-service/internal/store"
)

type Consumer struct {
	URL      string
	Exchange string
	Queue    string
	Store    store.Store
	Mailer   *notify.Mailer
	Log      *slog.Logger
}

// Run connects and consumes until ctx is cancelled, reconnecting on failure.
func (c *Consumer) Run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := c.consume(ctx); err != nil {
			c.Log.Warn("consumer loop ended, retrying in 3s", "err", err)
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
			}
		}
	}
}

func (c *Consumer) consume(ctx context.Context) error {
	conn, err := amqp.Dial(c.URL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(c.Exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	q, err := ch.QueueDeclare(c.Queue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err := ch.QueueBind(q.Name, "shipment.#", c.Exchange, false, nil); err != nil {
		return err
	}
	if err := ch.Qos(20, 0, false); err != nil {
		return err
	}

	deliveries, err := ch.Consume(q.Name, "tracking-service", false, false, false, false, nil)
	if err != nil {
		return err
	}
	c.Log.Info("consuming", "queue", q.Name, "exchange", c.Exchange)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return amqp.ErrClosed
			}
			c.handle(ctx, d)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, d amqp.Delivery) {
	var msg projection.Message
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		c.Log.Error("bad message, dropping", "err", err)
		metrics.EventsConsumed.WithLabelValues("invalid").Inc()
		_ = d.Nack(false, false) // don't requeue a poison message
		return
	}

	current, err := c.Store.Get(ctx, msg.TrackingNumber)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		c.Log.Error("store get failed", "err", err)
		metrics.EventsConsumed.WithLabelValues("error").Inc()
		_ = d.Nack(false, true)
		return
	}

	next, changed := projection.Reduce(current, msg, time.Now().UTC().Format(time.RFC3339))
	if changed {
		if err := c.Store.Save(ctx, next); err != nil {
			c.Log.Error("store save failed", "err", err)
			metrics.EventsConsumed.WithLabelValues("error").Inc()
			_ = d.Nack(false, true)
			return
		}
		c.maybeNotify(ctx, current.Status, next)
	}

	metrics.EventsConsumed.WithLabelValues("ok").Inc()
	_ = d.Ack(false)
}

func (c *Consumer) maybeNotify(ctx context.Context, prevStatus string, st projection.State) {
	if st.Status != "DELIVERED" || prevStatus == "DELIVERED" || c.Mailer == nil {
		return
	}
	sent, err := c.Mailer.SendDelivered(st.RecipientName, st.RecipientEmail, st.TrackingNumber, st.Destination)
	if err != nil {
		c.Log.Warn("delivery email failed", "err", err)
		return
	}
	metrics.NotificationsSent.Inc()
	_ = c.Store.AddNotification(ctx, store.Notification{
		TrackingNumber: st.TrackingNumber,
		To:             sent.To,
		Subject:        sent.Subject,
		SentAt:         sent.SentAt.Format(time.RFC3339),
	})
}
