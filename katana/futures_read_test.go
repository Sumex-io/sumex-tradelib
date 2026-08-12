package katana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// --- Test fixtures / helpers shared by the action tests below ---

// muxServer builds an httptest server that dispatches to per-path handlers, mirroring Katana's own
// multi-endpoint REST API (a client only has one BaseURL, so every test that exercises more than
// one endpoint needs this instead of the single-handler httptest.NewServer used by the simpler
// tests in katana_test.go).
func muxServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, h := range handlers {
		mux.HandleFunc(path, h)
	}
	return httptest.NewServer(mux)
}

func writeJSON(t *testing.T, w http.ResponseWriter, body string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
}

// newTestFuturesClient is the read path's client: no delegated key, because nothing on this path
// signs anything. The write tests build their own with testPrivKey.
func newTestFuturesClient(baseURL string) *FuturesClient {
	c := NewFuturesClient("k", "s", "")
	c.BaseURL = baseURL
	return c
}

const singleWalletFixture = `[{"wallet":"0xWALLET","equity":"10000.00000000","freeCollateral":"9000.00000000","quoteBalance":"9500.00000000","unrealizedPnL":"100.00000000"}]`

const twoMarketsFixture = `[
	{
		"market": "ETH-USD", "type": "perpetual", "status": "active", "baseAsset": "ETH", "quoteAsset": "USD",
		"stepSize": "0.00000100", "tickSize": "0.01000000", "minimumPositionSize": "0.01000000",
		"maximumPositionSize": "500.00000000", "initialMarginFraction": "0.05000000",
		"maintenanceMarginFraction": "0.03000000"
	},
	{
		"market": "BTC-USD", "type": "perpetual", "status": "active", "baseAsset": "BTC", "quoteAsset": "USD",
		"stepSize": "0.00001000", "tickSize": "0.10000000", "minimumPositionSize": "0.00100000",
		"maximumPositionSize": "50.00000000", "initialMarginFraction": "0.02000000",
		"maintenanceMarginFraction": "0.01000000"
	},
	{
		"market": "GOLD-USD", "type": "intermittent", "status": "active", "baseAsset": "GOLD", "quoteAsset": "USD"
	}
]`

// --- getPositionMode / getMarginMode: answered from Katana invariants, no network call ---

func TestGetPositionModeIsAlwaysOneWay(t *testing.T) {
	c := NewFuturesClient("k", "s", "")
	got, err := c.NewGetPositionMode().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.HedgeMode {
		t.Fatalf("hedgeMode = true, want false (Katana has no hedge mode)")
	}
}

func TestGetMarginModeIsAlwaysCross(t *testing.T) {
	c := NewFuturesClient("k", "s", "")
	got, err := c.NewGetMarginMode().Symbol("ETH-USD").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.MarginMode != "CROSS" {
		t.Fatalf("marginMode = %q, want CROSS", got.MarginMode)
	}
}

// --- getBalance ---

func TestGetBalanceFetchesScopedWalletAndMapsToUSDC(t *testing.T) {
	var gotWallet string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			gotWallet = r.URL.Query().Get("wallet")
			writeJSON(t, w, singleWalletFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetBalance().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotWallet != "0xWALLET" {
		t.Fatalf("wallet query param = %q, want 0xWALLET (getBalance must scope to the resolved wallet)", gotWallet)
	}
	want := []entity.FuturesBalance{{
		Asset: "USDC", Balance: "9500.00000000", Equity: "10000.00000000",
		Available: "9000.00000000", UnrealizedProfit: "100.00000000",
	}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("getBalance = %+v, want %+v", got, want)
	}
}

func TestGetBalanceErrorsWhenWalletMissingFromResponse(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetBalance().Do(context.Background()); err == nil {
		t.Fatal("expected an error when GET /v1/wallets returns no entries for the resolved wallet")
	}
}

// --- getInstrumentsInfo ---

func TestGetInstrumentsInfoSortsFiltersAndDerivesPrecision(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, twoMarketsFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetInstrumentsInfo().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d instruments, want 2 (GOLD-USD is intermittent and must be filtered)", len(got))
	}
	// Alphabetical: BTC-USD before ETH-USD.
	want := []entity.Futures_InstrumentsInfo{
		{
			Symbol: "BTC-USD", Base: "BTC", Quote: "USD", MinQty: "0.00100000",
			PricePrecision: "1", SizePrecision: "5", State: "LIVE", MaxLeverage: "50",
			ContractSize: "1", IsSizeContract: false,
		},
		{
			Symbol: "ETH-USD", Base: "ETH", Quote: "USD", MinQty: "0.01000000",
			PricePrecision: "2", SizePrecision: "6", State: "LIVE", MaxLeverage: "20",
			ContractSize: "1", IsSizeContract: false,
		},
	}
	if got[0] != want[0] {
		t.Fatalf("got[0] = %+v, want %+v", got[0], want[0])
	}
	if got[1] != want[1] {
		t.Fatalf("got[1] = %+v, want %+v", got[1], want[1])
	}
}

func TestGetInstrumentsInfoPropagatesMarketsEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetInstrumentsInfo().Do(context.Background()); err == nil {
		t.Fatal("expected getInstrumentsInfo to propagate a GET /v1/markets failure")
	}
}

func TestDecimalPlaces(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"0.00000100", "6"},
		{"0.01000000", "2"},
		{"0.10000000", "1"},
		{"1.00000000", "0"},
		{"5", "0"},
		{"not-a-number", "0"},
	}
	for _, tc := range cases {
		if got := decimalPlaces(tc.in); got != tc.want {
			t.Fatalf("decimalPlaces(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// --- getPositions ---

const positionsWithPhantomFixture = `[
	{"market":"ETH-USD","quantity":"2.50000000","entryPrice":"2000.00000000","markPrice":"2100.00000000","realizedPnL":"0.00000000","unrealizedPnL":"250.00000000","time":1700000000000},
	{"market":"ETH-USD","quantity":"0.00000000","entryPrice":"0.00000000","markPrice":"2100.00000000","realizedPnL":"-15.00000000","unrealizedPnL":"0.00000000","time":1699000000000},
	{"market":"ETH-USD","quantity":"-1.00000000","entryPrice":"2200.00000000","markPrice":"2100.00000000","realizedPnL":"5.00000000","unrealizedPnL":"100.00000000","time":1698000000000}
]`

const ethOverrideFixture = `[{"wallet":"0xWALLET","market":"ETH-USD","initialMarginFractionOverride":"0.20000000"}]`

func TestGetPositionsFiltersPhantomAndAppliesOverrideLeverage(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/positions": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, positionsWithPhantomFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, ethOverrideFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetPositions().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d positions, want 2 (the zero-quantity phantom row must be dropped)", len(got))
	}
	for _, p := range got {
		if p.Leverage != "5" {
			t.Fatalf("leverage = %s, want 5 (1 / override 0.20000000, not the market default 0.05 -> 20)", p.Leverage)
		}
	}
	if got[0].PositionSide != "LONG" || got[1].PositionSide != "SHORT" {
		t.Fatalf("position sides = [%s, %s], want [LONG, SHORT]", got[0].PositionSide, got[1].PositionSide)
	}
}

// adlQuintile is documented as a quoted string but production sends a bare number. A string target
// failed the whole decode ("cannot unmarshal number into Go struct field katanaPosition.adlQuintile
// of type string"), emptying the positions panel on a field nothing reads. Both shapes must decode.
const positionsAdlQuintileShapesFixture = `[
	{"market":"ETH-USD","quantity":"1.00000000","entryPrice":"2000.00000000","markPrice":"2100.00000000","realizedPnL":"0.00000000","unrealizedPnL":"100.00000000","adlQuintile":3,"time":1700000000000},
	{"market":"ETH-USD","quantity":"-2.00000000","entryPrice":"2200.00000000","markPrice":"2100.00000000","realizedPnL":"0.00000000","unrealizedPnL":"200.00000000","adlQuintile":"1","time":1699000000000}
]`

func TestGetPositionsDecodesAdlQuintileAsEitherNumberOrString(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/positions": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, positionsAdlQuintileShapesFixture)
		},
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetPositions().Do(context.Background())
	if err != nil {
		t.Fatalf("decode failed on a valid positions page: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d positions, want 2 (neither adlQuintile shape may drop a row)", len(got))
	}
	if got[0].PositionSide != "LONG" || got[1].PositionSide != "SHORT" {
		t.Fatalf("position sides = [%s, %s], want [LONG, SHORT]", got[0].PositionSide, got[1].PositionSide)
	}
}

func TestGetPositionsScopesToTheCallerSymbol(t *testing.T) {
	var gotMarket string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/positions": func(w http.ResponseWriter, r *http.Request) {
			gotMarket = r.URL.Query().Get("market")
			writeJSON(t, w, `[]`)
		},
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetPositions().Symbol("ETH-USD").Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotMarket != "ETH-USD" {
		t.Fatalf("market query param = %q, want ETH-USD", gotMarket)
	}
}

func TestGetPositionsPropagatesPositionsEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/positions": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetPositions().Do(context.Background()); err == nil {
		t.Fatal("expected getPositions to propagate a GET /v1/positions failure, not swallow it")
	}
}

// The IMF override lookup is cosmetic (it only changes a displayed leverage number) and must never
// take down the whole positions list.
func TestGetPositionsSurvivesOverrideEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/positions": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"market":"ETH-USD","quantity":"1.00000000","entryPrice":"2000.00000000","markPrice":"2100.00000000","realizedPnL":"0.00000000","unrealizedPnL":"0.00000000","time":1700000000000}]`)
		},
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetPositions().Do(context.Background())
	if err != nil {
		t.Fatalf("getPositions must survive an override-endpoint failure, got: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d positions, want 1", len(got))
	}
	if got[0].Leverage != "20" {
		t.Fatalf("leverage = %s, want 20 (market default 0.05000000; override lookup failed)", got[0].Leverage)
	}
}

func TestGetPositionsPropagatesMarketsEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetPositions().Do(context.Background()); err == nil {
		t.Fatal("expected getPositions to propagate a GET /v1/markets failure (unlike the override lookup, the market catalog is not optional)")
	}
}

// --- getOrderList (the source connector's getOpenOrders) ---

const openOrdersFixture = `[
	{"market":"ETH-USD","orderId":"open-1","time":1700000000000,"status":"active","type":"limit","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2000.00000000","reduceOnly":false}
]`

func TestGetOrderListRequestsActiveOrdersOnly(t *testing.T) {
	var gotClosed string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders": func(w http.ResponseWriter, r *http.Request) {
			gotClosed = r.URL.Query().Get("closed")
			writeJSON(t, w, openOrdersFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetOrderList().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotClosed != "false" {
		t.Fatalf("closed query param = %q, want \"false\"", gotClosed)
	}
	if len(got) != 1 || got[0].OrderID != "open-1" || got[0].Status != "ACTIVE" {
		t.Fatalf("getOrderList = %+v", got)
	}
}

func TestGetOrderListPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetOrderList().Do(context.Background()); err == nil {
		t.Fatal("expected getOrderList to propagate a GET /v1/orders failure")
	}
}

// TestGetOrderListRequestsTheDocumentedMaximumLimit: without an explicit `limit`, GET /v1/orders
// defaults to 50 (API_NOTES.md's parameter table) and truncates the open-orders panel with no error
// and no log. 1000 is the documented maximum for this endpoint, written here as a hand-written
// literal rather than read back from katanaOrdersPageCap.
func TestGetOrderListRequestsTheDocumentedMaximumLimit(t *testing.T) {
	var gotLimit string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders": func(w http.ResponseWriter, r *http.Request) {
			gotLimit = r.URL.Query().Get("limit")
			writeJSON(t, w, openOrdersFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetOrderList().Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotLimit != "1000" {
		t.Fatalf("limit query param = %q, want \"1000\" (the documented maximum; the endpoint otherwise defaults to 50)", gotLimit)
	}
}

// TestGetOrderListScopesToTheCallerSymbol proves the caller's symbol actually reaches the query
// string, so a symbol-scoped request is genuinely scoped rather than an account-wide one the caller
// then filters client-side.
func TestGetOrderListScopesToTheCallerSymbol(t *testing.T) {
	var gotMarket string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders": func(w http.ResponseWriter, r *http.Request) {
			gotMarket = r.URL.Query().Get("market")
			writeJSON(t, w, openOrdersFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetOrderList().Symbol("ETH-USD").Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotMarket != "ETH-USD" {
		t.Fatalf("market query param = %q, want ETH-USD", gotMarket)
	}
}

func TestGetOrderListThreadsADifferentSymbolIntoTheQuery(t *testing.T) {
	var gotMarket string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders": func(w http.ResponseWriter, r *http.Request) {
			gotMarket = r.URL.Query().Get("market")
			writeJSON(t, w, openOrdersFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetOrderList().Symbol("BTC-USD").Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if gotMarket != "BTC-USD" {
		t.Fatalf("market query param = %q, want BTC-USD", gotMarket)
	}
}

// TestGetOrderListLogsWhenAFullPageComesBack pins the truncation warning: a response of exactly the
// page cap means Katana had at least that many open orders and may have more, which the caller has
// no other way to notice.
func TestGetOrderListLogsWhenAFullPageComesBack(t *testing.T) {
	var page strings.Builder
	page.WriteString("[")
	for i := 0; i < 1000; i++ {
		if i > 0 {
			page.WriteString(",")
		}
		fmt.Fprintf(&page, `{"market":"ETH-USD","orderId":"open-%d","time":1700000000000,"status":"active","type":"limit","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"2000.00000000","reduceOnly":false}`, i)
	}
	page.WriteString("]")

	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, page.String()) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	buf := captureLog(t)

	got, err := c.NewGetOrderList().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1000 {
		t.Fatalf("got %d orders, want 1000", len(got))
	}
	if !strings.Contains(buf.String(), "full 1000-order page") {
		t.Fatalf("expected a truncation warning naming the full page, got: %q", buf.String())
	}
}

// TestGetOrderListDoesNotLogOnAPartialPage is the negative half of the pair: a response shorter
// than the cap is complete, and must not emit a warning that would otherwise fire on every poll.
func TestGetOrderListDoesNotLogOnAPartialPage(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, openOrdersFixture) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	buf := captureLog(t)

	if _, err := c.NewGetOrderList().Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected no log output for a partial page, got: %q", buf.String())
	}
}

// captureLog redirects the standard logger for the duration of one test and hands back the buffer
// it writes into, restoring the original writer and flags on cleanup.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	origOutput := log.Writer()
	origFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(origOutput)
		log.SetFlags(origFlags)
	})
	return &buf
}

// --- ordersHistory: fill aggregation + override leverage ---

// ordersHistoryFixture has three closed orders:
//   - ord-A: inline fills (two), aggregation must use them directly, no GET /v1/fills.
//   - ord-B: no inline fills but executedQuantity > 0, must trigger the GET /v1/fills fallback.
//   - ord-C: no inline fills and executedQuantity == 0 (never executed), must NOT trigger any
//     fallback fetch.
const ordersHistoryFixture = `[
	{
		"market":"ETH-USD","orderId":"ord-A","time":1700000000000,"status":"filled","type":"market","side":"buy",
		"originalQuantity":"2.00000000","executedQuantity":"2.00000000","avgExecutionPrice":"2000.00000000",
		"reduceOnly":false,
		"fills":[
			{"fillId":"f-a1","price":"2000.00000000","quantity":"1.00000000","quoteQuantity":"2000.00000000","time":1700000000000,"realizedPnL":"5.00000000","fee":"0.40000000"},
			{"fillId":"f-a2","price":"2001.00000000","quantity":"1.00000000","quoteQuantity":"2001.00000000","time":1700000000100,"realizedPnL":"1.75000000","fee":"0.10000000"}
		]
	},
	{
		"market":"ETH-USD","orderId":"ord-B","time":1700000001000,"status":"filled","type":"market","side":"sell",
		"originalQuantity":"1.00000000","executedQuantity":"1.00000000","avgExecutionPrice":"1990.00000000",
		"reduceOnly":true
	},
	{
		"market":"ETH-USD","orderId":"ord-C","time":1700000002000,"status":"canceled","type":"limit","side":"buy",
		"originalQuantity":"1.00000000","executedQuantity":"0.00000000","price":"1500.00000000",
		"reduceOnly":false
	}
]`

const ordBFillsFixture = `[{"fillId":"f-b1","orderId":"ord-B","price":"1990.00000000","quantity":"1.00000000","quoteQuantity":"1990.00000000","time":1700000001000,"realizedPnL":"-3.20000000","fee":"0.20000000"}]`

// TestOrdersHistoryAggregatesFillsUsingWindowedFallback covers the paginated windowed GET /v1/fills
// fetch (resolveMissingOrderFills -> fetchFillsPaged, instead of one request per order lacking
// inline fills) and, since ord-B's fill is found on the windowed fetch's first (and only, here)
// page, proves the per-order fallback is NOT invoked when the windowed fetch already has what's
// needed.
func TestOrdersHistoryAggregatesFillsUsingWindowedFallback(t *testing.T) {
	fillsFetchCount := 0

	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, ordersHistoryFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, ethOverrideFixture)
		},
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			fillsFetchCount++
			if r.URL.Query().Has("orderId") {
				t.Fatal("the windowed fallback must not scope by orderId — that would be an N+1 fetch")
			}
			writeJSON(t, w, ordBFillsFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewOrdersHistory().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d history rows, want 3", len(got))
	}

	byID := map[string]entity.Futures_OrdersHistory{}
	for _, o := range got {
		byID[o.OrderID] = o
	}

	// ord-A: inline fills, hand-computed sums: pnl 5.00000000 + 1.75000000 = 6.75000000;
	// fee 0.40000000 + 0.10000000 = 0.50000000.
	a := byID["ord-A"]
	if a.RealisedProfit != "6.75000000" {
		t.Fatalf("ord-A RealisedProfit = %s, want 6.75000000", a.RealisedProfit)
	}
	if a.Fee != "0.50000000" {
		t.Fatalf("ord-A Fee = %s, want 0.50000000", a.Fee)
	}
	if a.FeeAsset != "USDC" {
		t.Fatalf("ord-A FeeAsset = %s, want USDC", a.FeeAsset)
	}
	if a.Leverage != "5" {
		t.Fatalf("ord-A Leverage = %s, want 5 (override 0.20000000, not market default)", a.Leverage)
	}

	// ord-B: no inline fills, executed -> picked up from the single windowed GET /v1/fills call,
	// grouped by its own orderId.
	b := byID["ord-B"]
	if b.RealisedProfit != "-3.20000000" {
		t.Fatalf("ord-B RealisedProfit = %s, want -3.20000000 (from the windowed fill grouped by orderId)", b.RealisedProfit)
	}
	if b.Fee != "0.20000000" {
		t.Fatalf("ord-B Fee = %s, want 0.20000000", b.Fee)
	}

	// ord-C: never executed, no fills anywhere.
	c2 := byID["ord-C"]
	if c2.RealisedProfit != "" || c2.Fee != "" || c2.FeeAsset != "" {
		t.Fatalf("ord-C aggregation = (%q, %q, %q), want all empty (never executed)", c2.RealisedProfit, c2.Fee, c2.FeeAsset)
	}

	if fillsFetchCount != 1 {
		t.Fatalf("GET /v1/fills called %d times, want exactly 1 (one windowed call covers every order needing lookup)", fillsFetchCount)
	}

	// Wire order above is ascending (ord-A, ord-B, ord-C); output must be newest-first.
	gotOrder := []string{got[0].OrderID, got[1].OrderID, got[2].OrderID}
	want := []string{"ord-C", "ord-B", "ord-A"}
	if gotOrder[0] != want[0] || gotOrder[1] != want[1] || gotOrder[2] != want[2] {
		t.Fatalf("order = %v, want newest-first %v", gotOrder, want)
	}
}

