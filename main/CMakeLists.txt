idf_component_register(
    SRCS
        "main.c"
        "wifi.c"
        "accel_task.c"
        "time_sync.c"
    INCLUDE_DIRS "."
    REQUIRES
        nvs_flash
        esp_wifi
        esp_netif
        esp_event
        esp_driver_i2c
        esp_http_client
        esp_timer
        esp_hw_support
)
