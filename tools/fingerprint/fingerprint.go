package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Generate creates a unique client fingerprint based on IP and User-Agent
func Generate(ipAddress, userAgent string) string {
	data := fmt.Sprintf("%s%s", ipAddress, userAgent)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
