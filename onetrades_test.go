package onetrades

import (
	"context"
	"log"
	"testing"

	"github.com/Sumex-io/sumex-tradelib/binance"
	"github.com/Sumex-io/sumex-tradelib/bingx"
	"github.com/Sumex-io/sumex-tradelib/bitget"
	"github.com/Sumex-io/sumex-tradelib/blofin"
	"github.com/Sumex-io/sumex-tradelib/bullish"
	"github.com/Sumex-io/sumex-tradelib/bybit"
	"github.com/Sumex-io/sumex-tradelib/gateio"
	"github.com/Sumex-io/sumex-tradelib/huobi"
	"github.com/Sumex-io/sumex-tradelib/hyperliquid"
	"github.com/Sumex-io/sumex-tradelib/kucoin"
	"github.com/Sumex-io/sumex-tradelib/mexc"
	"github.com/Sumex-io/sumex-tradelib/okx"
	"github.com/Sumex-io/sumex-tradelib/weex"
	"github.com/Sumex-io/sumex-tradelib/whitebit"
	"github.com/spf13/viper"
)

var n = ""

func printAnswers(r interface{}, e error) {
	log.Printf("=%s= %+v %v", n, r, e)
}
func TestOnetrades(t *testing.T) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalln("Error ReadInConfig", err)
	}
	ctx := context.Background()
	//==========================KEYS==========================
	binanceKey := viper.GetString("BINANCE_API")
	binanceSecret := viper.GetString("BINANCE_SECRET")
	bingxKey := viper.GetString("BINGX_API")
	bingxSecret := viper.GetString("BINGX_SECRET")
	bybitKey := viper.GetString("BYBIT_API")
	bybitSecret := viper.GetString("BYBIT_SECRET")
	gateKey := viper.GetString("GATE_API")
	gateSecret := viper.GetString("GATE_SECRET")
	mexcKey := viper.GetString("MEX_API")
	mexcSecret := viper.GetString("MEX_SECRET")
	bitgetKey := viper.GetString("BITGET_API")
	bitgetSecret := viper.GetString("BITGET_SECRET")
	bitgetMemo := viper.GetString("BITGET_MEMO")
	okxKey := viper.GetString("OKX_API")
	okxSecret := viper.GetString("OKX_SECRET")
	okxMemo := viper.GetString("OKX_MEMO")
	huobiKey := viper.GetString("HUOBI_API")
	huobiSecret := viper.GetString("HUOBI_SECRET")
	bullishKey := viper.GetString("BULLISH_API")
	bullishSecret := viper.GetString("BULLISH_SECRET")
	bullishJWT := viper.GetString("BULLISH_JWT")
	kucoinKey := viper.GetString("KUCOIN_API")
	kucoinSecret := viper.GetString("KUCOIN_SECRET")
	kucoinMemo := viper.GetString("KUCOIN_MEMO")
	blofinKey := viper.GetString("BLOFIN_API")
	blofinSecret := viper.GetString("BLOFIN_SECRET")
	blofinMemo := viper.GetString("BLOFIN_MEMO")
	whitebitKey := viper.GetString("WHITEBIT_API")
	whitebitSecret := viper.GetString("WHITEBIT_SECRET")
	hyperliquidKey := viper.GetString("HYPERLIQUID_API")
	hyperliquidSecret := viper.GetString("HYPERLIQUID_SECRET")
	weexKey := viper.GetString("WEEX_API")
	weexSecret := viper.GetString("WEEX_SECRET")
	weexMemo := viper.GetString("WEEX_MEMO")
	//==========================CLIENTS==========================

	binanceSpot := binance.NewSpotClient(binanceKey, binanceSecret)
	binanceFutures := binance.NewFuturesClient(binanceKey, binanceSecret)
	bingxSpot := bingx.NewSpotClient(bingxKey, bingxSecret)
	bingxFutures := bingx.NewFuturesClient(bingxKey, bingxSecret)
	bybitSpot := bybit.NewSpotClient(bybitKey, bybitSecret)
	bybitFutures := bybit.NewFuturesClient(bybitKey, bybitSecret)
	gateioSpot := gateio.NewSpotClient(gateKey, gateSecret)
	gateioFutures := gateio.NewFuturesClient(gateKey, gateSecret)
	mexcSpot := mexc.NewSpotClient(mexcKey, mexcSecret)
	bitgetSpot := bitget.NewSpotClient(bitgetKey, bitgetSecret, bitgetMemo)
	bitgetFutures := bitget.NewFuturesClient(bitgetKey, bitgetSecret, bitgetMemo)
	okxSpot := okx.NewSpotClient(okxKey, okxSecret, okxMemo)
	okxFutures := NewOKXFutures(Credentials{
		APIKey:    okxKey,
		SecretKey: okxSecret,
		Memo:      okxMemo,
	})
	// okxFutures := okx.NewFuturesClient(okxKey, okxSecret, okxMemo)
	huobiSpot := huobi.NewSpotClient(huobiKey, huobiSecret)
	huobiFutures := huobi.NewFuturesClient(huobiKey, huobiSecret)
	bullishFutures := bullish.NewFuturesClient(bullishKey, bullishSecret, bullishJWT)
	kucoinSpot := kucoin.NewSpotClient(kucoinKey, kucoinSecret, kucoinMemo)
	kucoinFutures := kucoin.NewFuturesClient(kucoinKey, kucoinSecret, kucoinMemo)
	blofinSpot := blofin.NewSpotClient(blofinKey, blofinSecret, blofinMemo)
	blofinFutures := blofin.NewFuturesClient(blofinKey, blofinSecret, blofinMemo)
	whitebitSpot := whitebit.NewSpotClient(whitebitKey, whitebitSecret)
	whitebitFutures := whitebit.NewFuturesClient(whitebitKey, whitebitSecret)
	hyperliquidSpot := hyperliquid.NewSpotClient(hyperliquidKey, hyperliquidSecret)
	hyperliquidFutures := hyperliquid.NewFuturesClient(hyperliquidKey, hyperliquidSecret)
	weexSpot := weex.NewSpotClient(weexKey, weexSecret, weexMemo)
	weexFutures := weex.NewFuturesClient(weexKey, weexSecret, weexMemo)

	binanceSpot.Debug = false
	binanceFutures.Debug = false
	bingxSpot.Debug = false
	bingxFutures.Debug = false
	bybitSpot.Debug = false
	bybitFutures.Debug = false
	gateioSpot.Debug = false
	gateioFutures.Debug = false
	mexcSpot.Debug = false
	bitgetSpot.Debug = false
	bitgetFutures.Debug = false
	okxSpot.Debug = false
	okxFutures.Debug = false
	huobiSpot.Debug = false
	huobiFutures.Debug = false
	bullishFutures.Debug = false
	kucoinSpot.Debug = false
	kucoinFutures.Debug = false
	blofinFutures.Debug = false
	whitebitSpot.Debug = false
	whitebitFutures.Debug = false
	hyperliquidSpot.Debug = false
	hyperliquidFutures.Debug = false
	weexSpot.Debug = false
	weexFutures.Debug = false
	bullishFutures.Proxy = "http://localhost:1080"
	bitgetSpot.Proxy = "http://localhost:1080"
	bitgetFutures.Proxy = "http://localhost:8080"
	okxSpot.Proxy = "http://localhost:1080"
	// bingxFutures.Proxy = "http://localhost:1080"
	okxFutures.Proxy = "http://localhost:8080"
	// bybitSpot.Proxy = "http://localhost:1080"
	// bybitFutures.Proxy = "http://localhost:1080"
	binanceSpot.Proxy = "http://localhost:1080"
	// bitgetFutures.Proxy = "http://localhost:1080"
	binanceFutures.Proxy = "http://localhost:8080"
	blofinSpot.Proxy = "http://localhost:1080"
	blofinFutures.Proxy = "http://localhost:8080"
	whitebitSpot.Proxy = "http://localhost:1080"
	whitebitFutures.Proxy = "http://localhost:1080"
	kucoinSpot.Proxy = "http://localhost:1080"
	kucoinFutures.Proxy = "http://localhost:1080"
	// binanceFutures.IsCOINM(true)
	binanceFutures.GateWay = ""
	weexSpot.Proxy = "http://localhost:8080"
	weexFutures.Proxy = "http://localhost:8080"

	// =======================AccountInfo
	n = "AccountInfo"
	//SPOT
	// printAnswers(binanceSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(bingxSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(bybitSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(gateioSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(mexcSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(bitgetSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(okxSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(huobiSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(kucoinSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(whitebitSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(hyperliquidSpot.NewGetAccountInfo().Do(ctx))
	// printAnswers(weexSpot.NewGetAccountInfo().Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(bingxFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(bybitFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(gateioFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(bitgetFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(okxFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(huobiFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(kucoinFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(blofinFutures.NewGetAccountInfo().Do(ctx))
	// printAnswers(whitebitFutures.NewGetAccountInfo().Do(ctx))

	//=======================GetBalance
	n = "GetBalance"
	//SPOT
	// printAnswers(binanceSpot.NewGetBalance().Do(ctx))
	// printAnswers(bingxSpot.NewGetBalance().Do(ctx))
	// printAnswers(bybitSpot.NewGetBalance().Do(ctx))
	// printAnswers(gateioSpot.NewGetBalance().Do(ctx))
	// printAnswers(mexcSpot.NewGetBalance().Do(ctx))
	// printAnswers(bitgetSpot.NewGetBalance().Do(ctx))
	// printAnswers(okxSpot.NewGetBalance().Do(ctx))
	// printAnswers(huobiSpot.NewGetBalance().UID("53799773").Do(ctx))
	// printAnswers(huobiSpot.NewGetBalance().UID("69069265").Do(ctx))
	// printAnswers(kucoinSpot.NewGetBalance().Do(ctx))
	// printAnswers(whitebitSpot.NewGetBalance().Do(ctx))
	// printAnswers(hyperliquidSpot.NewGetBalance().Do(ctx))
	// printAnswers(weexSpot.NewGetBalance().Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewGetBalance().Do(ctx))
	// printAnswers(bingxFutures.NewGetBalance().Do(ctx))
	// printAnswers(bybitFutures.NewGetBalance().Do(ctx))
	// printAnswers(gateioFutures.NewGetBalance().Do(ctx))
	// printAnswers(bitgetFutures.NewGetBalance().Do(ctx))
	// printAnswers(okxFutures.NewGetBalance().Do(ctx))
	// printAnswers(huobiFutures.NewGetBalance().UID("53799773").Do(ctx))
	// printAnswers(bullishFutures.NewGetBalance().UID("111872616831896").Do(ctx))
	// printAnswers(kucoinFutures.NewGetBalance().Do(ctx))
	// printAnswers(blofinFutures.NewGetBalance().Do(ctx))
	// printAnswers(whitebitFutures.NewGetBalance().Do(ctx))
	// printAnswers(hyperliquidFutures.NewGetBalance().Do(ctx))
	// printAnswers(weexFutures.NewGetBalance().Do(ctx))

	//=======================InstrumentsInfo
	n = "InstrumentsInfo"
	//SPOT
	// printAnswers(binanceSpot.NewGetInstrumentsInfo().Symbol("TONUSDT").Do(ctx))
	// printAnswers(bybitSpot.NewGetInstrumentsInfo().Symbol("ETHBTC").Do(ctx))
	// printAnswers(bingxSpot.NewGetInstrumentsInfo().Symbol("TRX-USDT").Do(ctx))
	// printAnswers(gateioSpot.NewGetInstrumentsInfo().Symbol("ATOM_USDT").Do(ctx))
	// printAnswers(mexcSpot.NewGetInstrumentsInfo().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(bitgetSpot.NewGetInstrumentsInfo().Symbol("DOGEUSDT").Do(ctx))
	// printAnswers(okxSpot.NewGetInstrumentsInfo().Symbol("BTC-USDT").Do(ctx))
	// printAnswers(huobiSpot.NewGetInstrumentsInfo().Do(ctx))
	// huobiSpot.NewGetInstrumentsInfo().Do(ctx)
	// printAnswers(kucoinSpot.NewGetInstrumentsInfo().Symbol("BTC-USDT").Do(ctx))
	// printAnswers(whitebitSpot.NewGetInstrumentsInfo().Symbol("BTC_USDT").Do(ctx))
	// printAnswers(hyperliquidSpot.NewGetInstrumentsInfo().Symbol("USOL").Do(ctx))
	// printAnswers(weexSpot.NewGetInstrumentsInfo().Symbol("ATOMUSDT").Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewGetInstrumentsInfo().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(binanceFutures.NewGetInstrumentsInfo().Symbol("SOLUSD_PERP").Do(ctx))
	// printAnswers(bybitFutures.NewGetInstrumentsInfo().Symbol("BTCUSD").Do(ctx))
	// printAnswers(bybitFutures.NewGetInstrumentsInfo().Symbol("BTCUSD").Category("inverse").Do(ctx))
	// printAnswers(bingxFutures.NewGetInstrumentsInfo().Symbol("BTC-USDT").Do(ctx))
	// printAnswers(gateioFutures.NewGetInstrumentsInfo().Symbol("JDON_USDT").Do(ctx))
	// printAnswers(bitgetFutures.NewGetInstrumentsInfo().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(okxFutures.NewGetInstrumentsInfo().Symbol("FARTCOIN-USDT-SWAP").Do(ctx))
	// printAnswers(okxFutures.NewGetInstrumentsInfo().Symbol("BTC-USD-SWAP").Do(ctx))
	// printAnswers(huobiFutures.NewGetInstrumentsInfo().Symbol("BTC-USDT").Do(ctx))
	// printAnswers(bullishFutures.NewGetInstrumentsInfo().Symbol("BTC-USDC-PERP").Do(ctx))
	// printAnswers(kucoinFutures.NewGetInstrumentsInfo().Symbol("DOGEUSDTM").Do(ctx))
	// printAnswers(blofinFutures.NewGetInstrumentsInfo().Symbol("BTC-USDT").Do(ctx))
	// printAnswers(whitebitFutures.NewGetInstrumentsInfo().Symbol("BTC_PERP").Do(ctx))
	// printAnswers(hyperliquidFutures.NewGetInstrumentsInfo().Symbol("SOL").Do(ctx))
	// printAnswers(weexFutures.NewGetInstrumentsInfo().Symbol("ATOMUSDT").Do(ctx))

	//=======================MarketCandle
	//FUTURES
	n = "MarketCandle"
	// printAnswers(bullishFutures.NewGetMarketCandle().Symbol("BTC-USDC-PERP").TimeFrame(entity.TimeFrameType1H).Do(ctx))
	// printAnswers(binanceFutures.NewGetMarketCandle().Symbol("BTCUSD").TimeFrame(entity.TimeFrameType1H).Limit(3).Do(ctx))

	//=======================PlaceOrder
	n = "PlaceOrder"
	//SPOT
	// printAnswers(binanceSpot.NewPlaceOrder().Symbol("BTCUSDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("0.002").Do(ctx))
	// printAnswers(bingxSpot.NewPlaceOrder().Symbol("TRX-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Price("0.2892").Size("10").Do(ctx))
	// printAnswers(bingxSpot.NewPlaceOrder().Symbol("DOGE-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("10").Do(ctx))
	// printAnswers(bybitSpot.NewPlaceOrder().Symbol("BTCUSDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("0.002").Do(ctx))
	// printAnswers(gateioSpot.NewPlaceOrder().Symbol("DOGE_USDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Size("19").Price("0.18250").Do(ctx))
	// printAnswers(mexcSpot.NewPlaceOrder().Symbol("DOGEUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Price("0.17288").Size("10").Do(ctx))
	// printAnswers(bitgetSpot.NewPlaceOrder().Symbol("DOGEUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Price("0.17250").Size("11").Do(ctx))
	// printAnswers(okxSpot.NewPlaceOrder().Symbol("BTC-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("200").Do(ctx))
	// printAnswers(okxSpot.NewPlaceOrder().Symbol("DOGE-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("10").Do(ctx))
	// printAnswers(huobiSpot.NewPlaceOrder().Symbol("DOGEUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Size("56").Price("0.20899").UID("69069265").Do(ctx))
	// printAnswers(kucoinSpot.NewPlaceOrder().Symbol("BTC-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Price("116910.5").Size("0.00001").ClientOrderID("3").Do(ctx))
	// printAnswers(whitebitSpot.NewPlaceOrder().Symbol("BTC_USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeLimit).Size("0.00006").Price("92100.5").Do(ctx))
	// printAnswers(hyperliquidSpot.NewPlaceOrder().Symbol("10156").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Size("0.15").Price("91").ClientOrderID("123").Do(ctx))
	// printAnswers(weexSpot.NewPlaceOrder().Symbol("SOLUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Size("0.12").Price("91.55").Do(ctx))

	//FUTURES
	//TEST MARKET
	// printAnswers(binanceFutures.NewPlaceOrder().Symbol("BTCUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Price("69950").PositionSide(entity.PositionSideTypeLong).Size("0.002").Do(ctx))
	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("SOLUSDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeShort).Size("0.1").Price("82.82").HedgeMode(true).TpOrder(true).Do(ctx))

	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("SOLUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeShort).Size("0.1").HedgeMode(true).Do(ctx))
	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("BTCUSD").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).Size("100").HedgeMode(false).Category("inverse").Do(ctx))
	// printAnswers(okxFutures.NewPlaceOrder().Symbol("BTC-USDT-SWAP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).Size("2283").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(okxFutures.NewPlaceOrder().Symbol("BTC-USD-SWAP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Size("2280").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(okxFutures.NewPlaceOrder().Symbol("DOGE-USD-SWAP").Side(entity.SideTypeSell).PositionSide(entity.PositionSideTypeShort).OrderType(entity.OrderTypeMarket).Size("1").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(blofinFutures.NewPlaceOrder().Symbol("BTC-USDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Size("0.2").MarginMode(entity.MarginModeTypeCross).PositionSide(entity.PositionSideTypeLong).Do(ctx))
	// printAnswers(whitebitFutures.NewPlaceOrder().Symbol("BTC_PERP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Size("0.001").PositionSide(entity.PositionSideTypeShort).Do(ctx))

	//TEST LIMIT
	// printAnswers(binanceFutures.NewPlaceOrder().Symbol("BTCUSDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeLimit).PositionSide(entity.PositionSideTypeLong).Price("67200").Size("0.002").Do(ctx))
	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("BTCUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).PositionSide(entity.PositionSideTypeLong).Size("100").HedgeMode(true).Price("124200").Do(ctx))
	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("BTCUSD").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).PositionSide(entity.PositionSideTypeLong).Size("100").HedgeMode(false).Category("inverse").Price("125000").Do(ctx))
	// printAnswers(okxFutures.NewPlaceOrder().Symbol("BTC-USDT-SWAP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Price("125000").PositionSide(entity.PositionSideTypeLong).Size("0.05").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(blofinFutures.NewPlaceOrder().Symbol("BTC-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeLimit).Price("90555.5").Size("0.2").MarginMode(entity.MarginModeTypeCross).PositionSide(entity.PositionSideTypeLong).Do(ctx))
	// printAnswers(whitebitFutures.NewPlaceOrder().Symbol("BTC_PERP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Size("0.001").Price("91550.5").PositionSide(entity.PositionSideTypeShort).Do(ctx))
	//==========

	// printAnswers(binanceFutures.NewPlaceOrder().Symbol("BTCUSD_PERP").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).Size("1").Do(ctx))
	// printAnswers(binanceFutures.NewPlaceOrder().Symbol("DOGEUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Size("35").Do(ctx))
	// printAnswers(bingxFutures.NewPlaceOrder().Symbol("TRX-USDT").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Price("0.28550").Size("10").HedgeMode(false).Do(ctx))

	// printAnswers(bingxFutures.NewPlaceOrder().Symbol("DOGE-USDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Size("20").Price("0.077").HedgeMode(true).PositionSide(entity.PositionSideTypeLong).SlOrder(true).Do(ctx))

	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("BTCUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).Size("0.1").HedgeMode(true).Do(ctx))
	// printAnswers(bybitFutures.NewPlaceOrder().Symbol("BTCUSD").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("100").HedgeMode(true).Category("inverse").Do(ctx))
	// printAnswers(gateioFutures.NewPlaceOrder().Symbol("DOGE_USDT").Side(entity.SideTypeSell).Size("50").Price("0.085").HedgeMode(true).PositionSide(entity.PositionSideTypeLong).OrderType(entity.OrderTypeLimit).SlOrder(true).Do(ctx))

	// printAnswers(bitgetFutures.NewPlaceOrder().Symbol("DOGEUSDT").Side(entity.SideTypeSell).PositionSide(entity.PositionSideTypeLong).OrderType(entity.OrderTypeMarket).Price("0.089").Size("60").HedgeMode(true).MarginMode(entity.MarginModeTypeCross).TpOrder(true).Do(ctx))
	// printAnswers(bitgetFutures.NewPlaceOrder().Symbol("DOGEUSDT").Side(entity.SideTypeSell).PositionSide(entity.PositionSideTypeLong).OrderType(entity.OrderTypeMarket).Price("0.08").Size("10").HedgeMode(true).MarginMode(entity.MarginModeTypeCross).SlOrder(true).Do(ctx))

	// printAnswers(okxFutures.NewPlaceOrder().Symbol("BTC-USDT-SWAP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).Price("118230").PositionSide(entity.PositionSideTypeLong).Size("0.01").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(okxFutures.NewPlaceOrder().Symbol("BTC-USD-SWAP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).Size("1").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(bullishFutures.NewPlaceOrder().UID("111872616831896").Symbol("BTC-USDC-PERP").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeLimit).Size("0.001").Price("111750").ClientOrderID("test123").Do(ctx))
	// printAnswers(bullishFutures.NewPlaceOrder().UID("111872616831896").Symbol("BTC-USDC-PERP").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Size("0.00750000").Do(ctx))

	// printAnswers(kucoinFutures.NewPlaceOrder().Symbol("DOGEUSDTM").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).PositionSide(entity.PositionSideTypeLong).Price("0.085").Size("2").MarginMode(entity.MarginModeTypeCross).MarginMode(entity.MarginModeTypeCross).Leverage("75").ClientOrderID("tt14").SlOrder(true).Do(ctx))
	// printAnswers(kucoinFutures.NewPlaceOrder().Symbol("ETHUSDTM").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).Size("1").MarginMode(entity.MarginModeTypeCross).ClientOrderID("4").MarginMode(entity.MarginModeTypeCross).Leverage("75").Do(ctx))

	// printAnswers(hyperliquidFutures.NewPlaceOrder().Symbol("5").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).Size("0.2").Price("80.445").SlOrder(true).Do(ctx))
	// printAnswers(hyperliquidFutures.NewPlaceOrder().Symbol("5").Side(entity.SideTypeBuy).OrderType(entity.OrderTypeMarket).Size("0.2").Do(ctx))

	// printAnswers(weexFutures.NewPlaceOrder().Symbol("SOLUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeShort).ClientOrderID("test1").Size("0.1").Do(ctx))
	// printAnswers(weexFutures.NewPlaceOrder().Symbol("SOLUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeMarket).PositionSide(entity.PositionSideTypeLong).ClientOrderID("test1").Size("0.1").Price("86.7").TpOrder(true).Do(ctx))
	// printAnswers(weexFutures.NewPlaceOrder().Symbol("SOLUSDT").Side(entity.SideTypeSell).OrderType(entity.OrderTypeLimit).PositionSide(entity.PositionSideTypeLong).ClientOrderID("test2").Size("0.1").Price("84.84").Do(ctx))

	//=======================GetPositions
	n = "GetPositions"
	//FUTURES
	// printAnswers(binanceFutures.NewGetPositions().Do(ctx))

	// printAnswers(bingxFutures.NewGetPositions().Do(ctx))
	// printAnswers(bybitFutures.NewGetPositions().Do(ctx))
	// printAnswers(bybitFutures.NewGetPositions().Category("inverse").Do(ctx))
	// printAnswers(gateioFutures.NewGetPositions().Do(ctx))
	// printAnswers(bitgetFutures.NewGetPositions().Do(ctx))
	// printAnswers(okxFutures.NewGetPositions().Do(ctx))
	// printAnswers(bullishFutures.NewGetPositions().UID("111872616831896").Do(ctx))
	// printAnswers(kucoinFutures.NewGetPositions().Do(ctx))
	// printAnswers(blofinFutures.NewGetPositions().Do(ctx))
	// printAnswers(whitebitFutures.NewGetPositions().Do(ctx))
	// printAnswers(hyperliquidFutures.NewGetPositions().Do(ctx))
	// printAnswers(weexFutures.NewGetPositions().Do(ctx))

	//=======================CancelOrder
	n = "CancelOrder"
	//SPOT
	// printAnswers(binanceSpot.NewCancelOrder().Symbol("DOGEUSDT").OrderID("10968968313").Do(ctx))
	// printAnswers(bingxSpot.NewCancelOrder().Symbol("TRX-USDT").OrderID("1942168112220504064").Do(ctx))
	// printAnswers(bybitSpot.NewCancelOrder().OrderID("1993102065488195328").Do(ctx))
	// printAnswers(gateioSpot.NewCancelOrder().Symbol("DOGE_USDT").OrderID("872370092892").Do(ctx))
	// printAnswers(mexcSpot.NewCancelOrder().Symbol("DOGEUSDT").OrderID("C02__571234536180977664120").Do(ctx))
	// printAnswers(bitgetSpot.NewCancelOrder().Symbol("DOGEUSDT").OrderID("1326467275447353344").Do(ctx))
	// printAnswers(okxSpot.NewCancelOrder().Symbol("DOGE-USDT").OrderID("2669952276160045056").Do(ctx))
	// printAnswers(huobiSpot.NewCancelOrder().OrderID("1371171052466066").Do(ctx))
	// printAnswers(kucoinSpot.NewCancelOrder().OrderID("68cc0554a4aad100075896a5").Do(ctx))
	// printAnswers(whitebitSpot.NewCancelOrder().Symbol("BTC_USDT").OrderID("1981703663512").Do(ctx))
	// printAnswers(hyperliquidSpot.NewCancelOrder().Symbol("10156").OrderID("345212875305").Do(ctx))
	// printAnswers(weexSpot.NewCancelOrder().OrderID("731217364401521319").Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewCancelOrder().Symbol("BTCUSDT").OrderID("934062304663").Do(ctx))
	// printAnswers(bingxFutures.NewCancelOrder().Symbol("DOGE-USDT").OrderID("2028437663832969216").Do(ctx))
	// printAnswers(bybitFutures.NewCancelOrder().Symbol("DOGEUSDT").OrderID("0b4308fa-058b-4a26-9477-1d95ef29fd57").Do(ctx))
	// printAnswers(gateioFutures.NewCancelOrder().OrderID("2028505823805702144").IsTpSl(true).Do(ctx))
	// printAnswers(bitgetFutures.NewCancelOrder().Symbol("DOGEUSDT").OrderID("1326805187463618574").Do(ctx))
	// printAnswers(bitgetFutures.NewCancelOrder().Symbol("DOGEUSDT").OrderID("GiveError").IsSl(true).Do(ctx))
	// printAnswers(okxFutures.NewCancelOrder().Symbol("DOGE-USDT-SWAP").OrderID("2672372971099906048").Do(ctx))
	// printAnswers(okxFutures.NewCancelOrder().Symbol("XRP-USDT-SWAP").OrderID("3345313022222200832").IsTpSl(true).Do(ctx))
	// printAnswers(bullishFutures.NewCancelOrder().UID("111872616831896").Symbol("BTC-USDC-PERP").OrderID("870221330058314753").Do(ctx))
	// printAnswers(kucoinFutures.NewCancelOrder().OrderID("417970584002015232").IsTpSl(true).Do(ctx))
	// printAnswers(blofinFutures.NewCancelOrder().OrderID("5000054382203").Do(ctx))
	// printAnswers(whitebitFutures.NewCancelOrder().Symbol("BTC_PERP").OrderID("1979248560459").Do(ctx))
	// printAnswers(weexFutures.NewCancelOrder().OrderID("733404581681168807").Do(ctx))
	// printAnswers(weexFutures.NewCancelOrder().OrderID("733475053085131175").IsTpSl(true).Do(ctx))

	//=======================AmendOrder
	n = "AmendOrder"
	//SPOT
	// printAnswers(binanceSpot.NewAmendOrder().Symbol("TONUSDT").OrderID("957322613").NewSize("2.85").NewPrice("2.89").Do(ctx))
	// printAnswers(bybitSpot.NewAmendOrder().Symbol("DOGEUSDT").OrderID("1993102065488195328").NewPrice("0.19855").NewSize("31").Do(ctx))
	// printAnswers(gateioSpot.NewAmendOrder().Symbol("TRX_USDT").OrderID("186336434877910305").NewSize("3").NewPrice("0.2314").Do(ctx))
	// printAnswers(okxSpot.NewAmendOrder().Symbol("DOGE-USDT").OrderID("2669952276160045056").NewPrice("0.17101").NewSize("11.11").Do(ctx))
	// printAnswers(whitebitSpot.NewAmendOrder().Symbol("BTC_USDT").OrderID("1981688045654").NewPrice("91910.7").NewSize("0.00006").Do(ctx))
	// printAnswers(hyperliquidSpot.NewAmendOrder().Symbol("10156").OrderID("345210167974").Side(entity.SideTypeSell).NewPrice("90.05").NewSize("0.17").ClientOrderID("test1").Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewAmendOrder().Symbol("DOGEUSDT").OrderID("76780426418").NewSize("39").NewPrice("0.23522").Side(entity.SideTypeBuy).Do(ctx))
	// printAnswers(gateioFutures.NewAmendOrder().OrderID("186336434877910305").NewSize("3").NewPrice("0.2314").Do(ctx))
	// printAnswers(bybitFutures.NewAmendOrder().Symbol("DOGEUSDT").OrderID("0c88fc3c-7896-4b62-aacc-b5c9c52122cd").NewPrice("0.19897").NewSize("31").Do(ctx))
	// printAnswers(bitgetFutures.NewAmendOrder().Symbol("BNBUSDT").OrderID("1335231994933182473").NewSize("500").NewPrice("0.17").NewСlientOrderID("12345few").Do(ctx))
	// printAnswers(okxFutures.NewAmendOrder().Symbol("DOGE-USDT-SWAP").OrderID("2672372971099906048").NewSize("0.02").NewPrice("0.1787").Do(ctx))
	// printAnswers(bullishFutures.NewAmendOrder().UID("111872616831896").Symbol("BTC-USDC-PERP").OrderID("869492653091717121").NewSize("0.0012").NewPrice("118820").Do(ctx))
	// printAnswers(blofinFutures.NewAmendOrder().Symbol("BTC-USDT").OrderID("5000054376599").NewSize("0.2").NewPrice("90999.99").Do(ctx)) // не работает

	//=======================OrderList
	n = "OrderList"
	//SPOT
	// printAnswers(binanceSpot.NewGetOrderList().Do(ctx))
	// printAnswers(bingxSpot.NewGetOrderList().Do(ctx))
	// printAnswers(bybitSpot.NewGetOrderList().Do(ctx))
	// printAnswers(gateioSpot.NewGetOrderList().Do(ctx))
	// printAnswers(mexcSpot.NewGetOrderList().Do(ctx))
	// printAnswers(bitgetSpot.NewGetOrderList().Do(ctx))
	// printAnswers(okxSpot.NewGetOrderList().Do(ctx))
	// printAnswers(huobiSpot.NewGetOrderList().Do(ctx))
	// printAnswers(kucoinSpot.NewGetOrderList().Do(ctx))
	// printAnswers(whitebitSpot.NewGetOrderList().Do(ctx))
	// printAnswers(hyperliquidSpot.NewGetOrderList().Limit(5).Do(ctx))
	// printAnswers(weexSpot.NewGetOrderList().Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewGetOrderList().Do(ctx))
	// printAnswers(bingxFutures.NewGetOrderList().Do(ctx))
	// printAnswers(bybitFutures.NewGetOrderList().Category("inverse").Do(ctx))
	// printAnswers(bybitFutures.NewGetOrderList().Do(ctx))
	// printAnswers(gateioFutures.NewGetOrderList().Do(ctx))
	// printAnswers(mexcSpot.NewGetOrderList().Symbol("MXUSDT").Do(ctx))
	// printAnswers(bitgetFutures.NewGetOrderList().Do(ctx))
	// printAnswers(okxFutures.NewGetOrderList().Do(ctx))
	// printAnswers(bullishFutures.NewGetOrderList().UID("111872616831896").Symbol("BTC-USDC-PERP").Do(ctx))
	// printAnswers(bullishFutures.NewGetOrderList().UID("111872616831896").Do(ctx))
	// printAnswers(kucoinFutures.NewGetOrderList().Do(ctx))
	// printAnswers(blofinFutures.NewGetOrderList().Do(ctx))
	// printAnswers(whitebitFutures.NewGetOrderList().Do(ctx))
	// printAnswers(hyperliquidFutures.NewGetOrderList().Do(ctx))
	// printAnswers(weexFutures.NewGetOrderList().Do(ctx))

	//=======================OrdersHistory
	n = "OrdersHistory"
	//SPOT
	// printAnswers(binanceSpot.NewOrdersHistory().Symbol("DOGEUSDT").Do(ctx))
	// printAnswers(bingxSpot.NewOrdersHistory().Do(ctx))
	// printAnswers(bybitSpot.NewOrdersHistory().Do(ctx))
	// printAnswers(gateioSpot.NewOrdersHistory().StartTime((time.Now().UnixMilli() - (60 * 60 * 24 * 1000)) / 1000).EndTime(time.Now().UnixMilli() / 1000).Do(ctx))
	// printAnswers(mexcSpot.NewOrdersHistory().Symbol("DOGEUSDT").Do(ctx))
	// printAnswers(bitgetSpot.NewOrdersHistory().Symbol("XRPUSDT").Do(ctx))
	// printAnswers(okxSpot.NewOrdersHistory().Do(ctx))
	// printAnswers(huobiSpot.NewOrdersHistory().Do(ctx))
	// printAnswers(kucoinSpot.NewOrdersHistory().Limit(10).Do(ctx))
	// printAnswers(whitebitSpot.NewOrdersHistory().Do(ctx))
	// printAnswers(hyperliquidSpot.NewOrdersHistory().Do(ctx))
	// printAnswers(weexSpot.NewOrdersHistory().Symbol("SOLUSDT").Do(ctx))

	//FUTURES
	// printAnswers(binanceFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(binanceFutures.NewOrdersHistory().OrderID("8389765814458701896").Limit(2).Do(ctx))
	// printAnswers(bingxFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(bybitFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(bybitFutures.NewOrdersHistory().OrderID("fddec6d3-a8e7-4f52-a732-0832c034be36").Limit(2).Do(ctx))
	// printAnswers(gateioFutures.NewOrdersHistory().Limit(3).Do(ctx))
	// printAnswers(bitgetFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(okxFutures.NewOrdersHistory().OrderID("2896365455043928064").Limit(2).Do(ctx))
	// printAnswers(okxFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(kucoinFutures.NewOrdersHistory().Limit(50).StartTime(1758362413105).EndTime(1758382413105).Do(ctx))
	// printAnswers(kucoinFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(blofinFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(whitebitFutures.NewOrdersHistory().Do(ctx))
	// printAnswers(hyperliquidFutures.NewOrdersHistory().StartTime(1773221695300).EndTime(1773221695500).Symbol("SOL").Do(ctx))
	// printAnswers(weexFutures.NewOrdersHistory().Do(ctx))

	//=======================ExecutionsHistory
	n = "ExecutionsHistory"
	// printAnswers(bybitFutures.NewExecutionsHistory().Limit(50).Do(ctx))

	//=======================PositionsHistory
	n = "PositionsHistory"
	//FUTURES
	// printAnswers(binanceFutures.NewPositionsHistory().Limit(1).Do(ctx))
	// printAnswers(bingxFutures.NewPositionsHistory().Symbol("1000PEPE-USDT").StartTime(time.Now().UnixMilli() - (60 * 60 * 24 * 90 * 1000)).EndTime(time.Now().UnixMilli()).Do(ctx))
	// printAnswers(bingxFutures.NewPositionsHistory().Symbol("TRX-USDT").StartTime(time.Now().UnixMilli() - (60 * 60 * 24 * 1000)).EndTime(time.Now().UnixMilli()).Do(ctx))
	// printAnswers(bybitFutures.NewPositionsHistory().Limit(1).Do(ctx))
	// printAnswers(gateioFutures.NewPositionsHistory().Do(ctx))
	// printAnswers(bitgetFutures.NewPositionsHistory().Symbol("RAVEUSDT").Do(ctx))
	// printAnswers(okxFutures.NewPositionsHistory().Symbol("BNB-USDT-SWAP").Do(ctx))
	// printAnswers(okxFutures.NewPositionsHistory().Do(ctx))
	// printAnswers(kucoinFutures.NewPositionsHistory().Do(ctx))
	// printAnswers(blofinFutures.NewPositionsHistory().Do(ctx))
	// printAnswers(whitebitFutures.NewPositionsHistory().Do(ctx))
	// printAnswers(hyperliquidFutures.NewPositionsHistory().Do(ctx))
	// printAnswers(weexFutures.NewPositionsHistory().Do(ctx))

	//=======================UserTrades
	n = "UserTrades"
	//FUTURES
	// printAnswers(weexFutures.NewUserTrades().Do(ctx))
	printAnswers(hyperliquidSpot.NewUserTrades().Do(ctx))
	// printAnswers(hyperliquidFutures.NewUserTrades().Do(ctx))

	//=======================SetPositionMode
	n = "SetPositionMode"
	//FUTURES
	// printAnswers(binanceFutures.NewSetPositionMode().Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(bingxFutures.NewSetPositionMode().Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(bybitFutures.NewSetPositionMode().Symbol("BTCUSD").Category("inverse").Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(gateioFutures.NewSetPositionMode().Mode(entity.PositionModeTypeOneWay).Do(ctx))
	// printAnswers(bitgetFutures.NewSetPositionMode().Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(okxFutures.NewSetPositionMode().Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(kucoinFutures.NewSetPositionMode().Mode(entity.PositionModeTypeOneWay).Do(ctx))
	// printAnswers(blofinFutures.NewSetPositionMode().Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(whitebitFutures.NewSetPositionMode().Mode(entity.PositionModeTypeHedge).Do(ctx))
	// printAnswers(weexFutures.NewSetPositionMode().Symbol("BTCUSDT").MarginMode(entity.MarginModeTypeCross).Mode(entity.PositionModeTypeOneWay).Do(ctx))

	//=======================GetPositionMode
	n = "GetPositionMode"
	//FUTURES
	// printAnswers(binanceFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(bingxFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(bybitFutures.NewGetPositionMode().Symbol("BTCUSD").Category("inverse").Do(ctx))
	// printAnswers(bybitFutures.NewGetPositionMode().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(gateioFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(bitgetFutures.NewGetPositionMode().Symbol("BNBUSDT").Do(ctx))
	// printAnswers(bitgetFutures.NewGetPositionMode().Symbol("DOGEUSDT").Do(ctx))
	// printAnswers(okxFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(kucoinFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(blofinFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(whitebitFutures.NewGetPositionMode().Do(ctx))
	// printAnswers(weexFutures.NewGetPositionMode().Symbol("SOLUSDT").Do(ctx))

	//=======================SetLeverage
	n = "SetLeverage"
	//FUTURES
	// printAnswers(binanceFutures.NewSetLeverage().Symbol("BTCUSDT").Leverage("5").Do(ctx))
	// printAnswers(bingxFutures.NewSetLeverage().Symbol("ATOM-USDT").Leverage("17").PositionSide(entity.PositionSideTypeLong).Do(ctx))
	// printAnswers(bybitFutures.NewSetLeverage().Symbol("BTCUSDT").LongLeverage("13").ShortLeverage("14").Do(ctx))
	// printAnswers(bybitFutures.NewSetLeverage().Symbol("BTCUSD").Category("inverse").Leverage("100").Do(ctx))
	// printAnswers(gateioFutures.NewSetLeverage().Symbol("DOGE_USDT").Leverage("30").Do(ctx))
	// printAnswers(gateioFutures.NewSetLeverage().Symbol("ANKR_USDT").Leverage("10").MarginMode(entity.MarginModeTypeIsolated).Do(ctx))
	// printAnswers(bitgetFutures.NewSetLeverage().Symbol("ATOMUSDT").Leverage("15").PositionSide(entity.PositionSideTypeLong).Do(ctx))
	// printAnswers(bitgetFutures.NewSetLeverage().Symbol("ATOMUSDT").LongLeverage("12").ShortLeverage("10").Do(ctx))
	// printAnswers(okxFutures.NewSetLeverage().Symbol("BTC-USDT-SWAP").Leverage("100").PositionSide(entity.PositionSideTypeShort).MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(kucoinFutures.NewSetLeverage().Symbol("DOGEUSDTM").Leverage("5").MarginMode(entity.MarginModeTypeIsolated).Do(ctx))
	// printAnswers(blofinFutures.NewSetLeverage().Symbol("BTC-USDT").Leverage("10").PositionSide(entity.PositionSideTypeBoth).MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(whitebitFutures.NewSetLeverage().Leverage("50").Do(ctx))
	// printAnswers(hyperliquidFutures.NewSetLeverage().Leverage("5").MarginMode(entity.MarginModeTypeIsolated).Symbol("10").Do(ctx))
	// printAnswers(weexFutures.NewSetLeverage().Leverage("25").MarginMode(entity.MarginModeTypeCross).Symbol("DOGEUSDT").Do(ctx))

	//=======================GetLeverage
	n = "GetLeverage"
	//FUTURES
	// printAnswers(binanceFutures.NewGetLeverage().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(bingxFutures.NewGetLeverage().Symbol("DOGE-USDT").Do(ctx))
	// printAnswers(bybitFutures.NewGetLeverage().Symbol("BTCUSD").Category("inverse").Do(ctx))
	// printAnswers(bybitFutures.NewGetLeverage().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(gateioFutures.NewGetLeverage().Symbol("ANKR_USDT").MarginMode(entity.MarginModeTypeIsolated).Do(ctx))
	// printAnswers(bitgetFutures.NewGetLeverage().Symbol("ATOMUSDT").Do(ctx))
	// printAnswers(okxFutures.NewGetLeverage().Symbol("BTC-USDT-SWAP").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(kucoinFutures.NewGetLeverage().Symbol("DOGEUSDTM").Do(ctx))
	// printAnswers(blofinFutures.NewGetLeverage().Symbol("BTC-USDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(whitebitFutures.NewGetLeverage().Do(ctx))
	// printAnswers(hyperliquidFutures.NewGetLeverage().Symbol("SOL/USDC").Do(ctx))
	// printAnswers(weexFutures.NewGetLeverage().Symbol("DOGEUSDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))

	//=======================SetMarginMode
	n = "SetMarginMode"
	//FUTURES
	// printAnswers(binanceFutures.NewSetMarginMode().Symbol("BTCUSDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(bingxFutures.NewSetMarginMode().Symbol("ATOM-USDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(bybitFutures.NewSetMarginMode().MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(gateioFutures.NewSetMarginMode().Symbol("DOGE_USDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(bitgetFutures.NewSetMarginMode().Symbol("DOGEUSDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(kucoinFutures.NewSetMarginMode().Symbol("DOGEUSDTM").MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(blofinFutures.NewSetMarginMode().MarginMode(entity.MarginModeTypeCross).Do(ctx))
	// printAnswers(weexFutures.NewSetMarginMode().Symbol("DOGEUSDT").MarginMode(entity.MarginModeTypeCross).Do(ctx))

	//=======================GetMarginMode
	n = "GetMarginMode"
	//FUTURES
	// printAnswers(binanceFutures.NewGetMarginMode().Symbol("BTCUSDT").Do(ctx))
	// printAnswers(bingxFutures.NewGetMarginMode().Symbol("ATOM-USDT").Do(ctx))
	// printAnswers(bybitFutures.NewGetMarginMode().Do(ctx))
	// printAnswers(gateioFutures.NewGetMarginMode().Symbol("DOGE_USDT").Do(ctx))
	// printAnswers(bitgetFutures.NewGetMarginMode().Symbol("ATOMUSDT").Do(ctx))
	// printAnswers(kucoinFutures.NewGetMarginMode().Symbol("DOGEUSDTM").Do(ctx))
	// printAnswers(blofinFutures.NewGetMarginMode().Do(ctx))
	// printAnswers(weexFutures.NewGetMarginMode().Symbol("DOGEUSDT").Do(ctx))

	//=======================GetListenKey
	n = "GetListenKey"
	//SPOT
	// printAnswers(bingxSpot.NewGetListenKey().Do(ctx))
	// printAnswers(mexcSpot.NewGetListenKey().Do(ctx))
	// printAnswers(kucoinSpot.NewGetListenKey().Do(ctx))

	//FUTURES
	// printAnswers(bingxFutures.NewGetListenKey().Do(ctx))
	// printAnswers(kucoinFutures.NewGetListenKey().Do(ctx))
	// printAnswers(whitebitFutures.NewGetListenKey().Do(ctx))

	//=======================ExtendListenKey
	n = "ExtendListenKey"
	//SPOT
	// printAnswers(bingxSpot.NewExtendListenKey().ListenKey("0e81716a1c67128521a48f066800b24420a14194b6637e43968d75e3a7293fce").Do(ctx))
	// printAnswers(mexcSpot.NewExtendListenKey().ListenKey("802935b2c04071cff608424de8cb0416585ab71783e9c932ea748f73c6efe223").Do(ctx))

	//FUTURES
	// printAnswers(bingxFutures.NewExtendListenKey().ListenKey("0e81716a1c67128521a48f066800b24420a14194b6637e43968d75e3a7293fce").Do(ctx))
	// printAnswers(bingxFutures.NewExtendListenKey().Do(ctx))

	//=======================SignAuthStream
	n = "SignAuthStream"
	// printAnswers(bybitSpot.NewSignAuthStream().TimeStamp(1753350824193).Do(ctx))
	// printAnswers(gateioSpot.NewSignAuthStream().TimeStamp(1753352964).Channel("futures.orders").Event("subscribe").Do(ctx))
	// printAnswers(bitgetSpot.NewSignAuthStream().TimeStamp(1753356762807).Do(ctx))
	// printAnswers(okxSpot.NewSignAuthStream().TimeStamp(1753356762807).Do(ctx))
	// printAnswers(blofinFutures.NewSignAuthStream().TimeStamp(1753356762807).Do(ctx))
	// printAnswers(weexSpot.NewSignAuthStream().TimeStamp(1753356762807).Do(ctx))

	//=======================GenerateJWT
	n = "GenerateJWT"
	// printAnswers(bullishFutures.NewGenerateJWT().Do(ctx))

	//===========Not Exit
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// for {
	// 	<-c
	// 	return
	// }
}
