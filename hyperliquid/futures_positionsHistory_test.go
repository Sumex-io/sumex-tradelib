package hyperliquid

import (
	"encoding/json"
	"testing"
)

func positionsFromFills(t *testing.T, payload string) []entityRow {
	t.Helper()

	var fills []hlUserFill
	if err := json.Unmarshal([]byte(payload), &fills); err != nil {
		t.Fatalf("unmarshal fills: %v", err)
	}

	c := futures_converts{}
	converted := c.convertPositionsHistory(fills)

	rows := make([]entityRow, 0, len(converted))
	for _, item := range converted {
		rows = append(rows, entityRow{
			Symbol:       item.Symbol,
			PositionSide: item.PositionSide,
			PositionAmt:  item.PositionAmt,
			ClosedAmt:    item.ExecutedPositionAmt,
			AvgPrice:     item.AvgPrice,
			ExitPrice:    item.ExecutedAvgPrice,
			Realised:     item.RealisedProfit,
			Fee:          item.Fee,
			CreateTime:   item.CreateTime,
			UpdateTime:   item.UpdateTime,
		})
	}

	return rows
}

type entityRow struct {
	Symbol       string
	PositionSide string
	PositionAmt  string
	ClosedAmt    string
	AvgPrice     string
	ExitPrice    string
	Realised     string
	Fee          string
	CreateTime   int64
	UpdateTime   int64
}

// `/info` answers newest first, so the fills below are listed in that order: two
// opens at 10 and 12 (average entry 11) closed by two fills at 15 and 16.
func TestConvertPositionsHistoryAggregatesFillsIntoOnePosition(t *testing.T) {
	rows := positionsFromFills(t, `[
		{"coin":"SOL","px":"16","sz":"50.0","side":"A","time":1780000004000,"startPosition":"50.0","dir":"Close Long","closedPnl":"250.0","fee":"0.4","tid":4},
		{"coin":"SOL","px":"15","sz":"150.0","side":"A","time":1780000003000,"startPosition":"200.0","dir":"Close Long","closedPnl":"600.0","fee":"0.3","tid":3},
		{"coin":"SOL","px":"12","sz":"100.0","side":"B","time":1780000002000,"startPosition":"100.0","dir":"Open Long","closedPnl":"0.0","fee":"0.2","tid":2},
		{"coin":"SOL","px":"10","sz":"100.0","side":"B","time":1780000001000,"startPosition":"0.0","dir":"Open Long","closedPnl":"0.0","fee":"0.1","tid":1}
	]`)

	if len(rows) != 1 {
		t.Fatalf("expected 1 positions history row, got %d (%+v)", len(rows), rows)
	}

	expected := entityRow{
		Symbol: "SOL/USDC", PositionSide: "LONG",
		PositionAmt: "200", ClosedAmt: "200",
		AvgPrice: "11", ExitPrice: "15.25",
		Realised: "850", Fee: "1",
		CreateTime: 1780000001000, UpdateTime: 1780000004000,
	}
	if rows[0] != expected {
		t.Errorf("expected %+v, got %+v", expected, rows[0])
	}
}

// A position opened before the served fills has no open time and no entry fills,
// so the entry price has to come back out of the realised PnL.
func TestConvertPositionsHistoryDerivesEntryOfPositionOpenedEarlier(t *testing.T) {
	rows := positionsFromFills(t, `[
		{"coin":"XPL","px":"0.71","sz":"500.0","side":"B","time":1780000002000,"startPosition":"-500.0","dir":"Auto-Deleveraging","closedPnl":"3.2","fee":"0.0","tid":2},
		{"coin":"XPL","px":"0.70","sz":"500.0","side":"B","time":1780000001000,"startPosition":"-1000.0","dir":"Close Short","closedPnl":"8.0","fee":"0.1","tid":1}
	]`)

	if len(rows) != 1 {
		t.Fatalf("expected 1 positions history row, got %d (%+v)", len(rows), rows)
	}

	expected := entityRow{
		Symbol: "XPL/USDC", PositionSide: "SHORT",
		PositionAmt: "1000", ClosedAmt: "1000",
		AvgPrice: "0.7162", ExitPrice: "0.705",
		Realised: "11.2", Fee: "0.1",
		CreateTime: 0, UpdateTime: 1780000002000,
	}
	if rows[0] != expected {
		t.Errorf("expected %+v, got %+v", expected, rows[0])
	}
}

