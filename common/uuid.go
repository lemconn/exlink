package common

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/lemconn/exlink/types"
)

// GenerateClientOrderID generates a client order ID based on exchange type and side
// Format: EL<EX><B/S><timestamp_ms><2 random>
// Example: ELOKXS173209570112398 (S=Sell), ELBINB1732095709956F2 (B=Buy)
// Special case: Gate exchange requires "t-" prefix: t-ELGTEB1732095709956A9
// exchange: Exchange name (e.g., binance, okx, gate, bybit)
// side: Order side (buy or sell)
func GenerateClientOrderID(exchange string, side types.OrderSide) string {
	// Map exchange names to three-letter abbreviations
	exchangeMap := map[string]string{
		"okx":     "OKX",
		"binance": "BIN",
		"bybit":   "BYB",
		"gate":    "GTE",
	}

	// Convert to lowercase to match the mapping
	exchangeLower := strings.ToLower(exchange)
	exchangeCode, ok := exchangeMap[exchangeLower]
	if !ok {
		// If mapping not found, use the first three characters of the original name (uppercase)
		if len(exchangeLower) >= 3 {
			exchangeCode = strings.ToUpper(exchangeLower[:3])
		} else {
			exchangeCode = strings.ToUpper(exchangeLower)
		}
	}

	// Order side: buy -> B, sell -> S
	sideCode := "B"
	if strings.ToLower(string(side)) == "sell" {
		sideCode = "S"
	}

	// Get millisecond timestamp
	timestampMs := time.Now().UnixMilli()

	// Generate two random characters (uppercase letters + digits)
	// Character set: 0-9, A-Z (36 characters)
	charset := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	randomChars := make([]byte, 2)
	for i := range randomChars {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(fmt.Sprintf("failed to generate random number: %v", err))
		}
		randomChars[i] = charset[n.Int64()]
	}

	// Build order ID
	orderID := fmt.Sprintf("EL%s%s%d%s", exchangeCode, sideCode, timestampMs, string(randomChars))

	// Gate exchange requires "t-" prefix
	if exchangeLower == "gate" {
		orderID = "t-" + orderID
	}

	return orderID
}
