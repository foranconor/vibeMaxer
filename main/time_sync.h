#pragma once
#include "esp_err.h"

// Fetch current time from TIME_SYNC_URL, call settimeofday, and set the
// TZ environment variable.  Returns ESP_OK on success.
esp_err_t time_sync(void);