func TestOrdersHistoryPropagatesWindowedFillsFetchError(t *testing.T) {
	// o-1 has no inline fills and a non-zero executedQuantity, so ordersHistory must attempt the
	// windowed GET /v1/fills fetch (fetchFillsPaged, via resolveMissingOrderFills) — and must
	// propagate its failure rather than silently leaving RealisedProfit/Fee empty.
	fixture := `[{"market":"ETH-USD","orderId":"o-1","time":1700000000000,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"1.00000000","reduceOnly":false}]`
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewOrdersHistory().Do(context.Background()); err == nil {
		t.Fatal("expected ordersHistory to propagate a windowed GET /v1/fills failure")
	}
}

// --- Fills backfill: hybrid windowed fetch + per-order fallback ---

// TestOrdersHistoryFallsBackPerOrderWhenWindowedFetchIsTruncated: the windowed fetch genuinely runs
// out of page budget (maxFillsPagingPages full pages, never a short/empty one) before it ever sees
// ord-Z's fill, which a single-request windowed fetch would have silently missed (rendering ord-Z's
// RealisedProfit/Fee as ""). The per-order fallback must resolve it instead.
func TestOrdersHistoryFallsBackPerOrderWhenWindowedFetchIsTruncated(t *testing.T) {
	ordersFixture := `[{"market":"ETH-USD","orderId":"ord-Z","time":1700000000000,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"1.00000000","reduceOnly":false}]`

	windowCalls := 0
	perOrderCalls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, ordersFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("orderId") == "ord-Z" {
				perOrderCalls++
				writeJSON(t, w, `[{"fillId":"f-z1","orderId":"ord-Z","price":"1990.00000000","quantity":"1.00000000","quoteQuantity":"1990.00000000","time":1700000000500,"realizedPnL":"9.00000000","fee":"0.15000000"}]`)
				return
			}
			// Windowed fetch: always a full page of fills unrelated to ord-Z (no OrderId set), so
			// the page bound is hit and ord-Z is never seen — a genuine truncation, not a
			// short/empty page signalling exhaustion.
			windowCalls++
			page := genFillsPage(windowCalls-1, katanaFillsPageCap)
			raw, err := json.Marshal(page)
			if err != nil {
				t.Fatal(err)
			}
			writeJSON(t, w, string(raw))
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewOrdersHistory().Do(context.Background())
	if err != nil {
		t.Fatalf("expected the per-order fallback to resolve ord-Z after windowed truncation, got error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1", len(got))
	}
	if got[0].RealisedProfit != "9.00000000" || got[0].Fee != "0.15000000" {
		t.Fatalf("RealisedProfit=%s Fee=%s, want 9.00000000 / 0.15000000 (from the per-order fallback, not left blank)", got[0].RealisedProfit, got[0].Fee)
	}
	if windowCalls != maxFillsPagingPages {
		t.Fatalf("windowed GET /v1/fills called %d times, want %d (the page bound)", windowCalls, maxFillsPagingPages)
	}
	if perOrderCalls != 1 {
		t.Fatalf("per-order GET /v1/fills?orderId=ord-Z called %d times, want 1", perOrderCalls)
	}
}

// TestOrdersHistoryFallbackResolvesOrderWhoseFillsFallOutsideWindow: ord-Y's fill genuinely
// executed outside the orders query's [start, end] window (no truncation — the windowed fetch's
// page is short, i.e. the window is legitimately exhausted with nothing for ord-Y in it). The
// per-order fallback does not scope by start/end at all, so it finds the fill regardless.
func TestOrdersHistoryFallbackResolvesOrderWhoseFillsFallOutsideWindow(t *testing.T) {
	ordersFixture := `[{"market":"ETH-USD","orderId":"ord-Y","time":1700000000000,"status":"filled","type":"market","side":"sell","originalQuantity":"2.00000000","executedQuantity":"2.00000000","reduceOnly":false}]`

	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, ordersFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("orderId") == "ord-Y" {
				writeJSON(t, w, `[{"fillId":"f-y1","orderId":"ord-Y","price":"2050.00000000","quantity":"2.00000000","quoteQuantity":"4100.00000000","time":1700099999000,"realizedPnL":"-1.50000000","fee":"0.40000000"}]`)
				return
			}
			// Windowed fetch, bounded by [start, end]: genuinely nothing for ord-Y.
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewOrdersHistory().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1", len(got))
	}
	if got[0].RealisedProfit != "-1.50000000" || got[0].Fee != "0.40000000" {
		t.Fatalf("RealisedProfit=%s Fee=%s, want -1.50000000 / 0.40000000 (from the per-order fallback, not left blank)", got[0].RealisedProfit, got[0].Fee)
	}
}

// TestOrdersHistoryErrorsWhenTooManyOrdersRemainUnmatched: more unmatched orders than
// maxPerOrderFillsFallback allows must error, naming the count, rather than silently emitting blank
// RealisedProfit/Fee for the orders beyond the cap.
func TestOrdersHistoryErrorsWhenTooManyOrdersRemainUnmatched(t *testing.T) {
	orders := make([]string, 0, 21)
	for i := 0; i < 21; i++ {
		orders = append(orders, fmt.Sprintf(
			`{"market":"ETH-USD","orderId":"ord-%02d","time":%d,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"1.00000000","reduceOnly":false}`,
			i, 1700000000000+int64(i)*1000,
		))
	}
	ordersFixture := "[" + strings.Join(orders, ",") + "]"

	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, ordersFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	_, err := c.NewOrdersHistory().Do(context.Background())
	if err == nil {
		t.Fatal("expected an error when more orders remain unmatched than the per-order fallback cap allows")
	}
	if !strings.Contains(err.Error(), "21") {
		t.Fatalf("err = %q, want it to name the unmatched count (21)", err.Error())
	}
}

// --- resolveMissingOrderFills / fetchFillsForOrder unit tests ---

func TestResolveMissingOrderFillsFallsBackToPerOrderForUnmatched(t *testing.T) {
	needFills := []katanaOrder{
		{OrderId: "ord-1", ExecutedQuantity: "1.00000000"},
		{OrderId: "ord-2", ExecutedQuantity: "1.00000000"},
	}
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			if oid := r.URL.Query().Get("orderId"); oid != "" {
				writeJSON(t, w, fmt.Sprintf(`[{"fillId":"f-%s","orderId":"%s","price":"1","quantity":"1","quoteQuantity":"1","time":1700000000000,"realizedPnL":"1.00000000","fee":"0.10000000"}]`, oid, oid))
				return
			}
			// Windowed fetch only ever finds ord-1.
			writeJSON(t, w, `[{"fillId":"f-w1","orderId":"ord-1","price":"1","quantity":"1","quoteQuantity":"1","time":1700000000000,"realizedPnL":"2.00000000","fee":"0.20000000"}]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := resolveMissingOrderFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, needFills, map[string]Market{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["ord-1"]) != 1 || got["ord-1"][0].FillId != "f-w1" {
		t.Fatalf("ord-1 = %v, want the windowed fill f-w1 (found without needing the fallback)", got["ord-1"])
	}
	if len(got["ord-2"]) != 1 || got["ord-2"][0].FillId != "f-ord-2" {
		t.Fatalf("ord-2 = %v, want the per-order fallback fill f-ord-2", got["ord-2"])
	}
}

func TestResolveMissingOrderFillsErrorsOverCap(t *testing.T) {
	needFills := make([]katanaOrder, 0, 21)
	for i := 0; i < 21; i++ {
		needFills = append(needFills, katanaOrder{OrderId: fmt.Sprintf("ord-%02d", i), ExecutedQuantity: "1.00000000"})
	}
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	_, err := resolveMissingOrderFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, needFills, map[string]Market{})
	if err == nil {
		t.Fatal("expected an error when more than maxPerOrderFillsFallback orders are unmatched")
	}
	if !strings.Contains(err.Error(), "21") {
		t.Fatalf("err = %q, want it to name the count (21)", err.Error())
	}
}

func TestResolveMissingOrderFillsPropagatesWindowedFetchError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	needFills := []katanaOrder{{OrderId: "ord-1", ExecutedQuantity: "1.00000000"}}
	if _, err := resolveMissingOrderFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, needFills, map[string]Market{}); err == nil {
		t.Fatal("expected an error when the windowed GET /v1/fills fetch fails")
	}
}

// --- Fills backfill: reconcile by summed quantity, not bucket presence ---

// TestResolveMissingOrderFillsFallsBackWhenWindowedQuantityIsPartial: the windowed fetch DOES have
// a bucket for the order, but it only covers part of ExecutedQuantity (e.g. the rest straddled the
// window's [start, end] boundary or the page bound) — a presence-only check would have accepted
// this bucket and understated the order's fee/PnL. The quantity reconciliation must reject it and
// route to the per-order fallback, whose result (the FULL picture) replaces the partial bucket
// rather than merging with it.
func TestResolveMissingOrderFillsFallsBackWhenWindowedQuantityIsPartial(t *testing.T) {
	needFills := []katanaOrder{{OrderId: "ord-1", Market: "ETH-USD", ExecutedQuantity: "1.00000000"}}
	mkts := map[string]Market{"ETH-USD": {Symbol: "ETH-USD", StepSize: "0.00000100"}}

	perOrderCalls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("orderId") == "ord-1" {
				perOrderCalls++
				// Authoritative, full picture: both fills that made up the order.
				writeJSON(t, w, `[
					{"fillId":"f-1","orderId":"ord-1","price":"2000.00000000","quantity":"0.40000000","quoteQuantity":"800.00000000","time":1700000000000,"realizedPnL":"1.00000000","fee":"0.10000000"},
					{"fillId":"f-2","orderId":"ord-1","price":"2001.00000000","quantity":"0.60000000","quoteQuantity":"1200.60000000","time":1700000000100,"realizedPnL":"2.00000000","fee":"0.15000000"}
				]`)
				return
			}
			// Windowed fetch only captured a PARTIAL fill: 0.4 of the order's 1.0 executedQuantity.
			writeJSON(t, w, `[{"fillId":"f-1","orderId":"ord-1","price":"2000.00000000","quantity":"0.40000000","quoteQuantity":"800.00000000","time":1700000000000,"realizedPnL":"1.00000000","fee":"0.10000000"}]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := resolveMissingOrderFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, needFills, mkts)
	if err != nil {
		t.Fatal(err)
	}
	if perOrderCalls != 1 {
		t.Fatalf("per-order fallback called %d times, want 1 (the windowed bucket only covered 0.4 of 1.0 executedQuantity)", perOrderCalls)
	}
	pnl, fee, _ := aggregateFills(got["ord-1"])
	if pnl != "3.00000000" || fee != "0.25000000" {
		t.Fatalf("aggregated pnl=%s fee=%s, want 3.00000000 / 0.25000000 (the FULL fallback result must replace, not merge with, the partial windowed bucket)", pnl, fee)
	}
}

