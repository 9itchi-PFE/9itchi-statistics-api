package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewEventConsumerWithoutURLIsNoop(t *testing.T) {
	c := NewEventConsumer("", "q", []string{"a"})
	assert.Equal(t, NoopEventConsumer{}, c)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx, nil) }()
	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("the no-op consumer did not stop")
	}
}

func TestNewEventConsumerWithURLIsRabbit(t *testing.T) {
	assert.IsType(t, &RabbitConsumer{}, NewEventConsumer("amqp://guest:guest@localhost:5672/", "q", nil))
}

func TestRedactAMQPHidesURLAndPassword(t *testing.T) {
	const url = "amqp://app:s3cr3t-pw@broker.internal:5672/"
	msg := redactAMQP(errors.New(`parse "`+url+`": invalid port; password s3cr3t-pw`), url)
	assert.NotContains(t, msg, "s3cr3t-pw")
	assert.NotContains(t, msg, "broker.internal")
	assert.Empty(t, redactAMQP(nil, url))
}
