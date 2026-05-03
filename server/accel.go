package main

import (
	"encoding/binary"
	"math"
	"time"
)

const (
	countsToMS2 = 0.0039 * 9.80665 // 3.9 mg/count → m/s²
	accelHz     = 100.0
)

func handleAccelPayload(payload []byte, nSamples int, deviceTsMs uint32, ts time.Time) {
	if len(payload) < nSamples*6 {
		return
	}

	mag := make([]float64, nSamples)
	var peakSq, sumSq float64

	for i := range mag {
		xi := int16(binary.LittleEndian.Uint16(payload[i*6:]))
		yi := int16(binary.LittleEndian.Uint16(payload[i*6+2:]))
		zi := int16(binary.LittleEndian.Uint16(payload[i*6+4:]))

		xms2 := float64(xi) * countsToMS2
		yms2 := float64(yi) * countsToMS2
		zms2 := float64(zi) * countsToMS2

		m2 := xms2*xms2 + yms2*yms2 + zms2*zms2
		mag[i] = math.Sqrt(m2)
		sumSq += m2
		if m2 > peakSq {
			peakSq = m2
		}
	}

	rms := math.Sqrt(sumSq / float64(nSamples))
	peak := math.Sqrt(peakSq)
	domHz := dominantHz(mag, accelHz)

	dbInsertAccel(ts, deviceTsMs, rms, peak, domHz)
}

// dominantHz returns the frequency bin with the most power in the DFT of mag.
// Skips bin 0 (DC) since the HP filter on the device already removes it.
func dominantHz(mag []float64, sampleHz float64) float64 {
	n := len(mag)
	if n == 0 {
		return 0
	}
	maxPow := 0.0
	maxBin := 1
	for k := 1; k <= n/2; k++ {
		var re, im float64
		for j, v := range mag {
			angle := 2 * math.Pi * float64(k) * float64(j) / float64(n)
			re += v * math.Cos(angle)
			im -= v * math.Sin(angle)
		}
		pow := re*re + im*im
		if pow > maxPow {
			maxPow = pow
			maxBin = k
		}
	}
	return float64(maxBin) * sampleHz / float64(n)
}