// TestResolveMissingOrderFillsIssuesNoFallbackWhenWindowedQuantityFullyCovers is the counterpart:
// when the windowed bucket's summed quantity already reaches ExecutedQuantity, no per-order call
// should ever be made.
func TestResolveMissingOrderFillsIssuesNoFallbackWhenWindowedQuantityFullyCovers(t *testing.T) {
	needFills := []katanaOrder{{OrderId: "ord-1", Market: "ETH-USD", ExecutedQuantity: "1.00000000"}}
	mkts := map[string]Market{"ETH-USD": {Symbol: "ETH-USD", StepSize: "0.00000100"}}

	perOrderCalls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("orderId") != "" {
				perOrderCalls++
				writeJSON(t, w, `[]`)
				return
			}
			writeJSON(t, w, `[{"fillId":"f-1","orderId":"ord-1","price":"2000.00000000","quantity":"1.00000000","quoteQuantity":"2000.00000000","time":1700000000000,"realizedPnL":"5.00000000","fee":"0.40000000"}]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := resolveMissingOrderFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, needFills, mkts)
	if err != nil {
		t.Fatal(err)
	}
	if perOrderCalls != 0 {
		t.Fatalf("per-order fallback called %d times, want 0 (the windowed bucket already covers the full executedQuantity)", perOrderCalls)
	}
	if len(got["ord-1"]) != 1 || got["ord-1"][0].FillId != "f-1" {
		t.Fatalf("got %v, want the single windowed fill f-1 unchanged", got["ord-1"])
	}
}

// An empty per-order fallback result — the exchange reports executedQuantity != "0" but the
// unscoped, authoritative per-order fetch found nothing — must be logged by order id, not silently
// swallowed.
func TestResolveMissingOrderFillsLogsWhenPerOrderFallbackFindsNothing(t *testing.T) {
	needFills := []katanaOrder{{OrderId: "ord-1", Market: "ETH-USD", ExecutedQuantity: "1.00000000"}}
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	buf := captureLog(t)

	got, err := resolveMissingOrderFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, needFills, map[string]Market{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["ord-1"]) != 0 {
		t.Fatalf("got %v, want no fills for ord-1 (the exchange genuinely has none)", got["ord-1"])
	}
	if !strings.Contains(buf.String(), "ord-1") {
		t.Fatalf("expected a log line naming order ord-1, got: %q", buf.String())
	}
}

func TestQuantityReconcilesAcceptsShortfallUnderOneStepSize(t *testing.T) {
	fills := []katanaFill{{Quantity: "0.99500000"}}
	// executed 1.00000000, stepSize 0.01000000: shortfall 0.00500000 < stepSize.
	if !quantityReconciles(fills, "1.00000000", "0.01000000") {
		t.Fatal("expected a shortfall smaller than one stepSize to reconcile")
	}
}

func TestQuantityReconcilesRejectsShortfallOverOneStepSize(t *testing.T) {
	fills := []katanaFill{{Quantity: "0.50000000"}}
	if quantityReconciles(fills, "1.00000000", "0.01000000") {
		t.Fatal("expected a shortfall larger than one stepSize to NOT reconcile")
	}
}

func TestQuantityReconcilesExactMatchWithZeroTolerance(t *testing.T) {
	fills := []katanaFill{{Quantity: "1.00000000"}}
	if !quantityReconciles(fills, "1.00000000", "") {
		t.Fatal("expected an exact quantity match to reconcile even with an unknown stepSize (zero tolerance)")
	}
}

func TestQuantityReconcilesTreatsInvalidExecutedQuantityAsUnreconciled(t *testing.T) {
	if quantityReconciles(nil, "not-a-number", "0.01000000") {
		t.Fatal("expected an unparseable executedQuantity to force the fallback, not silently reconcile")
	}
}

func TestQuantityReconcilesSumsAcrossMultipleFills(t *testing.T) {
	fills := []katanaFill{{Quantity: "0.30000000"}, {Quantity: "0.30000000"}, {Quantity: "0.40000000"}}
	if !quantityReconciles(fills, "1.00000000", "0.00000001") {
		t.Fatal("expected 0.3 + 0.3 + 0.4 = 1.0 to reconcile against executedQuantity 1.0")
	}
}

func TestFetchFillsForOrderSendsOrderIdScope(t *testing.T) {
	var gotQuery url.Values
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := fetchFillsForOrder(context.Background(), c.callAPI, "0xWALLET", "ord-1"); err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("wallet") != "0xWALLET" || gotQuery.Get("orderId") != "ord-1" {
		t.Fatalf("query = %v, want wallet=0xWALLET orderId=ord-1", gotQuery)
	}
}

func TestFetchFillsForOrderPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := fetchFillsForOrder(context.Background(), c.callAPI, "0xWALLET", "ord-1"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestOrdersHistorySortsNewestFirstFromAscendingWireOrder(t *testing.T) {
	// Deliberately ascending wire order (oldest first) — the same order Katana's own samples happen
	// to use, per API_NOTES.md, but which must never be trusted as guaranteed.
	fixture := `[
		{"market":"ETH-USD","orderId":"o-old","time":1700000000000,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","reduceOnly":false},
		{"market":"ETH-USD","orderId":"o-mid","time":1700000001000,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","reduceOnly":false},
		{"market":"ETH-USD","orderId":"o-new","time":1700000002000,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","reduceOnly":false}
	]`
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewOrdersHistory().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3", len(got))
	}
	gotOrder := []string{got[0].OrderID, got[1].OrderID, got[2].OrderID}
	want := []string{"o-new", "o-mid", "o-old"}
	if gotOrder[0] != want[0] || gotOrder[1] != want[1] || gotOrder[2] != want[2] {
		t.Fatalf("order = %v, want newest-first %v", gotOrder, want)
	}
}

func TestOrdersHistorySurvivesOverrideEndpointError(t *testing.T) {
	fixture := `[{"market":"ETH-USD","orderId":"o-1","time":1700000000000,"status":"filled","type":"market","side":"buy","originalQuantity":"1.00000000","executedQuantity":"0.00000000","reduceOnly":false}]`
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders":  func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewOrdersHistory().Do(context.Background())
	if err != nil {
		t.Fatalf("ordersHistory must survive an override-endpoint failure, got: %v", err)
	}
	if len(got) != 1 || got[0].Leverage != "20" {
		t.Fatalf("got = %+v, want 1 row with leverage 20 (market default, override lookup failed)", got)
	}
}

func TestOrdersHistoryPropagatesMarketsEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewOrdersHistory().Do(context.Background()); err == nil {
		t.Fatal("expected ordersHistory to propagate a GET /v1/markets failure")
	}
}

func TestOrdersHistoryPagingParamsMapToStartEndLimit(t *testing.T) {
	var gotQuery url.Values
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/orders": func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			writeJSON(t, w, `[]`)
		},
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewOrdersHistory().
		Symbol("ETH-USD").
		StartTime(1700000000000).
		EndTime(1700100000000).
		Limit(5000). // above the 1000 cap, must be clamped
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	if gotQuery.Get("market") != "ETH-USD" {
		t.Fatalf("market = %s, want ETH-USD", gotQuery.Get("market"))
	}
	if gotQuery.Get("start") != "1700000000000" {
		t.Fatalf("start = %s, want 1700000000000", gotQuery.Get("start"))
	}
	if gotQuery.Get("end") != "1700100000000" {
		t.Fatalf("end = %s, want 1700100000000", gotQuery.Get("end"))
	}
	if gotQuery.Get("limit") != "1000" {
		t.Fatalf("limit = %s, want 1000 (5000 must be clamped to the documented cap)", gotQuery.Get("limit"))
	}
	if gotQuery.Has("page") || gotQuery.Has("cursor") {
		t.Fatal("page/cursor have no Katana equivalent and must not be sent")
	}
}

// --- positionsHistory ---

const fillsForPositionsHistoryFixture = `[
	{"fillId":"f-open","market":"ETH-USD","price":"2000.00000000","quantity":"1.00000000","quoteQuantity":"2000.00000000","time":1700000000000,"side":"buy","position":"long","action":"open"},
	{"fillId":"f-close","market":"ETH-USD","price":"2100.00000000","quantity":"1.00000000","quoteQuantity":"2100.00000000","time":1700000005000,"side":"sell","position":"long","action":"close","realizedPnL":"100.00000000","fee":"0.80000000"}
]`

func TestPositionsHistoryKeepsOnlyClosingFillsAndResolvesLeverage(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fillsForPositionsHistoryFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, ethOverrideFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewPositionsHistory().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1 (the opening fill with no realizedPnL must be dropped)", len(got))
	}
	want := entity.Futures_PositionsHistory{
		Symbol: "ETH-USD", PositionID: "", PositionSide: "LONG", PositionAmt: "1.00000000",
		ExecutedPositionAmt: "1.00000000", AvgPrice: "2100.00000000", ExecutedAvgPrice: "2100.00000000",
		RealisedProfit: "100.00000000", Leverage: "5", Fee: "0.80000000", Funding: "",
		MarginMode: "CROSS", CreateTime: 1700000005000, UpdateTime: 1700000005000,
	}
	if got[0] != want {
		t.Fatalf("positionsHistory[0] = %+v, want %+v", got[0], want)
	}
}

// TestPositionsHistoryResolvesLeveragePerFillMarket: a page of closing fills can span markets with
// different margin fractions. Each row must report ITS OWN market's leverage — one leverage applied
// to the whole page would report another market's number on half of it.
func TestPositionsHistoryResolvesLeveragePerFillMarket(t *testing.T) {
	fixture := `[
		{"fillId":"f-eth","market":"ETH-USD","price":"2100.00000000","quantity":"1.00000000","quoteQuantity":"2100.00000000","time":1700000005000,"side":"sell","position":"long","realizedPnL":"100.00000000","fee":"0.80000000"},
		{"fillId":"f-btc","market":"BTC-USD","price":"60000.00000000","quantity":"0.10000000","quoteQuantity":"6000.00000000","time":1700000006000,"side":"sell","position":"long","realizedPnL":"50.00000000","fee":"0.40000000"}
	]`
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewPositionsHistory().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	byMarket := map[string]entity.Futures_PositionsHistory{}
	for _, row := range got {
		byMarket[row.Symbol] = row
	}
	if byMarket["ETH-USD"].Leverage != "20" {
		t.Fatalf("ETH-USD leverage = %q, want 20 (1 / 0.05000000)", byMarket["ETH-USD"].Leverage)
	}
	if byMarket["BTC-USD"].Leverage != "50" {
		t.Fatalf("BTC-USD leverage = %q, want 50 (1 / 0.02000000) — not ETH's number", byMarket["BTC-USD"].Leverage)
	}
}

// Sending Katana's own `limit` straight through and filtering client-side after the fact can
// under-fill a requested page — a 50/50 open/close mix at limit N previously returned roughly N/2
// rows, which a pager's short-page check misreads as "no more data". Here the wallet requests 2
// rows and the single window response contains exactly 2 closing fills among 2 opening ones (all
// within one short/exhausted page); both closing rows must come back.
const mixedFillsFullPageFixture = `[
	{"fillId":"f-o1","market":"ETH-USD","price":"2000.00000000","quantity":"1.00000000","quoteQuantity":"2000.00000000","time":1700000000000,"side":"buy","position":"long"},
	{"fillId":"f-c1","market":"ETH-USD","price":"2010.00000000","quantity":"1.00000000","quoteQuantity":"2010.00000000","time":1700000001000,"side":"sell","position":"long","realizedPnL":"10.00000000","fee":"0.20000000"},
	{"fillId":"f-o2","market":"ETH-USD","price":"2020.00000000","quantity":"1.00000000","quoteQuantity":"2020.00000000","time":1700000002000,"side":"buy","position":"long"},
	{"fillId":"f-c2","market":"ETH-USD","price":"2030.00000000","quantity":"1.00000000","quoteQuantity":"2030.00000000","time":1700000003000,"side":"sell","position":"long","realizedPnL":"20.00000000","fee":"0.30000000"}
]`

func TestPositionsHistoryReturnsFullRequestedPageDespiteMixedOpenCloseFills(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, mixedFillsFullPageFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewPositionsHistory().Limit(2).Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2 (both closing fills in the mixed set)", len(got))
	}
	// Sorted newest-first — f-c2 (time 1700000003000) before f-c1 (time 1700000001000).
	if got[0].RealisedProfit != "20.00000000" || got[1].RealisedProfit != "10.00000000" {
		t.Fatalf("got = %+v, want newest-first [20.00000000, 10.00000000]", got)
	}
}

func TestPositionsHistorySurvivesOverrideEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fillsForPositionsHistoryFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewPositionsHistory().Do(context.Background())
	if err != nil {
		t.Fatalf("positionsHistory must survive an override-endpoint failure, got: %v", err)
	}
	if len(got) != 1 || got[0].Leverage != "20" {
		t.Fatalf("got = %+v, want 1 row with leverage 20 (market default, override lookup failed)", got)
	}
}

func TestPositionsHistoryPropagatesFillsEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewPositionsHistory().Do(context.Background()); err == nil {
		t.Fatal("expected positionsHistory to propagate a GET /v1/fills failure")
	}
}

func TestPositionsHistoryPropagatesMarketsEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewPositionsHistory().Do(context.Background()); err == nil {
		t.Fatal("expected positionsHistory to propagate a GET /v1/markets failure")
	}
}

// --- fetchClosingFills / fetchFillsPaged (the pagination loop behind positionsHistory and
// ordersHistory's fills fallback) ---

// 4 closing fills in one short (window-exhausting) page, target=2. Trimming to target BEFORE
// sorting would keep whichever 2 came first (f-1, f-2) and merely re-sort that already-wrong pair.
// The result must be the NEWEST two by Time (f-4, f-3), not the first two encountered.
func TestFetchClosingFillsReturnsNewestTargetNotFirstEncountered(t *testing.T) {
	fixture := `[
		{"fillId":"f-1","market":"ETH-USD","price":"1","quantity":"1","quoteQuantity":"1","time":1700000000000,"realizedPnL":"1.00000000","fee":"0.10000000"},
		{"fillId":"f-2","market":"ETH-USD","price":"1","quantity":"1","quoteQuantity":"1","time":1700000001000,"realizedPnL":"2.00000000","fee":"0.10000000"},
		{"fillId":"f-3","market":"ETH-USD","price":"1","quantity":"1","quoteQuantity":"1","time":1700000002000,"realizedPnL":"3.00000000","fee":"0.10000000"},
		{"fillId":"f-4","market":"ETH-USD","price":"1","quantity":"1","quoteQuantity":"1","time":1700000003000,"realizedPnL":"4.00000000","fee":"0.10000000"}
	]`
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fixture) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := fetchClosingFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d closing fills, want 2", len(got))
	}
	if got[0].FillId != "f-4" || got[1].FillId != "f-3" {
		t.Fatalf("got fillIds [%s, %s], want [f-4, f-3] (the NEWEST two, not the first two encountered)", got[0].FillId, got[1].FillId)
	}
}

func TestFetchClosingFillsSingleShortPageReturnsAllMatchingClosingFills(t *testing.T) {
	calls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			calls++
			writeJSON(t, w, mixedFillsFullPageFixture) // 4 raw fills, 2 closing
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := fetchClosingFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d closing fills, want 2", len(got))
	}
	if calls != 1 {
		t.Fatalf("GET /v1/fills called %d times, want 1 (a single short page already exhausts the window)", calls)
	}
}

// TestFetchClosingFillsTrimsToNewestTargetAcrossMultiplePages: the newest-target trim must hold
// across page boundaries, not just within one page. 1500 closing fills spread across two pages
// (1000 + 500, mirroring the real endpoint's inclusive fromId semantics), target=100: the newest
// 100 (fill seq 1400-1499) must come back, spanning what would have been split across both pages
// during the scan, not whichever page happened to be fetched first.
func TestFetchClosingFillsTrimsToNewestTargetAcrossMultiplePages(t *testing.T) {
	const total = 1500
	master := genFillsRange(0, total) // every fill closing, ascending time/id

	calls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			calls++
			start := 0
			if fromID := r.URL.Query().Get("fromId"); fromID != "" {
				for i, f := range master {
					if f.FillId == fromID {
						start = i // inclusive, same real-endpoint semantics as the dedup test
						break
					}
				}
			}
			end := start + katanaFillsPageCap
			if end > len(master) {
				end = len(master)
			}
			raw, err := json.Marshal(master[start:end])
			if err != nil {
				t.Fatal(err)
			}
			writeJSON(t, w, string(raw))
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	const target = 100
	got, err := fetchClosingFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != target {
		t.Fatalf("got %d closing fills, want exactly %d", len(got), target)
	}
	if got[0].FillId != "f-01499" {
		t.Fatalf("got[0] = %s, want f-01499 (the single newest fill across both pages)", got[0].FillId)
	}
	if got[target-1].FillId != "f-01400" {
		t.Fatalf("got[%d] = %s, want f-01400 (the 100th-newest fill)", target-1, got[target-1].FillId)
	}
	if calls != 2 {
		t.Fatalf("GET /v1/fills called %d times, want 2 (1000 + 500, page 2 short and exhausts the window)", calls)
	}
}

func TestFetchClosingFillsStopsWhenWindowExhaustedBeforeTarget(t *testing.T) {
	calls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			calls++
			// Short page (5 raw fills, well under katanaFillsPageCap): 2 closing. Must be treated
			// as "window exhausted" even though target (10) is never reached, not retried forever.
			writeJSON(t, w, `[
				{"fillId":"f-1","market":"ETH-USD","price":"1.00000000","quantity":"1.00000000","quoteQuantity":"1.00000000","time":1700000000000},
				{"fillId":"f-2","market":"ETH-USD","price":"1.00000000","quantity":"1.00000000","quoteQuantity":"1.00000000","time":1700000001000,"realizedPnL":"1.00000000","fee":"0.10000000"},
				{"fillId":"f-3","market":"ETH-USD","price":"1.00000000","quantity":"1.00000000","quoteQuantity":"1.00000000","time":1700000002000},
				{"fillId":"f-4","market":"ETH-USD","price":"1.00000000","quantity":"1.00000000","quoteQuantity":"1.00000000","time":1700000003000,"realizedPnL":"2.00000000","fee":"0.20000000"},
				{"fillId":"f-5","market":"ETH-USD","price":"1.00000000","quantity":"1.00000000","quoteQuantity":"1.00000000","time":1700000004000}
			]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := fetchClosingFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d closing fills, want 2 (window exhausted after one short page)", len(got))
	}
	if calls != 1 {
		t.Fatalf("GET /v1/fills called %d times, want 1 (a short page signals exhaustion)", calls)
	}
}

func TestFetchClosingFillsStopsOnEmptyPage(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, `[]`) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := fetchClosingFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d closing fills, want 0", len(got))
	}
}

func TestFetchClosingFillsPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := fetchClosingFills(context.Background(), c.callAPI, "0xWALLET", "", nil, nil, 5); err == nil {
		t.Fatal("expected fetchClosingFills to propagate a GET /v1/fills failure")
	}
}

