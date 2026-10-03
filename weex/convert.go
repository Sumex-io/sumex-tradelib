package weex

import (
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
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
			LiquidationPrice: item.LiquidatePrice,
			UnRealizedProfit: item.UnrealizePnl,
			RealizedProfit:   weexPositionRealisedProfit(item),
			Notional:         item.OpenValue,
			Margin:           weexPositionMargin(item),
			// HedgeMode:        hedgeMode,
			HedgeMode:  true,
			MarginMode: strings.ToUpper(item.MarginType),
			CreateTime: item.CreatedTime,
			UpdateTime: item.UpdatedTime,
		})
	}
	return res
}

func weexPositionMargin(item futures_Position) string {
	if item.MarginType == string(entity.MarginModeTypeIsolated) && utils.StringToFloat(item.IsolatedMargin) > 0 {
		return item.IsolatedMargin
	}
	return item.MarginSize
}

// WeEx realised PnL: closed PnL less every fee paid, funding included (positive = paid).
func weexPositionRealisedProfit(item futures_Position) string {
	profit := new(big.Rat)
	if openSize := weexDecimal(item.CumOpenSize); openSize.Sign() != 0 {
		entryValue := new(big.Rat).Mul(weexDecimal(item.CumCloseSize), new(big.Rat).Quo(weexDecimal(item.CumOpenValue), openSize))
		profit.Sub(weexDecimal(item.CumCloseValue), entryValue)
		if strings.EqualFold(item.Side, "SHORT") {
			profit.Neg(profit)
		}
	}
	for _, fee := range []string{item.CumOpenFee, item.CumCloseFee, item.CumFundingFee, item.CumLiquidateFee} {
		profit.Sub(profit, weexDecimal(fee))
	}
	return weexDecimalString(profit)
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
		if utils.StringToFloat(price) == 0 {
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

func (c *futures_converts) convertAlgoOrdersHistory(in []futures_algoOrder, executed map[string]entity.Futures_OrdersHistory) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		status := strings.ToUpper(item.AlgoStatus)
		if status != "FILLED" && status != "TRIGGERED" {
			continue
		}

		typ := strings.ToUpper(item.OrderType)
		positionSide := strings.ToUpper(item.PositionSide)

		price := item.Price
		if utils.StringToFloat(price) == 0 {
			price = item.TriggerPrice
		}

		executedPrice := item.ActualPrice
		if utils.StringToFloat(executedPrice) == 0 {
			executedPrice = price
		}

		updateTime := item.TriggerTime
		if updateTime == 0 {
			updateTime = item.UpdateTime
		}

		row := entity.Futures_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       strconv.FormatInt(item.AlgoId, 10),
			ClientOrderID: item.ClientAlgoId,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  positionSide,
			PositionSize:  item.Quantity,
			ExecutedSize:  item.Quantity,
			Price:         price,
			ExecutedPrice: executedPrice,
			HedgeMode:     positionSide != "" && positionSide != "BOTH",
			Type:          typ,
			Status:        "FILLED",
			CreateTime:    item.CreateTime,
			UpdateTime:    updateTime,
			TpOrder:       strings.HasPrefix(typ, "TAKE_PROFIT"),
			SlOrder:       typ == "STOP" || typ == "STOP_MARKET",
			ReduceOnly:    item.ReduceOnly || item.ClosePosition,
		}

		// A full-position TP/SL has quantity 0; the triggered order holds the fill.
		if item.ActualOrderId != nil {
			if order, ok := executed[strconv.FormatInt(*item.ActualOrderId, 10)]; ok {
				if utils.StringToFloat(row.PositionSize) == 0 {
					row.PositionSize = order.PositionSize
				}
				row.ExecutedSize = order.ExecutedSize
				row.ExecutedPrice = order.ExecutedPrice
				row.RealisedProfit = order.RealisedProfit
				row.Fee = order.Fee
				row.FeeAsset = order.FeeAsset
			}
		}

		out = append(out, row)
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
			ReduceOnly:     item.ReduceOnly,
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

const weexHistoryDecimals = 8

type weexPositionAccumulator struct {
	symbol         string
	positionSide   string
	positionID     string
	held           *big.Rat
	peakSize       *big.Rat
	closedSize     *big.Rat
	exitNotional   *big.Rat
	realisedProfit *big.Rat
	fee            *big.Rat
	openTime       int64
	closeTime      int64
}

// Fills carry no prior size: a position ends at zero size, or at its next opening fill if opened before the window.
func (c *futures_converts) convertPositionsHistory(in []entity.Futures_UserTrades) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	fills := make([]entity.Futures_UserTrades, len(in))
	copy(fills, in)
	sort.SliceStable(fills, func(i, j int) bool { return fills[i].Time < fills[j].Time })

	open := make(map[string]*weexPositionAccumulator)

	for _, item := range fills {
		side := strings.ToUpper(strings.TrimSpace(item.PositionSide))
		if side != "LONG" && side != "SHORT" {
			continue
		}

		qty := weexDecimal(item.Qty)
		qty.Abs(qty)
		if qty.Sign() == 0 {
			continue
		}

		key := item.Symbol + "|" + side
		position := open[key]

		if strings.EqualFold(strings.TrimSpace(item.Side), "BUY") == (side == "LONG") {
			if position != nil && position.closedSize.Sign() > 0 && position.held.Sign() <= 0 {
				out = append(out, position.toEntity())
				position = nil
			}
			if position == nil {
				position = newWeexPositionAccumulator(item.Symbol, side, item.Time)
				open[key] = position
			}
			position.addOpen(item, qty)
			continue
		}

		if position == nil {
			position = newWeexPositionAccumulator(item.Symbol, side, 0)
			open[key] = position
		}
		position.addClose(item, qty)

		if position.openTime != 0 && position.held.Sign() == 0 {
			out = append(out, position.toEntity())
			delete(open, key)
		}
	}

	for _, position := range open {
		if position.closedSize.Sign() == 0 || (position.openTime != 0 && position.held.Sign() > 0) {
			continue
		}
		out = append(out, position.toEntity())
	}

	return out
}

