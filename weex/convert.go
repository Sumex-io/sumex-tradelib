package weex

import (
	"strconv"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================

type account_converts struct{}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		tickSize := item.TickSize.String()
		stepSize := item.StepSize.String()
		minTradeAmount := item.MinTradeAmount.String()

		out = append(out, entity.Spot_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseAsset,
			Quote:          item.QuoteAsset,
			MinQty:         minTradeAmount,
			PricePrecision: utils.GetPrecisionFromStr(tickSize),
			SizePrecision:  utils.GetPrecisionFromStr(stepSize),
			State:          strings.ToUpper(item.Status),
		})
	}

	return out
}

func (c *spot_converts) convertBalance(in []spot_Balance) (out []entity.AssetsBalance) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.AssetsBalance{
			Asset:   item.Asset,
			Balance: item.Free,
			Locked:  item.Locked,
		})
	}

	return out
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       strconv.FormatInt(in.OrderId, 10),
		ClientOrderID: in.ClientOrderId,
		Ts:            in.TransactTime,
	})
	return out
}

func (c *spot_converts) convertOrderList(answ []spot_orderList) (res []entity.Spot_OrdersList) {
	for _, item := range answ {
		res = append(res, entity.Spot_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       strconv.FormatInt(item.OrderId, 10),
			ClientOrderID: item.ClientOrderId,
			Size:          item.OrigQty,
			ExecutedSize:  item.ExecutedQty,
			Side:          strings.ToUpper(item.Side),
			Price:         item.Price,
			Type:          strings.ToUpper(item.Type),
			Status:        strings.ToUpper(item.Status),
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
		})
	}
	return res
}

func (c *spot_converts) convertCancelOrder(in cancelOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderID,
		ClientOrderID: in.ClientOrderID,
	})
	return out
}

func (c *spot_converts) convertOrdersHistory(in []spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		if strings.ToUpper(item.Status) != "FILLED" {
			continue
		}

		executedPrice := item.Price

		executedQty := utils.StringToFloat(item.ExecutedQty)
		cumQuote := utils.StringToFloat(item.CummulativeQuoteQty)
		if executedQty > 0 && cumQuote > 0 {
			executedPrice = strconv.FormatFloat(cumQuote/executedQty, 'f', -1, 64)
		}

		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       strconv.FormatInt(item.OrderId, 10),
			ClientOrderID: item.ClientOrderId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.OrigQty,
			ExecutedSize:  item.ExecutedQty,
			Price:         item.Price,
			ExecutedPrice: executedPrice,
			Type:          strings.ToUpper(item.Type),
			Status:        strings.ToUpper(item.Status),
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
		})
	}

	return out
}

// ===============FUTURES=================

type futures_converts struct{}

func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {

		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseAsset,
			Quote:          item.QuoteAsset,
			MinQty:         item.MinOrderSize.String(),
			PricePrecision: strconv.Itoa(item.PricePrecision),
			SizePrecision:  utils.GetPrecisionFromStr(item.MinOrderSize.String()),
			// SizePrecision:  strconv.Itoa(item.QuantityPrecision),
			MaxLeverage:    strconv.Itoa(item.MaxLeverage),
			State:          strings.ToUpper("LIVE"),
			IsSizeContract: false,
			// Multiplier:     "1",
			// ContractSize:   item.ContractVal.String(),
		})
	}

	return out
}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.FuturesBalance{
			Asset:            item.Asset,
			Balance:          item.Balance,
			Equity:           item.Balance,
			Available:        item.AvailableBalance,
			UnrealizedProfit: item.UnrealizePnl,
		})
	}
	return out
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.ClientOrderId,
		Ts:            utils.CurrentTimestamp(),
	})
	return out
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {
		// hedgeMode := false
		// if item.SeparatedMode == "SEPARATED" {
		// 	hedgeMode = true
		// }

		if strings.ToUpper(item.MarginType) == "CROSSED" {
			item.MarginType = string(entity.MarginModeTypeCross)
		} else {
			item.MarginType = string(entity.MarginModeTypeIsolated)

		}

		entryPrice := calcWeexPositionEntryPrice(item)

		res = append(res, entity.Futures_Positions{
			Symbol:           item.Symbol,
			PositionSide:     strings.ToUpper(item.Side),
			PositionSize:     item.Size,
			Leverage:         item.Leverage,
			PositionID:       utils.Int64ToString(item.ID),
			EntryPrice:       entryPrice,
			MarkPrice:        "",
			UnRealizedProfit: item.UnrealizePnl,
			RealizedProfit:   "",
			Notional:         item.OpenValue,
			// HedgeMode:        hedgeMode,
			HedgeMode:  true,
			MarginMode: strings.ToUpper(item.MarginType),
			CreateTime: item.CreatedTime,
			UpdateTime: item.UpdatedTime,
		})
	}
	return res
}

