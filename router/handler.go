package router

import (
	"fmt"

	"api.umn.edu/route/interfaces"
	"code.google.com/p/goprotobuf/proto"
)

// The handle function is the callback for the event listener and handles
// incoming messages on the given topic.
func (router *Router) handle(topic string, code int, payload []byte) {
	switch {
	case topic == "auth":
		// Auth message
		data := &interfaces.Auth{}

		// Extract message into struct
		err := proto.Unmarshal(payload, data)
		if err != nil {
			fmt.Println("Failed to unmarshal auth message")
			return
		}

		// Store in appropriate collection based on topic
		publicKey := data.GetPublicKey()
		privateKey := data.GetPrivateKey()

		// Create a cache payload
		// This is an association from public key to private key
		payload := map[string]string{
			publicKey: privateKey,
		}

		router.cache.Set(fmt.Sprintf("key:%s", publicKey), payload)
	}
}
