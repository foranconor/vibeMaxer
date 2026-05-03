#pragma once

// WiFi
#define WIFI_SSID "conor"
#define WIFI_PASSWORD "password"

// Server endpoint
#define SERVER_URL "http://10.236.192.62:8080/data"

// I2S pins (INMP441)
#define I2S_BCLK_GPIO 4
#define I2S_WS_GPIO 5
#define I2S_DIN_GPIO 6

// I2C pins (ADXL345)
// Wire: CS -> 3.3V (I2C mode), SDO -> GND (addr 0x53)
#define I2C_SDA_GPIO 8
#define I2C_SCL_GPIO 9
#define ADXL345_I2C_ADDR 0x53

// Packet framing
#define PACKET_MAGIC_0 0xAB
#define PACKET_MAGIC_1 0xCD
#define SENSOR_ID_MIC 0x01
#define SENSOR_ID_ACCEL 0x02

// Mic: 16 kHz, 1024 samples per POST (~64 ms batches)
#define MIC_SAMPLE_RATE_HZ 16000
#define MIC_SAMPLES_PER_POST 1024

// Accel: 100 Hz, 100 samples per POST (1 s batches)
#define ACCEL_POLL_MS 10
#define ACCEL_SAMPLES_PER_POST 100

// Work hours (local time, 24-hour)
#define WORK_HOUR_START 8
#define WORK_HOUR_END 16

// POSIX TZ string for New Zealand (NZST/NZDT)
#define TIMEZONE "NZST-12NZDT,M9.5.0,M4.1.0/3"

#define NTP_SERVER "pool.ntp.org"
