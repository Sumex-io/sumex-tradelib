package katana

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// --- Wire-format and mapper contract ---

func TestLeverageToIMFRoundTrip(t *testing.T) {
	imf, err := leverageToIMF("10")
	if err != nil {
		t.Fatal(err)
	}
	if imf != "0.10000000" {
		t.Fatalf("leverageToIMF(10) = %s, want 0.10000000", imf)
	}
	lev, err := imfToLeverage("0.05000000")
	if err != nil {
		t.Fatal(err)
	}
	if lev != "20" {
		t.Fatalf("imfToLeverage(0.05) = %s, want 20", lev)
	}
}

func TestLeverageToIMFRejectsNonPositive(t *testing.T) {
	if _, err := leverageToIMF("0"); err == nil {
		t.Fatal("expected error for zero leverage")
	}
}

func TestToPositionDerivesSideAndKeepsCrossMargin(t *testing.T) {
	got := toPosition(katanaPosition{
		Market: "ETH-USD", Quantity: "-2.50000000", EntryPrice: "2250.00000000",
		MarkPrice: "2300.00000000", UnrealizedPnL: "-125.00000000", RealizedPnL: "10.00000000",
		Time: 1704067200000,
	}, Market{Symbol: "ETH-USD", InitialMarginFraction: "0.10000000"})

	if got.PositionSide != "SHORT" {
		t.Fatalf("positionSide = %s, want SHORT for negative quantity", got.PositionSide)
	}
	if got.PositionSize != "2.50000000" {
		t.Fatalf("positionSize = %s, want the absolute value", got.PositionSize)
	}
	if got.MarginMode != "CROSS" {
		t.Fatalf("marginMode = %s, want CROSS", got.MarginMode)
	}
	if got.HedgeMode {
		t.Fatal("hedgeMode must always be false on Katana")
	}
	if got.Leverage != "10" {
		t.Fatalf("leverage = %s, want 10 (1 / initialMarginFraction)", got.Leverage)
	}
}

func TestToBalanceMapsVbUsdcToUsdc(t *testing.T) {
	got := toBalance(katanaWallet{
		Wallet: "0x1", EquityUSD: "1500.00000000", FreeCollateral: "1200.00000000",
		UnrealizedPnL: "50.00000000", Quantity: "1450.00000000",
	})
	if len(got) != 1 {
		t.Fatalf("got %d balances, want 1", len(got))
	}
	if got[0].Asset != "USDC" {
		t.Fatalf("asset = %s, want USDC (vbUSDC is mapped)", got[0].Asset)
	}
	if got[0].Equity != "1500.00000000" {
		t.Fatalf("equity = %s", got[0].Equity)
	}
}

// --- Parsing and precision ---

func TestToBalanceMapsEveryFieldLiterally(t *testing.T) {
	got := toBalance(katanaWallet{
		Wallet: "0xabc", EquityUSD: "1500.00000000", FreeCollateral: "1200.00000000",
		UnrealizedPnL: "50.00000000", Quantity: "1450.00000000",
	})
	want := entity.FuturesBalance{
		Asset:            "USDC",
		Balance:          "1450.00000000",
		Equity:           "1500.00000000",
		Available:        "1200.00000000",
		UnrealizedProfit: "50.00000000",
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("toBalance = %+v, want [%+v]", got, want)
	}
}

func TestLeverageToIMFRejectsNonNumericAndNaN(t *testing.T) {
	for _, in := range []string{"abc", "NaN", "", "-5"} {
		if _, err := leverageToIMF(in); err == nil {
			t.Fatalf("leverageToIMF(%q) should have errored", in)
		}
	}
}

func TestImfToLeverageRejectsNonPositiveAndNonNumeric(t *testing.T) {
	for _, in := range []string{"0", "-0.05000000", "abc", "NaN", ""} {
		if _, err := imfToLeverage(in); err == nil {
			t.Fatalf("imfToLeverage(%q) should have errored", in)
		}
	}
}

func TestLeverageToIMFAndImfToLeverageTableDriven(t *testing.T) {
	cases := []struct {
		leverage string
		imf      string
	}{
		{"1", "1.00000000"},
		{"2", "0.50000000"},
		{"4", "0.25000000"},
		{"20", "0.05000000"},
		{"50", "0.02000000"},
		{"100", "0.01000000"},
	}
	for _, tc := range cases {
		gotIMF, err := leverageToIMF(tc.leverage)
		if err != nil {
			t.Fatalf("leverageToIMF(%s): %v", tc.leverage, err)
		}
		if gotIMF != tc.imf {
			t.Fatalf("leverageToIMF(%s) = %s, want %s", tc.leverage, gotIMF, tc.imf)
		}

		gotLev, err := imfToLeverage(tc.imf)
		if err != nil {
			t.Fatalf("imfToLeverage(%s): %v", tc.imf, err)
		}
		if gotLev != tc.leverage {
			t.Fatalf("imfToLeverage(%s) = %s, want %s", tc.imf, gotLev, tc.leverage)
		}
	}
}

