package kucoin

import (
	"encoding/json"
	"testing"
)

// KuCoin returns `leverage` as a decimal number (cross positions report the
// effective leverage, e.g. 9.51), so the payload must not be bound to an
// integer field.
func TestFuturesPositionsLeverageDecimal(t *testing.T) {
	payload := []byte(`{"code":"200000","data":[{
		"id":"500000000000000001",
		"symbol":"XBTUSDTM",
		"marginMode":"CROSS",
		"positionSide":"BOTH",
		"crossMode":true,
		"isOpen":true,
		"markPrice":64000.1,
		"currentQty":3,
		"currentCost":192.0003,
		"realisedPnl":-0.115,
		"unrealisedPnl":1.23,
		"avgEntryPrice":63500.5,
		"leverage":9.51,
		"openingTimestamp":1750000000000,
		"currentTimestamp":1750000060000
	}]}`)

	var answ struct {
		Result []futures_Position `json:"data"`
	}
	if err := json.Unmarshal(payload, &answ); err != nil {
		t.Fatalf("unmarshal positions: %v", err)
	}

	c := futures_converts{}
	res := c.convertPositions(answ.Result)
	if len(res) != 1 {
		t.Fatalf("expected 1 position, got %d", len(res))
	}
	if res[0].Leverage != "9.51" {
		t.Errorf("expected leverage %q, got %q", "9.51", res[0].Leverage)
	}
	if res[0].PositionSide != "LONG" {
		t.Errorf("expected position side %q, got %q", "LONG", res[0].PositionSide)
	}
}

func TestFuturesPositionsLeverageWholeNumber(t *testing.T) {
	payload := []byte(`{"data":[{"symbol":"ETHUSDTM","marginMode":"ISOLATED","positionSide":"BOTH","isOpen":true,"currentQty":-2,"leverage":10}]}`)

	var answ struct {
		Result []futures_Position `json:"data"`
	}
	if err := json.Unmarshal(payload, &answ); err != nil {
		t.Fatalf("unmarshal positions: %v", err)
	}

	c := futures_converts{}
	res := c.convertPositions(answ.Result)
	if len(res) != 1 {
		t.Fatalf("expected 1 position, got %d", len(res))
	}
	if res[0].Leverage != "10" {
		t.Errorf("expected leverage %q, got %q", "10", res[0].Leverage)
	}
	if res[0].PositionSide != "SHORT" {
		t.Errorf("expected position side %q, got %q", "SHORT", res[0].PositionSide)
	}
}
