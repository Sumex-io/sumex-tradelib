package katana

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// Note on two source tests with no subject here: the original connector's dispatcher answered
// "setPositionMode"/"setMarginMode" with an explicit "Exchange does not support this method." error.
// This library has no dispatcher and the katana client exposes no NewSetPositionMode/NewSetMarginMode
// factory at all, so the rejection is structural — there is no call to make and therefore no error
// string to assert.

// --- resolveOrderType / resolveOrderTypeName ---

func TestResolveOrderTypeMapsTriggerOrders(t *testing.T) {
	cases := []struct {
		orderType string
		tp, sl    bool
		want      uint8
	}{
		{"MARKET", false, false, orderTypeMarket},
		{"LIMIT", false, false, orderTypeLimit},
		{"MARKET", true, false, orderTypeTakeProfitMarket},
		{"LIMIT", true, false, orderTypeTakeProfitLimit},
		{"MARKET", false, true, orderTypeStopLossMarket},
		{"LIMIT", false, true, orderTypeStopLossLimit},
	}
	for _, tc := range cases {
		got, err := resolveOrderType(tc.orderType, tc.tp, tc.sl)
		if err != nil {
			t.Fatalf("%+v: %v", tc, err)
		}
		if got != tc.want {
			t.Fatalf("resolveOrderType(%q, tp=%v, sl=%v) = %d, want %d", tc.orderType, tc.tp, tc.sl, got, tc.want)
		}
	}
}

func TestResolveOrderTypeRejectsTpAndSlTogether(t *testing.T) {
	if _, err := resolveOrderType("MARKET", true, true); err == nil {
		t.Fatal("an order cannot be both take-profit and stop-loss")
	}
}