func TestLeverageToIMFRoundsRepeatingDecimal(t *testing.T) {
	// 1/3 = 0.3333...; rounded to 8 decimals the 9th digit (3) rounds down.
	got, err := leverageToIMF("3")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.33333333" {
		t.Fatalf("leverageToIMF(3) = %s, want 0.33333333", got)
	}
}

func TestToPositionLongSideAndFullFieldMapping(t *testing.T) {
	got := toPosition(katanaPosition{
		Market: "BTC-USD", Quantity: "4.00000000", EntryPrice: "29000.00000000",
		MarkPrice: "30000.00000000", UnrealizedPnL: "4000.00000000", RealizedPnL: "0.00000000",
		Time: 1700000000000,
	}, Market{Symbol: "BTC-USD", InitialMarginFraction: "0.05000000"})

	want := entity.Futures_Positions{
		Symbol:           "BTC-USD",
		PositionSide:     "LONG",
		PositionSize:     "4.00000000",
		Leverage:         "20",
		PositionID:       "",
		EntryPrice:       "29000.00000000",
		MarkPrice:        "30000.00000000",
		UnRealizedProfit: "4000.00000000",
		RealizedProfit:   "0.00000000",
		Notional:         "120000.00000000",
		HedgeMode:        false,
		MarginMode:       "CROSS",
		CreateTime:       1700000000000,
		UpdateTime:       1700000000000,
	}
	if got != want {
		t.Fatalf("toPosition = %+v, want %+v", got, want)
	}
}

// tieredMarket uses the numbers from API_NOTES.md's own GET /v1/markets sample: base tier 0.05 IMF
// (20x) up to a position size of 25, then +0.01 IMF for each further step of 5.
var tieredMarket = Market{
	Symbol:                           "ETH-USD",
	InitialMarginFraction:            "0.05000000",
	BasePositionSize:                 "25.00000000",
	IncrementalPositionSize:          "5.00000000",
	IncrementalInitialMarginFraction: "0.01000000",
}

// TestEffectiveIMFForSizeAppliesTheTierSchedule works the schedule by hand:
//
//	  10 <= 25            -> base tier, 0.05
//	  25 == 25            -> base tier (the boundary is inclusive), 0.05
//	  26 -> excess 1, ceil(1/5)  = 1 step  -> 0.05 + 0.01 = 0.06
//	  30 -> excess 5, ceil(5/5)  = 1 step  -> 0.06
//	  31 -> excess 6, ceil(6/5)  = 2 steps -> 0.07
//	  40 -> excess 15, ceil(15/5)= 3 steps -> 0.08
//	-40 -> size is the absolute value       -> 0.08
//
// Every expected value below is written out from that arithmetic, never produced by the function
// under test.
func TestEffectiveIMFForSizeAppliesTheTierSchedule(t *testing.T) {
	cases := []struct {
		quantity string
		want     string
	}{
		{"10.00000000", "0.05000000"},
		{"25.00000000", "0.05000000"},
		{"26.00000000", "0.06000000"},
		{"30.00000000", "0.06000000"},
		{"31.00000000", "0.07000000"},
		{"40.00000000", "0.08000000"},
		{"-40.00000000", "0.08000000"},
	}
	for _, tc := range cases {
		got, err := effectiveIMFForSize(tieredMarket, tc.quantity)
		if err != nil {
			t.Fatalf("effectiveIMFForSize(%s): %v", tc.quantity, err)
		}
		if got != tc.want {
			t.Fatalf("effectiveIMFForSize(%s) = %s, want %s", tc.quantity, got, tc.want)
		}
	}
}

