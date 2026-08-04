package katana

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const marketsFixture = `[
	{
		"market": "ETH-USD",
		"type": "perpetual",
		"status": "active",
		"baseAsset": "ETH",
		"quoteAsset": "USD",
		"stepSize": "0.00000100",
		"tickSize": "0.01000000",
		"indexPrice": "2236.32000000",
		"minimumPositionSize": "0.01000000",
		"maximumPositionSize": "500.00000000",
		"initialMarginFraction": "0.05000000",
		"maintenanceMarginFraction": "0.03000000",
		"basePositionSize": "25.00000000",
		"incrementalPositionSize": "5.00000000",
		"incrementalInitialMarginFraction": "0.01000000",
		"makerFeeRate": "-0.00010000",
		"takerFeeRate": "0.00040000"
	},
	{
		"market": "BTC-USD",
		"type": "perpetual",
		"status": "active",
		"baseAsset": "BTC",
		"quoteAsset": "USD",
		"stepSize": "0.00001000",
		"tickSize": "0.10000000",
		"indexPrice": "60000.00000000",
		"minimumPositionSize": "0.001000000",
		"maximumPositionSize": "50.00000000",
		"initialMarginFraction": "0.02000000",
		"maintenanceMarginFraction": "0.01000000",
		"basePositionSize": "5.00000000",
		"incrementalPositionSize": "1.00000000",
		"incrementalInitialMarginFraction": "0.01000000",
		"makerFeeRate": "-0.00010000",
		"takerFeeRate": "0.00040000"
	},
	{
		"market": "GOLD-USD",
		"type": "intermittent",
		"status": "active",
		"baseAsset": "GOLD",
		"quoteAsset": "USD"
	}
]`

func TestMarketsFiltersToPerpetualAndCaches(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(marketsFixture))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.markets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d markets, want 2 (intermittent market must be filtered out)", len(got))
	}
	eth, ok := got["ETH-USD"]
	if !ok {
		t.Fatal("ETH-USD missing from markets map")
	}
	if eth.InitialMarginFraction != "0.05000000" {
		t.Fatalf("ETH-USD InitialMarginFraction = %s, want 0.05000000", eth.InitialMarginFraction)
	}
	if _, ok := got["GOLD-USD"]; ok {
		t.Fatal("GOLD-USD is intermittent and must not appear in markets()")
	}
	if hits != 1 {
		t.Fatalf("server hit %d times for the first call, want 1", hits)
	}

	if _, err := c.markets(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("server hit %d times after a second call within the cache TTL, want 1 (cache hit)", hits)
	}

	// Backdate the cache past its TTL and confirm a refetch happens.
	c.marketsMu.Lock()
	c.marketsCachedAt = time.Now().Add(-(cacheTTL + time.Minute))
	c.marketsMu.Unlock()

	if _, err := c.markets(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("server hit %d times after the cache expired, want 2 (refetch)", hits)
	}
}

// haltedMarketFixture holds a perpetual market whose status is anything other than "active". Such a
// market must not be offered as tradable, or its orders come back TRADING_DISABLED /
// MARKET_DEACTIVATED and the user only finds out at order time.
const haltedMarketFixture = `[
	{
		"market": "ETH-USD",
		"type": "perpetual",
		"status": "active",
		"baseAsset": "ETH",
		"quoteAsset": "USD",
		"initialMarginFraction": "0.05000000"
	},
	{
		"market": "DEAD-USD",
		"type": "perpetual",
		"status": "inactive",
		"baseAsset": "DEAD",
		"quoteAsset": "USD",
		"initialMarginFraction": "0.05000000"
	}
]`

func TestMarketsFiltersOutNonActiveMarkets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(haltedMarketFixture))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.markets(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d markets, want 1 (the non-active perpetual must be filtered out)", len(got))
	}
	if _, ok := got["ETH-USD"]; !ok {
		t.Fatal("ETH-USD is active and must be present")
	}
	if _, ok := got["DEAD-USD"]; ok {
		t.Fatal("DEAD-USD has status \"inactive\" and must not be offered as a tradable market")
	}
}

// TestInstrumentsInfoOmitsNonActiveMarkets is the same guard seen from what actually ships: a
// halted market must not reach the instrument list at all, rather than reaching it with a state
// field nothing downstream reads. The source asserted this through getInstrumentsInfo; that action
// belongs to the per-action pass, so the same path is exercised through the two pieces of it this
// foundation owns.
func TestInstrumentsInfoOmitsNonActiveMarkets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(haltedMarketFixture))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	mkts, err := c.markets(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	convert := futures_converts{}
	got := convert.convertInstrumentsInfo(mkts)
	if len(got) != 1 {
		t.Fatalf("convertInstrumentsInfo returned %d instruments, want 1", len(got))
	}
	if got[0].Symbol != "ETH-USD" {
		t.Fatalf("instrument = %q, want ETH-USD (the only active market)", got[0].Symbol)
	}
	if got[0].State != "LIVE" {
		t.Fatalf("state = %q, want \"LIVE\" — the platform-wide active-instrument sentinel consumers gate on, not Katana's own \"online\" vocabulary", got[0].State)
	}
}

func walletsFixture(entries string) string {
	return "[" + entries + "]"
}

