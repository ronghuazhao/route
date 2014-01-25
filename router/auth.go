package router

import (
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

func Authenticate(other string, user string, time string, path string, method string) bool {
    var key string

    db, _ := sql.Open("sqlite3", "/Users/ben/Code/api-auth/db/development.sqlite3")
    db.QueryRow("SELECT private_key FROM keystore WHERE public_key=?", user).Scan(&key)

    mac := hmac.New(sha256.New, []byte(key))
    signature := user + time + path + method
    mac.Write([]byte(signature))
    sum := mac.Sum(nil)

    local := fmt.Sprintf("%x", []byte(sum))

    return hmac.Equal([]byte(local), []byte(other))
}
