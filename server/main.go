package main

import (
	"encoding/binary"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	magic0      = 0xAB
	magic1      = 0xCD
	sensorAccel = 0x02
	headerSize  = 9
	listenAddr  = ":8080"
)

func main() {
	// PostgreSQL is optional — set DATABASE_URL to enable.
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		if err := initDB(dsn); err != nil {
			log.Printf("postgres unavailable (%v) — running without DB", err)
		} else {
			log.Printf("postgres connected")
		}
	}

	http.HandleFunc("/data", handleData)

	log.Printf("listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func handleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("read body: %v", err)
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	if len(body) < headerSize {
		log.Printf("packet too short: %d bytes", len(body))
		http.Error(w, "packet too short", http.StatusBadRequest)
		return
	}
	if body[0] != magic0 || body[1] != magic1 {
		log.Printf("bad magic: %02x %02x", body[0], body[1])
		http.Error(w, "bad magic", http.StatusBadRequest)
		return
	}

	sensorID  := body[2]
	deviceTsMs := binary.BigEndian.Uint32(body[3:7])
	nSamples  := int(binary.BigEndian.Uint16(body[7:9]))
	payload   := body[headerSize:]
	now       := time.Now()

	switch sensorID {
	case sensorAccel:
		log.Printf("accel device_ts=%dms  samples=%d  payload=%d B", deviceTsMs, nSamples, len(payload))
		handleAccelPayload(payload, nSamples, deviceTsMs, now)
	default:
		log.Printf("unknown sensor 0x%02x", sensorID)
	}

	w.WriteHeader(http.StatusOK)
}
