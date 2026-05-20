-- +goose Up
CREATE TABLE IF NOT EXISTS metric (
	id varchar NOT NULL,
	metric_type varchar NOT NULL,
	delta bigint NULL,
	value double precision NULL,
	updated_at timestamp with time zone NOT NULL DEFAULT now(),
	CONSTRAINT newtable_pk PRIMARY KEY (id, metric_type)
);
CREATE INDEX IF NOT EXISTS metric_type_idx ON metric (metric_type);

-- +goose Down
DROP INDEX IF EXISTS metric_type_idx;
DROP TABLE IF EXISTS metric;