// genFillsPage builds size synthetic katanaFill entries for pageIndex (fillIds
// "f-00000".."f-00999" for page 0, "f-01000".."f-01999" for page 1, and so on, times strictly
// ascending with the index), every other one a closing fill (non-empty RealizedPnL/Fee) — used by
// the multi-page tests, the only ones that need a full katanaFillsPageCap-sized page.
func genFillsPage(pageIndex, size int) []katanaFill {
	out := make([]katanaFill, 0, size)
	for i := 0; i < size; i++ {
		seq := pageIndex*size + i
		f := katanaFill{
			FillId: fmt.Sprintf("f-%05d", seq), Market: "ETH-USD", Price: "2000.00000000",
			Quantity: "1.00000000", QuoteQuantity: "2000.00000000", Time: int64(1700000000000 + seq),
		}
		if i%2 == 0 {
			f.RealizedPnL = "1.00000000"
			f.Fee = "0.10000000"
		}
		out = append(out, f)
	}
	return out
}

// genFillsRange builds count synthetic, EVERY-fill-closing katanaFill entries starting at global
// sequence startSeq (fillIds "f-00000", "f-00001", ..., times strictly ascending with the sequence)
// — used by the dedup test, where every fill must count toward the expected total for the
// arithmetic to be exact.
func genFillsRange(startSeq, count int) []katanaFill {
	out := make([]katanaFill, 0, count)
	for i := 0; i < count; i++ {
		seq := startSeq + i
		out = append(out, katanaFill{
			FillId: fmt.Sprintf("f-%05d", seq), Market: "ETH-USD", Price: "2000.00000000",
			Quantity: "1.00000000", QuoteQuantity: "2000.00000000", Time: int64(1700000000000 + seq),
			RealizedPnL: "1.00000000", Fee: "0.10000000",
		})
	}
	return out
}

