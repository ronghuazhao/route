// Copyright 2014 Regents of the University of Minnesota. All rights reserved.
// The University of Minnesota is an equal opportunity educator and employer.
// Use of this file is governed by a license found in the license.md file.

package util

import (
	"os"
)

// GetenvDefault wraps Getenv with a fallback value if the environment variable is not set.
func GetenvDefault(key, fallback string) (value string) {
	value = os.Getenv(key)

	if value == "" {
		value = fallback
	}

	return value
}
