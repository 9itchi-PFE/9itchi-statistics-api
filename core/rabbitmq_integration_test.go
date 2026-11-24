//go:build integration

package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests need a RabbitMQ broker: RABBITMQ_TEST_URL=amqp://guest:guest@localhost:5672/ . They are skipped without it.
func brokerURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("RABBITMQ_TEST_URL")
	if u == "" {
		t.Skip("RABBITMQ_TEST_URL is not set")
	}
	return u
}

// publishRaw sends one message the way another service's publisher does.
func publishRaw(t *testing.T, url, key, id string, body []byte) {
	t.Helper()
	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	require.NoError(t, err)
	require.NoError(t, ch.ExchangeDeclare(EventsExchange, amqp.ExchangeTopic, true, false, false, false, nil))
	require.NoError(t, ch.Confirm(false))
	c, err := ch.PublishWithDeferredConfirmWithContext(context.Background(), EventsExchange, key, false, false,
		amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: id, Body: body})
	require.NoError(t, err)
	require.True(t, c.Wait())
}

type received struct {
	event   string
	payload map[string]any
}

// startConsumer runs a consumer on its own queue and returns what the handler saw. fail decides the handler's answer.
func startConsumer(t *testing.T, url, queue string, keys []string, fail func(n int, r received) error) (seen func() []received, stop func()) {
	seen, halt := runConsumer(t, url, queue, keys, fail)
	return seen, func() {
		halt()
		deleteQueue(url, queue)
	}
}

func deleteQueue(url, queue string) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	if ch, err := conn.Channel(); err == nil {
		_, _ = ch.QueueDelete(queue, false, false, false)
	}
}

// runConsumer is startConsumer without the cleanup: halt stops the consumer and leaves its queue in place.
func runConsumer(t *testing.T, url, queue string, keys []string, fail func(n int, r received) error) (seen func() []received, halt func()) {
	t.Helper()
	var mu sync.Mutex
	var got []received
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = NewEventConsumer(url, queue, keys).Run(ctx, func(_ context.Context, event string, payload map[string]any) error {
			mu.Lock()
			got = append(got, received{event, payload})
			n := len(got)
			mu.Unlock()
			if fail != nil {
				return fail(n, received{event, payload})
			}
			return nil
		})
	}()
	halt = func() {
		cancel()
		<-done
	}
	seen = func() []received {
		mu.Lock()
		defer mu.Unlock()
		return append([]received(nil), got...)
	}
	return seen, halt
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestConsumerDeliversBoundEventsOnly(t *testing.T) {
	url := brokerURL(t)
	queue := fmt.Sprintf("test.%d", time.Now().UnixNano())
	seen, stop := startConsumer(t, url, queue, []string{"test.wanted"}, nil)
	defer stop()
	// The queue must exist before the publish: a topic exchange drops what no queue is bound for.
	waitForQueue(t, url, queue)

	body, _ := json.Marshal(map[string]any{"user_id": "u-1", "count": 2})
	publishRaw(t, url, "test.other", "evt-0", body)
	publishRaw(t, url, "test.wanted", "evt-1", body)

	waitFor(t, "the wanted event", func() bool { return len(seen()) == 1 })
	got := seen()[0]
	assert.Equal(t, "test.wanted", got.event)
	assert.Equal(t, "u-1", got.payload["user_id"])
	assert.Equal(t, "evt-1", got.payload[EventIDKey], "the message id reaches the handler")
	time.Sleep(300 * time.Millisecond)
	assert.Len(t, seen(), 1, "an event the queue is not bound to never arrives")
}

func TestConsumerRedeliversOnceThenDrops(t *testing.T) {
	url := brokerURL(t)
	queue := fmt.Sprintf("test.%d", time.Now().UnixNano())
	seen, stop := startConsumer(t, url, queue, []string{"test.failing"}, func(int, received) error { return errors.New("always failing") })
	defer stop()
	waitForQueue(t, url, queue)

	publishRaw(t, url, "test.failing", "evt-1", []byte(`{"user_id":"u-1"}`))
	waitFor(t, "two deliveries", func() bool { return len(seen()) == 2 })
	time.Sleep(500 * time.Millisecond)
	assert.Len(t, seen(), 2, "a second failure drops the message instead of looping")
}

func TestConsumerDropsWhatIsNotJSON(t *testing.T) {
	url := brokerURL(t)
	queue := fmt.Sprintf("test.%d", time.Now().UnixNano())
	seen, stop := startConsumer(t, url, queue, []string{"test.shape"}, nil)
	defer stop()
	waitForQueue(t, url, queue)

	publishRaw(t, url, "test.shape", "evt-1", []byte(`not json`))
	publishRaw(t, url, "test.shape", "evt-2", []byte(`[1,2]`))
	publishRaw(t, url, "test.shape", "evt-3", []byte(`{"user_id":"u-1"}`))
	waitFor(t, "the valid event", func() bool { return len(seen()) == 1 })
	assert.Equal(t, "evt-3", seen()[0].payload[EventIDKey])
}

func TestConsumerKeepsEventsPublishedWhileItIsDown(t *testing.T) {
	url := brokerURL(t)
	queue := fmt.Sprintf("test.%d", time.Now().UnixNano())
	_, halt := runConsumer(t, url, queue, []string{"test.durable"}, nil)
	waitForQueue(t, url, queue)
	// Stop the consumer but keep the queue, as a restart of the service does.
	halt()

	publishRaw(t, url, "test.durable", "evt-1", []byte(`{"user_id":"u-1"}`))
	seen, stop := startConsumer(t, url, queue, []string{"test.durable"}, nil)
	defer stop()
	waitFor(t, "the event published while down", func() bool { return len(seen()) == 1 })
}

func TestConsumerStopsWithItsContextWhenTheBrokerIsUnreachable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- NewEventConsumer("amqp://guest:guest@127.0.0.1:1/", "test.unreachable", nil).Run(ctx, func(context.Context, string, map[string]any) error { return nil })
	}()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the consumer did not stop")
	}
}

func waitForQueue(t *testing.T, url, queue string) {
	t.Helper()
	conn, err := amqp.Dial(url)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	waitFor(t, "the queue", func() bool {
		ch, err := conn.Channel()
		if err != nil {
			return false
		}
		defer func() { _ = ch.Close() }()
		q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
		return err == nil && q.Consumers > 0
	})
}
