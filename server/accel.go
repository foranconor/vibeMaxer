package main

import (
	"log"
	"time"
)

func handleAccelPayload(payload []byte, nSamples int, deviceTsMs uint32, ts time.Time) {
	if len(payload) < nSamples*6 {
		log.Printf("accel payload too short: %d < %d", len(payload), nSamples*6)
		return
	}
	dbInsertAccel(ts, deviceTsMs, nSamples, payload[:nSamples*6])
}
