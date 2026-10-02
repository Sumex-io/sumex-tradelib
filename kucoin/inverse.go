package kucoin

import "strings"

// isInverseSymbol: linear contracts settle in USDT/USDC (XBTUSDTM), coin-margined ones do not (XBTUSDM, XBTMZ26).
func isInverseSymbol(symbol string) bool {
	return !strings.HasSuffix(symbol, "USDTM") && !strings.HasSuffix(symbol, "USDCM")
}

func dropInverse[T any](in []T, symbol func(T) string) []T {
	out := make([]T, 0, len(in))
	for _, item := range in {
		if !isInverseSymbol(symbol(item)) {
			out = append(out, item)
		}
	}
	return out
}
