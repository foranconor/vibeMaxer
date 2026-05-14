#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "nvs_flash.h"
#include "esp_log.h"
#include "esp_sleep.h"
#include "wifi.h"
#include "time_sync.h"
#include "accel_task.h"
#include "config.h"
#include <time.h>

#define TAG "main"

// Return seconds until WORK_HOUR_START tomorrow (or today if still before it).
static uint64_t secs_until_work(const struct tm *t)
{
    int secs_into_day = t->tm_hour * 3600 + t->tm_min * 60 + t->tm_sec;
    int start_secs    = WORK_HOUR_START * 3600;

    if (secs_into_day < start_secs) {
        return (uint64_t)(start_secs - secs_into_day);
    }
    // After WORK_HOUR_END — sleep until tomorrow's start.
    return (uint64_t)(86400 - secs_into_day + start_secs);
}

void app_main(void)
{
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES || ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    ESP_ERROR_CHECK(ret);

    ESP_LOGI(TAG, "connecting to wifi...");
    wifi_init_sta();

    // Sync RTC via server.  Retry a few times before giving up.
    esp_err_t ts_err = ESP_FAIL;
    for (int i = 0; i < 5 && ts_err != ESP_OK; i++) {
        ts_err = time_sync();
        if (ts_err != ESP_OK) vTaskDelay(pdMS_TO_TICKS(2000));
    }

    if (ts_err != ESP_OK) {
        ESP_LOGW(TAG, "time sync failed — running without work-hour check");
    } else {
        time_t now = time(NULL);
        struct tm t;
        localtime_r(&now, &t);

        bool in_hours = (t.tm_hour >= WORK_HOUR_START && t.tm_hour < WORK_HOUR_END);
        if (!in_hours) {
            uint64_t sleep_s = secs_until_work(&t);
            ESP_LOGI(TAG, "outside work hours — sleeping %llu s until %02d:00",
                     (unsigned long long)sleep_s, WORK_HOUR_START);
            esp_sleep_enable_timer_wakeup(sleep_s * 1000000ULL);
            esp_deep_sleep_start();
        }
    }

    ESP_LOGI(TAG, "starting sensor tasks");
    xTaskCreate(accel_task, "accel", 12288, NULL, 5, NULL);

    // Monitor for end of work day — deep sleep until tomorrow's start.
    while (1) {
        vTaskDelay(pdMS_TO_TICKS(60000));

        if (ts_err != ESP_OK) continue; // no clock, run forever

        time_t now = time(NULL);
        struct tm t;
        localtime_r(&now, &t);

        if (t.tm_hour >= WORK_HOUR_END) {
            uint64_t sleep_s = secs_until_work(&t);
            ESP_LOGI(TAG, "end of work day — sleeping %llu s until %02d:00",
                     (unsigned long long)sleep_s, WORK_HOUR_START);
            esp_sleep_enable_timer_wakeup(sleep_s * 1000000ULL);
            esp_deep_sleep_start();
        }
    }
}
