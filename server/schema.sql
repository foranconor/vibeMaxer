CREATE TABLE IF NOT EXISTS audio_clips (
    id          BIGSERIAL PRIMARY KEY,
    started_at  TIMESTAMPTZ NOT NULL,
    duration_ms INT         NOT NULL,
    opus_path   TEXT        NOT NULL,
    raw_path    TEXT
);

CREATE TABLE IF NOT EXISTS accel_batches (
    id           BIGSERIAL PRIMARY KEY,
    ts           TIMESTAMPTZ NOT NULL,
    device_ts_ms INT         NOT NULL,
    rms_ms2      REAL        NOT NULL,
    peak_ms2     REAL        NOT NULL,
    dom_hz       REAL        NOT NULL
);
