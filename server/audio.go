package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/hraban/opus"
)

const (
	micSampleRate  = 16000
	opusFrameSize  = 320  // 20 ms at 16 kHz
	samples48kHz   = 960  // 20 ms at 48 kHz (for OGG granule)
	clipDuration   = 60 * time.Second
	rawSampleRate  = 0.10 // 10 % of clips saved as raw WAV
	audioDir       = "data/audio"
)

type audioSession struct {
	mu        sync.Mutex
	enc       *opus.Encoder
	ogg       *OggWriter
	pcmBuf    []int16
	rawBuf    []int16 // non-nil when saving raw WAV for this clip
	startedAt time.Time
	opusPath  string
	rawPath   string
	nSamples  int // 16 kHz samples encoded so far
}

var audio audioSession

func initAudio() error {
	if err := os.MkdirAll(audioDir, 0755); err != nil {
		return err
	}
	var err error
	audio.enc, err = opus.NewEncoder(micSampleRate, 1, opus.AppAudio)
	if err != nil {
		return err
	}
	audio.enc.SetBitrate(32000)
	return nil
}

// handleMicPayload processes the raw int32 PCM payload from one mic packet.
func handleMicPayload(payload []byte, nSamples int, ts time.Time) {
	if len(payload) < nSamples*4 {
		log.Printf("mic payload too short: %d < %d", len(payload), nSamples*4)
		return
	}

	// Convert left-justified 24-bit int32 to int16.
	pcm := make([]int16, nSamples)
	for i := range pcm {
		raw := int32(binary.LittleEndian.Uint32(payload[i*4:]))
		pcm[i] = int16(raw >> 16)
	}

	audio.mu.Lock()
	defer audio.mu.Unlock()

	if audio.ogg == nil {
		startClip(ts)
	}

	audio.pcmBuf = append(audio.pcmBuf, pcm...)
	if audio.rawBuf != nil {
		audio.rawBuf = append(audio.rawBuf, pcm...)
	}
	audio.nSamples += nSamples

	// Encode all complete frames.
	outBuf := make([]byte, 4000)
	for len(audio.pcmBuf) >= opusFrameSize {
		n, err := audio.enc.Encode(audio.pcmBuf[:opusFrameSize], outBuf)
		if err != nil {
			log.Printf("opus encode: %v", err)
			audio.pcmBuf = audio.pcmBuf[opusFrameSize:]
			continue
		}
		if err := audio.ogg.WriteFrame(outBuf[:n], samples48kHz); err != nil {
			log.Printf("ogg write: %v", err)
		}
		audio.pcmBuf = audio.pcmBuf[opusFrameSize:]
	}

	// Rotate clip after clipDuration.
	if time.Since(audio.startedAt) >= clipDuration {
		finaliseClip()
	}
}

func startClip(ts time.Time) {
	stamp := ts.Format("20060102_150405")
	audio.opusPath = fmt.Sprintf("%s/%s.ogg", audioDir, stamp)
	audio.startedAt = ts
	audio.nSamples = 0

	var err error
	audio.ogg, err = NewOggWriter(audio.opusPath, micSampleRate)
	if err != nil {
		log.Printf("new ogg writer: %v", err)
		audio.ogg = nil
		return
	}

	if rand.Float64() < rawSampleRate {
		audio.rawPath = fmt.Sprintf("%s/%s_raw.wav", audioDir, stamp)
		audio.rawBuf = make([]int16, 0, micSampleRate*60)
	} else {
		audio.rawPath = ""
		audio.rawBuf = nil
	}

	log.Printf("audio clip started: %s (raw=%v)", audio.opusPath, audio.rawPath != "")
}

func finaliseClip() {
	if audio.ogg == nil {
		return
	}

	// Zero-pad to complete the last frame.
	if len(audio.pcmBuf) > 0 {
		padded := make([]int16, opusFrameSize)
		copy(padded, audio.pcmBuf)
		outBuf := make([]byte, 4000)
		if n, err := audio.enc.Encode(padded, outBuf); err == nil {
			_ = audio.ogg.WriteFrame(outBuf[:n], samples48kHz)
		}
		audio.pcmBuf = audio.pcmBuf[:0]
	}

	if err := audio.ogg.Close(); err != nil {
		log.Printf("close ogg: %v", err)
	}

	rawPath := ""
	if audio.rawBuf != nil {
		rawPath = audio.rawPath
		if err := writeWAV(rawPath, audio.rawBuf, micSampleRate); err != nil {
			log.Printf("write wav: %v", err)
			rawPath = ""
		}
	}

	durationMs := int(time.Since(audio.startedAt).Milliseconds())
	dbInsertAudioClip(audio.startedAt, durationMs, audio.opusPath, rawPath)
	log.Printf("audio clip done: %s  dur=%ds  raw=%v", audio.opusPath, durationMs/1000, rawPath != "")

	audio.ogg = nil
	audio.rawBuf = nil
	audio.rawPath = ""
	audio.opusPath = ""
}

// FinaliseAudioClip is called on server shutdown to flush the in-progress clip.
func FinaliseAudioClip() {
	audio.mu.Lock()
	defer audio.mu.Unlock()
	finaliseClip()
}

func writeWAV(path string, pcm []int16, rate int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dataLen := len(pcm) * 2
	// RIFF header
	binary.Write(f, binary.LittleEndian, [4]byte{'R', 'I', 'F', 'F'})
	binary.Write(f, binary.LittleEndian, uint32(36+dataLen))
	binary.Write(f, binary.LittleEndian, [4]byte{'W', 'A', 'V', 'E'})
	// fmt chunk
	binary.Write(f, binary.LittleEndian, [4]byte{'f', 'm', 't', ' '})
	binary.Write(f, binary.LittleEndian, uint32(16))   // chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))    // PCM
	binary.Write(f, binary.LittleEndian, uint16(1))    // mono
	binary.Write(f, binary.LittleEndian, uint32(rate))
	binary.Write(f, binary.LittleEndian, uint32(rate*2)) // byte rate
	binary.Write(f, binary.LittleEndian, uint16(2))    // block align
	binary.Write(f, binary.LittleEndian, uint16(16))   // bits per sample
	// data chunk
	binary.Write(f, binary.LittleEndian, [4]byte{'d', 'a', 't', 'a'})
	binary.Write(f, binary.LittleEndian, uint32(dataLen))
	return binary.Write(f, binary.LittleEndian, pcm)
}