func TestResolveOrderTypeIsCaseInsensitiveAndTrims(t *testing.T) {
	got, err := resolveOrderType("  market  ", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != orderTypeMarket {
		t.Fatalf("resolveOrderType(lowercase, padded) = %d, want %d", got, orderTypeMarket)
	}
}

func TestResolveOrderTypeRejectsUnknownBaseType(t *testing.T) {
	if _, err := resolveOrderType("STOP", false, false); err == nil {
		t.Fatal("expected an error for an order type that is neither MARKET nor LIMIT")
	}
}

func TestResolveOrderTypeNameMapsTriggerOrders(t *testing.T) {
	cases := []struct {
		orderType string
		tp, sl    bool
		want      string
	}{
		{"MARKET", false, false, orderTypeNameMarket},
		{"LIMIT", false, false, orderTypeNameLimit},
		{"MARKET", true, false, orderTypeNameTakeProfitMarket},
		{"LIMIT", true, false, orderTypeNameTakeProfitLimit},
		{"MARKET", false, true, orderTypeNameStopLossMarket},
		{"LIMIT", false, true, orderTypeNameStopLossLimit},
	}
	for _, tc := range cases {
		got, err := resolveOrderTypeName(tc.orderType, tc.tp, tc.sl)
		if err != nil {
			t.Fatalf("%+v: %v", tc, err)
		}
		if got != tc.want {
			t.Fatalf("resolveOrderTypeName(%q, tp=%v, sl=%v) = %q, want %q", tc.orderType, tc.tp, tc.sl, got, tc.want)
		}
	}
}

func TestResolveOrderTypeNameRejectsTpAndSlTogether(t *testing.T) {
	if _, err := resolveOrderTypeName("MARKET", true, true); err == nil {
		t.Fatal("an order cannot be both take-profit and stop-loss")
	}
}

func TestResolveOrderTypeNameRejectsUnknownBaseType(t *testing.T) {
	if _, err := resolveOrderTypeName("STOP", false, false); err == nil {
		t.Fatal("expected an error for an order type that is neither MARKET nor LIMIT")
	}
}

// TestResolveOrderTypeAndNameNeverDisagree is a property check over every valid
// (orderType, tp, sl) combination: resolveOrderTypeName must always be exactly the
// orderTypeNameByCode entry for what resolveOrderType returned for the same inputs. This is the
// structural guarantee that the REST string and the signed integer never describe two different
// orders; it would catch a hand-edited orderTypeNameByCode entry that drifted from the numeric
// table.
func TestResolveOrderTypeAndNameNeverDisagree(t *testing.T) {
	bases := []string{"MARKET", "LIMIT"}
	flags := []struct{ tp, sl bool }{{false, false}, {true, false}, {false, true}}
	for _, base := range bases {
		for _, f := range flags {
			code, err := resolveOrderType(base, f.tp, f.sl)
			if err != nil {
				t.Fatalf("resolveOrderType(%q, %v, %v): %v", base, f.tp, f.sl, err)
			}
			name, err := resolveOrderTypeName(base, f.tp, f.sl)
			if err != nil {
				t.Fatalf("resolveOrderTypeName(%q, %v, %v): %v", base, f.tp, f.sl, err)
			}
			if orderTypeNameByCode[code] != name {
				t.Fatalf("code %d maps to name %q in the table but resolveOrderTypeName(%q,%v,%v) returned %q", code, orderTypeNameByCode[code], base, f.tp, f.sl, name)
			}
		}
	}
}

// --- resolveOrderSide ---

func TestResolveOrderSide(t *testing.T) {
	cases := []struct {
		in       string
		wantCode uint8
		wantName string
	}{
		{"BUY", orderSideBuy, orderSideNameBuy},
		{"buy", orderSideBuy, orderSideNameBuy},
		{" SELL ", orderSideSell, orderSideNameSell},
	}
	for _, tc := range cases {
		code, name, err := resolveOrderSide(tc.in)
		if err != nil {
			t.Fatalf("resolveOrderSide(%q): %v", tc.in, err)
		}
		if code != tc.wantCode || name != tc.wantName {
			t.Fatalf("resolveOrderSide(%q) = (%d, %q), want (%d, %q)", tc.in, code, name, tc.wantCode, tc.wantName)
		}
	}
}

func TestResolveOrderSideRejectsUnknown(t *testing.T) {
	if _, _, err := resolveOrderSide("LONG"); err == nil {
		t.Fatal("expected an error for a side that is neither BUY nor SELL")
	}
}

// TestEnumConstantsMatchAPINotesLiteralValues: the encoding-split tests below express their
// expectations using this package's OWN named constants (orderTypeTakeProfitMarket,
// triggerTypeLast, ...) and recompute the expected signature through the same signOrder the code
// under test calls — so swapping the *values* of two constants (e.g. orderTypeStopLossMarket <->
// orderTypeTakeProfitMarket) would leave every one of those tests green while every stop-loss order
// actually sent to Katana carried the take-profit enum code. This test pins every constant this
// package defines to a bare literal copied directly from API_NOTES.md section A/B — no named
// constant appears anywhere on the "want" side, by construction.
func TestEnumConstantsMatchAPINotesLiteralValues(t *testing.T) {
	intCases := []struct {
		name string
		got  uint8
		want uint8
	}{
		// Order Type Enum (API_NOTES.md section B).
		{"orderTypeMarket", orderTypeMarket, 0},
		{"orderTypeLimit", orderTypeLimit, 1},
		{"orderTypeStopLossMarket", orderTypeStopLossMarket, 2},
		{"orderTypeStopLossLimit", orderTypeStopLossLimit, 3},
		{"orderTypeTakeProfitMarket", orderTypeTakeProfitMarket, 4},
		{"orderTypeTakeProfitLimit", orderTypeTakeProfitLimit, 5},
		// Order Side Enum.
		{"orderSideBuy", orderSideBuy, 0},
		{"orderSideSell", orderSideSell, 1},
		// Order Trigger Type Enum.
		{"triggerTypeNone", triggerTypeNone, 0},
		{"triggerTypeLast", triggerTypeLast, 1},
		{"triggerTypeIndex", triggerTypeIndex, 2},
		// Order Time-In-Force Enum.
		{"timeInForceGTC", timeInForceGTC, 0},
		{"timeInForceGTX", timeInForceGTX, 1},
		{"timeInForceIOC", timeInForceIOC, 2},
		{"timeInForceFOK", timeInForceFOK, 3},
		// Order Self-Trade Prevention Enum.
		{"selfTradePreventionDC", selfTradePreventionDC, 0},
		{"selfTradePreventionCO", selfTradePreventionCO, 1},
		{"selfTradePreventionCN", selfTradePreventionCN, 2},
		{"selfTradePreventionCB", selfTradePreventionCB, 3},
	}
	for _, tc := range intCases {
		if tc.got != tc.want {
			t.Fatalf("%s = %d, want the literal %d (API_NOTES.md section B)", tc.name, tc.got, tc.want)
		}
	}

	// REST-wire string spellings (API_NOTES.md section A's "... Values" lists, plus section B's
	// enum table parentheticals).
	stringCases := []struct {
		name string
		got  string
		want string
	}{
		{"orderTypeNameMarket", orderTypeNameMarket, "market"},
		{"orderTypeNameLimit", orderTypeNameLimit, "limit"},
		{"orderTypeNameStopLossMarket", orderTypeNameStopLossMarket, "stopLossMarket"},
		{"orderTypeNameStopLossLimit", orderTypeNameStopLossLimit, "stopLossLimit"},
		{"orderTypeNameTakeProfitMarket", orderTypeNameTakeProfitMarket, "takeProfitMarket"},
		{"orderTypeNameTakeProfitLimit", orderTypeNameTakeProfitLimit, "takeProfitLimit"},
		{"orderSideNameBuy", orderSideNameBuy, "buy"},
		{"orderSideNameSell", orderSideNameSell, "sell"},
		{"triggerTypeNameLast", triggerTypeNameLast, "last"},
		{"triggerTypeNameIndex", triggerTypeNameIndex, "index"},
		{"timeInForceNameGTC", timeInForceNameGTC, "gtc"},
		{"timeInForceNameGTX", timeInForceNameGTX, "gtx"},
		{"timeInForceNameIOC", timeInForceNameIOC, "ioc"},
		{"timeInForceNameFOK", timeInForceNameFOK, "fok"},
		{"selfTradePreventionNameDC", selfTradePreventionNameDC, "dc"},
		{"selfTradePreventionNameCO", selfTradePreventionNameCO, "co"},
		{"selfTradePreventionNameCN", selfTradePreventionNameCN, "cn"},
		{"selfTradePreventionNameCB", selfTradePreventionNameCB, "cb"},
	}
	for _, tc := range stringCases {
		if tc.got != tc.want {
			t.Fatalf("%s = %q, want the literal %q (API_NOTES.md section A/B)", tc.name, tc.got, tc.want)
		}
	}
}

// --- clampIMFToMarketFloor ---

func TestClampIMFToMarketFloorPassesThroughATighteningOverride(t *testing.T) {
	// Requesting 5x leverage (IMF 0.20000000) against a market whose default is 10x (IMF
	// 0.10000000): the requested IMF is ABOVE the floor (tighter), so it passes through unchanged.
	got, err := clampIMFToMarketFloor("0.20000000", "0.10000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.20000000" {
		t.Fatalf("clampIMFToMarketFloor = %s, want the requested value unchanged (0.20000000)", got)
	}
}

func TestClampIMFToMarketFloorClampsALooseningRequestToTheMarketMaximum(t *testing.T) {
	// Requesting 50x leverage (IMF 0.02000000) against a market capped at 10x (IMF 0.10000000): the
	// requested IMF is BELOW the floor (more leverage than allowed), so it is clamped UP to the
	// market's own default — its maximum leverage — rather than rejected.
	got, err := clampIMFToMarketFloor("0.02000000", "0.10000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.10000000" {
		t.Fatalf("clampIMFToMarketFloor = %s, want clamped to the market floor (0.10000000)", got)
	}
}

func TestClampIMFToMarketFloorExactMatchPassesThrough(t *testing.T) {
	got, err := clampIMFToMarketFloor("0.10000000", "0.10000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0.10000000" {
		t.Fatalf("clampIMFToMarketFloor = %s, want 0.10000000", got)
	}
}

func TestClampIMFToMarketFloorRejectsUnparseableInput(t *testing.T) {
	if _, err := clampIMFToMarketFloor("not-a-number", "0.10000000"); err == nil {
		t.Fatal("expected an error for an unparseable requested IMF")
	}
	if _, err := clampIMFToMarketFloor("0.10000000", "not-a-number"); err == nil {
		t.Fatal("expected an error for an unparseable market default IMF")
	}
}

// API_NOTES.md documents the override as "between 1 and the market's initialMarginFraction" —
// leverageToIMF("0.5") (i.e. requesting HALF-x leverage) produces an IMF of "2.00000000", which is
// above 1 and which Katana rejects. The clamp caps the IMF at 1 regardless of how far below 1x the
// requested leverage is.
func TestClampIMFToMarketFloorClampsBelowOneXLeverageToOne(t *testing.T) {
	got, err := clampIMFToMarketFloor("2.00000000", "0.10000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.00000000" {
		t.Fatalf("clampIMFToMarketFloor = %s, want 1.00000000 (the documented upper bound)", got)
	}
}

func TestClampIMFToMarketFloorUpperBoundExactMatchPassesThrough(t *testing.T) {
	got, err := clampIMFToMarketFloor("1.00000000", "0.10000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.00000000" {
		t.Fatalf("clampIMFToMarketFloor = %s, want 1.00000000 unchanged", got)
	}
}

// --- formatMoneyField / truncateClientOrderID ---

func TestFormatMoneyField(t *testing.T) {
	got, err := formatMoneyField("size", "1.5")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.50000000" {
		t.Fatalf("formatMoneyField = %s, want 1.50000000", got)
	}
}

func TestFormatMoneyFieldRejectsNonPositive(t *testing.T) {
	if _, err := formatMoneyField("size", "0"); err == nil {
		t.Fatal("expected an error for a zero size")
	}
	if _, err := formatMoneyField("size", "-1"); err == nil {
		t.Fatal("expected an error for a negative size")
	}
}

func TestTruncateClientOrderIDLeavesShortIDsUnchanged(t *testing.T) {
	if got := truncateClientOrderID("abc123"); got != "abc123" {
		t.Fatalf("truncateClientOrderID = %q, want abc123 unchanged", got)
	}
}

func TestTruncateClientOrderIDCapsAtFortyBytes(t *testing.T) {
	long := strings.Repeat("x", 55)
	got := truncateClientOrderID(long)
	if len(got) != 40 {
		t.Fatalf("truncateClientOrderID length = %d, want 40", len(got))
	}
	if got != long[:40] {
		t.Fatalf("truncateClientOrderID did not keep the leading 40 bytes")
	}
}

// --- Test fixtures / helpers for placeOrder/cancelOrder/amendOrder/setLeverage ---

// nonceBigIntFromString converts a nonce UUID string (as captured off the wire) back into the
// uint128 big.Int the EIP-712 signature is computed over, using the exact same conversion newNonce()
// performs (SetBytes over the raw 16 UUID bytes) — used by every signature cross-check test below
// to independently recompute "what should have been signed" using the nonce the code under test
// actually generated and sent.
func nonceBigIntFromString(t *testing.T, s string) *big.Int {
	t.Helper()
	u, err := uuid.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return new(big.Int).SetBytes(u[:])
}

func readSignedBody[T any](t *testing.T, r *http.Request) katanaSignedRequest[T] {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var out katanaSignedRequest[T]
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// validWalletAddress/signedWalletFixture replace the read path's "0xWALLET" for every write-path
// test in this file: EIP-712 typed data validates `address`-typed fields strictly
// (apitypes.TypedDataAndHash), and "0xWALLET" is not a well-formed 20-byte hex address — fine for
// the read-path tests (which never sign anything) but rejected outright the moment signOrder/
// signCancelByOrderIds/signIMFOverride tries to hash it. The address itself is API_NOTES.md's own
// sample wallet, copied verbatim.
const validWalletAddress = "0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d"

const signedWalletFixture = `[{"wallet":"0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d","equity":"10000.00000000","freeCollateral":"9000.00000000","quoteBalance":"9500.00000000","unrealizedPnL":"100.00000000"}]`

const placedOrderFixture = `{
	"market":"ETH-USD","orderId":"ord-1","clientOrderId":"abc123","wallet":"0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d",
	"time":1700000000000,"status":"filled","type":"market","side":"buy",
	"originalQuantity":"1.00000000","executedQuantity":"1.00000000","reduceOnly":false,
	"selfTradePrevention":"dc"
}`

// newSigningFuturesClient is the write path's client: it carries the delegated private key, so
// every EIP-712 signature this file cross-checks is produced by the same key the code under test
// signs with.
func newSigningFuturesClient(baseURL string) *FuturesClient {
	c := NewFuturesClient("k", "s", testPrivKey)
	c.BaseURL = baseURL
	return c
}

func mustDelegatedAddress(t *testing.T, c *FuturesClient) string {
	t.Helper()
	addr, err := c.signer().delegatedAddress()
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

// --- placeOrder ---

// TestPlaceOrderMarketBuySendsStringBodyAndIntegerSignature is the most important test in this
// file: a single logical MARKET BUY order must put the STRING form ("market"/"buy"/"gtc"/"dc") in
// the REST body and the INTEGER form (0/0/0/0) in the EIP-712 signature. The signature half is
// verified by independently recomputing what signOrder should have produced for the hand-written
// expected integers plus the nonce/wallet actually sent, and asserting it equals the signature
// actually sent — if resolveOrderType/resolveOrderSide ever signed the wrong integer, this
// recomputed value would differ and the test would fail.
func TestPlaceOrderMarketBuySendsStringBodyAndIntegerSignature(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeBuy).
		Size("1.00000000").
		OrderType(entity.OrderTypeMarket).
		ClientOrderID("abc123").
		Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []entity.PlaceOrder{{OrderID: "ord-1", ClientOrderID: "abc123", Ts: 1700000000000}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("placeOrder response = %+v, want %+v", got, want)
	}

	p := captured.Parameters
	if p.Market != "ETH-USD" {
		t.Fatalf("market = %q, want ETH-USD", p.Market)
	}
	if p.Type != "market" {
		t.Fatalf("type = %q, want the STRING \"market\", not a numeric code", p.Type)
	}
	if p.Side != "buy" {
		t.Fatalf("side = %q, want the STRING \"buy\", not a numeric code", p.Side)
	}
	if p.Quantity != "1.00000000" {
		t.Fatalf("quantity = %q, want 1.00000000", p.Quantity)
	}
	if p.ReduceOnly {
		t.Fatal("reduceOnly = true, want false for a plain (non-TP/SL) order with no reduceOnly requested")
	}
	if p.TimeInForce != "gtc" {
		t.Fatalf("timeInForce = %q, want gtc", p.TimeInForce)
	}
	if p.SelfTradePrevention != "dc" {
		t.Fatalf("selfTradePrevention = %q, want dc", p.SelfTradePrevention)
	}
	if p.Price != "" || p.TriggerPrice != "" || p.TriggerType != "" {
		t.Fatalf("a MARKET order must omit price/triggerPrice/triggerType, got %+v", p)
	}
	if p.ClientOrderId != "abc123" {
		t.Fatalf("clientOrderId = %q, want abc123", p.ClientOrderId)
	}
	if p.Wallet != validWalletAddress {
		t.Fatalf("wallet = %q, want %q", p.Wallet, validWalletAddress)
	}
	delegated := mustDelegatedAddress(t, c)
	if p.DelegatedKey != delegated {
		t.Fatalf("delegatedKey = %q, want %q (must equal delegatedAddress(), never re-derived)", p.DelegatedKey, delegated)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signOrder(orderSignPayload{
		Nonce:               nonce,
		Wallet:              validWalletAddress,
		MarketSymbol:        "ETH-USD",
		OrderType:           orderTypeMarket, // 0 — the INTEGER, not "market"
		OrderSide:           orderSideBuy,    // 0 — the INTEGER, not "buy"
		Quantity:            "1.00000000",
		LimitPrice:          "0.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         triggerTypeNone, // 0
		IsReduceOnly:        false,
		TimeInForce:         timeInForceGTC,        // 0 — the INTEGER, not "gtc"
		SelfTradePrevention: selfTradePreventionDC, // 0 — the INTEGER, not "dc"
		ClientOrderId:       "abc123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (recomputed from the expected numeric enum values 0/0/0/0/0)", captured.Signature, wantSig)
	}
}

// TestPlaceOrderTakeProfitMarketForcesReduceOnlyAndSignsTheTriggerEnums mirrors the test above for
// a take-profit MARKET SELL order: the wire body must carry type="takeProfitMarket"/
// triggerType="last" (strings) and reduceOnly=true (forced, even though the request explicitly set
// Reduce(false)), while the signature must cover orderType=4 (Take profit market) and triggerType=1
// (Last price) — the INTEGER forms, per API_NOTES.md section B's numeric enum table.
func TestPlaceOrderTakeProfitMarketForcesReduceOnlyAndSignsTheTriggerEnums(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-2","time":1700000001000,"status":"active","type":"takeProfitMarket","side":"sell","originalQuantity":"2.00000000","executedQuantity":"0.00000000","reduceOnly":true,"selfTradePrevention":"dc","triggerPrice":"2500.00000000","triggerType":"last"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeSell).
		Size("2.00000000").
		OrderType(entity.OrderTypeMarket).
		TpOrder(true).
		TpPrice("2500").
		Reduce(false). // deliberately false: must be forced to true anyway
		Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].OrderID != "ord-2" {
		t.Fatalf("placeOrder response = %+v, want orderId ord-2", got)
	}

	p := captured.Parameters
	if p.Type != "takeProfitMarket" {
		t.Fatalf("type = %q, want takeProfitMarket", p.Type)
	}
	if p.Side != "sell" {
		t.Fatalf("side = %q, want sell", p.Side)
	}
	if !p.ReduceOnly {
		t.Fatal("reduceOnly = false, want true (forced for take-profit orders regardless of the request)")
	}
	if p.TriggerPrice != "2500.00000000" {
		t.Fatalf("triggerPrice = %q, want 2500.00000000 (from tpPrice, formatted to 8 decimals)", p.TriggerPrice)
	}
	if p.TriggerType != "last" {
		t.Fatalf("triggerType = %q, want last", p.TriggerType)
	}
	if p.Price != "" {
		t.Fatalf("price = %q, want omitted for a *Market trigger variant", p.Price)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signOrder(orderSignPayload{
		Nonce:               nonce,
		Wallet:              validWalletAddress,
		MarketSymbol:        "ETH-USD",
		OrderType:           orderTypeTakeProfitMarket, // 4
		OrderSide:           orderSideSell,             // 1
		Quantity:            "2.00000000",
		LimitPrice:          "0.00000000",
		TriggerPrice:        "2500.00000000",
		TriggerType:         triggerTypeLast, // 1
		IsReduceOnly:        true,
		TimeInForce:         timeInForceGTC,
		SelfTradePrevention: selfTradePreventionDC,
		ClientOrderId:       "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (recomputed from orderType=4, orderSide=1, triggerType=1, isReduceOnly=true)", captured.Signature, wantSig)
	}
}

// TestPlaceOrderStopLossLimitMatchesTheDocumentedSample uses API_NOTES.md's own verbatim
// stopLossLimit request sample (section A, POST /v1/orders) as the hand-written expected values —
// market/type/side/quantity/price/triggerPrice/triggerType/clientOrderId are copied
// character-for-character from the docs, not derived from this code. reduceOnly is the one
// deliberate divergence: the docs' illustrative sample doesn't set it (so it would default false),
// but this connector forces reduceOnly true for every TP/SL order.
func TestPlaceOrderStopLossLimitMatchesTheDocumentedSample(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-3","clientOrderId":"199283","time":1705779328613,"status":"active","type":"stopLossLimit","side":"sell","originalQuantity":"4.95500000","executedQuantity":"0.00000000","price":"2440.00000000","triggerPrice":"2450.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeSell).
		Size("4.95500000").
		Price("2440.00000000").
		OrderType(entity.OrderTypeLimit).
		SlOrder(true).
		SlPrice("2450.00000000").
		ClientOrderID("199283").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	p := captured.Parameters
	if p.Market != "ETH-USD" || p.Type != "stopLossLimit" || p.Side != "sell" || p.Quantity != "4.95500000" {
		t.Fatalf("body = %+v, want the docs' sample market/type/side/quantity", p)
	}
	if p.Price != "2440.00000000" {
		t.Fatalf("price = %q, want 2440.00000000 (the docs' sample)", p.Price)
	}
	if p.TriggerPrice != "2450.00000000" {
		t.Fatalf("triggerPrice = %q, want 2450.00000000 (the docs' sample)", p.TriggerPrice)
	}
	if p.TriggerType != "last" {
		t.Fatalf("triggerType = %q, want last", p.TriggerType)
	}
	if p.ClientOrderId != "199283" {
		t.Fatalf("clientOrderId = %q, want 199283", p.ClientOrderId)
	}
	if !p.ReduceOnly {
		t.Fatal("reduceOnly = false, want true (forced for stop-loss orders)")
	}
}

// --- TP/SL trigger price falls back to `price` ---
//
// The common caller convention for "set a take-profit/stop-loss" is
// {orderType: LIMIT, price, tpOrder|slOrder: true} — nothing populates the dedicated
// TpPrice/SlPrice inputs. The three tests below cover the two shapes that actually arrive plus the
// explicit-field path that must keep winning.

// TestPlaceOrderTakeProfitWithOnlyPriceUsesItAsTheTriggerPrice sends exactly that: LIMIT + price +
// tpOrder, no tpPrice. Hard-requiring the dedicated field would reject it before any network call,
// so every Katana take-profit would fail instantly.
func TestPlaceOrderTakeProfitWithOnlyPriceUsesItAsTheTriggerPrice(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-tp-fallback","time":1700000002000,"status":"active","type":"takeProfitLimit","side":"sell","originalQuantity":"1.50000000","executedQuantity":"0.00000000","price":"2700.00000000","triggerPrice":"2700.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeSell).
		Size("1.5").
		OrderType(entity.OrderTypeLimit).
		Price("2700").
		TpOrder(true).
		// TpPrice deliberately absent — this is the only shape most callers can send.
		Do(context.Background())
	if err != nil {
		t.Fatalf("placeOrder returned %v, want a take-profit built from `price` alone", err)
	}
	if len(got) != 1 || got[0].OrderID != "ord-tp-fallback" {
		t.Fatalf("placeOrder response = %+v, want orderId ord-tp-fallback", got)
	}

	p := captured.Parameters
	if p.Type != "takeProfitLimit" {
		t.Fatalf("type = %q, want takeProfitLimit", p.Type)
	}
	if p.TriggerPrice != "2700.00000000" {
		t.Fatalf("triggerPrice = %q, want 2700.00000000 (fell back to the order's own price)", p.TriggerPrice)
	}
	if p.Price != "2700.00000000" {
		t.Fatalf("price = %q, want 2700.00000000 (a *Limit variant still sends its limit price)", p.Price)
	}
	if p.TriggerType != "last" {
		t.Fatalf("triggerType = %q, want last", p.TriggerType)
	}
	if !p.ReduceOnly {
		t.Fatal("reduceOnly = false, want true (forced for take-profit orders)")
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signOrder(orderSignPayload{
		Nonce:               nonce,
		Wallet:              validWalletAddress,
		MarketSymbol:        "ETH-USD",
		OrderType:           orderTypeTakeProfitLimit, // 5
		OrderSide:           orderSideSell,            // 1
		Quantity:            "1.50000000",
		LimitPrice:          "2700.00000000",
		TriggerPrice:        "2700.00000000",
		TriggerType:         triggerTypeLast, // 1
		IsReduceOnly:        true,
		TimeInForce:         timeInForceGTC,
		SelfTradePrevention: selfTradePreventionDC,
		ClientOrderId:       "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (recomputed with limitPrice=triggerPrice=2700.00000000)", captured.Signature, wantSig)
	}
}

// TestPlaceOrderStopLossWithOnlyPriceUsesItAsTheTriggerPrice is the stop-loss half of the same
// defect — the one whose failure scenario leaves an open position unprotected.
func TestPlaceOrderStopLossWithOnlyPriceUsesItAsTheTriggerPrice(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-sl-fallback","time":1700000003000,"status":"active","type":"stopLossLimit","side":"sell","originalQuantity":"3.00000000","executedQuantity":"0.00000000","price":"1800.00000000","triggerPrice":"1800.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeSell).
		Size("3").
		OrderType(entity.OrderTypeLimit).
		Price("1800").
		SlOrder(true).
		// SlPrice deliberately absent.
		Do(context.Background()); err != nil {
		t.Fatalf("placeOrder returned %v, want a stop-loss built from `price` alone", err)
	}

	p := captured.Parameters
	if p.Type != "stopLossLimit" {
		t.Fatalf("type = %q, want stopLossLimit", p.Type)
	}
	if p.TriggerPrice != "1800.00000000" {
		t.Fatalf("triggerPrice = %q, want 1800.00000000 (fell back to the order's own price)", p.TriggerPrice)
	}
	if p.Price != "1800.00000000" {
		t.Fatalf("price = %q, want 1800.00000000", p.Price)
	}
	if p.TriggerType != "last" {
		t.Fatalf("triggerType = %q, want last", p.TriggerType)
	}
	if !p.ReduceOnly {
		t.Fatal("reduceOnly = false, want true (forced for stop-loss orders)")
	}
}

// TestPlaceOrderExplicitTpPriceStillWinsOverPrice pins the other half of the fallback ruling: it
// must not shadow a caller that DOES send the dedicated field. price and tpPrice are deliberately
// different so a regression that preferred `price` could not pass.
func TestPlaceOrderExplicitTpPriceStillWinsOverPrice(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-tp-explicit","time":1700000004000,"status":"active","type":"takeProfitLimit","side":"sell","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2600.00000000","triggerPrice":"2900.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeSell).
		Size("1").
		OrderType(entity.OrderTypeLimit).
		Price("2600").
		TpOrder(true).
		TpPrice("2900").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	p := captured.Parameters
	if p.TriggerPrice != "2900.00000000" {
		t.Fatalf("triggerPrice = %q, want 2900.00000000 (the explicit tpPrice, not the limit price)", p.TriggerPrice)
	}
	if p.Price != "2600.00000000" {
		t.Fatalf("price = %q, want 2600.00000000 (the limit price is untouched by the trigger)", p.Price)
	}
}

// TestResolveTriggerPriceRawPrefersTheDedicatedFieldAndTrims covers the helper directly, including
// the blank-string-counts-as-absent rule the pointer-shaped inputs make reachable.
func TestResolveTriggerPriceRawPrefersTheDedicatedFieldAndTrims(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	cases := []struct {
		name             string
		dedicated, price *string
		want             string
	}{
		{"dedicated wins", strPtr("2900"), strPtr("2600"), "2900"},
		{"falls back to price", nil, strPtr("2600"), "2600"},
		{"blank dedicated falls back to price", strPtr("   "), strPtr("2600"), "2600"},
		{"both absent", nil, nil, ""},
		{"both blank", strPtr(""), strPtr("  "), ""},
		{"trims the winner", strPtr("  2900  "), nil, "2900"},
	}
	for _, tc := range cases {
		if got := resolveTriggerPriceRaw(tc.dedicated, tc.price); got != tc.want {
			t.Fatalf("%s: resolveTriggerPriceRaw = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPlaceOrderRejectsTakeProfitAndStopLossTogetherWithoutAnyNetworkCall(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0") // unreachable — a network call here fails

	_, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeBuy).
		Size("1").
		TpOrder(true).
		SlOrder(true).
		TpPrice("2500").
		SlPrice("2000").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected an error for an order that is both take-profit and stop-loss")
	}
}

func TestPlaceOrderRequiresSymbolSideSize(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")

	if _, err := c.NewPlaceOrder().Side(entity.SideTypeBuy).Size("1").OrderType(entity.OrderTypeMarket).
		Do(context.Background()); err == nil {
		t.Fatal("expected an error when symbol is missing")
	}
	if _, err := c.NewPlaceOrder().Symbol("ETH-USD").Size("1").OrderType(entity.OrderTypeMarket).
		Do(context.Background()); err == nil {
		t.Fatal("expected an error when side is missing")
	}
	if _, err := c.NewPlaceOrder().Symbol("ETH-USD").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).
		Do(context.Background()); err == nil {
		t.Fatal("expected an error when size is missing")
	}
}

func TestPlaceOrderLimitOrderRequiresPrice(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")
	_, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeBuy).
		Size("1").
		OrderType(entity.OrderTypeLimit).
		Do(context.Background())
	if err == nil {
		t.Fatal("expected an error when a LIMIT order has no price")
	}
}

// TestPlaceOrderTakeProfitRequiresATriggerPriceFromSomewhere: a take-profit accepts either tpPrice
// or the order's own price, but a request carrying NEITHER still has no trigger to place and must
// be rejected before any network call.
func TestPlaceOrderTakeProfitRequiresATriggerPriceFromSomewhere(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")
	_, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeBuy).
		Size("1").
		OrderType(entity.OrderTypeMarket).
		TpOrder(true).
		Do(context.Background())
	if err == nil {
		t.Fatal("expected an error when a take-profit order has neither tpPrice nor price")
	}
	if err.Error() != "katana: a take-profit order requires tpPrice or price" {
		t.Fatalf("err = %q, want %q", err.Error(), "katana: a take-profit order requires tpPrice or price")
	}
}

// TestPlaceOrderRejectsAnOrderTypeKatanaCannotExpress: Katana's six order types are the base type
// (MARKET/LIMIT) crossed with the TP/SL flags, so entity.OrderTypeStop and OrderTypeTakeProfit have
// no Katana counterpart. They are rejected before any network call rather than guessed at.
func TestPlaceOrderRejectsAnOrderTypeKatanaCannotExpress(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")
	for _, ot := range []entity.OrderType{entity.OrderTypeStop, entity.OrderTypeTakeProfit} {
		_, err := c.NewPlaceOrder().
			Symbol("ETH-USD").
			Side(entity.SideTypeBuy).
			Size("1").
			Price("2000").
			OrderType(ot).
			Do(context.Background())
		if err == nil {
			t.Fatalf("expected an error for order type %q, which Katana cannot express as a base type", string(ot))
		}
	}
}

func TestPlaceOrderClientOrderIdIsPrefixedWithBrokerIDAndTruncated(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)
	c.SetBrokerID(strings.Repeat("B", 30) + "-")

	if _, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeBuy).
		Size("1").
		OrderType(entity.OrderTypeMarket).
		ClientOrderID(strings.Repeat("c", 20)).
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	got := captured.Parameters.ClientOrderId
	if len(got) != maxClientOrderIDBytes {
		t.Fatalf("clientOrderId length = %d, want the 40-byte cap after builder-code prefixing", len(got))
	}
	if !strings.HasPrefix(got, strings.Repeat("B", 30)+"-") {
		t.Fatalf("clientOrderId = %q, want it prefixed with the builder code", got)
	}
}

