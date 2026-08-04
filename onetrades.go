package onetrades

import (
	"github.com/Betarost/onetrades/binance"
	"github.com/Betarost/onetrades/bingx"
	"github.com/Betarost/onetrades/bitget"
	"github.com/Betarost/onetrades/bullish"
	"github.com/Betarost/onetrades/bybit"
	"github.com/Betarost/onetrades/gateio"
	"github.com/Betarost/onetrades/huobi"
	"github.com/Betarost/onetrades/kucoin"
	"github.com/Betarost/onetrades/mexc"
	"github.com/Betarost/onetrades/okx"
)

// Общие креды для всех бирж
type Credentials struct {
	APIKey    string
	SecretKey string
	Memo      string
}

// Общие опции
type Options struct {
	COINM      bool   // для фьючерсов у поддерживаемых бирж
	Proxy      string // http(s)://host:port
	UserAgent  string // кастомный UA
	Debug      bool
	BrokerID   string
	TimeOffset int64
}

type HasProxy interface{ SetProxy(string) }
type HasUserAgent interface{ SetUserAgent(string) }
type HasDebug interface{ SetDebug(bool) }
type HasBrokerID interface{ SetBrokerID(string) }
type HasTimeOffset interface{ SetTimeOffset(int64) }
type HasCOINM interface{ SetCOINM(bool) }

// ===================== BINANCE =====================

func NewBinanceSpot(cred Credentials, opts ...Options) *binance.SpotClient {
	o := pickOpts(opts...)
	c := binance.NewSpotClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

func NewBinanceFutures(cred Credentials, opts ...Options) *binance.FuturesClient {
	o := pickOpts(opts...)
	c := binance.NewFuturesClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

// ===================== BYBIT =====================

func NewBybitSpot(cred Credentials, opts ...Options) *bybit.SpotClient {
	o := pickOpts(opts...)
	c := bybit.NewSpotClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}
func NewBybitFutures(cred Credentials, opts ...Options) *bybit.FuturesClient {
	o := pickOpts(opts...)
	c := bybit.NewFuturesClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

// ===================== OKX =====================

func NewOKXSpot(cred Credentials, opts ...Options) *okx.SpotClient {
	o := pickOpts(opts...)
	c := okx.NewSpotClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}
func NewOKXFutures(cred Credentials, opts ...Options) *okx.FuturesClient {
	o := pickOpts(opts...)
	c := okx.NewFuturesClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}

// ===================== BINGX =====================

func NewBingXSpot(cred Credentials, opts ...Options) *bingx.SpotClient {
	o := pickOpts(opts...)
	c := bingx.NewSpotClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

func NewBingXFutures(cred Credentials, opts ...Options) *bingx.FuturesClient {
	o := pickOpts(opts...)
	c := bingx.NewFuturesClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

// ===================== BITGET =====================

func NewBitgetSpot(cred Credentials, opts ...Options) *bitget.SpotClient {
	o := pickOpts(opts...)
	c := bitget.NewSpotClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}

func NewBitgetFutures(cred Credentials, opts ...Options) *bitget.FuturesClient {
	o := pickOpts(opts...)
	c := bitget.NewFuturesClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}

// ===================== BULLISH =====================

func NewBullishFutures(cred Credentials, opts ...Options) *bullish.FuturesClient {
	o := pickOpts(opts...)
	c := bullish.NewFuturesClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}

// ===================== GATE.IO =====================

func NewGateIOSpot(cred Credentials, opts ...Options) *gateio.SpotClient {
	o := pickOpts(opts...)
	c := gateio.NewSpotClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

func NewGateIOFutures(cred Credentials, opts ...Options) *gateio.FuturesClient {
	o := pickOpts(opts...)
	c := gateio.NewFuturesClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

// ===================== HUOBI =====================

func NewHuobiSpot(cred Credentials, opts ...Options) *huobi.SpotClient {
	o := pickOpts(opts...)
	c := huobi.NewSpotClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

func NewHuobiFutures(cred Credentials, opts ...Options) *huobi.FuturesClient {
	o := pickOpts(opts...)
	c := huobi.NewFuturesClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

// ===================== KUCOIN =====================

func NewKucoinSpot(cred Credentials, opts ...Options) *kucoin.SpotClient {
	o := pickOpts(opts...)
	c := kucoin.NewSpotClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}

func NewKucoinFutures(cred Credentials, opts ...Options) *kucoin.FuturesClient {
	o := pickOpts(opts...)
	c := kucoin.NewFuturesClient(cred.APIKey, cred.SecretKey, cred.Memo)
	applyOptions(c, o)
	return c
}

// ===================== MEXC =====================

func NewMEXCSpot(cred Credentials, opts ...Options) *mexc.SpotClient {
	o := pickOpts(opts...)
	c := mexc.NewSpotClient(cred.APIKey, cred.SecretKey)
	applyOptions(c, o)
	return c
}

// ===================== HELPERS =====================

func pickOpts(opts ...Options) Options {
	if len(opts) > 0 {
		return opts[0]
	}
	return Options{}
}

func applyOptions(cli any, opts Options) {
	if v, ok := cli.(HasProxy); ok && opts.Proxy != "" {
		v.SetProxy(opts.Proxy)
	}
	if v, ok := cli.(HasUserAgent); ok && opts.UserAgent != "" {
		v.SetUserAgent(opts.UserAgent)
	}
	if v, ok := cli.(HasDebug); ok {
		v.SetDebug(opts.Debug)
	}
	if v, ok := cli.(HasBrokerID); ok && opts.BrokerID != "" {
		v.SetBrokerID(opts.BrokerID)
	}
	if v, ok := cli.(HasTimeOffset); ok && opts.TimeOffset != 0 {
		v.SetTimeOffset(opts.TimeOffset)
	}
	if v, ok := cli.(HasCOINM); ok {
		v.SetCOINM(opts.COINM)
	}
}
