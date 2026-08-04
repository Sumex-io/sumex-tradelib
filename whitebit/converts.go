package whitebit

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type account_converts struct{}

// convertAccountInfo: WhiteBit не даёт UID / permissions у ключа,
// поэтому заполняем только то, что можем гарантировать.
func (c *account_converts) convertAccountInfo(_ whitebitAccountSummary) (out entity.AccountInformation) {
	// UID/Label/IP — данных нет, оставляем пустыми
	out.UID = ""
	out.Label = ""
	out.IP = ""

	// Раз запрос к collateral-account/summary прошёл — ключ
	// как минимум умеет читать и торговать фьючерсы.
	out.CanRead = true
	out.CanTrade = true

	// Про переводы данных по API информации нет — ставим false.
	out.CanTransfer = false

	// Мы сейчас работаем именно с collateral / futures:
	out.PermSpot = true
	out.PermFutures = true

	return out
}

type futures_converts struct{}

func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		state := "DISABLED"
		if item.TradesEnabled {
			state = "LIVE"
		}

		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         item.Name,
			Base:           item.Stock,
			Quote:          item.Money,
			MinQty:         item.MinAmount,
			MinNotional:    item.MinTotal,
			PricePrecision: item.MoneyPrec,
			SizePrecision:  item.StockPrec,
			State:          strings.ToUpper(state),

			MaxLeverage:  item.MaxLeverage,
			Multiplier:   "",
			ContractSize: "",

			// WhiteBIT: размер указываем в base (BTC), не в контрактах
			IsSizeContract: false,
		})
	}

	return out
}

func (c *futures_converts) convertBalance(in futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for asset, bal := range in {
		if utils.StringToFloat(bal) == 0 {
			continue
		}

		out = append(out, entity.FuturesBalance{
			Asset:            asset,
			Balance:          bal,
			Equity:           bal,
			Available:        bal,
			UnrealizedProfit: "0",
		})
	}

	return out
}

func (c *futures_converts) convertLeverage(in futures_leverageSummary) (out entity.Futures_Leverage) {
	lvStr := strconv.Itoa(in.Leverage)

	out.Symbol = "" // плечо аккаунт-уровня, без конкретного инструмента
	out.Leverage = lvStr
	out.LongLeverage = lvStr
	out.ShortLeverage = lvStr

	return out
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrderResponse) (out []entity.PlaceOrder) {
	if in.OrderID == 0 {
		return out
	}

	ts := tsFloatToMillis(in.Timestamp)
	if ts == 0 {
		ts = time.Now().UTC().UnixMilli()
	}

	out = append(out, entity.PlaceOrder{
		OrderID:       strconv.FormatInt(in.OrderID, 10),
		ClientOrderID: in.ClientOrderID,
		// PositionID у WhiteBIT при создании ордера не возвращается
		Ts: ts,
	})

	return out
}

func (c *futures_converts) convertOrderList(in []futures_orderListWB) (out []entity.Futures_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		side := strings.ToUpper(item.Side)
		posSide := strings.ToUpper(item.PositionSide)

		// Нормализуем тип ордера: "MARGIN LIMIT" / "MARGIN MARKET" → LIMIT / MARKET
		ordType := "UNKNOWN"
		lowerType := strings.ToLower(item.Type)
		switch {
		case strings.Contains(lowerType, "market"):
			ordType = "MARKET"
		case strings.Contains(lowerType, "limit"):
			ordType = "LIMIT"
		default:
			ordType = strings.ToUpper(item.Type)
		}

		// Исполненный объём = dealStock, общий = amount, остаток = left
		executedSize := item.DealStock
		positionSize := item.Amount

		// timestamp в секундах → ms
		tsMs := int64(item.Timestamp * 1000)

		// MarginMode на WhiteBit для collateral-фьючей по сути всегда CROSS
		marginMode := string(entity.MarginModeTypeCross)

		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Market,
			OrderID:       fmt.Sprintf("%d", item.OrderId),
			ClientOrderID: item.ClientOrderId,
			PositionID:    "",
			Side:          side,
			PositionSide:  posSide,
			PositionSize:  positionSize,
			ExecutedSize:  executedSize,
			Price:         item.Price,
			Leverage:      "", // WhiteBit здесь не отдаёт плечо per-order
			Type:          ordType,
			Status:        strings.ToUpper(item.Status),
			CreateTime:    tsMs,
			UpdateTime:    tsMs, // отдельного поля нет — дублируем creation
			MarginMode:    marginMode,
		})
	}

	return out
}

