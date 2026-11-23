package ingest

import (
	"crypto/sha256"
	"encoding/hex"
)

func hexSHA(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
