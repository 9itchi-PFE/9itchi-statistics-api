package core

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"9itchi-statistics-api/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	// EventsExchange is the topic exchange every service publishes its events to; the routing key is the event name.
	EventsExchange = "9itchi.events"
	// EventIDKey is where the consumer puts the id of the message in the payload, so a handler can ignore a replay.
	EventIDKey = "_event_id"

	rabbitDialTimeout  = 5 * time.Second
	rabbitMinBackoff   = time.Second
	rabbitMaxBackoff   = 30 * time.Second
	rabbitPrefetch     = 10
	rabbitHandlerLimit = 30 * time.Second
)

// EventHandler handles one event. An error asks for one more delivery; a second failure drops the message.
type EventHandler func(ctx context.Context, event string, payload map[string]any) error

// EventConsumer delivers the events of the queue to handler until ctx ends.
type EventConsumer interface {
	Run(ctx context.Context, handler EventHandler) error
}

// NoopEventConsumer is the consumer when RABBITMQ_URL is empty: it waits for the end and delivers nothing.
type NoopEventConsumer struct{}

func (NoopEventConsumer) Run(ctx context.Context, _ EventHandler) error {
	<-ctx.Done()
	return nil
}

// RabbitConsumer reads a durable queue bound to EventsExchange for the given event names. It reconnects on its own.
type RabbitConsumer struct {
	url   string
	queue string
	keys  []string
}

// NewEventConsumer is the consumer factory: RabbitMQ when url is set, else the no-op.
func NewEventConsumer(url, queue string, keys []string) EventConsumer {
	if url == "" {
		return NoopEventConsumer{}
	}
	return &RabbitConsumer{url: url, queue: queue, keys: keys}
}

func (c *RabbitConsumer) Run(ctx context.Context, handler EventHandler) error {
	backoff := rabbitMinBackoff
	for {
		connected, err := c.session(ctx, handler)
		if ctx.Err() != nil {
			return nil
		}
		if connected {
			backoff = rabbitMinBackoff
		}
		utils.Logger.Warn("rabbitmq consumer disconnected, retrying", zap.String("queue", c.queue), zap.String("cause", redactAMQP(err, c.url)), zap.Duration("in", backoff))
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		if backoff *= 2; backoff > rabbitMaxBackoff {
			backoff = rabbitMaxBackoff
		}
	}
}

// session consumes until the connection drops or ctx ends. connected says whether the queue was reached.
func (c *RabbitConsumer) session(ctx context.Context, handler EventHandler) (connected bool, err error) {
	conn, err := amqp.DialConfig(c.url, amqp.Config{Dial: amqp.DefaultDial(rabbitDialTimeout)})
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		return false, err
	}
	if err := ch.ExchangeDeclare(EventsExchange, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return false, err
	}
	if _, err := ch.QueueDeclare(c.queue, true, false, false, false, nil); err != nil {
		return false, err
	}
	for _, key := range c.keys {
		if err := ch.QueueBind(c.queue, key, EventsExchange, false, nil); err != nil {
			return false, err
		}
	}
	if err := ch.Qos(rabbitPrefetch, 0, false); err != nil {
		return false, err
	}
	deliveries, err := ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return false, err
	}
	utils.Logger.Info("rabbitmq consumer ready", zap.String("queue", c.queue))
	for {
		select {
		case <-ctx.Done():
			return true, nil
		case d, ok := <-deliveries:
			if !ok {
				return true, errors.New("rabbitmq: delivery channel closed")
			}
			c.handle(ctx, handler, d)
		}
	}
}

// handle runs the handler on one delivery and settles it. The payload is never logged.
func (c *RabbitConsumer) handle(ctx context.Context, handler EventHandler, d amqp.Delivery) {
	var payload map[string]any
	if err := json.Unmarshal(d.Body, &payload); err != nil || payload == nil {
		utils.Logger.Warn("event dropped: not a JSON object", zap.String("event", d.RoutingKey))
		_ = d.Ack(false)
		return
	}
	if d.MessageId != "" {
		payload[EventIDKey] = d.MessageId
	}
	hctx, cancel := context.WithTimeout(ctx, rabbitHandlerLimit)
	err := handler(hctx, d.RoutingKey, payload)
	cancel()
	switch {
	case err == nil:
		_ = d.Ack(false)
	case d.Redelivered:
		utils.Logger.Error("event dropped after a second failure", zap.String("event", d.RoutingKey), zap.Error(err))
		_ = d.Ack(false)
	default:
		utils.Logger.Warn("event failed, asking for one more delivery", zap.String("event", d.RoutingKey), zap.Error(err))
		_ = d.Nack(false, true)
	}
}

// redactAMQP writes an error without the broker URL or its password, which a parse error can carry.
func redactAMQP(err error, rawURL string) string {
	if err == nil {
		return ""
	}
	msg := strings.ReplaceAll(err.Error(), rawURL, "RABBITMQ_URL")
	if u, perr := url.Parse(rawURL); perr == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok && pw != "" {
			msg = strings.ReplaceAll(msg, pw, "REDACTED")
		}
	}
	return msg
}
