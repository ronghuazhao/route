// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

package router

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
)

// Authenticate takes in several request parameters along with a private key and verifies the message integrity.
func Authenticate(digest string, public_key string, private_key string, now string, path string, method string) bool {
	// Build a new hash from the public key
	mac := hmac.New(sha256.New, []byte(private_key))

	// Build a signature
	signature := public_key + now + path + method
	mac.Write([]byte(signature))

	// Compute the hash
	sum := mac.Sum(nil)

	// Get hex representation of the sum
	local := fmt.Sprintf("%x", []byte(sum))

	// Safely compare the hashes
	return hmac.Equal([]byte(local), []byte(digest))
}

// Authorize verifies that the entity requesting a resource is allowed to do so.
func Authorize() {
	return
}
