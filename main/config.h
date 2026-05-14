#pragma once

// WiFi
#define WIFI_SSID "!GreatStairs!"
#define WIFI_PASSWORD "8tetrexadraxe"

// Server endpoint
#define SERVER_URL "http://10.1.1.238:7788/data"

// I2C pins (ADXL345)
// Wire: CS -> 3.3V (I2C mode), SDO -> GND (addr 0x53)
#define I2C_SDA_GPIO 8
#define I2C_SCL_GPIO 9
#define ADXL345_I2C_ADDR 0x53

// Packet framing
#define PACKET_MAGIC_0 0xAB
#define PACKET_MAGIC_1 0xCD
#define SENSOR_ID_ACCEL 0x02

// Accel: 200 Hz polling, 200 samples per POST (1 s batches)
#define ACCEL_SAMPLE_RATE_HZ 200
#define ACCEL_SAMPLES_PER_POST 200
#define ACCEL_POLL_MS 5

// Work hours (local time, 24-hour)
#define WORK_HOUR_START 8
#define WORK_HOUR_END 16

// POSIX TZ string for New Zealand (NZST/NZDT)
#define TIMEZONE "NZST-12NZDT,M9.5.0,M4.1.0/3"

#define NTP_SERVER "pool.ntp.org"