const fundedWalletJSON = `{"wallet":"0xFUNDED","equity":"1000.00000000","freeCollateral":"900.00000000","quoteBalance":"950.00000000","unrealizedPnL":"50.00000000"}`
const emptyWalletJSON = `{"wallet":"0xEMPTY","equity":"0.00000000","freeCollateral":"0.00000000","quoteBalance":"0.00000000","unrealizedPnL":"0.00000000"}`

func TestResolveWalletReturnsTheSingleFundedWalletAndCaches(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(walletsFixture(fundedWalletJSON + "," + emptyWalletJSON)))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.resolveWallet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "0xFUNDED" {
		t.Fatalf("resolveWallet = %s, want 0xFUNDED (the only wallet with a positive quoteBalance)", got)
	}
	if hits != 1 {
		t.Fatalf("server hit %d times, want 1", hits)
	}

	if _, err := c.resolveWallet(context.Background()); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("server hit %d times after a second call within the cache TTL, want 1 (cache hit)", hits)
	}
}

func TestResolveWalletErrorsOnMultipleFundedWallets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(walletsFixture(fundedWalletJSON + "," + `{"wallet":"0xOTHER","equity":"500.00000000","freeCollateral":"400.00000000","quoteBalance":"450.00000000","unrealizedPnL":"0.00000000"}`)))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	_, err := c.resolveWallet(context.Background())
	if err == nil {
		t.Fatal("expected an error when more than one wallet holds collateral")
	}
	if err.Error() != "multiple funded Katana wallets on this API account" {
		t.Fatalf("err = %q, want the exact brief-mandated message", err.Error())
	}
}

// Two empty wallets: the funded filter cannot disambiguate between them, so this must still error.
// (A *single* empty wallet is a different case — see TestResolveWalletSingleWalletIsUsedEvenIfEmpty.)
func TestResolveWalletErrorsOnNoFundedWalletsAmongMultiple(t *testing.T) {
	otherEmptyWalletJSON := `{"wallet":"0xEMPTY2","equity":"0.00000000","freeCollateral":"0.00000000","quoteBalance":"0.00000000","unrealizedPnL":"0.00000000"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(walletsFixture(emptyWalletJSON + "," + otherEmptyWalletJSON)))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	if _, err := c.resolveWallet(context.Background()); err == nil {
		t.Fatal("expected an error when no wallet among several holds collateral")
	}
}

// A brand-new account's only wallet legitimately starts out empty. resolveWallet must still resolve
// it rather than erroring the connector unusable.
func TestResolveWalletSingleWalletIsUsedEvenIfEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(walletsFixture(emptyWalletJSON)))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.resolveWallet(context.Background())
	if err != nil {
		t.Fatalf("expected the single wallet to be used even though it holds no collateral, got error: %v", err)
	}
	if got != "0xEMPTY" {
		t.Fatalf("resolveWallet = %s, want 0xEMPTY (the account's only wallet)", got)
	}
}

// An unparseable balance among 2+ wallets must not abort resolution for the whole account — it is
// simply treated as not-funded.
func TestResolveWalletTreatsUnparseableBalanceAsNotFundedNotAsError(t *testing.T) {
	badWalletJSON := `{"wallet":"0xBAD","equity":"0.00000000","freeCollateral":"0.00000000","quoteBalance":"not-a-number","unrealizedPnL":"0.00000000"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(walletsFixture(fundedWalletJSON + "," + badWalletJSON)))
	}))
	defer server.Close()

	c := NewFuturesClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.resolveWallet(context.Background())
	if err != nil {
		t.Fatalf("an unparseable balance on a wallet other than the funded one must not error, got: %v", err)
	}
	if got != "0xFUNDED" {
		t.Fatalf("resolveWallet = %s, want 0xFUNDED", got)
	}
}

// TestSpotClientResolvesWalletTheSameWayAsFutures guards the one duplication this layout forces:
// SpotClient and FuturesClient each carry their own copy of the resolver and its cache, the way
// hyperliquid's two clients each carry their own callAPI.
func TestSpotClientResolvesWalletTheSameWayAsFutures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(walletsFixture(fundedWalletJSON + "," + emptyWalletJSON)))
	}))
	defer server.Close()

	c := NewSpotClient("k", "s", "")
	c.BaseURL = server.URL

	got, err := c.resolveWallet(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != "0xFUNDED" {
		t.Fatalf("SpotClient.resolveWallet = %s, want 0xFUNDED", got)
	}
}

func TestIsFundedWalletTreatsUnparseableBalanceAsNotFunded(t *testing.T) {
	if isFundedWallet(katanaWallet{Wallet: "0xBAD", Quantity: "not-a-number"}) {
		t.Fatal("an unparseable quoteBalance must be treated as not-funded, not funded")
	}
	if isFundedWallet(katanaWallet{Wallet: "0xEMPTY", Quantity: "0.00000000"}) {
		t.Fatal("a zero quoteBalance must be treated as not-funded")
	}
	if !isFundedWallet(katanaWallet{Wallet: "0xFUNDED", Quantity: "1.00000000"}) {
		t.Fatal("a positive quoteBalance must be treated as funded")
	}
}