// TestEffectiveIMFForSizeFallsBackToTheBaseTierWithoutASchedule covers a market that publishes no
// tier fields at all.
func TestEffectiveIMFForSizeFallsBackToTheBaseTierWithoutASchedule(t *testing.T) {
	got, err := effectiveIMFForSize(Market{Symbol: "BTC-USD", InitialMarginFraction: "0.02000000"}, "1000.00000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.02000000" {
		t.Fatalf("effectiveIMFForSize = %s, want the market's own 0.02000000 when no tier schedule is published", got)
	}
}

func TestEffectiveIMFForSizeErrorsOnUnusableInput(t *testing.T) {
	if _, err := effectiveIMFForSize(tieredMarket, "not-a-number"); err == nil {
		t.Fatal("expected an error for an unparseable quantity, not a confident base-tier answer")
	}
	if _, err := effectiveIMFForSize(Market{InitialMarginFraction: "0"}, "1.00000000"); err == nil {
		t.Fatal("expected an error for a non-positive initialMarginFraction")
	}
	// Exotic literal forms big.Rat.SetString would otherwise accept must be rejected, the same way
	// parseBigFloat rejects them.
	if _, err := effectiveIMFForSize(tieredMarket, "3/2"); err == nil {
		t.Fatal("expected a quotient literal to be rejected")
	}
	if _, err := effectiveIMFForSize(tieredMarket, "1e3"); err == nil {
		t.Fatal("expected an exponent literal to be rejected")
	}
}

// TestToPositionUsesTheTieredLeverageAboveBasePositionSize: a 40-unit position on the fixture
// market sits three increments above basePositionSize, so its IMF is 0.08 and its leverage is
// 1/0.08 = 12.5 — not the 20x the base tier alone would have reported.
func TestToPositionUsesTheTieredLeverageAboveBasePositionSize(t *testing.T) {
	got := toPosition(katanaPosition{
		Market: "ETH-USD", Quantity: "40.00000000", EntryPrice: "2000.00000000",
		MarkPrice: "2000.00000000", UnrealizedPnL: "0.00000000", RealizedPnL: "0.00000000",
	}, tieredMarket)

	if got.Leverage != "12.5" {
		t.Fatalf("leverage = %q, want 12.5 (IMF 0.08 at three increments above basePositionSize)", got.Leverage)
	}
}

// The paired case below the boundary: 10 units is inside the base tier, IMF 0.05, leverage 20.
func TestToPositionUsesTheBaseTierLeverageBelowBasePositionSize(t *testing.T) {
	got := toPosition(katanaPosition{
		Market: "ETH-USD", Quantity: "10.00000000", EntryPrice: "2000.00000000",
		MarkPrice: "2000.00000000", UnrealizedPnL: "0.00000000", RealizedPnL: "0.00000000",
	}, tieredMarket)

	if got.Leverage != "20" {
		t.Fatalf("leverage = %q, want 20 (base tier)", got.Leverage)
	}
}

// TestToInstrumentInfoMaxLeverageStaysAtTheBaseTier pins the deliberate asymmetry: the instrument's
// maxLeverage is the base tier's 20x even for a market whose tier schedule requires more margin
// above basePositionSize, because that IS the maximum leverage the market ever offers.
func TestToInstrumentInfoMaxLeverageStaysAtTheBaseTier(t *testing.T) {
	got := toInstrumentInfo(tieredMarket)
	if got.MaxLeverage != "20" {
		t.Fatalf("maxLeverage = %q, want 20 (the base tier is the market's genuine maximum)", got.MaxLeverage)
	}
}

// TestMulAbsDecimal8ReportsUnknownRatherThanZero: "0.00000000" for input it could not parse is
// indistinguishable from a genuinely zero notional. An empty string says "unknown" and cannot be
// mistaken for a computed zero. A real zero must still render as a real zero (second case).
func TestMulAbsDecimal8ReportsUnknownRatherThanZero(t *testing.T) {
	if got := mulAbsDecimal8("not-a-number", "2000.00000000"); got != "" {
		t.Fatalf("mulAbsDecimal8 = %q, want \"\" (unknown) for an unparseable quantity", got)
	}
	if got := mulAbsDecimal8("1.00000000", "oops"); got != "" {
		t.Fatalf("mulAbsDecimal8 = %q, want \"\" (unknown) for an unparseable price", got)
	}
	if got := mulAbsDecimal8("0.00000000", "2000.00000000"); got != "0.00000000" {
		t.Fatalf("mulAbsDecimal8 = %q, want 0.00000000 — a genuine zero is still a computed answer", got)
	}
}

// The same guard at the field that ships: notional must be empty, not a confident zero, when the
// wire values it multiplies are unusable.
func TestToPositionLeavesNotionalEmptyWhenItCannotBeComputed(t *testing.T) {
	got := toPosition(katanaPosition{
		Market: "ETH-USD", Quantity: "1.00000000", MarkPrice: "not-a-price",
	}, Market{Symbol: "ETH-USD", InitialMarginFraction: "0.05000000"})

	if got.Notional != "" {
		t.Fatalf("notional = %q, want \"\" — an uncomputable notional must not read as zero", got.Notional)
	}
}

func TestToPositionFallsBackToEmptyLeverageOnInvalidMarketIMF(t *testing.T) {
	got := toPosition(katanaPosition{Market: "ETH-USD", Quantity: "1.00000000", MarkPrice: "1.00000000"}, Market{InitialMarginFraction: "0"})
	if got.Leverage != "" {
		t.Fatalf("leverage = %q, want empty string when the market IMF is invalid", got.Leverage)
	}
}

func TestToOrderRegularBuyLimitOrderIsLongPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{
		Market: "ETH-USD", OrderId: "abc123", ClientOrderId: "myorder1", Time: 1700000000000,
		Status: "active", Type: "limit", Side: "buy",
		OriginalQuantity: "1.50000000", ExecutedQuantity: "0.00000000", Price: "2500.00000000",
	})
	want := entity.Futures_OrdersList{
		Symbol: "ETH-USD", OrderID: "abc123", ClientOrderID: "myorder1", Side: "BUY",
		ExecutedSize: "0.00000000", Price: "2500.00000000", Type: "LIMIT", Status: "ACTIVE",
		CreateTime: 1700000000000, UpdateTime: 1700000000000,
		PositionID: "", PositionSide: "LONG", PositionSize: "1.50000000", Leverage: "",
		MarginMode: "CROSS", TpOrder: false, SlOrder: false,
	}
	if got != want {
		t.Fatalf("toOrder = %+v, want %+v", got, want)
	}
}

func TestToOrderRegularSellOrderIsShortPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{Market: "ETH-USD", Type: "market", Side: "sell", OriginalQuantity: "1.00000000"})
	if got.PositionSide != "SHORT" {
		t.Fatalf("positionSide = %s, want SHORT for a plain sell order", got.PositionSide)
	}
	if got.TpOrder || got.SlOrder {
		t.Fatal("a plain market order must not be classified as tp/sl")
	}
}

func TestToOrderTakeProfitSellInvertsToLongPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{Market: "ETH-USD", Type: "takeProfitLimit", Side: "sell", OriginalQuantity: "1.00000000"})
	if !got.TpOrder || got.SlOrder {
		t.Fatalf("tpOrder=%v slOrder=%v, want tpOrder=true slOrder=false", got.TpOrder, got.SlOrder)
	}
	if got.PositionSide != "LONG" {
		t.Fatalf("positionSide = %s, want LONG (a sell TP closes a long)", got.PositionSide)
	}
}

func TestToOrderStopLossBuyInvertsToShortPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{Market: "ETH-USD", Type: "stopLossMarket", Side: "buy", OriginalQuantity: "1.00000000"})
	if got.TpOrder || !got.SlOrder {
		t.Fatalf("tpOrder=%v slOrder=%v, want tpOrder=false slOrder=true", got.TpOrder, got.SlOrder)
	}
	if got.PositionSide != "SHORT" {
		t.Fatalf("positionSide = %s, want SHORT (a buy SL closes a short)", got.PositionSide)
	}
}

func TestClassifyTriggerOrder(t *testing.T) {
	cases := []struct {
		orderType string
		wantTP    bool
		wantSL    bool
	}{
		{"market", false, false},
		{"limit", false, false},
		{"takeProfitMarket", true, false},
		{"takeProfitLimit", true, false},
		{"stopLossMarket", false, true},
		{"stopLossLimit", false, true},
	}
	for _, tc := range cases {
		tp, sl := classifyTriggerOrder(tc.orderType)
		if tp != tc.wantTP || sl != tc.wantSL {
			t.Fatalf("classifyTriggerOrder(%s) = (%v, %v), want (%v, %v)", tc.orderType, tp, sl, tc.wantTP, tc.wantSL)
		}
	}
}

// --- Numeric-parsing and order-derivation edge cases ---

// big.Float.SetString (base 0) accepts "Inf" and exotic Go numeric literal forms (hex floats,
// underscore digit grouping) that Katana's wire format never sends. parseBigFloat/leverageToIMF/
// imfToLeverage must reject all of them explicitly.
func TestParseBigFloatRejectsInfAndExoticLiteralForms(t *testing.T) {
	for _, in := range []string{"Inf", "+Inf", "-Inf", "inf", "0x1p4", "1_0"} {
		if _, err := parseBigFloat(in); err == nil {
			t.Fatalf("parseBigFloat(%q) should have errored", in)
		}
	}
}

func TestLeverageToIMFRejectsInfAndExoticLiteralForms(t *testing.T) {
	for _, in := range []string{"Inf", "-Inf", "0x1p4", "1_0"} {
		if _, err := leverageToIMF(in); err == nil {
			t.Fatalf("leverageToIMF(%q) should have errored", in)
		}
	}
}

func TestImfToLeverageRejectsInfAndExoticLiteralForms(t *testing.T) {
	for _, in := range []string{"Inf", "-Inf", "0x1p4", "1_0"} {
		if _, err := imfToLeverage(in); err == nil {
			t.Fatalf("imfToLeverage(%q) should have errored", in)
		}
	}
}