func (c *futures_converts) convertPositions(in []futures_positionWB) (out []entity.Futures_Positions) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		side := strings.ToUpper(item.PositionSide)

		// На WhiteBIT:
		//  - hedge mode включён → позиции LONG и/или SHORT
		//  - hedge mode выключен → BOTH (односторонний режим)
		hedgeMode := side != "BOTH"

		// Маржин-режим на WhiteBIT для collateral account — по сути CROSS
		marginMode := string(entity.MarginModeTypeCross)

		// openDate / modifyDate приходят в секундах (float), переводим в ms
		var cTimeMs, uTimeMs int64
		if item.OpenDate > 0 {
			cTimeMs = int64(item.OpenDate * 1000)
		}
		if item.ModifyDate > 0 {
			uTimeMs = int64(item.ModifyDate * 1000)
		}

		posSize := math.Abs(utils.StringToFloat(item.Amount))
		entry := utils.StringToFloat(item.BasePrice)

		notional := posSize * entry

		out = append(out, entity.Futures_Positions{
			Symbol:           item.Market,
			PositionSide:     side,        // LONG / SHORT / BOTH
			PositionSize:     item.Amount, // amount
			Leverage:         "",          // по позициям не приходит, только на аккаунте
			PositionID:       strconv.FormatInt(item.PositionId, 10),
			EntryPrice:       item.BasePrice, // basePrice
			MarkPrice:        "",             // в этом ответе нет markPrice
			UnRealizedProfit: item.Pnl,       // PnL в money
			RealizedProfit:   "",             // отдельного realized PnL нет
			Notional:         utils.FloatToStringAll(notional),
			HedgeMode:        hedgeMode,
			MarginMode:       marginMode,
			CreateTime:       cTimeMs,
			UpdateTime:       uTimeMs,
		})
	}

	return out
}

func (c *futures_converts) convertPositionsHistory(in []futures_positionsHistoryWB) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		positionIDStr := strconv.FormatInt(item.PositionId, 10)

		// --- определяем сторону позиции ---
		posSide := strings.ToUpper(item.PositionSide)

		if posSide == "" {
			amt := utils.StringToFloat(item.Amount)
			if amt > 0 {
				posSide = "LONG"
			} else if amt < 0 {
				posSide = "SHORT"
			}
			// если amt == 0 – позиция закрыта, сторону однозначно не восстановить, можно оставить пусто
		}

		positionAmt := item.Amount

		avgPrice := item.BasePrice

		var (
			executedPositionAmt string
			executedAvgPrice    string
			realisedProfit      string
			fee                 string
			funding             string
		)

		if item.OrderDetail != nil {
			od := item.OrderDetail
			executedPositionAmt = od.TradeAmount
			executedAvgPrice = od.Price
			fee = od.TradeFee

			if od.RealizedPnl != nil {
				realisedProfit = *od.RealizedPnl
			}
			if od.FundingFee != nil && *od.FundingFee != "" {
				funding = *od.FundingFee
			}
		}

		if funding == "" && item.RealizedFunding != "" {
			funding = item.RealizedFunding
		}

		createTimeMs := int64(item.OpenDate * 1000)
		updateTimeMs := int64(item.ModifyDate * 1000)

		out = append(out, entity.Futures_PositionsHistory{
			Symbol:              item.Market,
			PositionID:          positionIDStr,
			PositionSide:        posSide,
			PositionAmt:         positionAmt,
			ExecutedPositionAmt: executedPositionAmt,
			AvgPrice:            avgPrice,
			ExecutedAvgPrice:    executedAvgPrice,
			RealisedProfit:      realisedProfit,
			Fee:                 fee,
			Leverage:            "", // эндпоинт WB его не даёт
			Funding:             funding,
			MarginMode:          string(entity.MarginModeTypeCross),
			CreateTime:          createTimeMs,
			UpdateTime:          updateTimeMs,
		})
	}

	return out
}

func (c *futures_converts) convertOrdersHistoryWB(in []futures_positionsHistoryWB) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		// Нас интересуют только события, где есть конкретный ордер (orderDetail).
		if item.OrderDetail == nil {
			continue
		}

		od := item.OrderDetail

		// --- PositionID / OrderID ---
		positionID := strconv.FormatInt(item.PositionId, 10)
		orderID := strconv.FormatInt(od.Id, 10)

		// --- PositionSide ---
		posSide := strings.ToUpper(item.PositionSide)
		if posSide == "" {
			amt := utils.StringToFloat(item.Amount)
			if amt > 0 {
				posSide = "LONG"
			} else if amt < 0 {
				posSide = "SHORT"
			}
		}

		// --- Side (BUY/SELL) ---
		side := ""
		switch posSide {
		case "LONG":
			side = "BUY"
		case "SHORT":
			side = "SELL"
		}

		// --- HedgeMode ---
		hedgeMode := false
		if posSide == "LONG" || posSide == "SHORT" {
			hedgeMode = true
		}

		// --- размеры и цены ---
		positionSize := item.Amount    // текущий размер позиции после события
		executedSize := od.TradeAmount // сколько "проторговалось" этим ордером
		price := od.Price              // цена сделки
		executedPrice := od.Price      // другого значения WB нам не даёт, дублируем
		avgPrice := item.BasePrice     // средняя по позиции после сделки (для инфы, можно не тащить)
		_ = avgPrice                   // avgPrice нам здесь не нужен, просто на всякий

		realisedProfit := ""
		if od.RealizedPnl != nil {
			realisedProfit = *od.RealizedPnl
		}

		fee := od.TradeFee

		funding := ""
		if od.FundingFee != nil && *od.FundingFee != "" {
			funding = *od.FundingFee
		}
		if funding == "" && item.RealizedFunding != "" {
			funding = item.RealizedFunding
		}

		createTimeMs := int64(item.OpenDate * 1000)
		updateTimeMs := int64(item.ModifyDate * 1000)

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:         item.Market,
			OrderID:        orderID,
			ClientOrderID:  "", // WhiteBIT в этом эндпоинте не возвращает clientOrderId
			PositionID:     positionID,
			Side:           side,
			PositionSide:   posSide,
			PositionSize:   positionSize,
			ExecutedSize:   executedSize,
			Price:          price,
			ExecutedPrice:  executedPrice,
			RealisedProfit: realisedProfit,
			Fee:            fee,
			Leverage:       "",       // на уровне ордера WB не даёт плечо
			Type:           "",       // тип ордера (LIMIT/MARKET) тут не приходит, оставить пустым
			Status:         "FILLED", // это уже история исполнения, считаем FILLED
			HedgeMode:      hedgeMode,
			MarginMode:     string(entity.MarginModeTypeCross),
			CreateTime:     createTimeMs,
			UpdateTime:     updateTimeMs,
		})
	}

	return out
}

