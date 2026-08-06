package katana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Sumex-io/sumex-tradelib/utils"
)

var (
	tradeName_Spot    = "KATANA_SPOT"
	tradeName_Futures = "KATANA_FUTURES"
)

// Katana has no entry in utils.GetEndpoint because its host is not a constant: production and
// sandbox are two different hosts selected by SetDemo, which also selects the EIP-712 domain the
// wallet signature is produced under (see katanaSigner.domain in request.go). Keeping both ends of
// that pair behind one flag is what makes "signed for sandbox, sent to production" inexpressible.
const (
	katana_prod    = "https://api-perps.katana.network"
	katana_sandbox = "https://api-perps-sandbox.katana.network"
)

func baseURLFor(demo bool) string {
	if demo {
		return katana_sandbox
	}
	return katana_prod
}

// cacheTTL bounds how long a client reuses a resolved wallet address or market catalog.
const cacheTTL = 5 * time.Minute

// ===============SPOT=================

// SpotClient exists only to serve the two account-level actions every exchange in this library is
// expected to answer — account info and the private WebSocket auth token. Katana is perpetuals
// only: it lists no spot markets, so no spot trading action is offered here at all rather than
// quietly falling through to futures behaviour the caller never asked for.
type SpotClient struct {
	apiKey     string
	secretKey  string
	privateKey string
	keyType    string
	demo       bool
	BaseURL    string
	UserAgent  string
	Proxy      string
	BrokerID   string
	Debug      bool
	logger     *log.Logger
	TimeOffset int64

	// Only the wallet is cached here: no SpotClient action reads the market catalog, because
	// Katana lists no spot markets. The cache is per client, so a long-lived client refetches at
	// most once per cacheTTL.
	walletMu       sync.Mutex
	walletCache    string
	walletCachedAt time.Time
}

func (c *SpotClient) SetProxy(proxy string)  { c.Proxy = proxy }
func (c *SpotClient) SetUserAgent(ua string) { c.UserAgent = ua }
func (c *SpotClient) SetDebug(v bool)        { c.Debug = v }
func (c *SpotClient) SetBrokerID(id string)  { c.BrokerID = id }
func (c *SpotClient) SetTimeOffset(ms int64) { c.TimeOffset = ms }

// SetDemo switches the client between Katana's production and sandbox environments. It moves
// BaseURL and the EIP-712 domain together on purpose — see the comment on katana_prod.
func (c *SpotClient) SetDemo(v bool) {
	c.demo = v
	c.BaseURL = baseURLFor(v)
}

func (c *SpotClient) debug(format string, v ...interface{}) {
	if c.Debug {
		c.logger.Printf(format, v...)
	}
}

// NewSpotClient takes the three-credential form used by every exchange whose signing needs a third
// secret (okx, kucoin, bitget). Katana's third credential is the DELEGATED KEY'S PRIVATE KEY: the
// secp256k1 key that produces the EIP-712 wallet signature on every trade action.
func NewSpotClient(apiKey, secretKey, delegatedPrivateKey string) *SpotClient {
	return &SpotClient{
		apiKey:     apiKey,
		secretKey:  secretKey,
		privateKey: delegatedPrivateKey,
		keyType:    utils.KeyTypeHmac,
		BaseURL:    baseURLFor(false),
		UserAgent:  "Onetrades/golang",
		logger:     log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Spot), log.LstdFlags),
	}
}

// signer bundles the only two values EIP-712 signing needs. It is built per call rather than
// stored so the demo flag can never be stale relative to BaseURL.
func (c *SpotClient) signer() *katanaSigner {
	return &katanaSigner{privKeyHex: c.privateKey, demo: c.demo}
}

func (c *SpotClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{
		resolveWallet: c.resolveWallet,
		sign:          c.signer(),
	}
}

func (c *SpotClient) NewSignAuthStream() *signAuthStream {
	return &signAuthStream{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
	}
}

// ===============FUTURES=================

type FuturesClient struct {
	apiKey     string
	secretKey  string
	privateKey string
	keyType    string
	demo       bool
	BaseURL    string
	UserAgent  string
	Proxy      string
	BrokerID   string
	Debug      bool
	logger     *log.Logger
	TimeOffset int64

	marketsMu       sync.Mutex
	marketsCache    map[string]Market
	marketsCachedAt time.Time

	walletMu       sync.Mutex
	walletCache    string
	walletCachedAt time.Time
}

