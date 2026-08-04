package bybit

import (
	"fmt"
	"log"
	"os"

	"github.com/Betarost/onetrades/utils"
)

var (
	tradeName_Spot    = "BYBIT_SPOT"
	tradeName_Futures = "BYBIT_FUTURES"
)

// ===============SPOT=================

type SpotClient struct {
	apiKey     string
	secretKey  string
	keyType    string
	BaseURL    string
	UserAgent  string
	Proxy      string
	BrokerID   string
	Debug      bool
	logger     *log.Logger
	TimeOffset int64
}

func (c *SpotClient) SetProxy(proxy string)  { c.Proxy = proxy }
func (c *SpotClient) SetUserAgent(ua string) { c.UserAgent = ua }
func (c *SpotClient) SetDebug(v bool)        { c.Debug = v }
func (c *SpotClient) SetBrokerID(id string)  { c.BrokerID = id }
func (c *SpotClient) SetTimeOffset(ms int64) { c.TimeOffset = ms }

func (c *SpotClient) debug(format string, v ...interface{}) {
	if c.Debug {
		c.logger.Printf(format, v...)
	}
}

func NewSpotClient(apiKey, secretKey string) *SpotClient {
	return &SpotClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		keyType:   utils.KeyTypeHmac,
		BaseURL:   utils.GetEndpoint(tradeName_Spot),
		UserAgent: "Onetrades/golang",
		logger:    log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Spot), log.LstdFlags),
	}
}

func (c *SpotClient) NewGetBalance() *spot_getBalance {
	return &spot_getBalance{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetInstrumentsInfo() *spot_getInstrumentsInfo {
	return &spot_getInstrumentsInfo{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{callAPI: c.callAPI}
}

func (c *SpotClient) NewOrdersHistory() *spot_ordersHistory {
	return &spot_ordersHistory{callAPI: c.callAPI}
}

func (c *SpotClient) NewPlaceOrder() *spot_placeOrder {
	return &spot_placeOrder{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetOrderList() *spot_getOrderList {
	return &spot_getOrderList{callAPI: c.callAPI}
}

func (c *SpotClient) NewCancelOrder() *spot_cancelOrder {
	return &spot_cancelOrder{callAPI: c.callAPI}
}

func (c *SpotClient) NewAmendOrder() *spot_amendOrder {
	return &spot_amendOrder{callAPI: c.callAPI}
}

func (c *SpotClient) NewSignAuthStream() *signAuthStream {
	return &signAuthStream{sec: c.secretKey}
}

// ===============FUTURES=================

type FuturesClient struct {
	apiKey     string
	secretKey  string
	keyType    string
	BaseURL    string
	UserAgent  string
	Proxy      string
	BrokerID   string
	Debug      bool
	logger     *log.Logger
	TimeOffset int64
}

func (c *FuturesClient) SetProxy(proxy string)  { c.Proxy = proxy }
func (c *FuturesClient) SetUserAgent(ua string) { c.UserAgent = ua }
func (c *FuturesClient) SetDebug(v bool)        { c.Debug = v }
func (c *FuturesClient) SetBrokerID(id string)  { c.BrokerID = id }
func (c *FuturesClient) SetTimeOffset(ms int64) { c.TimeOffset = ms }

// func (c *FuturesClient) SetCOINM(is bool) { c.IsCOINM(is) }

func (c *FuturesClient) debug(format string, v ...interface{}) {
	if c.Debug {
		c.logger.Printf(format, v...)
	}
}

func NewFuturesClient(apiKey, secretKey string) *FuturesClient {
	return &FuturesClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		keyType:   utils.KeyTypeHmac,
		BaseURL:   utils.GetEndpoint(tradeName_Futures),
		UserAgent: "Onetrades/golang",
		logger:    log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Futures), log.LstdFlags),
	}
}

func (c *FuturesClient) NewGetInstrumentsInfo() *futures_getInstrumentsInfo {
	return &futures_getInstrumentsInfo{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetBalance() *futures_getBalance {
	return &futures_getBalance{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetPositionMode() *futures_getPositionMode {
	return &futures_getPositionMode{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSetPositionMode() *futures_setPositionMode {
	return &futures_setPositionMode{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetLeverage() *futures_getLeverage {
	return &futures_getLeverage{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSetLeverage() *futures_setLeverage {
	return &futures_setLeverage{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSetMarginMode() *futures_setMarginMode {
	return &futures_setMarginMode{callAPI: c.callAPI}
}

func (c *FuturesClient) NewPositionsHistory() *futures_positionsHistory {
	return &futures_positionsHistory{callAPI: c.callAPI}
}

func (c *FuturesClient) NewOrdersHistory() *futures_ordersHistory {
	return &futures_ordersHistory{callAPI: c.callAPI}
}

func (c *FuturesClient) NewExecutionsHistory() *futures_executionsHistory {
	return &futures_executionsHistory{callAPI: c.callAPI}
}

func (c *FuturesClient) NewPlaceOrder() *futures_placeOrder {
	return &futures_placeOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetOrderList() *futures_getOrderList {
	return &futures_getOrderList{callAPI: c.callAPI}
}

func (c *FuturesClient) NewCancelOrder() *futures_cancelOrder {
	return &futures_cancelOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewAmendOrder() *futures_amendOrder {
	return &futures_amendOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetPositions() *futures_getPositions {
	return &futures_getPositions{callAPI: c.callAPI}
}

func (c *FuturesClient) NewSignAuthStream() *signAuthStream {
	return &signAuthStream{sec: c.secretKey}
}

func (c *FuturesClient) NewGetMarginMode() *futures_getMarginMode {
	return &futures_getMarginMode{callAPI: c.callAPI}
}
