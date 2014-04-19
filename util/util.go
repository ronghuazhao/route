package util

import (
    "os"
)

func GetenvDefault(key, fallback string) (value string) {
    value = os.Getenv(key)

    if value == "" {
        value = fallback
    }

    return value
}
