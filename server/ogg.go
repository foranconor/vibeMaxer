package main

import (
	"encoding/binary"
	"math/rand/v2"
	"os"
)

// oggCRCTable is CRC-32/MPEG-2: poly=0x04C11DB7, init=0, no reflection.
var oggCRCTable [256]uint32

func init() {
	for i := 0; i < 256; i++ {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if r&0x80000000 != 0 {
				r = (r << 1) ^ 0x04C11DB7
			} else {
				r <<= 1
			}
		}
		oggCRCTable[i] = r
	}
}

func oggChecksum(data []byte) uint32 {
	var crc uint32
	for _, b := range data {
		crc = (crc << 8) ^ oggCRCTable[byte(crc>>24)^b]
	}
	return crc
}

// OggWriter writes a mono Opus stream into an OGG container (RFC 7845).
type OggWriter struct {
	f      *os.File
	serial uint32
	seq    uint32
	pos    int64 // granule in 48 kHz samples
}

func NewOggWriter(path string, inputRate int) (*OggWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := &OggWriter{f: f, serial: rand.Uint32()}
	if err := w.writeIDHeader(inputRate); err != nil {
		f.Close()
		return nil, err
	}
	if err := w.writeCommentHeader(); err != nil {
		f.Close()
		return nil, err
	}
	return w, nil
}

// WriteFrame appends one encoded Opus frame. samplesAt48k is the number of
// input PCM samples this frame represents, converted to 48 kHz (e.g. 960 for
// a 20 ms frame regardless of input rate).
func (w *OggWriter) WriteFrame(frame []byte, samplesAt48k int) error {
	w.pos += int64(samplesAt48k)
	return w.writePage(0x00, w.pos, frame)
}

func (w *OggWriter) Close() error {
	// Mark the last page as EOS by re-writing with the EOS flag.
	// Simpler: write a zero-length EOS page.
	if err := w.writePage(0x04, w.pos, nil); err != nil {
		w.f.Close()
		return err
	}
	return w.f.Close()
}

// writePage writes a single OGG page carrying one packet (data).
func (w *OggWriter) writePage(headerFlags byte, granule int64, data []byte) error {
	// Build segment table for data.
	segs := lacingOf(len(data))

	pageLen := 27 + len(segs) + len(data)
	page := make([]byte, pageLen)

	copy(page[0:], "OggS")
	page[4] = 0           // stream structure version
	page[5] = headerFlags // 0x02=BOS, 0x04=EOS
	binary.LittleEndian.PutUint64(page[6:], uint64(granule))
	binary.LittleEndian.PutUint32(page[14:], w.serial)
	binary.LittleEndian.PutUint32(page[18:], w.seq)
	// page[22:26] = CRC (zeroed for checksum computation)
	page[26] = byte(len(segs))
	copy(page[27:], segs)
	copy(page[27+len(segs):], data)

	crc := oggChecksum(page)
	binary.LittleEndian.PutUint32(page[22:], crc)

	w.seq++
	_, err := w.f.Write(page)
	return err
}

// lacingOf returns the OGG lacing byte sequence for a packet of length n.
func lacingOf(n int) []byte {
	if n == 0 {
		return []byte{0}
	}
	segs := []byte{}
	for n > 0 {
		s := n
		if s > 255 {
			s = 255
		}
		segs = append(segs, byte(s))
		n -= s
	}
	// If the last segment is exactly 255 the packet would be ambiguous —
	// append a 0-length terminator.
	if segs[len(segs)-1] == 255 {
		segs = append(segs, 0)
	}
	return segs
}

// Opus ID header as per RFC 7845 §5.1.
func (w *OggWriter) writeIDHeader(inputRate int) error {
	hdr := make([]byte, 19)
	copy(hdr[0:], "OpusHead")
	hdr[8] = 1                                           // version
	hdr[9] = 1                                           // channels (mono)
	binary.LittleEndian.PutUint16(hdr[10:], 312)         // pre-skip at 48 kHz
	binary.LittleEndian.PutUint32(hdr[12:], uint32(inputRate)) // input sample rate
	binary.LittleEndian.PutUint16(hdr[16:], 0)           // output gain
	hdr[18] = 0                                          // channel mapping family 0

	// BOS page, granule = 0
	return w.writeBOSPage(hdr)
}

func (w *OggWriter) writeBOSPage(data []byte) error {
	segs := lacingOf(len(data))
	pageLen := 27 + len(segs) + len(data)
	page := make([]byte, pageLen)

	copy(page[0:], "OggS")
	page[4] = 0
	page[5] = 0x02 // BOS
	binary.LittleEndian.PutUint64(page[6:], 0)
	binary.LittleEndian.PutUint32(page[14:], w.serial)
	binary.LittleEndian.PutUint32(page[18:], w.seq)
	page[26] = byte(len(segs))
	copy(page[27:], segs)
	copy(page[27+len(segs):], data)

	crc := oggChecksum(page)
	binary.LittleEndian.PutUint32(page[22:], crc)

	w.seq++
	_, err := w.f.Write(page)
	return err
}

// Opus comment header as per RFC 7845 §5.2.
func (w *OggWriter) writeCommentHeader() error {
	vendor := "vibeMaxer"
	hdr := make([]byte, 8+4+len(vendor)+4)
	copy(hdr[0:], "OpusTags")
	binary.LittleEndian.PutUint32(hdr[8:], uint32(len(vendor)))
	copy(hdr[12:], vendor)
	binary.LittleEndian.PutUint32(hdr[12+len(vendor):], 0) // zero comments
	return w.writePage(0x00, 0, hdr)
}
