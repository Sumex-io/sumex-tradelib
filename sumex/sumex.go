// Package sumex is the connector for Sumex's own perpetuals venue: perp-api, a service in front of
// Orderly Network. It speaks perp-api's REST contract, not Orderly's directly — perp-api holds the
// Orderly keys and exposes a per-user ed25519 API key pair, which is what this connector signs
// with (see request.go).
package sumex

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	tradeName_Spot    = "SUMEX_SPOT"
	tradeName_Futures = "SUMEX_FUTURES"
)

// perp-api has no public constant host: it is a Sumex deployment whose address differs per
// environment (and may be cluster-internal), so the embedding service supplies both at start-up.
// The demo host is a separate perp-api deployment wired to Orderly testnet — perp-api itself has no
// demo flag, the network is fixed per deployment.
var (
	hostsMu     sync.RWMutex
	prodBaseURL string
	demoBaseURL string
)

// SetBaseURLs configures the production and demo perp-api hosts. Either may be empty; a client
// selecting an unconfigured host fails every call with errHostNotConfigured instead of guessing.
func SetBaseURLs(prod, demo string) {
	hostsMu.Lock()
	defer hostsMu.Unlock()
	prodBaseURL = strings.TrimRight(strings.TrimSpace(prod), "/")
	demoBaseURL = strings.TrimRight(strings.TrimSpace(demo), "/")
}

func baseURLFor(demo bool) string {
	hostsMu.RLock()
	defer hostsMu.RUnlock()
	if demo {
		return demoBaseURL
	}
	return prodBaseURL
}

var errHostNotConfigured = errors.New("sumex: no perp-api host is configured for this environment")

// ===============SPOT=================

// SpotClient exists only to serve the two account-level actions every exchange in this library is
// expected to answer. perp-api lists perpetuals only, so no spot trading action is offered.
type SpotClient struct {
	transport
}

func NewSpotClient(apiKey, secretKey string) *SpotClient {
	return &SpotClient{transport: newTransport(apiKey, secretKey, tradeName_Spot)}
}

func (c *SpotClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{callAPI: c.callAPI}
}

func (c *SpotClient) NewSignAuthStream() *signAuthStream {
	return &signAuthStream{callAPI: c.callAPI}
}

// ===============FUTURES=================

type FuturesClient struct {
	transport
}

func NewFuturesClient(apiKey, secretKey string) *FuturesClient {
	return &FuturesClient{transport: newTransport(apiKey, secretKey, tradeName_Futures)}
}

func (c *FuturesClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSignAuthStream() *signAuthStream {
	return &signAuthStream{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetInstrumentsInfo() *futures_getInstrumentsInfo {
	return &futures_getInstrumentsInfo{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetBalance() *futures_getBalance {
	return &futures_getBalance{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetPositions() *futures_getPositions {
	return &futures_getPositions{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetOrderList() *futures_getOrderList {
	return &futures_getOrderList{callAPI: c.callAPI}
}

func (c *FuturesClient) NewOrdersHistory() *futures_ordersHistory {
	return &futures_ordersHistory{callAPI: c.callAPI}
}

func (c *FuturesClient) NewPositionsHistory() *futures_positionsHistory {
	return &futures_positionsHistory{callAPI: c.callAPI}
}

func (c *FuturesClient) NewUserTrades() *futures_userTrades {
	return &futures_userTrades{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetLeverage() *futures_getLeverage {
	return &futures_getLeverage{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSetLeverage() *futures_setLeverage {
	return &futures_setLeverage{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetMarginMode() *futures_getMarginMode {
	return &futures_getMarginMode{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSetMarginMode() *futures_setMarginMode {
	return &futures_setMarginMode{callAPI: c.callAPI}
}

// NewGetPositionMode answers from an invariant: Orderly nets one position per symbol, so there is
// no hedge mode to read and no set-position-mode builder at all.
func (c *FuturesClient) NewGetPositionMode() *futures_getPositionMode {
	return &futures_getPositionMode{}
}

func (c *FuturesClient) NewPlaceOrder() *futures_placeOrder {
	return &futures_placeOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewCancelOrder() *futures_cancelOrder {
	return &futures_cancelOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewAmendOrder() *futures_amendOrder {
	return &futures_amendOrder{callAPI: c.callAPI}
}

// ===============SHARED CLIENT STATE=================

type transport struct {
	apiKey     string
	secretKey  string
	demo       bool
	BaseURL    string
	UserAgent  string
	Proxy      string
	BrokerID   string
	Debug      bool
	TimeOffset int64
	logger     *log.Logger
}

func newTransport(apiKey, secretKey, name string) transport {
	return transport{
		apiKey:    strings.TrimSpace(apiKey),
		secretKey: strings.TrimSpace(secretKey),
		BaseURL:   baseURLFor(false),
		UserAgent: "Onetrades/golang",
		logger:    log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", name), log.LstdFlags),
	}
}

func (c *transport) SetProxy(proxy string)  { c.Proxy = proxy }
func (c *transport) SetUserAgent(ua string) { c.UserAgent = ua }
func (c *transport) SetDebug(v bool)        { c.Debug = v }
func (c *transport) SetTimeOffset(ms int64) { c.TimeOffset = ms }

// SetBrokerID is accepted for parity with the other connectors and ignored: the Orderly broker id
// is fixed by the perp-api deployment, not per request.
func (c *transport) SetBrokerID(id string) { c.BrokerID = id }

// SetDemo switches between the production and demo perp-api hosts configured by SetBaseURLs.
func (c *transport) SetDemo(v bool) {
	c.demo = v
	c.BaseURL = baseURLFor(v)
}

func (c *transport) debug(format string, v ...interface{}) {
	if c.Debug {
		c.logger.Printf(format, v...)
	}
}

// callAPIFunc is how every action builder reaches the transport, so tests can swap it out.
type callAPIFunc func(ctx context.Context, r *request) ([]byte, error)
