package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

func Authenticate(digest string, public_key string, private_key string, now string, path string, method string) bool {
    mac := hmac.New(sha256.New, []byte(private_key))
    signature := public_key + now + path + method
    mac.Write([]byte(signature))
    sum := mac.Sum(nil)

    local := fmt.Sprintf("%x", []byte(sum))

    return hmac.Equal([]byte(local), []byte(digest))
}

func Authorize() {
    // Method stub
    return
}