func newWeexPositionAccumulator(symbol, positionSide string, openTime int64) *weexPositionAccumulator {
	return &weexPositionAccumulator{
		symbol:         symbol,
		positionSide:   positionSide,
		held:           new(big.Rat),
		peakSize:       new(big.Rat),
		closedSize:     new(big.Rat),
		exitNotional:   new(big.Rat),
		realisedProfit: new(big.Rat),
		fee:            new(big.Rat),
		openTime:       openTime,
	}
}

func (p *weexPositionAccumulator) addOpen(item entity.Futures_UserTrades, qty *big.Rat) {
	p.held.Add(p.held, qty)
	if p.held.Cmp(p.peakSize) > 0 {
		p.peakSize.Set(p.held)
	}
	p.fee.Add(p.fee, weexDecimal(item.Commission))
}

func (p *weexPositionAccumulator) addClose(item entity.Futures_UserTrades, qty *big.Rat) {
	p.held.Sub(p.held, qty)
	p.closedSize.Add(p.closedSize, qty)
	p.exitNotional.Add(p.exitNotional, new(big.Rat).Mul(qty, weexDecimal(item.Price)))
	p.realisedProfit.Add(p.realisedProfit, weexDecimal(item.RealisedProfit))
	p.fee.Add(p.fee, weexDecimal(item.Commission))
	p.positionID = item.TradeID
	p.closeTime = item.Time
}

func (p *weexPositionAccumulator) toEntity() entity.Futures_PositionsHistory {
	// The opening fills are usually older than the window, so the entry comes out of the PnL.
	entryNotional := new(big.Rat).Sub(p.exitNotional, p.realisedProfit)
	if p.positionSide == "SHORT" {
		entryNotional = new(big.Rat).Add(p.exitNotional, p.realisedProfit)
	}

	peakSize := p.peakSize
	if p.closedSize.Cmp(peakSize) > 0 {
		peakSize = p.closedSize
	}

	return entity.Futures_PositionsHistory{
		Symbol:              p.symbol,
		PositionID:          p.positionID,
		PositionSide:        p.positionSide,
		PositionAmt:         weexDecimalString(peakSize),
		ExecutedPositionAmt: weexDecimalString(p.closedSize),
		AvgPrice:            weexDecimalString(new(big.Rat).Quo(entryNotional, p.closedSize)),
		ExecutedAvgPrice:    weexDecimalString(new(big.Rat).Quo(p.exitNotional, p.closedSize)),
		RealisedProfit:      weexDecimalString(p.realisedProfit),
		Fee:                 weexDecimalString(p.fee),
		CreateTime:          p.openTime,
		UpdateTime:          p.closeTime,
	}
}

func weexDecimal(value string) *big.Rat {
	parsed := new(big.Rat)
	if _, ok := parsed.SetString(strings.TrimSpace(value)); !ok {
		return new(big.Rat)
	}

	return parsed
}

func weexDecimalString(value *big.Rat) string {
	s := strings.TrimRight(value.FloatString(weexHistoryDecimals), "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}

	return s
}