func calcWeexPositionEntryPrice(item futures_Position) string {
	cumOpenSize := utils.StringToFloat(item.CumOpenSize)
	cumOpenValue := utils.StringToFloat(item.CumOpenValue)
	if cumOpenSize != 0 && cumOpenValue != 0 {
		return utils.FloatToStringAll(cumOpenValue / cumOpenSize)
	}

	size := utils.StringToFloat(item.Size)
	openValue := utils.StringToFloat(item.OpenValue)
	if size != 0 && openValue != 0 {
		return utils.FloatToStringAll(openValue / size)
	}

	return ""
}

func (c *futures_converts) convertOrderList(answ []futures_orderList) (res []entity.Futures_OrdersList) {
	for _, item := range answ {
		res = append(res, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       strconv.FormatInt(item.OrderId, 10),
			ClientOrderID: item.ClientOrderId,
			PositionSide:  strings.ToUpper(item.PositionSide),
			Side:          strings.ToUpper(item.Side),
			PositionSize:  item.OrigQty,
			ExecutedSize:  item.ExecutedQty,
			Price:         item.Price,
			Type:          strings.ToUpper(item.Type),
			Status:        strings.ToUpper(item.Status),
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
		})
	}
	return res
}

func (c *futures_converts) convertAlgoOrderList(in []futures_algoOrder) (out []entity.Futures_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		typ := strings.ToUpper(item.OrderType)
		tpOrder := typ == "TAKE_PROFIT" || typ == "TAKE_PROFIT_MARKET"
		slOrder := typ == "STOP" || typ == "STOP_MARKET"

		price := item.Price
		if price == "" || price == "0" || price == "0.0" {
			price = item.TriggerPrice
		}

		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       strconv.FormatInt(item.AlgoId, 10),
			ClientOrderID: item.ClientAlgoId,
			PositionSide:  strings.ToUpper(item.PositionSide),
			Side:          strings.ToUpper(item.Side),
			PositionSize:  item.Quantity,
			ExecutedSize:  "0",
			Price:         price,
			Type:          typ,
			Status:        strings.ToUpper(item.AlgoStatus),
			CreateTime:    item.CreateTime,
			UpdateTime:    item.UpdateTime,
			TpOrder:       tpOrder,
			SlOrder:       slOrder,
		})
	}

	return out
}

func (c *futures_converts) convertCancelOrder(in futures_cancelOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.OrigClientOrderId,
		Ts:            utils.CurrentTimestamp(),
	})
	return out
}