func (c *FuturesClient) SetProxy(proxy string)  { c.Proxy = proxy }
func (c *FuturesClient) SetUserAgent(ua string) { c.UserAgent = ua }
func (c *FuturesClient) SetDebug(v bool)        { c.Debug = v }
func (c *FuturesClient) SetBrokerID(id string)  { c.BrokerID = id }
func (c *FuturesClient) SetTimeOffset(ms int64) { c.TimeOffset = ms }

func (c *FuturesClient) SetDemo(v bool) {
	c.demo = v
	c.BaseURL = baseURLFor(v)
}

func (c *FuturesClient) debug(format string, v ...interface{}) {
	if c.Debug {
		c.logger.Printf(format, v...)
	}
}

func NewFuturesClient(apiKey, secretKey, delegatedPrivateKey string) *FuturesClient {
	return &FuturesClient{
		apiKey:     apiKey,
		secretKey:  secretKey,
		privateKey: delegatedPrivateKey,
		keyType:    utils.KeyTypeHmac,
		BaseURL:    baseURLFor(false),
		UserAgent:  "Onetrades/golang",
		logger:     log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Futures), log.LstdFlags),
	}
}

func (c *FuturesClient) signer() *katanaSigner {
	return &katanaSigner{privKeyHex: c.privateKey, demo: c.demo}
}

func (c *FuturesClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{
		resolveWallet: c.resolveWallet,
		sign:          c.signer(),
	}
}

func (c *FuturesClient) NewSignAuthStream() *signAuthStream {
	return &signAuthStream{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
	}
}

func (c *FuturesClient) NewGetInstrumentsInfo() *futures_getInstrumentsInfo {
	return &futures_getInstrumentsInfo{markets: c.markets}
}

func (c *FuturesClient) NewGetBalance() *futures_getBalance {
	return &futures_getBalance{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
	}
}

func (c *FuturesClient) NewGetPositions() *futures_getPositions {
	return &futures_getPositions{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		markets:       c.markets,
	}
}

func (c *FuturesClient) NewGetOrderList() *futures_getOrderList {
	return &futures_getOrderList{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
	}
}

func (c *FuturesClient) NewOrdersHistory() *futures_ordersHistory {
	return &futures_ordersHistory{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		markets:       c.markets,
	}
}

func (c *FuturesClient) NewPositionsHistory() *futures_positionsHistory {
	return &futures_positionsHistory{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		markets:       c.markets,
	}
}

func (c *FuturesClient) NewUserTrades() *futures_userTrades {
	return &futures_userTrades{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
	}
}

func (c *FuturesClient) NewGetLeverage() *futures_getLeverage {
	return &futures_getLeverage{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		markets:       c.markets,
	}
}

// NewGetPositionMode and NewGetMarginMode answer from Katana's own invariants — one-way,
// cross-margin only — so neither builder is wired to the transport at all.
func (c *FuturesClient) NewGetPositionMode() *futures_getPositionMode {
	return &futures_getPositionMode{}
}

func (c *FuturesClient) NewGetMarginMode() *futures_getMarginMode {
	return &futures_getMarginMode{}
}

func (c *FuturesClient) NewPlaceOrder() *futures_placeOrder {
	return &futures_placeOrder{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		sign:          c.signer(),
		brokerID:      c.BrokerID,
	}
}

func (c *FuturesClient) NewAmendOrder() *futures_amendOrder {
	return &futures_amendOrder{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		sign:          c.signer(),
		brokerID:      c.BrokerID,
	}
}

func (c *FuturesClient) NewCancelOrder() *futures_cancelOrder {
	return &futures_cancelOrder{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		sign:          c.signer(),
	}
}

// NewSetLeverage is Katana's only leverage control. There is no set-leverage endpoint: leverage is
// 1/initialMarginFraction, and the only writable input is an initialMarginFractionOverride that
// can tighten a market's margin requirement but never loosen it below the market's own floor.
func (c *FuturesClient) NewSetLeverage() *futures_setLeverage {
	return &futures_setLeverage{
		callAPI:       c.callAPI,
		resolveWallet: c.resolveWallet,
		markets:       c.markets,
		sign:          c.signer(),
	}
}

// ===============RESOLVERS=================
//
// markets and resolveWallet are the two values almost every Katana action needs before it can
// build a request, and both cost a round trip. They live on the client because the cache does,
// and reach the action builders the way hyperliquid passes `user`: injected by the factory.