// A flip realises the position it turns out of; the remainder stays open.
func TestConvertPositionsHistoryClosesFlippedPosition(t *testing.T) {
	rows := positionsFromFills(t, `[
		{"coin":"ETH","px":"12","sz":"150.0","side":"A","time":1780000002000,"startPosition":"100.0","dir":"Long > Short","closedPnl":"200.0","fee":"0.2","tid":2},
		{"coin":"ETH","px":"10","sz":"100.0","side":"B","time":1780000001000,"startPosition":"0.0","dir":"Open Long","closedPnl":"0.0","fee":"0.1","tid":1}
	]`)

	if len(rows) != 1 {
		t.Fatalf("expected 1 positions history row, got %d (%+v)", len(rows), rows)
	}

	expected := entityRow{
		Symbol: "ETH/USDC", PositionSide: "LONG",
		PositionAmt: "100", ClosedAmt: "100",
		AvgPrice: "10", ExitPrice: "12",
		Realised: "200", Fee: "0.3",
		CreateTime: 1780000001000, UpdateTime: 1780000002000,
	}
	if rows[0] != expected {
		t.Errorf("expected %+v, got %+v", expected, rows[0])
	}
}

// Break-even closes realise nothing but still ended a position, so they stay.
func TestConvertPositionsHistoryKeepsBreakEvenClose(t *testing.T) {
	rows := positionsFromFills(t, `[
		{"coin":"POPCAT","px":"0.2","sz":"100.0","side":"A","time":1780000002000,"startPosition":"100.0","dir":"Close Long","closedPnl":"0.0","fee":"0.0044","tid":2},
		{"coin":"POPCAT","px":"0.2","sz":"100.0","side":"B","time":1780000001000,"startPosition":"0.0","dir":"Open Long","closedPnl":"0.0","fee":"0.0044","tid":1}
	]`)

	if len(rows) != 1 {
		t.Fatalf("expected 1 positions history row, got %d (%+v)", len(rows), rows)
	}
	if rows[0].Realised != "0" || rows[0].AvgPrice != "0.2" || rows[0].ExitPrice != "0.2" {
		t.Errorf("expected a break-even row at 0.2, got %+v", rows[0])
	}
}

// Positions that are still open belong to the positions endpoint, and spot fills
// are not positions at all.
func TestConvertPositionsHistorySkipsOpenPositionsAndSpotFills(t *testing.T) {
	rows := positionsFromFills(t, `[
		{"coin":"BTC","px":"11","sz":"40.0","side":"A","time":1780000003000,"startPosition":"100.0","dir":"Close Long","closedPnl":"40.0","fee":"0.2","tid":3},
		{"coin":"BTC","px":"10","sz":"100.0","side":"B","time":1780000002000,"startPosition":"0.0","dir":"Open Long","closedPnl":"0.0","fee":"0.1","tid":2},
		{"coin":"PURR/USDC","px":"0.3","sz":"1000.0","side":"A","time":1780000001500,"startPosition":"1000.0","dir":"Sell","closedPnl":"12.0","fee":"0.1","tid":15},
		{"coin":"@107","px":"58.9","sz":"0.2","side":"B","time":1780000001000,"startPosition":"0.0","dir":"Buy","closedPnl":"0.0","fee":"0.00014","tid":1}
	]`)

	if len(rows) != 0 {
		t.Fatalf("expected no positions history rows, got %d (%+v)", len(rows), rows)
	}
}
