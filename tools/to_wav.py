#!/usr/bin/env python3
"""Convert mic .bin packets from server/data/ into a single WAV file."""

import glob
import os
import struct
import sys
import wave

DATA_DIR = os.path.join(os.path.dirname(__file__), '..', 'server', 'data')
HEADER = 9
SENSOR_MIC = 0x01
SAMPLE_RATE = 16000


def parse(path):
    data = open(path, 'rb').read()
    if len(data) < HEADER or data[0] != 0xAB or data[1] != 0xCD:
        return None
    sensor_id = data[2]
    ts_ms     = struct.unpack('>I', data[3:7])[0]
    n         = struct.unpack('>H', data[7:9])[0]
    return sensor_id, ts_ms, n, data[HEADER:]


def main():
    files = sorted(glob.glob(os.path.join(DATA_DIR, 'mic_*.bin')))
    if not files:
        sys.exit(f'no mic_*.bin files in {DATA_DIR}')

    packets = []
    for f in files:
        r = parse(f)
        if r and r[0] == SENSOR_MIC:
            packets.append(r)
    packets.sort(key=lambda r: r[1])
    print(f'{len(packets)} mic packets')

    # Decode all samples first so we can normalize
    all_samples = []
    for _, _, n, payload in packets:
        # INMP441: left-justified 24-bit audio in 32-bit LE frame, shift to int16 range
        raw = struct.unpack_from(f'<{n}i', payload)
        all_samples.extend(s >> 16 for s in raw)

    peak = max(abs(s) for s in all_samples) if all_samples else 1
    scale = 32767 / peak if peak > 0 else 1.0
    print(f'peak={peak}  scale={scale:.2f}')

    out = os.path.join(DATA_DIR, 'output.wav')
    with wave.open(out, 'w') as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(SAMPLE_RATE)
        pcm = struct.pack(f'<{len(all_samples)}h',
                          *(max(-32768, min(32767, int(s * scale))) for s in all_samples))
        w.writeframes(pcm)

    print(f'written: {out}')


if __name__ == '__main__':
    main()
