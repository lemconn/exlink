package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// UUID16 generates a 16-character hexadecimal random string (similar to CCXT's uuid16)
// Equivalent to Python: format(random.getrandbits(16 * 4), 'x')
func UUID16() string {
	// Generate 16 * 4 = 64 bits of random data, convert to hexadecimal
	// 64 bits = 8 bytes
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// In practice, rand.Read from crypto/rand should never fail
		// but we handle it to satisfy linter
		panic(fmt.Sprintf("failed to generate random bytes: %v", err))
	}
	return hex.EncodeToString(bytes)[:16] // 16 hexadecimal characters
}

// GenerateClientOrderID generates a client order ID based on exchange type
// Format: exlink-{exchange}-{UUID16}
// Example: exlink-binance-caa54b21bbabadd4
// exchange: Exchange name (e.g., binance, okx, gate, bybit)
func GenerateClientOrderID(exchange string) string {
	// Convert exchange name to lowercase to ensure consistent format
	exchange = strings.ToLower(exchange)
	return fmt.Sprintf("exlink-%s-%s", exchange, UUID16())
}
