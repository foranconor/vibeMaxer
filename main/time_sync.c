#include "time_sync.h"
#include "esp_netif_sntp.h"
#include "esp_log.h"
#include "config.h"
#include <time.h>

#define TAG "time_sync"

esp_err_t time_sync(void)
{
    esp_sntp_config_t cfg = ESP_NETIF_SNTP_DEFAULT_CONFIG(NTP_SERVER);
    esp_netif_sntp_init(&cfg);

    esp_err_t err = esp_netif_sntp_sync_wait(pdMS_TO_TICKS(10000));
    esp_netif_sntp_deinit();

    if (err != ESP_OK) {
        ESP_LOGE(TAG, "SNTP sync failed: %s", esp_err_to_name(err));
        return err;
    }

    setenv("TZ", TIMEZONE, 1);
    tzset();

    time_t now = time(NULL);
    struct tm t;
    localtime_r(&now, &t);
    ESP_LOGI(TAG, "synced: %04d-%02d-%02d %02d:%02d:%02d (local)",
             t.tm_year + 1900, t.tm_mon + 1, t.tm_mday,
             t.tm_hour, t.tm_min, t.tm_sec);
    return ESP_OK;
}
