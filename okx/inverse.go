package okx

import "strings"

// isInverseInstId: coin-margined instruments are quoted in USD (BTC-USD-SWAP), linear ones in USDT/USDC.
func isInverseInstId(instId string) bool {
	return strings.Contains(instId, "-USD-")
}

func dropInverse[T any](in []T, instId func(T) string) []T {
	out := make([]T, 0, len(in))
	for _, item := range in {
		if !isInverseInstId(instId(item)) {
			out = append(out, item)
		}
	}
	return out
}