// Katana omits `price` for stopLossMarket/takeProfitMarket orders; the FE's `order.price ??
// fallback` does not treat "" as nullish, so this must fall back to triggerPrice — and, per the
// hyperliquid precedent this mirrors, unconditionally for *any* trigger order (including the
// *Limit variants, where price is present but is the limit price, not the trigger price the UI
// wants to show).
func TestResolveOrderPriceFallsBackToTriggerPriceForMarketVariants(t *testing.T) {
	got := resolveOrderPrice(katanaOrder{Type: "stopLossMarket", Price: "", TriggerPrice: "1900.00000000"})
	if got != "1900.00000000" {
		t.Fatalf("resolveOrderPrice = %q, want the trigger price (wire price is empty for *Market trigger orders)", got)
	}
}

func TestResolveOrderPricePrefersTriggerPriceOverLimitPriceForLimitVariants(t *testing.T) {
	got := resolveOrderPrice(katanaOrder{Type: "takeProfitLimit", Price: "2600.00000000", TriggerPrice: "2550.00000000"})
	if got != "2550.00000000" {
		t.Fatalf("resolveOrderPrice = %q, want the trigger price 2550.00000000, not the limit price", got)
	}
}

func TestResolveOrderPriceUsesWirePriceForRegularOrders(t *testing.T) {
	got := resolveOrderPrice(katanaOrder{Type: "limit", Price: "2500.00000000", TriggerPrice: ""})
	if got != "2500.00000000" {
		t.Fatalf("resolveOrderPrice = %q, want the wire price for a non-trigger order", got)
	}
}

// A reduce-only sell that closes a long must report the side of the position it closes (LONG), not
// the side of the order itself (which would read SHORT and display as "Open Short" instead of
// "Close Long").
//
// NOTE: the source's twin of this test also asserted that the wire's reduceOnly flag reached the
// output row. entity.Futures_OrdersList has no reduceOnly field, so that half of the assertion has
// no subject in this library and is not restated here — see the report's cross-cutting section.
func TestToOrderReduceOnlySellReportsClosedLongPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{
		Market: "ETH-USD", Type: "market", Side: "sell", OriginalQuantity: "1.00000000", ReduceOnly: true,
	})
	if got.PositionSide != "LONG" {
		t.Fatalf("positionSide = %s, want LONG (a reduce-only sell closes a long)", got.PositionSide)
	}
}

func TestToOrderReduceOnlyBuyReportsClosedShortPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{
		Market: "ETH-USD", Type: "market", Side: "buy", OriginalQuantity: "1.00000000", ReduceOnly: true,
	})
	if got.PositionSide != "SHORT" {
		t.Fatalf("positionSide = %s, want SHORT (a reduce-only buy closes a short)", got.PositionSide)
	}
}

func TestToOrderNonReduceOnlySellReportsOpeningShortPositionSide(t *testing.T) {
	got := toOrder(katanaOrder{
		Market: "ETH-USD", Type: "market", Side: "sell", OriginalQuantity: "1.00000000", ReduceOnly: false,
	})
	if got.PositionSide != "SHORT" {
		t.Fatalf("positionSide = %s, want SHORT (a normal, non-reduce-only sell opens a short)", got.PositionSide)
	}
}

// Price fallback must also apply to order history rows, not just live orders.
func TestToOrderHistoryUsesTriggerPriceFallback(t *testing.T) {
	got := toOrderHistory(katanaOrder{
		Market: "ETH-USD", OrderId: "ord-9", Type: "stopLossMarket", Side: "sell",
		OriginalQuantity: "1.00000000", ExecutedQuantity: "1.00000000", Price: "", TriggerPrice: "1900.00000000",
		Status: "filled", Time: 1700000000000,
	})
	if got.Price != "1900.00000000" {
		t.Fatalf("Price = %q, want the trigger price fallback", got.Price)
	}
}

