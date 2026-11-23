-- +goose Up
CREATE TABLE metrics (
  name text NOT NULL,
  day date NOT NULL,
  value bigint NOT NULL DEFAULT 0 CHECK (value >= 0),
  PRIMARY KEY (name, day)
);

-- +goose Down
DROP TABLE metrics;