// The builder code is how Katana attributes builder rewards. It reaches the action from the
// client's BrokerID, which is how every connector in this library carries the same idea.
func TestResolveClientOrderIDPrefixesWithTheBrokerID(t *testing.T) {
	if got := resolveClientOrderID("SUMEX-", "abc"); got != "SUMEX-abc" {
		t.Fatalf("resolveClientOrderID = %q, want SUMEX-abc", got)
	}
}

func TestResolveClientOrderIDLeavesTheIDAloneWhenNoBrokerIDIsSet(t *testing.T) {
	if got := resolveClientOrderID("", "abc"); got != "abc" {
		t.Fatalf("resolveClientOrderID = %q, want abc (no prefix when no builder code is configured)", got)
	}
}

// The BrokerID must reach the SIGNED clientOrderId too, not just the wire body — the two encodings
// of an order must always describe the same order.
func TestPlaceOrderBrokerIDPrefixIsAlsoSigned(t *testing.T) {
	var captured katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)
	c.SetBrokerID("SUMEX-")

	if _, err := c.NewPlaceOrder().
		Symbol("ETH-USD").
		Side(entity.SideTypeBuy).
		Size("1").
		OrderType(entity.OrderTypeMarket).
		ClientOrderID("abc123").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	p := captured.Parameters
	if p.ClientOrderId != "SUMEX-abc123" {
		t.Fatalf("clientOrderId = %q, want SUMEX-abc123", p.ClientOrderId)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signOrder(orderSignPayload{
		Nonce:               nonce,
		Wallet:              validWalletAddress,
		MarketSymbol:        "ETH-USD",
		OrderType:           orderTypeMarket,
		OrderSide:           orderSideBuy,
		Quantity:            "1.00000000",
		LimitPrice:          "0.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         triggerTypeNone,
		IsReduceOnly:        false,
		TimeInForce:         timeInForceGTC,
		SelfTradePrevention: selfTradePreventionDC,
		ClientOrderId:       "SUMEX-abc123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (the prefixed clientOrderId must be the one that was signed)", captured.Signature, wantSig)
	}
}

func TestPlaceOrderPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":     func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnprocessableEntity) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewPlaceOrder().
		Symbol("ETH-USD").Side(entity.SideTypeBuy).Size("1").OrderType(entity.OrderTypeMarket).
		Do(context.Background()); err == nil {
		t.Fatal("expected placeOrder to propagate a non-2xx response from POST /v1/orders")
	}
}

// TestPlaceOrderErrorsOnForcedCancellationWithZeroExecutedQuantity: Katana returns HTTP 200 (not an
// HTTP error) with `errorCode`/`errorMessage` set on the response body when an order is accepted
// but immediately/fully canceled by the matching engine (API_NOTES.md's "Order Placement
// Cancellations" table, e.g. an order below the maker minimum -> QUANTITY_TOO_LOW). Without this
// check, that response maps into a normal-looking success — the caller gets an orderId and a
// timestamp back for an order that was never actually placed.
func TestPlaceOrderErrorsOnForcedCancellationWithZeroExecutedQuantity(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-1","time":1700000000000,"status":"canceled","errorCode":"QUANTITY_TOO_LOW","errorMessage":"Order quantity does not meet the maker minimum.","type":"limit","side":"buy","originalQuantity":"0.00001000","executedQuantity":"0.00000000","reduceOnly":false,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewPlaceOrder().
		Symbol("ETH-USD").Side(entity.SideTypeBuy).Size("0.00001").OrderType(entity.OrderTypeMarket).
		Do(context.Background())
	if err == nil {
		t.Fatal("expected placeOrder to error on a forced cancellation with zero executedQuantity")
	}
	if !strings.Contains(err.Error(), "QUANTITY_TOO_LOW") {
		t.Fatalf("error = %q, want it to include Katana's errorCode (the response shape has no field for it, so the error channel is the only route to the user)", err.Error())
	}
}

