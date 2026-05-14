CREATE TABLE IF NOT EXISTS accel_batches (
    id           BIGSERIAL PRIMARY KEY,
    ts           TIMESTAMPTZ NOT NULL,
    device_ts_ms INT         NOT NULL,
    num_samples  INT         NOT NULL,
    raw_data     BYTEA       NOT NULL
);
