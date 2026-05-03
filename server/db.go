package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func initDB(dsn string) error {
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	return db.Ping()
}

func dbInsertAccel(ts time.Time, deviceTsMs uint32, rms, peak, domHz float64) {
	if db == nil {
		return
	}
	_, err := db.Exec(
		`INSERT INTO accel_batches (ts, device_ts_ms, rms_ms2, peak_ms2, dom_hz)
		 VALUES ($1, $2, $3, $4, $5)`,
		ts, int(deviceTsMs), rms, peak, domHz,
	)
	if err != nil {
		log.Printf("db accel: %v", err)
	}
}

func dbInsertAudioClip(startedAt time.Time, durationMs int, opusPath, rawPath string) {
	if db == nil {
		return
	}
	var rp *string
	if rawPath != "" {
		rp = &rawPath
	}
	_, err := db.Exec(
		`INSERT INTO audio_clips (started_at, duration_ms, opus_path, raw_path)
		 VALUES ($1, $2, $3, $4)`,
		startedAt, durationMs, opusPath, rp,
	)
	if err != nil {
		log.Printf("db audio: %v", err)
	}
}