// markets fetches the tradable perpetual catalog (GET /v1/markets) and indexes it by symbol. Both
// filters are load-bearing: non-perpetual markets (equities, metals) are out of scope, and a
// non-active market must not ship to the frontend as tradable or its orders come back
// TRADING_DISABLED / MARKET_DEACTIVATED. The cost is that a position or fill on a market that has
// since gone non-active takes effectiveMarket's unknown path — the row is still returned, just
// with an empty leverage.
func (c *FuturesClient) markets(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error) {
	c.marketsMu.Lock()
	if c.marketsCache != nil && time.Since(c.marketsCachedAt) < cacheTTL {
		cached := c.marketsCache
		c.marketsMu.Unlock()
		return cached, nil
	}
	c.marketsMu.Unlock()

	data, _, err := c.callAPI(ctx, marketsRequest(), opts...)
	if err != nil {
		return nil, err
	}

	var raw []Market
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	out := filterTradableMarkets(raw)

	c.marketsMu.Lock()
	c.marketsCache = out
	c.marketsCachedAt = time.Now()
	c.marketsMu.Unlock()

	return out, nil
}

// resolveWallet returns the address to trade from (GET /v1/wallets). A sole wallet is used
// regardless of balance — a new account's only wallet legitimately starts empty, and refusing it
// would lock the user out before they could deposit. The funded filter only disambiguates two or
// more, and still-ambiguous resolution is an error: on a cross-margin exchange, silently picking
// one would place orders against the wrong wallet.
func (c *FuturesClient) resolveWallet(ctx context.Context, opts ...utils.RequestOption) (string, error) {
	c.walletMu.Lock()
	if c.walletCache != "" && time.Since(c.walletCachedAt) < cacheTTL {
		cached := c.walletCache
		c.walletMu.Unlock()
		return cached, nil
	}
	c.walletMu.Unlock()

	r, err := walletsRequest()
	if err != nil {
		return "", err
	}

	data, _, err := c.callAPI(ctx, r, opts...)
	if err != nil {
		return "", err
	}

	var wallets []katanaWallet
	if err := json.Unmarshal(data, &wallets); err != nil {
		return "", err
	}

	wallet, err := pickWallet(wallets)
	if err != nil {
		return "", err
	}
	return c.cacheWallet(wallet), nil
}

func (c *FuturesClient) cacheWallet(wallet string) string {
	c.walletMu.Lock()
	c.walletCache = wallet
	c.walletCachedAt = time.Now()
	c.walletMu.Unlock()
	return wallet
}

func (c *SpotClient) resolveWallet(ctx context.Context, opts ...utils.RequestOption) (string, error) {
	c.walletMu.Lock()
	if c.walletCache != "" && time.Since(c.walletCachedAt) < cacheTTL {
		cached := c.walletCache
		c.walletMu.Unlock()
		return cached, nil
	}
	c.walletMu.Unlock()

	r, err := walletsRequest()
	if err != nil {
		return "", err
	}

	data, _, err := c.callAPI(ctx, r, opts...)
	if err != nil {
		return "", err
	}

	var wallets []katanaWallet
	if err := json.Unmarshal(data, &wallets); err != nil {
		return "", err
	}

	wallet, err := pickWallet(wallets)
	if err != nil {
		return "", err
	}
	return c.cacheWallet(wallet), nil
}

func (c *SpotClient) cacheWallet(wallet string) string {
	c.walletMu.Lock()
	c.walletCache = wallet
	c.walletCachedAt = time.Now()
	c.walletMu.Unlock()
	return wallet
}

// walletsRequest is the request half the twin SpotClient/FuturesClient resolveWallet copies above
// share, so they cannot drift apart on an endpoint or a query parameter.
func marketsRequest() *utils.Request {
	return &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/markets",
		SecType:  utils.SecTypeSigned,
	}
}

func walletsRequest() (*utils.Request, error) {
	nonce, _, err := newNonce()
	if err != nil {
		return nil, err
	}
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/wallets",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":            nonce,
		"includePositions": "false",
	})
	return r, nil
}

func pickWallet(wallets []katanaWallet) (string, error) {
	if len(wallets) == 0 {
		return "", errors.New("katana: no wallet found on this API account")
	}
	if len(wallets) == 1 {
		return wallets[0].Wallet, nil
	}

	var funded []katanaWallet
	for _, w := range wallets {
		if isFundedWallet(w) {
			funded = append(funded, w)
		}
	}

	switch len(funded) {
	case 0:
		return "", errors.New("katana: no funded wallet found on this API account")
	case 1:
		return funded[0].Wallet, nil
	default:
		return "", errors.New("multiple funded Katana wallets on this API account")
	}
}
