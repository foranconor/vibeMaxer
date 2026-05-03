#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/i2c_master.h"
#include "esp_log.h"
#include "esp_http_client.h"
#include "esp_timer.h"
#include "config.h"
#include "accel_task.h"

#define TAG "accel"

// ADXL345 registers
#define REG_BW_RATE     0x2C
#define REG_POWER_CTL   0x2D
#define REG_DATA_FORMAT 0x31
#define REG_DATAX0      0x32

// Packet layout: magic(2) + sensor_id(1) + timestamp_ms(4) + num_samples(2) + samples
// Each sample: int16_t x, y, z = 6 bytes
#define HEADER_SIZE     9
#define SAMPLE_BYTES    (ACCEL_SAMPLES_PER_POST * 6)
#define PACKET_SIZE     (HEADER_SIZE + SAMPLE_BYTES)

static i2c_master_dev_handle_t s_dev;

static void build_header(uint8_t *buf, uint32_t ts_ms, uint16_t n)
{
    buf[0] = PACKET_MAGIC_0;
    buf[1] = PACKET_MAGIC_1;
    buf[2] = SENSOR_ID_ACCEL;
    buf[3] = (ts_ms >> 24) & 0xFF;
    buf[4] = (ts_ms >> 16) & 0xFF;
    buf[5] = (ts_ms >>  8) & 0xFF;
    buf[6] =  ts_ms        & 0xFF;
    buf[7] = (n >> 8) & 0xFF;
    buf[8] =  n       & 0xFF;
}

static esp_err_t reg_write(uint8_t reg, uint8_t val)
{
    uint8_t buf[2] = {reg, val};
    return i2c_master_transmit(s_dev, buf, 2, pdMS_TO_TICKS(100));
}

static esp_err_t reg_read(uint8_t reg, uint8_t *buf, size_t len)
{
    return i2c_master_transmit_receive(s_dev, &reg, 1, buf, len, pdMS_TO_TICKS(100));
}

static void adxl345_init(void)
{
    ESP_ERROR_CHECK(reg_write(REG_BW_RATE,     0x0A)); // 100 Hz output rate
    ESP_ERROR_CHECK(reg_write(REG_DATA_FORMAT, 0x0B)); // full resolution, ±16 g
    ESP_ERROR_CHECK(reg_write(REG_POWER_CTL,   0x08)); // start measuring
}

static void http_post(const uint8_t *data, size_t len)
{
    esp_http_client_config_t cfg = {
        .url                = SERVER_URL,
        .method             = HTTP_METHOD_POST,
        .timeout_ms = 3000,
    };
    esp_http_client_handle_t client = esp_http_client_init(&cfg);
    esp_http_client_set_header(client, "Content-Type", "application/octet-stream");
    esp_http_client_set_post_field(client, (const char *)data, (int)len);
    esp_err_t err = esp_http_client_perform(client);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "POST failed: %s", esp_err_to_name(err));
    }
    esp_http_client_cleanup(client);
}

void accel_task(void *pv)
{
    i2c_master_bus_config_t bus_cfg = {
        .i2c_port            = I2C_NUM_0,
        .sda_io_num          = I2C_SDA_GPIO,
        .scl_io_num          = I2C_SCL_GPIO,
        .clk_source          = I2C_CLK_SRC_DEFAULT,
        .glitch_ignore_cnt   = 7,
        .flags.enable_internal_pullup = true,
    };
    i2c_master_bus_handle_t bus;
    ESP_ERROR_CHECK(i2c_new_master_bus(&bus_cfg, &bus));

    i2c_device_config_t dev_cfg = {
        .dev_addr_length = I2C_ADDR_BIT_LEN_7,
        .device_address  = ADXL345_I2C_ADDR,
        .scl_speed_hz    = 400000,
    };
    ESP_ERROR_CHECK(i2c_master_bus_add_device(bus, &dev_cfg, &s_dev));

    adxl345_init();

    // First-order high-pass filter — removes DC gravity component.
    // alpha = RC / (RC + dt), RC = 1/(2*pi*fc), fc=0.5 Hz, dt=0.01 s → alpha≈0.97
    static const float HP_ALPHA = 0.97f;
    static float hp_x = 0.0f, hp_y = 0.0f, hp_z = 0.0f;
    static int16_t prev_x = 0, prev_y = 0, prev_z = 0;

    static uint8_t packet[PACKET_SIZE];
    int16_t *samples = (int16_t *)(packet + HEADER_SIZE);
    int count = 0;
    uint32_t batch_ts = 0;

    while (1) {
        if (count == 0) {
            batch_ts = (uint32_t)(esp_timer_get_time() / 1000);
        }

        uint8_t raw[6];
        if (reg_read(REG_DATAX0, raw, 6) == ESP_OK) {
            // ADXL345 is little-endian: [X_L, X_H, Y_L, Y_H, Z_L, Z_H]
            int16_t rx = (int16_t)((raw[1] << 8) | raw[0]);
            int16_t ry = (int16_t)((raw[3] << 8) | raw[2]);
            int16_t rz = (int16_t)((raw[5] << 8) | raw[4]);

            hp_x = HP_ALPHA * (hp_x + rx - prev_x);
            hp_y = HP_ALPHA * (hp_y + ry - prev_y);
            hp_z = HP_ALPHA * (hp_z + rz - prev_z);

            prev_x = rx; prev_y = ry; prev_z = rz;

            samples[count * 3 + 0] = (int16_t)hp_x;
            samples[count * 3 + 1] = (int16_t)hp_y;
            samples[count * 3 + 2] = (int16_t)hp_z;
            count++;
        } else {
            ESP_LOGW(TAG, "read failed");
        }

        if (count >= ACCEL_SAMPLES_PER_POST) {
            build_header(packet, batch_ts, ACCEL_SAMPLES_PER_POST);
            http_post(packet, PACKET_SIZE);
            count = 0;
        }

        vTaskDelay(pdMS_TO_TICKS(ACCEL_POLL_MS));
    }
}