// TestPlaceOrderSucceedsOnForcedCancellationWithPartialFill is the other half: API_NOTES.md
// documents that self-trade prevention can force-cancel an order AFTER it partially filled
// ("Order Placement Cancellations": SELF_TRADE, TIME_IN_FORCE). A non-zero executedQuantity means
// real money already changed hands — that is a genuine, successful (if partial) order and must NOT
// be turned into an error just because errorCode is also set.
func TestPlaceOrderSucceedsOnForcedCancellationWithPartialFill(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-1","time":1700000000000,"status":"canceled","errorCode":"SELF_TRADE","errorMessage":"Order matched another order placed by the same wallet or API account.","type":"limit","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.40000000","reduceOnly":false,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewPlaceOrder().
		Symbol("ETH-USD").Side(entity.SideTypeBuy).Size("1").OrderType(entity.OrderTypeMarket).
		Do(context.Background())
	if err != nil {
		t.Fatalf("expected placeOrder to succeed for a partial fill despite errorCode being set, got: %v", err)
	}
	if len(got) != 1 || got[0].OrderID != "ord-1" {
		t.Fatalf("placeOrder = %+v, want the partially-filled order ord-1", got)
	}
}

// --- cancelOrder ---

func TestCancelOrderSendsOrderIdsAndIntegerSignature(t *testing.T) {
	var captured katanaSignedRequest[katanaCancelByOrderIdsParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaCancelByOrderIdsParams](t, r)
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewCancelOrder().OrderID("ord-1").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].OrderID != "ord-1" {
		t.Fatalf("cancelOrder = %+v, want one entry with orderId ord-1", got)
	}

	p := captured.Parameters
	if len(p.OrderIds) != 1 || p.OrderIds[0] != "ord-1" {
		t.Fatalf("orderIds = %v, want [ord-1]", p.OrderIds)
	}
	if p.Wallet != validWalletAddress {
		t.Fatalf("wallet = %q, want %q", p.Wallet, validWalletAddress)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signCancelByOrderIds(nonce, validWalletAddress, []string{"ord-1"})
	if err != nil {
		t.Fatal(err)
	}
	if captured.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (recomputed via signCancelByOrderIds with the same nonce/wallet/orderIds)", captured.Signature, wantSig)
	}
}

func TestCancelOrderRequiresOrderID(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")
	if _, err := c.NewCancelOrder().Do(context.Background()); err == nil {
		t.Fatal("expected an error when orderID is missing")
	}
}

func TestCancelOrderPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewCancelOrder().OrderID("ord-1").Do(context.Background()); err == nil {
		t.Fatal("expected cancelOrder to propagate a non-2xx response from DELETE /v1/orders")
	}
}

func TestCancelOrderErrorsOnEmptyResponse(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewCancelOrder().OrderID("ord-1").Do(context.Background()); err == nil {
		t.Fatal("expected an error when DELETE /v1/orders returns an empty array")
	}
}

// TestCancelOrderErrorsOnNotFoundStatus: API_NOTES.md requires the client to "confirm cancellation
// by checking the orderId's presence/status in the response", and the only two documented statuses
// are canceled/notFound. Parsing `status` but never looking at it would let a race where the order
// fills an instant before the cancel reaches Katana (status notFound) be silently treated as a
// successful cancellation.
func TestCancelOrderErrorsOnNotFoundStatus(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[{"status":"notFound"}]`) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewCancelOrder().OrderID("ord-1").Do(context.Background())
	if err == nil {
		t.Fatal("expected an error when DELETE /v1/orders reports status notFound")
	}
	if !strings.Contains(err.Error(), "notFound") {
		t.Fatalf("error = %q, want it to mention the notFound status", err.Error())
	}
}

// TestCancelOrderReturnsOrderIdInErrorWhenResponseOmitsIt covers the documented case where
// `orderId` itself is absent (API_NOTES.md: "absent for notFound orders specified by
// clientOrderId") — the error must still name an order, falling back to the orderID the caller
// requested rather than leaving the message blank.
func TestCancelOrderReturnsOrderIdInErrorWhenResponseOmitsIt(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[{"status":"notFound"}]`) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	orderID := "client:199283"
	_, err := c.NewCancelOrder().OrderID(orderID).Do(context.Background())
	if err == nil || !strings.Contains(err.Error(), orderID) {
		t.Fatalf("error = %v, want it to name %q even though the response omitted orderId", err, orderID)
	}
}

// --- amendOrder ---

// plainOpenOrderFixture/reduceOnlyOpenOrderFixture/stopLossOpenOrderFixture are
// GET /v1/orders?orderId=... responses (an array, per API_NOTES.md) used by every amendOrder test
// below to answer fetchOrderByID's pre-cancel read.
const plainOpenOrderFixture = `[{"market":"ETH-USD","orderId":"ord-1","time":1699999999000,"status":"active","type":"limit","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2500.00000000","reduceOnly":false,"selfTradePrevention":"dc"}]`

const reduceOnlyOpenOrderFixture = `[{"market":"ETH-USD","orderId":"ord-1","time":1699999999000,"status":"active","type":"limit","side":"sell","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2500.00000000","reduceOnly":true,"selfTradePrevention":"dc"}]`

const stopLossOpenOrderFixture = `[{"market":"ETH-USD","orderId":"ord-1","time":1699999999000,"status":"active","type":"stopLossLimit","side":"sell","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2440.00000000","triggerPrice":"2450.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}]`

func TestAmendOrderCancelsThenPlacesALimitOrder(t *testing.T) {
	var cancelCalls, placeCalls int
	var cancelBody katanaSignedRequest[katanaCancelByOrderIdsParams]
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			cancelCalls++
			cancelBody = readSignedBody[katanaCancelByOrderIdsParams](t, r)
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeCalls++
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-2","time":1700000002000,"status":"active","type":"limit","side":"buy","originalQuantity":"2.00000000","executedQuantity":"0.00000000","price":"2600.00000000","reduceOnly":false,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewAmendOrder().
		Symbol("ETH-USD").
		OrderID("ord-1").
		Side(entity.SideTypeBuy).
		NewSize("2.00000000").
		NewPrice("2600.00000000").
		Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if cancelCalls != 1 {
		t.Fatalf("cancel calls = %d, want exactly 1", cancelCalls)
	}
	if placeCalls != 1 {
		t.Fatalf("place calls = %d, want exactly 1", placeCalls)
	}
	if len(cancelBody.Parameters.OrderIds) != 1 || cancelBody.Parameters.OrderIds[0] != "ord-1" {
		t.Fatalf("cancel orderIds = %v, want [ord-1]", cancelBody.Parameters.OrderIds)
	}
	if placeBody.Parameters.Type != "limit" {
		t.Fatalf("replacement order type = %q, want limit (original order was a plain limit order, not TP/SL)", placeBody.Parameters.Type)
	}
	if placeBody.Parameters.Side != "buy" || placeBody.Parameters.Quantity != "2.00000000" || placeBody.Parameters.Price != "2600.00000000" {
		t.Fatalf("replacement order = %+v, want side=buy quantity=2.00000000 price=2600.00000000", placeBody.Parameters)
	}
	if placeBody.Parameters.ReduceOnly {
		t.Fatal("reduceOnly = true, want false (the original order was not reduce-only)")
	}
	if placeBody.Parameters.ClientOrderId == "" {
		t.Fatal("replacement order must carry a fresh clientOrderId when none was supplied")
	}
	if len(got) != 1 || got[0].OrderID != "ord-2" {
		t.Fatalf("amendOrder = %+v, want the placed replacement order (ord-2)", got)
	}
}