func TestLatestFillID(t *testing.T) {
	fills := []katanaFill{
		{FillId: "a", Time: 100},
		{FillId: "c", Time: 300},
		{FillId: "b", Time: 200},
	}
	if got := latestFillID(fills); got != "c" {
		t.Fatalf("latestFillID = %s, want c (the greatest Time, not the last element)", got)
	}
	if got := latestFillID(nil); got != "" {
		t.Fatalf("latestFillID(nil) = %q, want empty", got)
	}
}

// TestLatestFillIDBreaksTiesByFirstOccurrence: two fills sharing the same (maximum) Time. The
// strict `>` comparison means only a STRICTLY greater Time replaces the running max, so the first
// fill to reach the max Time wins and later ties are ignored. Whichever fill wins the tie, the
// result is deterministic across calls with the same input — that determinism, not a particular
// "correct" winner, is what matters for fromId's cursor role: this pins the current behaviour so a
// future change to the comparison is a visible, deliberate decision.
func TestLatestFillIDBreaksTiesByFirstOccurrence(t *testing.T) {
	fills := []katanaFill{
		{FillId: "a", Time: 100},
		{FillId: "b", Time: 300},
		{FillId: "c", Time: 300}, // ties with b
		{FillId: "d", Time: 200},
	}
	if got := latestFillID(fills); got != "b" {
		t.Fatalf("latestFillID = %s, want b (first fill to reach the max Time)", got)
	}
}

// TestFetchFillsPagedAdvancesCursorByLatestTimeNotLastElement: API_NOTES.md documents no ordering
// guarantee for GET /v1/fills. Page 1 here is deliberately NOT time-ascending — its
// chronologically-latest fill sits one position before the literal last array element — proving the
// paging cursor is computed from `time` (latestFillID), not lifted from raw[len(raw)-1].
func TestFetchFillsPagedAdvancesCursorByLatestTimeNotLastElement(t *testing.T) {
	page1 := genFillsPage(0, katanaFillsPageCap)
	// Swap the last two elements: position len-1 no longer holds the max-time fill; len-2 does.
	page1[len(page1)-1], page1[len(page1)-2] = page1[len(page1)-2], page1[len(page1)-1]
	wantCursor := page1[len(page1)-2].FillId

	calls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				raw, err := json.Marshal(page1)
				if err != nil {
					t.Fatal(err)
				}
				writeJSON(t, w, string(raw))
				return
			}
			got := r.URL.Query().Get("fromId")
			if got != wantCursor {
				t.Fatalf("page 2 fromId = %q, want %q (the chronologically-latest fill, not the last array element)", got, wantCursor)
			}
			writeJSON(t, w, `[]`) // window exhausted
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := fetchFillsPaged(context.Background(), c.callAPI, "0xWALLET", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("GET /v1/fills called %d times, want 2", calls)
	}
}

// TestFetchFillsPagedDedupesInclusiveFromIdBoundaryAcrossPages: API_NOTES.md documents `fromId` as
// the fillId to resume from, and the fake server here implements the documented INCLUSIVE semantics
// — the fill named by `fromId` is deliberately served again as part of the next page, exactly like
// the real endpoint. Without dedup, that boundary fill would be double-counted.
func TestFetchFillsPagedDedupesInclusiveFromIdBoundaryAcrossPages(t *testing.T) {
	const total = 1500
	master := genFillsRange(0, total)

	calls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			calls++
			start := 0
			if fromID := r.URL.Query().Get("fromId"); fromID != "" {
				for i, f := range master {
					if f.FillId == fromID {
						start = i // inclusive: the next page begins AT fromId, not after it
						break
					}
				}
			}
			end := start + katanaFillsPageCap
			if end > len(master) {
				end = len(master)
			}
			raw, err := json.Marshal(master[start:end])
			if err != nil {
				t.Fatal(err)
			}
			writeJSON(t, w, string(raw))
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := fetchFillsPaged(context.Background(), c.callAPI, "0xWALLET", "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != total {
		t.Fatalf("got %d fills, want exactly %d unique fills (the inclusive fromId boundary must not be double-counted)", len(got), total)
	}
	seenIDs := make(map[string]bool, len(got))
	for _, f := range got {
		if seenIDs[f.FillId] {
			t.Fatalf("duplicate fillId %s in result", f.FillId)
		}
		seenIDs[f.FillId] = true
	}
	if calls != 2 {
		t.Fatalf("GET /v1/fills called %d times, want 2 (1000 + 501, page 2 short and exhausts the window)", calls)
	}
}