func (c *futures_converts) convertListenKey(in futures_listenKey) (out entity.Futures_ListenKey) {
	out.ListenKey = in.WebsocketToken
	return out
}

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentWB) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		state := "LIVE"
		if !item.TradesEnabled {
			state = "DISABLED"
		}

		out = append(out, entity.Spot_InstrumentsInfo{
			Symbol:         item.Name,      // например BTC_USDT
			Base:           item.Stock,     // BTC
			Quote:          item.Money,     // USDT
			MinQty:         item.MinAmount, // "0.0001"
			MinNotional:    item.MinTotal,  // "5"
			PricePrecision: item.MoneyPrec, // "2"
			SizePrecision:  item.StockPrec, // "6"
			State:          strings.ToUpper(state),
		})
	}

	return out
}

func (c *spot_converts) convertPlaceOrder(in spot_placeOrderResponseWB) (out []entity.PlaceOrder) {
	if in.OrderID == 0 {
		return out
	}

	out = append(out, entity.PlaceOrder{
		OrderID:       strconv.FormatInt(in.OrderID, 10),
		ClientOrderID: in.ClientOrderID,
		// как и на фьючах, используем локальное время в мс
		Ts: time.Now().UTC().UnixMilli(),
	})

	return out
}

func (c *spot_converts) convertSpotOrderList(in []spot_orderListWB) (out []entity.Spot_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		// Отфильтровать всё, что НЕ спот:
		// 1) фьючерсы: рынки типа BTC_PERP
		if strings.HasSuffix(strings.ToUpper(item.Market), "_PERP") {
			continue
		}
		// 2) маржинальные: тип "margin limit" / "margin market"
		if strings.HasPrefix(strings.ToLower(item.Type), "margin ") {
			continue
		}

		// executedSize берём из dealStock (для спота это количество base-актива)
		executedSize := item.DealStock

		// timestamp приходит в секундах с дробной частью -> переводим в ms
		tsMs := int64(item.Timestamp * 1000)

		out = append(out, entity.Spot_OrdersList{
			Symbol:        item.Market,
			OrderID:       strconv.FormatInt(item.OrderID, 10),
			ClientOrderID: item.ClientOrderID,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Amount,
			ExecutedSize:  executedSize,
			Price:         item.Price,
			Type:          strings.ToUpper(item.Type),
			Status:        strings.ToUpper(item.Status),
			CreateTime:    tsMs,
			UpdateTime:    tsMs, // у API нет отдельного поля, используем timestamp
		})
	}

	return out
}

func (c *spot_converts) convertSpotAmendOrder(in []spot_modifyOrderWB) (out []entity.PlaceOrder) {
	if len(in) == 0 {
		return out
	}

	nowMs := time.Now().UTC().UnixMilli()

	for _, item := range in {
		out = append(out, entity.PlaceOrder{
			OrderID:       strconv.FormatInt(item.OrderID, 10),
			ClientOrderID: item.ClientOrderID,
			Ts:            nowMs,
		})
	}

	return out
}

func (c *spot_converts) convertOrdersHistory(in []spot_tradeHistoryWB) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, t := range in {
		// Для spot-метода отсекаем фьючерсные рынки типа *_PERP
		if strings.HasSuffix(t.Market, "_PERP") {
			continue
		}

		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        t.Market,
			OrderID:       strconv.FormatInt(t.OrderID, 10),
			ClientOrderID: t.ClientOrderID,
			Side:          strings.ToUpper(t.Side),

			Size:         t.Amount,
			ExecutedSize: t.Amount,

			Price:         t.Price,
			ExecutedPrice: t.Price,

			Fee:      t.Fee,
			FeeAsset: t.FeeAsset,
			// Эндпоинт не отдаёт тип явно (market/limit), оставляем пустым
			Type:   "",
			Status: "FILLED",

			// time — float сек, переводим в ms.
			CreateTime: int64(t.Time * 1000),
			UpdateTime: int64(t.Time * 1000),
		})
	}

	return out
}