func TestAmendOrderUsesTheSuppliedNewClientOrderID(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").
		OrderID("ord-1").
		Side(entity.SideTypeBuy).
		NewSize("1").
		NewPrice("2600").
		NewClientOrderID("caller-supplied-id").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if placeBody.Parameters.ClientOrderId != "caller-supplied-id" {
		t.Fatalf("clientOrderId = %q, want the caller-supplied newClientOrderID", placeBody.Parameters.ClientOrderId)
	}
}

// TestAmendOrderPreservesReduceOnlyOnAPlainOrder: a plain (non-TP/SL) reduce-only limit order —
// e.g. one placed by a close-position modal — must still be reduce-only after an amend, even though
// the amend request carries no reduceOnly input at all. A naive replacement that always sent
// reduceOnly:false could let a user's "reduce my position" order flip into one that increases it.
func TestAmendOrderPreservesReduceOnlyOnAPlainOrder(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, reduceOnlyOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeSell).
		NewSize("1").NewPrice("2550").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !placeBody.Parameters.ReduceOnly {
		t.Fatal("reduceOnly = false, want true — the original order was reduce-only and the amend request has no field to re-supply it, so it MUST be read from the original order")
	}
	if placeBody.Parameters.Type != "limit" {
		t.Fatalf("type = %q, want limit (a plain reduce-only order is not classified take-profit/stop-loss)", placeBody.Parameters.Type)
	}
}

// TestAmendOrderPreservesTypeTriggerPriceAndTriggerTypeForAStopLossOrder: amending a stop-loss
// order must keep it a stop-loss order — same type/triggerPrice/triggerType/reduceOnly, only
// quantity and limit price change — never silently downgrade it to a plain limit order. The
// signature is independently recomputed with orderTypeStopLossLimit/triggerTypeLast (the INTEGER
// encoding) to prove the replacement's signed payload, not just its wire body, preserves the
// classification.
func TestAmendOrderPreservesTypeTriggerPriceAndTriggerTypeForAStopLossOrder(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, stopLossOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, `{"market":"ETH-USD","orderId":"ord-2","time":1700000003000,"status":"active","type":"stopLossLimit","side":"sell","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2430.00000000","triggerPrice":"2450.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeSell).
		NewSize("1").NewPrice("2430").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	p := placeBody.Parameters
	if p.Type != "stopLossLimit" {
		t.Fatalf("type = %q, want stopLossLimit (must NOT downgrade to a plain limit order)", p.Type)
	}
	if !p.ReduceOnly {
		t.Fatal("reduceOnly = false, want true")
	}
	if p.TriggerPrice != "2450.00000000" {
		t.Fatalf("triggerPrice = %q, want 2450.00000000 (the ORIGINAL order's trigger price, unchanged by the amend)", p.TriggerPrice)
	}
	if p.TriggerType != "last" {
		t.Fatalf("triggerType = %q, want last (the ORIGINAL order's trigger type)", p.TriggerType)
	}
	if p.Price != "2430.00000000" {
		t.Fatalf("price = %q, want 2430.00000000 (the NEW limit price from the amend request)", p.Price)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signOrder(orderSignPayload{
		Nonce:               nonce,
		Wallet:              validWalletAddress,
		MarketSymbol:        "ETH-USD",
		OrderType:           orderTypeStopLossLimit, // 3 — the INTEGER, preserved from the original
		OrderSide:           orderSideSell,          // 1
		Quantity:            "1.00000000",
		LimitPrice:          "2430.00000000",
		TriggerPrice:        "2450.00000000",
		TriggerType:         triggerTypeLast, // 1 — the INTEGER, preserved from the original
		IsReduceOnly:        true,
		TimeInForce:         timeInForceGTC,
		SelfTradePrevention: selfTradePreventionDC,
		DelegatedPublicKey:  mustDelegatedAddress(t, c),
		ClientOrderId:       p.ClientOrderId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if placeBody.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (recomputed with orderType=3, orderSide=1, triggerType=1, isReduceOnly=true — the preserved classification)", placeBody.Signature, wantSig)
	}
}

func TestAmendOrderPropagatesFetchOrderError(t *testing.T) {
	var deleteCalls, postCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { deleteCalls++ },
		"POST /v1/orders":   func(w http.ResponseWriter, r *http.Request) { postCalls++ },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2600").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to fail when the original order cannot be read")
	}
	if deleteCalls != 0 || postCalls != 0 {
		t.Fatalf("delete calls = %d, post calls = %d, want 0/0 (must never cancel or place without first reading the original order)", deleteCalls, postCalls)
	}
}

// stopLossOpenOrderMissingTriggerTypeFixture/stopLossOpenOrderMissingTriggerPriceFixture are
// GET /v1/orders responses for a stop-loss-classified order whose triggerType/triggerPrice field is
// absent — API_NOTES.md documents BOTH as optional response fields.
const stopLossOpenOrderMissingTriggerTypeFixture = `[{"market":"ETH-USD","orderId":"ord-1","time":1699999999000,"status":"active","type":"stopLossLimit","side":"sell","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2440.00000000","triggerPrice":"2450.00000000","reduceOnly":true,"selfTradePrevention":"dc"}]`

const stopLossOpenOrderMissingTriggerPriceFixture = `[{"market":"ETH-USD","orderId":"ord-1","time":1699999999000,"status":"active","type":"stopLossLimit","side":"sell","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2440.00000000","triggerType":"last","reduceOnly":true,"selfTradePrevention":"dc"}]`

// TestAmendOrderErrorsBeforeCancellingWhenOriginalTriggerTypeIsMissing: an original order
// classified take-profit/stop-loss but whose fetched payload omits the (documented-optional)
// triggerType must fail validation WITHOUT ever cancelling the original order — validating after
// the cancel would cancel the protective order and only then error, leaving the position
// unprotected.
func TestAmendOrderErrorsBeforeCancellingWhenOriginalTriggerTypeIsMissing(t *testing.T) {
	var deleteCalls, postCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, stopLossOpenOrderMissingTriggerTypeFixture)
		},
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { deleteCalls++ },
		"POST /v1/orders":   func(w http.ResponseWriter, r *http.Request) { postCalls++ },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeSell).
		NewSize("1").NewPrice("2430").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to error when the original order is take-profit/stop-loss but omits triggerType")
	}
	if deleteCalls != 0 || postCalls != 0 {
		t.Fatalf("delete calls = %d, post calls = %d, want 0/0 (validation must happen BEFORE the original order is cancelled)", deleteCalls, postCalls)
	}
}

// TestAmendOrderErrorsBeforeCancellingWhenOriginalTriggerPriceIsMissing is the triggerPrice-side
// sibling of the test above — same failure shape, the other documented-optional field.
func TestAmendOrderErrorsBeforeCancellingWhenOriginalTriggerPriceIsMissing(t *testing.T) {
	var deleteCalls, postCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, stopLossOpenOrderMissingTriggerPriceFixture)
		},
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { deleteCalls++ },
		"POST /v1/orders":   func(w http.ResponseWriter, r *http.Request) { postCalls++ },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeSell).
		NewSize("1").NewPrice("2430").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to error when the original order is take-profit/stop-loss but omits triggerPrice")
	}
	if deleteCalls != 0 || postCalls != 0 {
		t.Fatalf("delete calls = %d, post calls = %d, want 0/0 (validation must happen BEFORE the original order is cancelled)", deleteCalls, postCalls)
	}
}

// gtxOpenOrderFixture is a GET /v1/orders response for a resting post-only (gtx) order.
const gtxOpenOrderFixture = `[{"market":"ETH-USD","orderId":"ord-1","time":1699999999000,"status":"active","type":"limit","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2500.00000000","reduceOnly":false,"timeInForce":"gtx","selfTradePrevention":"dc"}]`

// TestAmendOrderPreservesTimeInForceAndSelfTradePreventionFromOriginal: a resting post-only (gtx)
// order — one that could only have been placed outside this connector's own placeOrder, which
// always sends gtc — must stay gtx after an amend. Silently downgrading it to gtc would let the
// order cross the spread and fill as taker at a worse price with taker fees, changing an execution
// policy the user never asked to change. The signature is independently recomputed with
// timeInForceGTX (1) — the INTEGER — to prove the signed payload, not just the wire body,
// preserved it.
func TestAmendOrderPreservesTimeInForceAndSelfTradePreventionFromOriginal(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, gtxOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2550").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	p := placeBody.Parameters
	if p.TimeInForce != "gtx" {
		t.Fatalf("timeInForce = %q, want gtx (preserved from the original resting post-only order, not silently defaulted to gtc)", p.TimeInForce)
	}
	if p.SelfTradePrevention != "dc" {
		t.Fatalf("selfTradePrevention = %q, want dc (preserved from the original)", p.SelfTradePrevention)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signOrder(orderSignPayload{
		Nonce:               nonce,
		Wallet:              validWalletAddress,
		MarketSymbol:        "ETH-USD",
		OrderType:           orderTypeLimit,
		OrderSide:           orderSideBuy,
		Quantity:            "1.00000000",
		LimitPrice:          "2550.00000000",
		TriggerPrice:        "0.00000000",
		TriggerType:         triggerTypeNone,
		IsReduceOnly:        false,
		TimeInForce:         timeInForceGTX, // 1 — the INTEGER, preserved from the original
		SelfTradePrevention: selfTradePreventionDC,
		DelegatedPublicKey:  mustDelegatedAddress(t, c),
		ClientOrderId:       p.ClientOrderId,
	})
	if err != nil {
		t.Fatal(err)
	}
	if placeBody.Signature != wantSig {
		t.Fatalf("signature = %s, want %s (recomputed with timeInForce=1, the preserved GTX integer)", placeBody.Signature, wantSig)
	}
}

// TestAmendOrderReplacementMarketComesFromOriginalOrder: the replacement's `market` must come from
// the original order's own record (original.Market), not just be echoed back from the caller's
// request symbol.
func TestAmendOrderReplacementMarketComesFromOriginalOrder(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2600").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if placeBody.Parameters.Market != "ETH-USD" {
		t.Fatalf("market = %q, want ETH-USD (sourced from the original order's own record)", placeBody.Parameters.Market)
	}
}

// TestAmendOrderErrorsWhenRequestSymbolDoesNotMatchOriginalMarket: a caller-supplied symbol that
// disagrees with the original order's actual market must be rejected outright, before any cancel or
// place call, rather than silently trusting either source.
func TestAmendOrderErrorsWhenRequestSymbolDoesNotMatchOriginalMarket(t *testing.T) {
	var deleteCalls, postCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) }, // market: ETH-USD
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { deleteCalls++ },
		"POST /v1/orders":   func(w http.ResponseWriter, r *http.Request) { postCalls++ },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("BTC-USD"). // deliberately wrong — the fetched order is on ETH-USD
		OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2600").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to error when the request symbol does not match the original order's actual market")
	}
	if deleteCalls != 0 || postCalls != 0 {
		t.Fatalf("delete calls = %d, post calls = %d, want 0/0 (must never cancel or place on a market mismatch)", deleteCalls, postCalls)
	}
}

// --- resolveTimeInForce / resolveSelfTradePrevention ---

func TestResolveTimeInForce(t *testing.T) {
	cases := []struct {
		in       string
		wantCode uint8
		wantName string
	}{
		{"", timeInForceGTC, timeInForceNameGTC},
		{"gtc", timeInForceGTC, timeInForceNameGTC},
		{"GTX", timeInForceGTX, timeInForceNameGTX},
		{"ioc", timeInForceIOC, timeInForceNameIOC},
		{"FOK", timeInForceFOK, timeInForceNameFOK},
	}
	for _, tc := range cases {
		code, name, err := resolveTimeInForce(tc.in)
		if err != nil {
			t.Fatalf("resolveTimeInForce(%q): %v", tc.in, err)
		}
		if code != tc.wantCode || name != tc.wantName {
			t.Fatalf("resolveTimeInForce(%q) = (%d, %q), want (%d, %q)", tc.in, code, name, tc.wantCode, tc.wantName)
		}
	}
}

func TestResolveTimeInForceRejectsUnknown(t *testing.T) {
	if _, _, err := resolveTimeInForce("bogus"); err == nil {
		t.Fatal("expected an error for an unknown timeInForce")
	}
}

func TestResolveSelfTradePrevention(t *testing.T) {
	cases := []struct {
		in       string
		wantCode uint8
		wantName string
	}{
		{"", selfTradePreventionDC, selfTradePreventionNameDC},
		{"dc", selfTradePreventionDC, selfTradePreventionNameDC},
		{"CO", selfTradePreventionCO, selfTradePreventionNameCO},
		{"cn", selfTradePreventionCN, selfTradePreventionNameCN},
		{"CB", selfTradePreventionCB, selfTradePreventionNameCB},
	}
	for _, tc := range cases {
		code, name, err := resolveSelfTradePrevention(tc.in)
		if err != nil {
			t.Fatalf("resolveSelfTradePrevention(%q): %v", tc.in, err)
		}
		if code != tc.wantCode || name != tc.wantName {
			t.Fatalf("resolveSelfTradePrevention(%q) = (%d, %q), want (%d, %q)", tc.in, code, name, tc.wantCode, tc.wantName)
		}
	}
}

func TestResolveSelfTradePreventionRejectsUnknown(t *testing.T) {
	if _, _, err := resolveSelfTradePrevention("bogus"); err == nil {
		t.Fatal("expected an error for an unknown selfTradePrevention")
	}
}

func TestResolveTriggerTypeRoundTripsAndRejectsUnknown(t *testing.T) {
	code, name, err := resolveTriggerType("LAST")
	if err != nil {
		t.Fatal(err)
	}
	if code != triggerTypeLast || name != triggerTypeNameLast {
		t.Fatalf("resolveTriggerType(LAST) = (%d, %q), want (%d, %q)", code, name, triggerTypeLast, triggerTypeNameLast)
	}
	code, name, err = resolveTriggerType(" index ")
	if err != nil {
		t.Fatal(err)
	}
	if code != triggerTypeIndex || name != triggerTypeNameIndex {
		t.Fatalf("resolveTriggerType(index) = (%d, %q), want (%d, %q)", code, name, triggerTypeIndex, triggerTypeNameIndex)
	}
	if _, _, err := resolveTriggerType(""); err == nil {
		t.Fatal("expected an error for a missing trigger type — it must never be silently defaulted")
	}
}

// TestAmendOrderDoesNotPlaceWhenCancelFails asserts the first half of the accepted failure mode: if
// the cancel itself fails, amendOrder must fail outright and never even attempt to place a
// replacement (an order that was never actually removed must not get a duplicate placed alongside
// it).
func TestAmendOrderDoesNotPlaceWhenCancelFails(t *testing.T) {
	var placeCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeCalls++
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2600").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to fail when the cancel step fails")
	}
	if placeCalls != 0 {
		t.Fatalf("place was called %d times, want 0 (must never place after a failed cancel)", placeCalls)
	}
}

// TestAmendOrderDoesNotPlaceWhenCancelReturnsNotFound: a cancel response of notFound (e.g. the
// order filled a moment before the cancel reached Katana) must abort the amend before any
// replacement is placed, exactly like a hard cancel failure — cancelOrder itself returns an error
// for this, so amendOrder gets it for free, but it is worth a dedicated regression test given how
// dangerous silently proceeding here is (a duplicate resting order on top of a position that
// already has the fill).
func TestAmendOrderDoesNotPlaceWhenCancelReturnsNotFound(t *testing.T) {
	var placeCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[{"status":"notFound"}]`) },
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeCalls++
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2600").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to fail when the cancel response is notFound rather than canceled")
	}
	if placeCalls != 0 {
		t.Fatalf("place was called %d times, want 0 (a notFound cancel must never be followed by a place)", placeCalls)
	}
}

