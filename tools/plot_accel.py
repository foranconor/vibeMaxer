#!/usr/bin/env python3
"""Plot ADXL345 accelerometer data from server/data/ accel_*.bin packets."""

import glob
import math
import os
import struct
import sys

import matplotlib.pyplot as plt

DATA_DIR     = os.path.join(os.path.dirname(__file__), '..', 'server', 'data')
HEADER       = 9
SENSOR_ACCEL = 0x02
RATE_HZ      = 100
COUNTS_TO_MS2 = 0.0039 * 9.80665   # 3.9 mg/count → m/s²


def parse(path):
    data = open(path, 'rb').read()
    if len(data) < HEADER or data[0] != 0xAB or data[1] != 0xCD:
        return None
    sensor_id = data[2]
    ts_ms     = struct.unpack('>I', data[3:7])[0]
    n         = struct.unpack('>H', data[7:9])[0]
    return sensor_id, ts_ms, n, data[HEADER:]


def wall_ms_from_filename(path):
    name = os.path.basename(path)   # accel_67303_1746050597123.bin
    return int(name.split('_')[2].split('.')[0])


def main():
    files = sorted(glob.glob(os.path.join(DATA_DIR, 'accel_*.bin')),
                   key=wall_ms_from_filename)
    if not files:
        sys.exit(f'no accel_*.bin files in {DATA_DIR}')

    packets = []
    for f in files:
        r = parse(f)
        if r and r[0] == SENSOR_ACCEL:
            packets.append((wall_ms_from_filename(f), *r))
    packets.sort(key=lambda r: r[0])
    print(f'{len(packets)} accel packets')

    t0 = packets[0][0] if packets else 0
    t, x, y, z, mag = [], [], [], [], []
    for wall_ms, _, ts_ms, n, payload in packets:
        for i in range(n):
            xi, yi, zi = struct.unpack_from('<3h', payload, i * 6)
            xms2 = xi * COUNTS_TO_MS2
            yms2 = yi * COUNTS_TO_MS2
            zms2 = zi * COUNTS_TO_MS2
            t.append((wall_ms - t0) / 1000.0 + i / RATE_HZ)
            x.append(xms2)
            y.append(yms2)
            z.append(zms2)
            mag.append(math.sqrt(xms2**2 + yms2**2 + zms2**2))

    fig, (ax1, ax2) = plt.subplots(2, 1, figsize=(14, 7), sharex=True)

    ax1.plot(t, x, linewidth=0.7, label='X')
    ax1.plot(t, y, linewidth=0.7, label='Y')
    ax1.plot(t, z, linewidth=0.7, label='Z')
    ax1.set_ylabel('m/s²')
    ax1.set_title('ADXL345 — axes')
    ax1.legend()
    ax1.grid(True, alpha=0.3)

    ax2.plot(t, mag, linewidth=0.7, color='black', label='|a|')
    ax2.set_xlabel('time (s)')
    ax2.set_ylabel('m/s²')
    ax2.set_title('resultant magnitude')
    ax2.legend()
    ax2.grid(True, alpha=0.3)

    plt.tight_layout()
    plt.show()


if __name__ == '__main__':
    main()
