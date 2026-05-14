#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "nvs_flash.h"
#include "esp_log.h"
#include "wifi.h"
#include "accel_task.h"

#define TAG "main"

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

    ESP_LOGI(TAG, "starting sensor tasks");
    xTaskCreate(accel_task, "accel", 12288, NULL, 5, NULL);

    while (1) vTaskDelay(pdMS_TO_TICKS(60000));
}
