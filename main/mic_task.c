#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/i2s_std.h"
#include "esp_log.h"
#include "esp_http_client.h"
#include "esp_timer.h"
#include "config.h"
#include "mic_task.h"

#define TAG "mic"

// Packet layout: magic(2) + sensor_id(1) + timestamp_ms(4) + num_samples(2) + samples
#define HEADER_SIZE     9
#define SAMPLE_BYTES    (MIC_SAMPLES_PER_POST * sizeof(int32_t))
#define PACKET_SIZE     (HEADER_SIZE + SAMPLE_BYTES)

static void build_header(uint8_t *buf, uint32_t ts_ms, uint16_t n)
{
    buf[0] = PACKET_MAGIC_0;
    buf[1] = PACKET_MAGIC_1;
    buf[2] = SENSOR_ID_MIC;
    buf[3] = (ts_ms >> 24) & 0xFF;
    buf[4] = (ts_ms >> 16) & 0xFF;
    buf[5] = (ts_ms >>  8) & 0xFF;
    buf[6] =  ts_ms        & 0xFF;
    buf[7] = (n >> 8) & 0xFF;
    buf[8] =  n       & 0xFF;
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

void mic_task(void *pv)
{
    i2s_chan_handle_t rx;
    i2s_chan_config_t chan_cfg = I2S_CHANNEL_DEFAULT_CONFIG(I2S_NUM_0, I2S_ROLE_MASTER);
    ESP_ERROR_CHECK(i2s_new_channel(&chan_cfg, NULL, &rx));

    i2s_std_slot_config_t slot_cfg = I2S_STD_PHILIPS_SLOT_DEFAULT_CONFIG(
                        I2S_DATA_BIT_WIDTH_32BIT, I2S_SLOT_MODE_MONO);
    slot_cfg.slot_mask = I2S_STD_SLOT_LEFT;

    i2s_std_config_t std_cfg = {
        .clk_cfg  = I2S_STD_CLK_DEFAULT_CONFIG(MIC_SAMPLE_RATE_HZ),
        .slot_cfg = slot_cfg,
        .gpio_cfg = {
            .mclk = I2S_GPIO_UNUSED,
            .bclk = I2S_BCLK_GPIO,
            .ws   = I2S_WS_GPIO,
            .dout = I2S_GPIO_UNUSED,
            .din  = I2S_DIN_GPIO,
        },
    };
    ESP_ERROR_CHECK(i2s_channel_init_std_mode(rx, &std_cfg));
    ESP_ERROR_CHECK(i2s_channel_enable(rx));

    // Discard the first buffer — I2S startup transient takes ~100ms to settle
    static int32_t flush[MIC_SAMPLES_PER_POST];
    size_t dummy;
    i2s_channel_read(rx, flush, sizeof(flush), &dummy, portMAX_DELAY);

    // Static so the 4 KB sample buffer doesn't live on the task stack
    static uint8_t packet[PACKET_SIZE];
    int32_t *samples = (int32_t *)(packet + HEADER_SIZE);
    size_t bytes_read;

    while (1) {
        uint32_t ts = (uint32_t)(esp_timer_get_time() / 1000);

        // Fill the sample region of the packet via I2S DMA — blocks until full
        size_t total = 0;
        while (total < SAMPLE_BYTES) {
            i2s_channel_read(rx, (uint8_t *)samples + total,
                             SAMPLE_BYTES - total, &bytes_read, portMAX_DELAY);
            total += bytes_read;
        }

        build_header(packet, ts, MIC_SAMPLES_PER_POST);
        http_post(packet, PACKET_SIZE);
    }
}
