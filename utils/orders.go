package utils

import (
	"strings"

	"github.com/Betarost/onetrades/entity"
)

// DeriveTPFromOpenOrders ищет TP по открытым ордерам под конкретную сторону позиции
func DeriveTPFromOpenOrders(openOrders []entity.Futures_OrdersList, symbol, posSide string) float64 {
	sym := strings.ToUpper(strings.TrimSpace(symbol))
	pos := strings.ToUpper(strings.TrimSpace(posSide))
	for _, oo := range openOrders {
		if strings.ToUpper(strings.TrimSpace(oo.Symbol)) != sym {
			continue
		}
		sideU := strings.ToUpper(strings.TrimSpace(oo.Side))
		ooPos := strings.ToUpper(strings.TrimSpace(oo.PositionSide))
		if pos == "LONG" && sideU == "SELL" && ooPos == "LONG" {
			return StringToFloat(oo.Price)
		}
		if pos == "SHORT" && sideU == "BUY" && ooPos == "SHORT" {
			return StringToFloat(oo.Price)
		}
	}
	return 0
}