// Futures_OrdersHistory must carry tpOrder/slOrder — the FE's tpsl.adapter.ts classifies history
// rows by those flags first, so a filled TP/SL order without them silently vanishes from the TP/SL
// history panel.
func TestToOrderHistoryClassifiesTakeProfitAndStopLoss(t *testing.T) {
	tp := toOrderHistory(katanaOrder{Market: "ETH-USD", Type: "takeProfitLimit", Side: "sell", Status: "filled"})
	if !tp.TpOrder || tp.SlOrder {
		t.Fatalf("takeProfitLimit history row: tpOrder=%v slOrder=%v, want tpOrder=true slOrder=false", tp.TpOrder, tp.SlOrder)
	}
	if tp.PositionSide != "LONG" {
		t.Fatalf("positionSide = %s, want LONG (a sell TP closes a long)", tp.PositionSide)
	}

	sl := toOrderHistory(katanaOrder{Market: "ETH-USD", Type: "stopLossMarket", Side: "buy", Status: "filled"})
	if sl.TpOrder || !sl.SlOrder {
		t.Fatalf("stopLossMarket history row: tpOrder=%v slOrder=%v, want tpOrder=false slOrder=true", sl.TpOrder, sl.SlOrder)
	}

	plain := toOrderHistory(katanaOrder{Market: "ETH-USD", Type: "limit", Side: "buy", Status: "filled"})
	if plain.TpOrder || plain.SlOrder {
		t.Fatal("a plain limit order must not be classified as tp/sl in history either")
	}
}

// --- Converts ---

func TestConvertPositionsDropsClosedRowsAndAppliesOverrides(t *testing.T) {
	convert := futures_converts{}
	mkts := map[string]Market{"ETH-USD": tieredMarket}
	overrides := map[string]string{"ETH-USD": "0.10000000"}

	got := convert.convertPositions([]katanaPosition{
		{Market: "ETH-USD", Quantity: "0.00000000", MarkPrice: "2000.00000000"},
		{Market: "ETH-USD", Quantity: "10.00000000", MarkPrice: "2000.00000000"},
	}, mkts, overrides)

	if len(got) != 1 {
		t.Fatalf("got %d positions, want 1 (a zero-quantity row is a closed position, not a size-0 LONG)", len(got))
	}
	if got[0].Leverage != "10" {
		t.Fatalf("leverage = %q, want 10 — the wallet's IMF override becomes the base tier", got[0].Leverage)
	}
}

func TestConvertPositionsLeavesLeverageEmptyForAnUnknownMarket(t *testing.T) {
	convert := futures_converts{}
	got := convert.convertPositions([]katanaPosition{
		{Market: "DELISTED-USD", Quantity: "1.00000000", MarkPrice: "10.00000000"},
	}, map[string]Market{}, nil)

	if len(got) != 1 {
		t.Fatalf("got %d positions, want the row to still be returned", len(got))
	}
	if got[0].Leverage != "" {
		t.Fatalf("leverage = %q, want empty for a market markets() does not know", got[0].Leverage)
	}
}

func TestConvertInstrumentsInfoIsSortedBySymbol(t *testing.T) {
	convert := futures_converts{}
	got := convert.convertInstrumentsInfo(map[string]Market{
		"ETH-USD": {Symbol: "ETH-USD", InitialMarginFraction: "0.05000000"},
		"BTC-USD": {Symbol: "BTC-USD", InitialMarginFraction: "0.02000000"},
		"SOL-USD": {Symbol: "SOL-USD", InitialMarginFraction: "0.10000000"},
	})
	want := []string{"BTC-USD", "ETH-USD", "SOL-USD"}
	if len(got) != len(want) {
		t.Fatalf("got %d instruments, want %d", len(got), len(want))
	}
	for i, sym := range want {
		if got[i].Symbol != sym {
			t.Fatalf("instrument %d = %q, want %q (map iteration order must not reach the wire)", i, got[i].Symbol, sym)
		}
	}
}

func TestConvertAccountInfoReportsPerpetualsOnly(t *testing.T) {
	convert := account_converts{}
	got := convert.convertAccountInfo("0xabc")

	if got.UID != "0xabc" {
		t.Fatalf("uid = %q, want the resolved wallet address", got.UID)
	}
	if got.PermSpot {
		t.Fatal("permSpot must be false — Katana lists no spot markets")
	}
	if !got.PermFutures {
		t.Fatal("permFutures must be true")
	}
	if got.CanTransfer {
		t.Fatal("canTransfer must be false")
	}
}

func TestConvertSignAuthStreamCarriesTheTokenInTheSignatureSlot(t *testing.T) {
	convert := account_converts{}
	got := convert.convertSignAuthStream(katanaWsTokenResponse{Token: "opaque-token"})
	if got.Signature != "opaque-token" {
		t.Fatalf("signature = %q, want the server-issued token — Katana has no WebSocket signature", got.Signature)
	}
}

func TestConvertPositionAndMarginModeAreKatanaInvariants(t *testing.T) {
	convert := futures_converts{}
	if convert.convertPositionMode().HedgeMode {
		t.Fatal("hedgeMode must be false — Katana is one-way only")
	}
	if got := convert.convertMarginMode().MarginMode; got != "CROSS" {
		t.Fatalf("marginMode = %q, want CROSS — Katana is cross-margin only", got)
	}
}