func (c *futures_converts) convertAlgoOrdersHistory(in []futures_algoOrdersHistoryItem) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		// по вашему правилу history = только исполненные
		// здесь status == "2" уже считается исполненным/сработавшим
		if item.Status != "2" {
			continue
		}

		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		symbol = strings.TrimPrefix(symbol, "CMT_")

		positionSide := ""
		side := ""

		switch item.Type {
		case "1":
			positionSide = "LONG"
			side = "BUY"
		case "2":
			positionSide = "SHORT"
			side = "SELL"
		case "3", "5", "7", "9":
			positionSide = "LONG"
			side = "SELL"
		case "4", "6", "8", "10":
			positionSide = "SHORT"
			side = "BUY"
		}

		orderType := strings.ToUpper(strings.TrimSpace(item.OrderType))
		tpOrder := strings.Contains(orderType, "TAKE_PROFIT")
		slOrder := orderType == "STOP" || orderType == "STOP_MARKET" || strings.Contains(orderType, "STOP_LOSS")

		if orderType == "" {
			orderType = "CONDITIONAL"
		}

		price := strings.TrimSpace(item.Price)
		if price == "" || price == "0" {
			price = strings.TrimSpace(item.TriggerPrice)
		}

		executedPrice := strings.TrimSpace(item.PriceAvg)
		if executedPrice == "" || executedPrice == "0" {
			executedPrice = price
		}

		updateTime := utils.StringToInt64(item.TriggerTime)
		if updateTime == 0 {
			updateTime = utils.StringToInt64(item.CreateTime)
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:         symbol,
			OrderID:        item.OrderID,
			ClientOrderID:  item.ClientOID,
			Side:           side,
			PositionSide:   positionSide,
			PositionSize:   item.Size,
			ExecutedSize:   item.FilledQty,
			Price:          price,
			ExecutedPrice:  executedPrice,
			RealisedProfit: item.TotalProfits,
			Fee:            item.Fee,
			FeeAsset:       "",
			Leverage:       "",
			HedgeMode:      false,
			MarginMode:     "",
			Type:           orderType,
			Status:         "FILLED",
			CreateTime:     utils.StringToInt64(item.CreateTime),
			UpdateTime:     updateTime,
			TpOrder:        tpOrder,
			SlOrder:        slOrder,
		})
	}

	return out
}

func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		if strings.ToUpper(item.Status) != "FILLED" {
			continue
		}

		hedgeMode := false
		if item.PositionSide != "" && strings.ToUpper(item.PositionSide) != "BOTH" {
			hedgeMode = true
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:         item.Symbol,
			OrderID:        utils.Int64ToString(item.OrderId),
			ClientOrderID:  item.ClientOrderId,
			Side:           strings.ToUpper(item.Side),
			PositionSide:   strings.ToUpper(item.PositionSide),
			PositionSize:   item.OrigQty,
			ExecutedSize:   item.ExecutedQty,
			Price:          item.Price,
			ExecutedPrice:  item.AvgPrice,
			RealisedProfit: "",
			Fee:            "",
			FeeAsset:       "",
			Leverage:       "",
			HedgeMode:      hedgeMode,
			MarginMode:     "",
			Type:           strings.ToUpper(item.Type),
			Status:         strings.ToUpper(item.Status),
			CreateTime:     item.Time,
			UpdateTime:     item.UpdateTime,
		})
	}
	return out
}

func (c *futures_converts) convertUserTrades(in []futures_userTrade) (out []entity.Futures_UserTrades) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.Futures_UserTrades{
			TradeID:         utils.Int64ToString(item.ID),
			OrderID:         utils.Int64ToString(item.OrderID),
			Symbol:          item.Symbol,
			Side:            strings.ToUpper(item.Side),
			PositionSide:    strings.ToUpper(item.PositionSide),
			Price:           item.Price,
			Qty:             item.Qty,
			QuoteQty:        item.QuoteQty,
			Commission:      item.Commission,
			CommissionAsset: item.CommissionAsset,
			RealisedProfit:  item.RealizedPnl,
			Buyer:           item.Buyer,
			Maker:           item.Maker,
			Time:            item.Time,
		})
	}

	return out
}

func (c *futures_converts) convertLeverage(in []futures_leverage) (out entity.Futures_Leverage) {
	if len(in) == 0 {
		return out
	}

	item := in[0]
	out.Symbol = strings.ToUpper(item.Symbol)

	if strings.ToUpper(item.MarginType) == "ISOLATED" {
		out.MarginMode = string(entity.MarginModeTypeIsolated)
		out.LongLeverage = item.IsolatedLongLeverage
		out.ShortLeverage = item.IsolatedShortLeverage

		if item.IsolatedLongLeverage == item.IsolatedShortLeverage {
			out.Leverage = item.IsolatedLongLeverage
		} else if utils.StringToFloat(item.IsolatedLongLeverage) < utils.StringToFloat(item.IsolatedShortLeverage) {
			out.Leverage = item.IsolatedLongLeverage
		} else {
			out.Leverage = item.IsolatedShortLeverage
		}
	} else {
		out.MarginMode = string(entity.MarginModeTypeCross)
		out.Leverage = item.CrossLeverage
		out.LongLeverage = item.CrossLeverage
		out.ShortLeverage = item.CrossLeverage
	}

	return out
}
