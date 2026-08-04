package bullish

import (
	"fmt"
	"log"
	"os"

	"github.com/Betarost/onetrades/utils"
)

var (
	tradeName_Spot    = "BULLISH_SPOT"
	tradeName_Futures = "BULLISH_FUTURES"
)

// ===============SPOT=================

// type SpotClient struct {
// 	apiKey     string
// 	secretKey  string
// 	memo       string
// 	keyType    string
// 	BaseURL    string
// 	UserAgent  string
// 	Debug      bool
// 	logger     *log.Logger
// 	TimeOffset int64
// }

// func (c *SpotClient) debug(format string, v ...interface{}) {
// 	if c.Debug {
// 		c.logger.Printf(format, v...)
// 	}
// }

// func NewSpotClient(apiKey, secretKey, memo string) *SpotClient {
// 	return &SpotClient{
// 		apiKey:    apiKey,
// 		secretKey: secretKey,
// 		memo:      memo,
// 		keyType:   utils.KeyTypeHmac,
// 		BaseURL:   utils.GetEndpoint(tradeName_Spot),
// 		UserAgent: "Onetrades/golang",
// 		logger:    log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Spot), log.LstdFlags),
// 	}
// }

// ===============FUTURES=================

type FuturesClient struct {
	apiKey     string
	secretKey  string
	memo       string
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

func (c *FuturesClient) UpdateJWT(jwt string) {
	c.memo = jwt
}

func NewFuturesClient(apiKey, secretKey, memo string) *FuturesClient {
	return &FuturesClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		memo:      memo,
		keyType:   utils.KeyTypeHmac,
		BaseURL:   utils.GetEndpoint(tradeName_Futures),
		UserAgent: "Onetrades/golang",
		logger:    log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Futures), log.LstdFlags),
	}
}

func (c *FuturesClient) NewGenerateJWT() *generateJWT {
	return &generateJWT{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetBalance() *futures_getBalance {
	return &futures_getBalance{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetInstrumentsInfo() *futures_getInstrumentsInfo {
	return &futures_getInstrumentsInfo{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetMarketCandle() *futures_getMarketCandle {
	return &futures_getMarketCandle{callAPI: c.callAPI}
}

func (c *FuturesClient) NewPlaceOrder() *futures_placeOrder {
	return &futures_placeOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetPositions() *futures_getPositions {
	return &futures_getPositions{callAPI: c.callAPI}
}

func (c *FuturesClient) NewGetOrderList() *futures_getOrderList {
	return &futures_getOrderList{callAPI: c.callAPI}
}

func (c *FuturesClient) NewAmendOrder() *futures_amendOrder {
	return &futures_amendOrder{callAPI: c.callAPI}
}

func (c *FuturesClient) NewCancelOrder() *futures_cancelOrder {
	return &futures_cancelOrder{callAPI: c.callAPI}
}