func TestFetchFillsPagedStopsAtPageBoundAndLogsWhenWindowNeverExhausts(t *testing.T) {
	calls := 0
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			calls++
			// Always a full page — simulates a window so deep it never runs out within the bound.
			page := genFillsPage(0, katanaFillsPageCap)
			raw, err := json.Marshal(page)
			if err != nil {
				t.Fatal(err)
			}
			writeJSON(t, w, string(raw))
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	buf := captureLog(t)

	got, err := fetchFillsPaged(context.Background(), c.callAPI, "0xWALLET", "ETH-USD", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != maxFillsPagingPages {
		t.Fatalf("GET /v1/fills called %d times, want exactly %d (the page bound)", calls, maxFillsPagingPages)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one fill to be collected before the bound was hit")
	}
	if !strings.Contains(buf.String(), "page bound") {
		t.Fatalf("expected a log line about hitting the page bound before window exhaustion, got: %q", buf.String())
	}
}

func TestFetchFillsPagedSendsWindowBoundsAndMaxLimit(t *testing.T) {
	var gotQuery url.Values
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.Query()
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	start := int64(1)
	end := int64(2)
	if _, err := fetchFillsPaged(context.Background(), c.callAPI, "0xWALLET", "ETH-USD", &start, &end); err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("market") != "ETH-USD" || gotQuery.Get("start") != "1" || gotQuery.Get("end") != "2" || gotQuery.Get("limit") != "1000" {
		t.Fatalf("query = %v, want market=ETH-USD start=1 end=2 limit=1000", gotQuery)
	}
	if gotQuery.Has("orderId") || gotQuery.Has("fromId") {
		t.Fatal("the first page must not send orderId or fromId")
	}
}

func TestFetchFillsPagedPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := fetchFillsPaged(context.Background(), c.callAPI, "0xWALLET", "", nil, nil); err == nil {
		t.Fatal("expected fetchFillsPaged to propagate a GET /v1/fills failure")
	}
}

// --- userTrades ---

const fillsForUserTradesFixture = `[
	{"fillId":"trade-1","orderId":"ord-1","market":"ETH-USD","price":"2000.00000000","quantity":"1.50000000","quoteQuantity":"3000.00000000","time":1700000000000,"side":"buy","position":"long","fee":"1.20000000","liquidity":"taker","realizedPnL":""}
]`

func TestUserTradesMapsFillsDirectly(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fillsForUserTradesFixture) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewUserTrades().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.Futures_UserTrades{
		TradeID: "trade-1", OrderID: "ord-1", Symbol: "ETH-USD", Side: "BUY", PositionSide: "LONG",
		Price: "2000.00000000", Qty: "1.50000000", QuoteQty: "3000.00000000", Commission: "1.20000000",
		CommissionAsset: "USDC", RealisedProfit: "", Buyer: true, Maker: false, Time: 1700000000000,
	}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("userTrades = %+v, want [%+v]", got, want)
	}
}

// TestUserTradesMapsBuilderFee pins the builder-fee passthrough: Katana reports it per fill and
// entity.Futures_UserTrades has a slot for it, so it must not be dropped on the floor.
func TestUserTradesMapsBuilderFee(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/fills": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[{"fillId":"trade-1","orderId":"ord-1","market":"ETH-USD","price":"2000.00000000","quantity":"1.00000000","quoteQuantity":"2000.00000000","time":1700000000000,"side":"buy","position":"long","fee":"1.20000000","builderFee":"0.05000000","liquidity":"taker"}]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewUserTrades().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].BuilderFee != "0.05000000" {
		t.Fatalf("builderFee = %q, want 0.05000000", got[0].BuilderFee)
	}
}

func TestUserTradesSortsNewestFirstFromAscendingWireOrder(t *testing.T) {
	fixture := `[
		{"fillId":"t-old","orderId":"o-1","market":"ETH-USD","price":"2000.00000000","quantity":"1.00000000","quoteQuantity":"2000.00000000","time":1700000000000,"side":"buy","position":"long"},
		{"fillId":"t-mid","orderId":"o-2","market":"ETH-USD","price":"2001.00000000","quantity":"1.00000000","quoteQuantity":"2001.00000000","time":1700000001000,"side":"buy","position":"long"},
		{"fillId":"t-new","orderId":"o-3","market":"ETH-USD","price":"2002.00000000","quantity":"1.00000000","quoteQuantity":"2002.00000000","time":1700000002000,"side":"buy","position":"long"}
	]`
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, fixture) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewUserTrades().Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d rows, want 3", len(got))
	}
	gotOrder := []string{got[0].TradeID, got[1].TradeID, got[2].TradeID}
	want := []string{"t-new", "t-mid", "t-old"}
	if gotOrder[0] != want[0] || gotOrder[1] != want[1] || gotOrder[2] != want[2] {
		t.Fatalf("order = %v, want newest-first %v", gotOrder, want)
	}
}

func TestUserTradesPropagatesEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/fills":   func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewUserTrades().Do(context.Background()); err == nil {
		t.Fatal("expected userTrades to propagate a GET /v1/fills failure")
	}
}

// --- getLeverage ---

func TestGetLeveragePrefersOverrideOverMarketDefault(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, ethOverrideFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetLeverage().Symbol("ETH-USD").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.Futures_Leverage{Symbol: "ETH-USD", Leverage: "5", MarginMode: "CROSS"}
	if got != want {
		t.Fatalf("getLeverage = %+v, want %+v", got, want)
	}
}

func TestGetLeverageFallsBackToMarketDefaultWithoutOverride(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, `[]`)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got, err := c.NewGetLeverage().Symbol("ETH-USD").Do(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := entity.Futures_Leverage{Symbol: "ETH-USD", Leverage: "20", MarginMode: "CROSS"}
	if got != want {
		t.Fatalf("getLeverage = %+v, want %+v (1 / market default 0.05000000)", got, want)
	}
}

func TestGetLeverageErrorsOnMissingSymbol(t *testing.T) {
	c := NewFuturesClient("k", "s", "")
	if _, err := c.NewGetLeverage().Do(context.Background()); err == nil {
		t.Fatal("expected an error when no symbol was set")
	}
}

func TestGetLeverageErrorsOnEmptySymbol(t *testing.T) {
	c := NewFuturesClient("k", "s", "")
	if _, err := c.NewGetLeverage().Symbol("   ").Do(context.Background()); err == nil {
		t.Fatal("expected an error for a blank symbol")
	}
}

func TestGetLeverageErrorsOnUnknownMarket(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetLeverage().Symbol("DOGE-USD").Do(context.Background()); err == nil {
		t.Fatal("expected an error for a market not present in markets()")
	}
}

// For getLeverage the override IS the requested answer, so silently substituting the market default
// on a lookup failure would tell a wallet that set 5x that it has 20x, with no error and no signal
// the number is unreliable. Unlike positions/history, the error must propagate.
func TestGetLeveragePropagatesOverrideEndpointError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/wallets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, singleWalletFixture) },
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) { writeJSON(t, w, twoMarketsFixture) },
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	if _, err := c.NewGetLeverage().Symbol("ETH-USD").Do(context.Background()); err == nil {
		t.Fatal("expected getLeverage to propagate an override-endpoint failure rather than silently answering with the market default leverage")
	}
}

// --- Pure helper unit tests ---