// TestAmendOrderReportsButDoesNotHidePlaceFailureAfterASuccessfulCancel is the harder half of the
// accepted failure mode: the cancel succeeds (the original order is genuinely gone) and the
// subsequent place fails. amendOrder does not attempt a rollback (see its doc comment for why one
// isn't possible here), but it must still return a clear error instead of silently reporting
// success, so the caller knows the position is now unprotected by any order.
func TestAmendOrderReportsButDoesNotHidePlaceFailureAfterASuccessfulCancel(t *testing.T) {
	var cancelCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			cancelCalls++
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnprocessableEntity) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).
		NewSize("1").NewPrice("2600").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to return an error when the replacement place fails")
	}
	if cancelCalls != 1 {
		t.Fatalf("cancel calls = %d, want exactly 1 (the cancel really happened before place failed)", cancelCalls)
	}
	if !strings.Contains(err.Error(), "ord-1") {
		t.Fatalf("error = %q, want it to name the canceled order id so the caller knows which order is now unprotected", err.Error())
	}
}

// TestAmendOrderRequiresAllFields covers only the fields amendOrder genuinely requires. `side` and
// `newSize` are deliberately absent from this list — both are optional and a chart's drag-to-amend
// omits both; their defaulting is covered by TestAmendOrderDefaultsSideAndSizeFromTheOriginalOrder
// below.
//
// Every case here must fail before any HTTP call, which is why the client points at an unreachable
// address: a case that reached the network would fail the assertion by erroring for the wrong
// reason only by luck, so the base URL is deliberately dead rather than a live test server.
func TestAmendOrderRequiresAllFields(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")

	if _, err := c.NewAmendOrder().OrderID("ord-1").Side(entity.SideTypeBuy).NewSize("1").NewPrice("2600").
		Do(context.Background()); err == nil {
		t.Fatal("expected an error when symbol is missing")
	}
	if _, err := c.NewAmendOrder().Symbol("ETH-USD").Side(entity.SideTypeBuy).NewSize("1").NewPrice("2600").
		Do(context.Background()); err == nil {
		t.Fatal("expected an error when orderID is missing")
	}
	if _, err := c.NewAmendOrder().Symbol("ETH-USD").OrderID("ord-1").Side(entity.SideTypeBuy).NewSize("1").
		Do(context.Background()); err == nil {
		t.Fatal("expected an error when newPrice is missing")
	}
}