func TestConvertLeverageReportsCrossMargin(t *testing.T) {
	convert := futures_converts{}
	got := convert.convertLeverage("ETH-USD", "20")
	if got.Symbol != "ETH-USD" || got.Leverage != "20" {
		t.Fatalf("convertLeverage = %+v", got)
	}
	if got.MarginMode != "CROSS" {
		t.Fatalf("marginMode = %q, want CROSS", got.MarginMode)
	}
}

// TestOutputStructsMatchTheJSONContractExactly guards the JSON contract with the TS client
// directly — the exact key set (and exact values) json.Marshal produces for one fully-populated
// instance of each entity type this connector emits, so a renamed/dropped tag fails a Go test
// instead of silently emptying a column in the trading UI.
//
// The expectations below are the LIBRARY's entity tags, which differ from the original connector's
// own output structs in three places, all of them additive or dropped-field and all reported:
// Futures_OrdersList has no "reduceOnly", Futures_UserTrades adds "crossed", and
// Futures_InstrumentsInfo emits minQty/minNotional/pricePrecision/sizePrecision unconditionally.
func TestOutputStructsMatchTheJSONContractExactly(t *testing.T) {
	cases := []struct {
		name string
		got  any
		want map[string]any
	}{
		{
			name: "Futures_Positions",
			got: entity.Futures_Positions{
				Symbol: "ETH-USD", PositionSide: "LONG", PositionSize: "2.50000000", Leverage: "10",
				PositionID: "pos-1", EntryPrice: "2250.00000000", MarkPrice: "2300.00000000",
				UnRealizedProfit: "125.00000000", RealizedProfit: "10.00000000", Notional: "5750.00000000",
				HedgeMode: true, MarginMode: "CROSS", CreateTime: 1700000000000, UpdateTime: 1700000000001,
			},
			want: map[string]any{
				"symbol": "ETH-USD", "positionSide": "LONG", "positionSize": "2.50000000", "leverage": "10",
				"positionID": "pos-1", "entryPrice": "2250.00000000", "markPrice": "2300.00000000",
				"unRealizedProfit": "125.00000000", "realizedProfit": "10.00000000", "notional": "5750.00000000",
				"hedgeMode": true, "marginMode": "CROSS", "createTime": 1.7e12, "updateTime": 1.700000000001e12,
			},
		},
		{
			name: "Futures_OrdersList",
			got: entity.Futures_OrdersList{
				Symbol: "ETH-USD", OrderID: "ord-1", ClientOrderID: "client-1", Side: "BUY",
				ExecutedSize: "1.00000000", Price: "2500.00000000", Type: "LIMIT", Status: "ACTIVE",
				CreateTime: 1700000000000, UpdateTime: 1700000000001, PositionID: "", PositionSide: "LONG",
				PositionSize: "1.50000000", Leverage: "10", MarginMode: "CROSS", TpOrder: true, SlOrder: false,
			},
			want: map[string]any{
				"symbol": "ETH-USD", "orderId": "ord-1", "clientOrderID": "client-1", "side": "BUY",
				"executedSize": "1.00000000", "price": "2500.00000000", "type": "LIMIT", "status": "ACTIVE",
				"createTime": 1.7e12, "updateTime": 1.700000000001e12, "positionID": "", "positionSide": "LONG",
				"positionSize": "1.50000000", "leverage": "10", "marginMode": "CROSS", "tpOrder": true,
				"slOrder": false,
			},
		},
		{
			name: "FuturesBalance",
			got: entity.FuturesBalance{
				Asset: "USDC", Balance: "1450.00000000", Equity: "1500.00000000",
				Available: "1200.00000000", UnrealizedProfit: "50.00000000",
			},
			want: map[string]any{
				"asset": "USDC", "balance": "1450.00000000", "equity": "1500.00000000",
				"available": "1200.00000000", "unrealizedProfit": "50.00000000",
			},
		},
		{
			name: "Futures_OrdersHistory",
			got: entity.Futures_OrdersHistory{
				Symbol: "ETH-USD", OrderID: "ord-2", ClientOrderID: "client-2", PositionID: "",
				Side: "SELL", PositionSide: "SHORT", PositionSize: "1.00000000", ExecutedSize: "1.00000000",
				Price: "2500.00000000", ExecutedPrice: "2495.00000000", RealisedProfit: "5.00000000",
				Fee: "0.10000000", FeeAsset: "USDC", Leverage: "10", Type: "MARKET", Status: "FILLED",
				HedgeMode: false, MarginMode: "CROSS", CreateTime: 1700000000000, UpdateTime: 1700000000001,
				TpOrder: false, SlOrder: true,
			},
			want: map[string]any{
				"symbol": "ETH-USD", "orderId": "ord-2", "clientOrderID": "client-2", "positionID": "",
				"side": "SELL", "positionSide": "SHORT", "positionSize": "1.00000000", "executedSize": "1.00000000",
				"price": "2500.00000000", "executedPrice": "2495.00000000", "realisedProfit": "5.00000000",
				"fee": "0.10000000", "feeAsset": "USDC", "leverage": "10", "type": "MARKET", "status": "FILLED",
				"hedgeMode": false, "marginMode": "CROSS", "createTime": 1.7e12, "updateTime": 1.700000000001e12,
				"tpOrder": false, "slOrder": true,
			},
		},
		{
			name: "Futures_PositionsHistory",
			got: entity.Futures_PositionsHistory{
				Symbol: "ETH-USD", PositionID: "", PositionSide: "LONG", PositionAmt: "2.00000000",
				ExecutedPositionAmt: "2.00000000", AvgPrice: "2000.00000000", ExecutedAvgPrice: "2000.00000000",
				RealisedProfit: "20.00000000", Leverage: "10", Fee: "0.50000000", Funding: "-0.10000000",
				MarginMode: "CROSS", CreateTime: 1700000000000, UpdateTime: 1700000000001,
			},
			want: map[string]any{
				"symbol": "ETH-USD", "positionID": "", "positionSide": "LONG", "positionAmt": "2.00000000",
				"executedPositionAmt": "2.00000000", "avgPrice": "2000.00000000", "executedAvgPrice": "2000.00000000",
				"realisedProfit": "20.00000000", "leverage": "10", "fee": "0.50000000", "funding": "-0.10000000",
				"marginMode": "CROSS", "createTime": 1.7e12, "updateTime": 1.700000000001e12,
			},
		},
		{
			name: "Futures_InstrumentsInfo",
			got: entity.Futures_InstrumentsInfo{
				Symbol: "ETH-USD", Base: "ETH", Quote: "USD", MinQty: "0.01000000", MinNotional: "10.00000000",
				PricePrecision: "2", SizePrecision: "6", State: "LIVE", TokenId: "eth-token",
				MaxLeverage: "20", Multiplier: "1", ContractSize: "1", IsSizeContract: true,
			},
			want: map[string]any{
				"symbol": "ETH-USD", "base": "ETH", "quote": "USD", "minQty": "0.01000000", "minNotional": "10.00000000",
				"pricePrecision": "2", "sizePrecision": "6", "state": "LIVE", "tokenId": "eth-token",
				"maxLeverage": "20", "multiplier": "1", "contractSize": "1", "isSizeContract": true,
			},
		},
		{
			name: "Futures_UserTrades",
			got: entity.Futures_UserTrades{
				TradeID: "trade-1", OrderID: "ord-3", Symbol: "ETH-USD", Side: "BUY", PositionSide: "LONG",
				Price: "2500.00000000", Qty: "1.00000000", QuoteQty: "2500.00000000", Commission: "1.00000000",
				CommissionAsset: "USDC", RealisedProfit: "0.00000000", Buyer: true, Maker: false, Time: 1700000000000,
			},
			want: map[string]any{
				"tradeId": "trade-1", "orderId": "ord-3", "symbol": "ETH-USD", "side": "BUY", "positionSide": "LONG",
				"price": "2500.00000000", "qty": "1.00000000", "quoteQty": "2500.00000000", "commission": "1.00000000",
				"commissionAsset": "USDC", "realisedProfit": "0.00000000", "buyer": true, "maker": false,
				"crossed": false, "time": 1.7e12,
			},
		},
		{
			name: "AccountInformation",
			got: entity.AccountInformation{
				UID: "0xabc", Label: "main", IP: "0.0.0.0/0", CanRead: true, CanTrade: true,
				CanTransfer: false, PermSpot: false, PermFutures: true,
			},
			want: map[string]any{
				"uid": "0xabc", "label": "main", "ip": "0.0.0.0/0", "canRead": true, "canTrade": true,
				"canTransfer": false, "permSpot": false, "permFutures": true,
			},
		},
		{
			name: "Futures_Leverage",
			got: entity.Futures_Leverage{
				Symbol: "ETH-USD", Leverage: "10", LongLeverage: "", ShortLeverage: "", MarginMode: "CROSS",
			},
			want: map[string]any{
				"symbol": "ETH-USD", "leverage": "10", "longLeverage": "", "shortLeverage": "", "marginMode": "CROSS",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.got)
			if err != nil {
				t.Fatal(err)
			}
			var gotMap map[string]any
			if err := json.Unmarshal(raw, &gotMap); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotMap, tc.want) {
				t.Fatalf("JSON contract mismatch for %s:\n got  %s\n want %#v", tc.name, raw, tc.want)
			}
		})
	}
}
