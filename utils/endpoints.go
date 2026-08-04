package utils

var (
	Binance_spot          = "https://api.binance.com"
	Binance_futures       = "https://fapi.binance.com"
	Binance_futures_coinm = "https://dapi.binance.com"
	Bingx_spot            = "https://open-api.bingx.com"
	Bingx_futures         = "https://open-api.bingx.com"
	Bybit_spot            = "https://api.bybit.com"
	Bybit_futures         = "https://api.bybit.com"
	Gateio_spot           = "https://api.gateio.ws"
	Gateio_futures        = "https://api.gateio.ws"
	Mexc_spot             = "https://api.mexc.com"
	Mexc_futures          = "https://contract.mexc.com"
	Bitget_spot           = "https://api.bitget.com"
	Bitget_futures        = "https://api.bitget.com"
	Okx_spot              = "https://www.okx.com"
	Okx_futures           = "https://www.okx.com"
	Huobi_spot            = "https://api.huobi.pro"
	Huobi_futures         = "https://api.hbdm.com"
	Bullish_spot          = "https://api.exchange.bullish.com"
	Bullish_futures       = "https://api.exchange.bullish.com"
	Kucoin_spot           = "https://api.kucoin.com"
	Kucoin_futures        = "https://api-futures.kucoin.com"
	Blofin_spot           = "https://openapi.blofin.com"
	Blofin_futures        = "https://openapi.blofin.com"
	Whitebit_spot         = "https://whitebit.com"
	Whitebit_futures      = "https://whitebit.com"
	Hyperliquid_spot      = "https://api.hyperliquid.xyz"
	Hyperliquid_futures   = "https://api.hyperliquid.xyz"
	Weex_spot             = "https://api-spot.weex.com"
	Weex_futures          = "https://api-contract.weex.com"
	//================
	BinanceFutureApiMainUrl = "https://fapi.binance.com"
	BybitFutureApiMainUrl   = "https://api.bybit.com"
	MexcFutureApiMainUrl    = "https://futures.mexc.com"
	BingxFutureApiMainUrl   = "https://open-api.bingx.com"
	GateFutureApiMainUrl    = "https://api.gateio.ws"
	BitgetFutureApiMainUrl  = "https://api.bitget.com"
	OKXFutureApiMainUrl     = "https://www.okx.com"
	HuobiFutureApiMainUrl   = "https://api.hbdm.com"

	OKXFutureWsPublicUrl  = "wss://ws.okx.com:8443/ws/v5/public"
	OKXFutureWsPrivateUrl = "wss://ws.okx.com:8443/ws/v5/private"
)

func GetEndpoint(trade string) string {
	switch trade {
	case "BINANCE_SPOT":
		return Binance_spot
	case "BINANCE_FUTURES":
		return Binance_futures
	case "BINANCE_FUTURES_COINM":
		return Binance_futures_coinm
	case "BINGX_SPOT":
		return Bingx_spot
	case "BINGX_FUTURES":
		return Bingx_futures
	case "BYBIT_SPOT":
		return Bybit_spot
	case "BYBIT_FUTURES":
		return Bybit_futures
	case "GATEIO_SPOT":
		return Gateio_spot
	case "GATEIO_FUTURES":
		return Gateio_futures
	case "MEXC_SPOT":
		return Mexc_spot
	case "MEXC_FUTURES":
		return Mexc_futures
	case "BITGET_SPOT":
		return Bitget_spot
	case "BITGET_FUTURES":
		return Bitget_futures
	case "OKX_SPOT":
		return Okx_spot
	case "OKX_FUTURES":
		return Okx_futures
	case "HUOBI_SPOT":
		return Huobi_spot
	case "HUOBI_FUTURES":
		return Huobi_futures
	case "BULLISH_SPOT":
		return Bullish_spot
	case "BULLISH_FUTURES":
		return Bullish_futures
	case "KUCOIN_SPOT":
		return Kucoin_spot
	case "KUCOIN_FUTURES":
		return Kucoin_futures
	case "BLOFIN_SPOT":
		return Blofin_spot
	case "BLOFIN_FUTURES":
		return Blofin_futures
	case "WHITEBIT_SPOT":
		return Whitebit_spot
	case "WHITEBIT_FUTURES":
		return Whitebit_futures
	case "HYPERLIQUID_SPOT":
		return Hyperliquid_spot
	case "HYPERLIQUID_FUTURES":
		return Hyperliquid_futures
	case "WEEX_SPOT":
		return Weex_spot
	case "WEEX_FUTURES":
		return Weex_futures
	default:
		return ""
	}
}
func GetApiEndpoint(trade string) string {
	switch trade {
	case "BINANCE":
		return BinanceFutureApiMainUrl
	case "BYBIT":
		return BybitFutureApiMainUrl
	case "MEXC":
		return MexcFutureApiMainUrl
	case "BINGX":
		return BingxFutureApiMainUrl
	case "GATE":
		return GateFutureApiMainUrl
	case "BITGET":
		return BitgetFutureApiMainUrl
	case "OKX":
		return OKXFutureApiMainUrl
	case "HUOBI":
		return HuobiFutureApiMainUrl
	default:
		return ""
	}
}

func GetApiEndpointOption(trade string) string {
	switch trade {
	case "OKX":
		return OKXFutureApiMainUrl
	default:
		return ""
	}
}

func GetWsPublicEndpoint(trade string) string {
	switch trade {
	case "OKX":
		return OKXFutureWsPublicUrl
	default:
		return ""
	}
}

func GetWsPrivateEndpoint(trade string) string {
	switch trade {
	case "OKX":
		return OKXFutureWsPrivateUrl
	default:
		return ""
	}
}
