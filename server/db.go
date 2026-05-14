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

func dbInsertAccel(ts time.Time, deviceTsMs uint32, nSamples int, raw []byte) {
	if db == nil {
		return
	}
	_, err := db.Exec(
		`INSERT INTO accel_batches (ts, device_ts_ms, num_samples, raw_data)
		 VALUES ($1, $2, $3, $4)`,
		ts, int(deviceTsMs), nSamples, raw,
	)
	if err != nil {
		log.Printf("db accel: %v", err)
	}
}

