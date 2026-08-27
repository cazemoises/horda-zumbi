package websocket

import (
	"crypto/rand"
	"encoding/hex"
)

// newClientID returns a short random identifier for a newly connected
// websocket client.
func newClientID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
