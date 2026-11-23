-- +goose Up
-- Draft schema generated from models/db. Refine (constraints, indexes, FKs) as features land;
-- nothing is deployed yet, so editing this file is fine until the first release.
CREATE TABLE metrics (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  key text NOT NULL DEFAULT '',
  date timestamptz NOT NULL DEFAULT now(),
  value numeric(12,2) NOT NULL DEFAULT 0,
  dimensions jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE metrics;
