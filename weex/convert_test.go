package weex

import (
	"encoding/json"
	"testing"
)

// After partial closes WEEX keeps the whole lifetime in cumOpenValue/cumOpenSize;
// the entry it shows for the open remainder is openValue/size.
func TestPositionEntryPriceIgnoresClosedParts(t *testing.T) {
	payload := []byte(`[{
		"id":1,
		"symbol":"BTCUSDT",
		"side":"LONG",
		"marginType":"CROSSED",
		"leverage":"400",
		"size":"2",
		"openValue":"165046.8",
		"cumOpenSize":"4",
		"cumOpenValue":"332069.2",
		"cumCloseSize":"2",
		"unrealizePnl":"-2750.6"
	}]`)

	var answ []futures_Position
	if err := json.Unmarshal(payload, &answ); err != nil {
		t.Fatalf("unmarshal positions: %v", err)
	}

	c := futures_converts{}
	res := c.convertPositions(answ)
	if len(res) != 1 {
		t.Fatalf("expected 1 position, got %d", len(res))
	}
	if res[0].EntryPrice != "82523.4" {
		t.Fatalf("expected entry price 82523.4, got %s", res[0].EntryPrice)
	}
}