// TestAmendOrderDefaultsSideAndSizeFromTheOriginalOrder is the exact request a chart's
// drag-to-amend produces when a user drags a resting limit order — symbol, orderId and newPrice
// only. Requiring side would fail every one of those with "amendOrder requires a side".
//
// plainOpenOrderFixture is a BUY of originalQuantity 1.00000000, so the expected wire values below
// are hand-written from that fixture, not recomputed by the code under test.
func TestAmendOrderDefaultsSideAndSizeFromTheOriginalOrder(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").
		OrderID("ord-1").
		NewPrice("2600").
		Do(context.Background()); err != nil {
		t.Fatalf("amendOrder with only symbol/orderID/newPrice must succeed: %v", err)
	}
	if placeBody.Parameters.Side != "buy" {
		t.Fatalf("side = %q, want buy (defaulted from the original order)", placeBody.Parameters.Side)
	}
	if placeBody.Parameters.Quantity != "1.00000000" {
		t.Fatalf("quantity = %q, want 1.00000000 (defaulted from the original order's originalQuantity)", placeBody.Parameters.Quantity)
	}
	if placeBody.Parameters.Price != "2600.00000000" {
		t.Fatalf("price = %q, want 2600.00000000 (the one field the caller did supply)", placeBody.Parameters.Price)
	}
}

// TestAmendOrderSuppliedNewSizeStillWins guards the other half of that default: supplying newSize
// must still change the size, since that is the only reason to supply it.
func TestAmendOrderSuppliedNewSizeStillWins(t *testing.T) {
	var placeBody katanaSignedRequest[katanaPlaceOrderParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) },
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"orderId":"ord-1","status":"canceled"}]`)
		},
		"POST /v1/orders": func(w http.ResponseWriter, r *http.Request) {
			placeBody = readSignedBody[katanaPlaceOrderParams](t, r)
			writeJSON(t, w, placedOrderFixture)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").
		NewSize("3.5").NewPrice("2600").
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if placeBody.Parameters.Quantity != "3.50000000" {
		t.Fatalf("quantity = %q, want 3.50000000 (the caller-supplied newSize)", placeBody.Parameters.Quantity)
	}
}

// TestAmendOrderErrorsWhenRequestSideDoesNotMatchOriginalSide is why the cross-check exists at all:
// a stale UI row sending the wrong side taken on trust would cancel a resting BUY and place a SELL
// in its place. Zero DELETE and zero POST calls is the assertion that matters — an order that
// flipped direction cannot be undone.
func TestAmendOrderErrorsWhenRequestSideDoesNotMatchOriginalSide(t *testing.T) {
	var deleteCalls, postCalls int
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":       func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"GET /v1/orders":    func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, plainOpenOrderFixture) }, // side: buy
		"DELETE /v1/orders": func(w http.ResponseWriter, r *http.Request) { deleteCalls++ },
		"POST /v1/orders":   func(w http.ResponseWriter, r *http.Request) { postCalls++ },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	_, err := c.NewAmendOrder().
		Symbol("ETH-USD").OrderID("ord-1").
		Side(entity.SideTypeSell). // deliberately wrong — the fetched order is a buy
		NewSize("1").NewPrice("2600").
		Do(context.Background())
	if err == nil {
		t.Fatal("expected amendOrder to error when the request side does not match the original order's actual side")
	}
	if !strings.Contains(err.Error(), "side") {
		t.Fatalf("error = %v, want it to name the side mismatch", err)
	}
	if deleteCalls != 0 || postCalls != 0 {
		t.Fatalf("delete calls = %d, post calls = %d, want 0/0 (must never cancel or place on a side mismatch)", deleteCalls, postCalls)
	}
}

// --- setLeverage ---

func TestSetLeverageTighteningRequestIsSignedAndPostedAsRequested(t *testing.T) {
	var captured katanaSignedRequest[katanaIMFOverrideParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"POST /v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaIMFOverrideParams](t, r)
			writeJSON(t, w, `{"wallet":"0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d","market":"ETH-USD","initialMarginFractionOverride":"0.20000000"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	// ETH-USD's default initialMarginFraction is 0.05000000 (20x). Requesting 5x (IMF 0.20000000)
	// is a valid tightening override.
	got, err := c.NewSetLeverage().Symbol("ETH-USD").Leverage("5").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.Futures_Leverage{Symbol: "ETH-USD", Leverage: "5", MarginMode: "CROSS"}
	if got != want {
		t.Fatalf("setLeverage = %+v, want %+v", got, want)
	}

	p := captured.Parameters
	if p.Market != "ETH-USD" || p.InitialMarginFractionOverride != "0.20000000" {
		t.Fatalf("body = %+v, want market=ETH-USD initialMarginFractionOverride=0.20000000", p)
	}

	nonce := nonceBigIntFromString(t, p.Nonce)
	wantSig, err := c.signer().signIMFOverride(nonce, validWalletAddress, "ETH-USD", "0.20000000")
	if err != nil {
		t.Fatal(err)
	}
	if captured.Signature != wantSig {
		t.Fatalf("signature = %s, want %s", captured.Signature, wantSig)
	}
}

func TestSetLeverageRequestAboveMarketMaximumClampsInsteadOfErroring(t *testing.T) {
	var captured katanaSignedRequest[katanaIMFOverrideParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"POST /v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaIMFOverrideParams](t, r)
			writeJSON(t, w, `{"wallet":"0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d","market":"ETH-USD","initialMarginFractionOverride":"0.05000000"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	// ETH-USD's maximum is 20x (IMF 0.05000000). Requesting 100x must NOT error — it is clamped to
	// the market maximum.
	got, err := c.NewSetLeverage().Symbol("ETH-USD").Leverage("100").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.Futures_Leverage{Symbol: "ETH-USD", Leverage: "20", MarginMode: "CROSS"}
	if got != want {
		t.Fatalf("setLeverage = %+v, want %+v (clamped to the market maximum, not an error)", got, want)
	}
	if captured.Parameters.InitialMarginFractionOverride != "0.05000000" {
		t.Fatalf("posted initialMarginFractionOverride = %q, want the clamped market floor 0.05000000, not the requested 0.01000000", captured.Parameters.InitialMarginFractionOverride)
	}
}

// TestSetLeverageRequestBelowOneXClampsToOneInsteadOfPostingAnInvalidOverride: requesting 0.5x
// leverage (leverageToIMF("0.5") = "2.00000000") must be clamped to the documented upper bound
// (IMF 1.00000000, i.e. 1x) before it is ever posted to Katana, not sent as-is.
func TestSetLeverageRequestBelowOneXClampsToOneInsteadOfPostingAnInvalidOverride(t *testing.T) {
	var captured katanaSignedRequest[katanaIMFOverrideParams]
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"POST /v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			captured = readSignedBody[katanaIMFOverrideParams](t, r)
			writeJSON(t, w, `{"wallet":"0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d","market":"ETH-USD","initialMarginFractionOverride":"1.00000000"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewSetLeverage().Symbol("ETH-USD").Leverage("0.5").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Leverage != "1" {
		t.Fatalf("Leverage = %s, want 1 (clamped to the 1x/IMF-1.0 upper bound)", got.Leverage)
	}
	if captured.Parameters.InitialMarginFractionOverride != "1.00000000" {
		t.Fatalf("posted initialMarginFractionOverride = %q, want the clamped upper bound 1.00000000, not the raw requested 2.00000000", captured.Parameters.InitialMarginFractionOverride)
	}
}

func TestSetLeverageUsesTheServersConfirmedOverrideWhenPresent(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"POST /v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			// Server echoes back a DIFFERENT value than what was requested/posted.
			writeJSON(t, w, `{"wallet":"0xA71C4aeeAabBBB8D2910F41C2ca3964b81F7310d","market":"ETH-USD","initialMarginFractionOverride":"0.10000000"}`)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	got, err := c.NewSetLeverage().Symbol("ETH-USD").Leverage("5").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Leverage != "10" {
		t.Fatalf("Leverage = %s, want 10 (1 / the server-confirmed 0.10000000, not the locally requested 5x)", got.Leverage)
	}
}

func TestSetLeverageRequiresSymbolAndLeverage(t *testing.T) {
	c := newSigningFuturesClient("http://127.0.0.1:0")
	if _, err := c.NewSetLeverage().Leverage("5").Do(context.Background()); err == nil {
		t.Fatal("expected an error when symbol is missing")
	}
	if _, err := c.NewSetLeverage().Symbol("ETH-USD").Do(context.Background()); err == nil {
		t.Fatal("expected an error when leverage is missing")
	}
}

func TestSetLeverageErrorsOnUnknownMarket(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewSetLeverage().Symbol("DOGE-USD").Leverage("5").Do(context.Background()); err == nil {
		t.Fatal("expected an error for a market not present in markets()")
	}
}

func TestSetLeveragePropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, signedWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"POST /v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newSigningFuturesClient(server.URL)

	if _, err := c.NewSetLeverage().Symbol("ETH-USD").Leverage("5").Do(context.Background()); err == nil {
		t.Fatal("expected setLeverage to propagate a non-2xx response")
	}
}
