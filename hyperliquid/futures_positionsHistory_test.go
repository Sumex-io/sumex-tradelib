package hyperliquid

import (
	"encoding/json"
	"testing"
)

// Fills as `/info` returns them: an opening fill carries closedPnl "0.0" and a
// non-zero fee, so only the fills trading against the open position may become
// positions history rows.
func TestConvertPositionsHistorySkipsOpeningFills(t *testing.T) {
	payload := []byte(`[
		{"coin":"XPL","px":"0.7031","sz":"1500.0","side":"A","time":1780944613087,"startPosition":"-262.46","dir":"Open Short","closedPnl":"0.0","fee":"0.004194","tid":724473335954239},
		{"coin":"XPL","px":"0.6934","sz":"1500.0","side":"B","time":1780944713087,"startPosition":"-1762.46","dir":"Close Short","closedPnl":"-14.55","fee":"0.004161","tid":724473335954240},
		{"coin":"TON","px":"1.3622","sz":"2000.0","side":"A","time":1770112166927,"startPosition":"2000.0","dir":"Close Long","closedPnl":"-10432.56","fee":"0.858185","tid":681866443270310}
	]`)

	var fills []hlUserFill
	if err := json.Unmarshal(payload, &fills); err != nil {
		t.Fatalf("unmarshal fills: %v", err)
	}

	c := futures_converts{}
	res := c.convertPositionsHistory(fills)
	if len(res) != 2 {
		t.Fatalf("expected 2 positions history rows, got %d", len(res))
	}
	if res[0].RealisedProfit != "-14.55" || res[0].PositionSide != "SHORT" {
		t.Errorf("expected the short close row, got %+v", res[0])
	}
	if res[1].RealisedProfit != "-10432.56" || res[1].PositionSide != "LONG" {
		t.Errorf("expected the long close row, got %+v", res[1])
	}
}

// The first fill of a position reports startPosition "0.0" - nothing to realise.
func TestConvertPositionsHistorySkipsFirstFillOfPosition(t *testing.T) {
	payload := []byte(`[
		{"coin":"POPCAT","px":"0.2","sz":"100.0","side":"B","time":1780944610650,"startPosition":"0.0","dir":"Open Long","closedPnl":"0.0","fee":"0.0044","tid":335975793901354}
	]`)

	var fills []hlUserFill
	if err := json.Unmarshal(payload, &fills); err != nil {
		t.Fatalf("unmarshal fills: %v", err)
	}

	c := futures_converts{}
	if res := c.convertPositionsHistory(fills); len(res) != 0 {
		t.Fatalf("expected no positions history rows, got %d", len(res))
	}
}

// Break-even closes and fee-free exits (ADL, settlement) still realise the
// position, so they must survive the filter.
func TestConvertPositionsHistoryKeepsBreakEvenAndFeeFreeCloses(t *testing.T) {
	payload := []byte(`[
		{"coin":"POPCAT","px":"0.2","sz":"100.0","side":"A","time":1780944710650,"startPosition":"100.0","dir":"Close Long","closedPnl":"0.0","fee":"0.0044","tid":335975793901355},
		{"coin":"XPL","px":"0.71","sz":"500.0","side":"B","time":1780944810650,"startPosition":"-500.0","dir":"Auto-Deleveraging","closedPnl":"3.2","fee":"0.0","tid":335975793901356}
	]`)

	var fills []hlUserFill
	if err := json.Unmarshal(payload, &fills); err != nil {
		t.Fatalf("unmarshal fills: %v", err)
	}

	c := futures_converts{}
	res := c.convertPositionsHistory(fills)
	if len(res) != 2 {
		t.Fatalf("expected 2 positions history rows, got %d", len(res))
	}
	if res[0].RealisedProfit != "0.0" {
		t.Errorf("expected the break-even close to be kept, got %+v", res[0])
	}
	if res[1].RealisedProfit != "3.2" || res[1].PositionSide != "SHORT" {
		t.Errorf("expected the fee-free ADL close, got %+v", res[1])
	}
}
