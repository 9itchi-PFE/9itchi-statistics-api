-- +goose Up
-- The id of every event already counted (the message id the publisher sets). The broker can deliver an event
-- twice; the second time its id is here and nothing is added. Ids only: no user, no payload.
CREATE TABLE counted_events (
  event_id text PRIMARY KEY CHECK (length(event_id) BETWEEN 1 AND 200),
  counted_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE counted_events;
