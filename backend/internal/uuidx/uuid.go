// Package uuidx generates RFC 4122 version-4 UUIDs using only the
// standard library, so the project does not need an extra dependency
// just for ID generation.
package uuidx

import (
	"crypto/rand"
	"fmt"
)

func New() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("uuidx: failed to read random bytes: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
