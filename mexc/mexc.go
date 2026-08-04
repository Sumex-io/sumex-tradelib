package mexc

import (
	"fmt"
	"log"
	"os"

	"github.com/Betarost/onetrades/utils"
)

var (
	tradeName_Spot    = "MEXC_SPOT"
	tradeName_Futures = "MEXC_FUTURES"
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

func (c *SpotClient) NewGetAccountInfo() *getAccountInfo {
	return &getAccountInfo{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetBalance() *spot_getBalance {
	return &spot_getBalance{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetInstrumentsInfo() *spot_getInstrumentsInfo {
	return &spot_getInstrumentsInfo{callAPI: c.callAPI}
}

func (c *SpotClient) NewPlaceOrder() *spot_placeOrder {
	return &spot_placeOrder{callAPI: c.callAPI}
}

func (c *SpotClient) NewCancelOrder() *spot_cancelOrder {
	return &spot_cancelOrder{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetOrderList() *spot_getOrderList {
	return &spot_getOrderList{callAPI: c.callAPI}
}

func (c *SpotClient) NewOrdersHistory() *spot_ordersHistory {
	return &spot_ordersHistory{callAPI: c.callAPI}
}

func (c *SpotClient) NewGetListenKey() *spot_getListenKey {
	return &spot_getListenKey{callAPI: c.callAPI}
}

func (c *SpotClient) NewExtendListenKey() *spot_extendListenKey {
	return &spot_extendListenKey{callAPI: c.callAPI}
}

// ===============FUTURES=================

// type FuturesClient struct {
// 	apiKey     string
// 	secretKey  string
// 	keyType    string
// 	BaseURL    string
// 	UserAgent  string
// 	Debug      bool
// 	logger     *log.Logger
// 	TimeOffset int64
// }

// func (c *FuturesClient) debug(format string, v ...interface{}) {
// 	if c.Debug {
// 		c.logger.Printf(format, v...)
// 	}
// }

// func NewFuturesClient(apiKey, secretKey string) *FuturesClient {
// 	return &FuturesClient{
// 		apiKey:    apiKey,
// 		secretKey: secretKey,
// 		keyType:   utils.KeyTypeHmac,
// 		BaseURL:   utils.GetEndpoint(tradeName_Futures),
// 		UserAgent: "Onetrades/golang",
// 		logger:    log.New(os.Stderr, fmt.Sprintf("%s-onetrades ", tradeName_Futures), log.LstdFlags),
// 	}
// }

// func (c *FuturesClient) NewGetInstrumentsInfo() *futures_getInstrumentsInfo {
// 	return &futures_getInstrumentsInfo{callAPI: c.callAPI}
// }

// func (c *FuturesClient) NewGetAccountInfo() *getAccountInfo {
// 	return &getAccountInfo{callAPI: c.callAPI}
// }

// func (c *FuturesClient) NewGetFundingAccountBalance() *getFundingAccountBalance {
// 	return &getFundingAccountBalance{callAPI: c.callAPI}
// }