func TestIsZeroDecimal(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"0.00000000", true},
		{"0", true},
		{"-0.00000000", true},
		{"0.00000001", false},
		{"-1.00000000", false},
		{"not-a-number", false},
	}
	for _, tc := range cases {
		if got := isZeroDecimal(tc.in); got != tc.want {
			t.Fatalf("isZeroDecimal(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestClampLimit(t *testing.T) {
	five := int64(5)
	zero := int64(0)
	negative := int64(-10)
	huge := int64(1_000_000)
	thousand := int64(1000)

	cases := []struct {
		name string
		in   *int64
		want string
	}{
		{"nil", nil, ""},
		{"typical", &five, "5"},
		{"zero", &zero, ""},
		{"negative", &negative, ""},
		{"above cap", &huge, "1000"},
		{"exactly at cap", &thousand, "1000"},
	}
	for _, tc := range cases {
		if got := clampLimit(tc.in); got != tc.want {
			t.Fatalf("clampLimit(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestAggregateFillsSumsAcrossMultipleFillsAtDifferentPricesAndFees(t *testing.T) {
	// Hand-computed: pnl 12.34500000 + (-4.10000000) + 0.00000100 = 8.24500100
	// fee 1.11111111 + 0.88888889 + 0.00000010 = 2.00000010
	fills := []katanaFill{
		{RealizedPnL: "12.34500000", Fee: "1.11111111"},
		{RealizedPnL: "-4.10000000", Fee: "0.88888889"},
		{RealizedPnL: "0.00000100", Fee: "0.00000010"},
	}
	pnl, fee, asset := aggregateFills(fills)
	if pnl != "8.24500100" {
		t.Fatalf("realisedProfit = %s, want 8.24500100", pnl)
	}
	if fee != "2.00000010" {
		t.Fatalf("fee = %s, want 2.00000010", fee)
	}
	if asset != "USDC" {
		t.Fatalf("feeAsset = %s, want USDC", asset)
	}
}

func TestAggregateFillsSkipsUnparseableAndOmittedFields(t *testing.T) {
	// Second fill has no realizedPnL (open action) and a garbage fee; only the first and third
	// fills' values should land in the sums.
	fills := []katanaFill{
		{RealizedPnL: "5.00000000", Fee: "0.50000000"},
		{RealizedPnL: "", Fee: "garbage"},
		{RealizedPnL: "1.00000000", Fee: "0.25000000"},
	}
	pnl, fee, asset := aggregateFills(fills)
	if pnl != "6.00000000" {
		t.Fatalf("realisedProfit = %s, want 6.00000000", pnl)
	}
	if fee != "0.75000000" {
		t.Fatalf("fee = %s, want 0.75000000", fee)
	}
	if asset != "USDC" {
		t.Fatalf("feeAsset = %s, want USDC", asset)
	}
}

func TestAggregateFillsEmptyReturnsAllEmpty(t *testing.T) {
	pnl, fee, asset := aggregateFills(nil)
	if pnl != "" || fee != "" || asset != "" {
		t.Fatalf("aggregateFills(nil) = (%q, %q, %q), want all empty", pnl, fee, asset)
	}
}

func TestEffectiveMarketPrefersOverride(t *testing.T) {
	mkts := map[string]Market{"ETH-USD": {Symbol: "ETH-USD", InitialMarginFraction: "0.05000000"}}
	overrides := map[string]string{"ETH-USD": "0.25000000"}

	m, ok := effectiveMarket(mkts, overrides, "ETH-USD")
	if !ok {
		t.Fatal("expected ETH-USD to resolve")
	}
	if m.InitialMarginFraction != "0.25000000" {
		t.Fatalf("InitialMarginFraction = %s, want the override 0.25000000", m.InitialMarginFraction)
	}
}

func TestEffectiveMarketFallsBackToDefaultWithoutOverride(t *testing.T) {
	mkts := map[string]Market{"ETH-USD": {Symbol: "ETH-USD", InitialMarginFraction: "0.05000000"}}

	m, ok := effectiveMarket(mkts, map[string]string{}, "ETH-USD")
	if !ok {
		t.Fatal("expected ETH-USD to resolve")
	}
	if m.InitialMarginFraction != "0.05000000" {
		t.Fatalf("InitialMarginFraction = %s, want the market default 0.05000000", m.InitialMarginFraction)
	}
}

func TestEffectiveMarketReportsUnknownSymbol(t *testing.T) {
	_, ok := effectiveMarket(map[string]Market{}, map[string]string{}, "DOGE-USD")
	if ok {
		t.Fatal("expected ok=false for a symbol absent from markets()")
	}
}

func TestToPositionHistoryMapsFieldsLiterally(t *testing.T) {
	got := toPositionHistory(katanaFill{
		Market: "ETH-USD", Quantity: "2.00000000", Price: "2050.00000000",
		RealizedPnL: "42.00000000", Fee: "0.30000000", Position: "short", Time: 1700000000000,
	}, "10")
	want := entity.Futures_PositionsHistory{
		Symbol: "ETH-USD", PositionID: "", PositionSide: "SHORT", PositionAmt: "2.00000000",
		ExecutedPositionAmt: "2.00000000", AvgPrice: "2050.00000000", ExecutedAvgPrice: "2050.00000000",
		RealisedProfit: "42.00000000", Leverage: "10", Fee: "0.30000000", Funding: "",
		MarginMode: "CROSS", CreateTime: 1700000000000, UpdateTime: 1700000000000,
	}
	if got != want {
		t.Fatalf("toPositionHistory = %+v, want %+v", got, want)
	}
}

func TestToUserTradeMapsSellMakerFill(t *testing.T) {
	got := toUserTrade(katanaFill{
		FillId: "f-1", OrderId: "o-1", Market: "ETH-USD", Price: "1999.00000000", Quantity: "0.50000000",
		QuoteQuantity: "999.50000000", Time: 1700000000000, Side: "sell", Position: "short",
		Fee: "0.05000000", Liquidity: "maker", RealizedPnL: "3.00000000",
	})
	want := entity.Futures_UserTrades{
		TradeID: "f-1", OrderID: "o-1", Symbol: "ETH-USD", Side: "SELL", PositionSide: "SHORT",
		Price: "1999.00000000", Qty: "0.50000000", QuoteQty: "999.50000000", Commission: "0.05000000",
		CommissionAsset: "USDC", RealisedProfit: "3.00000000", Buyer: false, Maker: true, Time: 1700000000000,
	}
	if got != want {
		t.Fatalf("toUserTrade = %+v, want %+v", got, want)
	}
}

func TestToInstrumentInfoFallsBackToEmptyMaxLeverageOnInvalidIMF(t *testing.T) {
	got := toInstrumentInfo(Market{Symbol: "ETH-USD", InitialMarginFraction: "0"})
	if got.MaxLeverage != "" {
		t.Fatalf("MaxLeverage = %q, want empty when the market IMF is invalid", got.MaxLeverage)
	}
}

// --- Helper unit tests ---

func TestSortDescendingByTime(t *testing.T) {
	items := []int64{100, 300, 200}
	sortDescendingByTime(items, func(v int64) int64 { return v })
	want := []int64{300, 200, 100}
	if items[0] != want[0] || items[1] != want[1] || items[2] != want[2] {
		t.Fatalf("got %v, want %v", items, want)
	}
}

func TestSortDescendingByTimeIsStableForEqualTimestamps(t *testing.T) {
	type row struct {
		id   string
		time int64
	}
	items := []row{{"a", 100}, {"b", 100}, {"c", 200}}
	sortDescendingByTime(items, func(r row) int64 { return r.time })
	// c (200) must sort first; a and b (both 100) must keep their original relative order (a before
	// b) rather than an unspecified one.
	if items[0].id != "c" || items[1].id != "a" || items[2].id != "b" {
		t.Fatalf("got %v, want [c a b]", items)
	}
}

func TestResolvedLimit(t *testing.T) {
	five := int64(5)
	zero := int64(0)
	negative := int64(-1)
	huge := int64(5000)

	cases := []struct {
		name string
		in   *int64
		def  int
		want int
	}{
		{"nil uses default", nil, 50, 50},
		{"zero uses default", &zero, 50, 50},
		{"negative uses default", &negative, 50, 50},
		{"typical value passes through", &five, 50, 5},
		{"clamped to 1000", &huge, 50, 1000},
	}
	for _, tc := range cases {
		if got := resolvedLimit(tc.in, tc.def); got != tc.want {
			t.Fatalf("%s: resolvedLimit = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestGroupFillsByOrderID(t *testing.T) {
	fills := []katanaFill{
		{FillId: "f1", OrderId: "o1"},
		{FillId: "f2", OrderId: "o1"},
		{FillId: "f3", OrderId: "o2"},
		{FillId: "f4", OrderId: ""}, // liquidation/ADL fill, no orderId -> dropped
	}
	got := groupFillsByOrderID(fills)
	if len(got["o1"]) != 2 {
		t.Fatalf("o1 fills = %d, want 2", len(got["o1"]))
	}
	if len(got["o2"]) != 1 {
		t.Fatalf("o2 fills = %d, want 1", len(got["o2"]))
	}
	if _, ok := got[""]; ok {
		t.Fatal("fills with no orderId must not create a \"\" bucket")
	}
}

func TestResolveIMFOverridesBestEffortReturnsEmptyMapOnError(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got := resolveIMFOverridesBestEffort(context.Background(), c.callAPI, "0xWALLET", "")
	if len(got) != 0 {
		t.Fatalf("got %v, want an empty map on endpoint failure", got)
	}
}

// resetIMFOverrideLogThrottle clears the package-level rate-limit state, so a test's own log
// expectations do not depend on which earlier test last tripped it. Restores the previous value on
// cleanup for the same reason.
func resetIMFOverrideLogThrottle(t *testing.T) {
	t.Helper()
	imfOverrideLogMu.Lock()
	original := imfOverrideLoggedAt
	imfOverrideLoggedAt = time.Time{}
	imfOverrideLogMu.Unlock()
	t.Cleanup(func() {
		imfOverrideLogMu.Lock()
		imfOverrideLoggedAt = original
		imfOverrideLogMu.Unlock()
	})
}

// TestResolveIMFOverridesBestEffortLogsAtMostOncePerInterval: this runs on the positions poll path
// against an endpoint whose GET shape is undocumented, so an unthrottled failure line was roughly
// one line per second per user, forever, into a persistent on-disk log.
func TestResolveIMFOverridesBestEffortLogsAtMostOncePerInterval(t *testing.T) {
	resetIMFOverrideLogThrottle(t)

	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	buf := captureLog(t)

	for i := 0; i < 25; i++ {
		if got := resolveIMFOverridesBestEffort(context.Background(), c.callAPI, "0xWALLET", ""); len(got) != 0 {
			t.Fatalf("got %v, want an empty map on endpoint failure", got)
		}
	}

	lines := strings.Count(buf.String(), "\n")
	if lines != 1 {
		t.Fatalf("logged %d lines for 25 consecutive failures, want exactly 1: %q", lines, buf.String())
	}
}

func TestResolveIMFOverridesBestEffortPassesThroughOnSuccess(t *testing.T) {
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/initialMarginFractionOverride": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, ethOverrideFixture)
		},
	})
	defer server.Close()

	c := newTestFuturesClient(server.URL)

	got := resolveIMFOverridesBestEffort(context.Background(), c.callAPI, "0xWALLET", "ETH-USD")
	if got["ETH-USD"] != "0.20000000" {
		t.Fatalf("got %v, want ETH-USD override 0.20000000", got)
	}
}

// Asserted on the wire: a credential attached to a Public endpoint is only visible in the headers
// that actually left, and Katana rejects a key it cannot validate.
func TestGetInstrumentsInfoSendsNoCredentialsOnThePublicMarketsEndpoint(t *testing.T) {
	var gotAPIKey, gotSignature string
	server := muxServer(t, map[string]http.HandlerFunc{
		"/v1/markets": func(w http.ResponseWriter, r *http.Request) {
			gotAPIKey = r.Header.Get("KP-API-KEY")
			gotSignature = r.Header.Get("KP-HMAC-SIGNATURE")
			writeJSON(t, w, twoMarketsFixture)
		},
	})
	defer server.Close()

	if _, err := newTestFuturesClient(server.URL).NewGetInstrumentsInfo().Do(context.Background()); err != nil {
		t.Fatal(err)
	}

	if gotAPIKey != "" {
		t.Fatalf("KP-API-KEY = %q, want it absent on a public endpoint", gotAPIKey)
	}
	if gotSignature != "" {
		t.Fatalf("KP-HMAC-SIGNATURE = %q, want it absent on a public endpoint", gotSignature)
	}
}
