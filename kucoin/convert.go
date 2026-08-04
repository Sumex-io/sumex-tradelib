package kucoin

import (
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================
type account_converts struct{}

func (c *account_converts) convertAccountInfo(in accountInfo) (out entity.AccountInformation) {

	out.UID = utils.Int64ToString(in.UID)
	out.Label = in.Remark

	if strings.Contains(in.Permission, "General") {
		out.CanRead = true
	}

	if strings.Contains(in.Permission, "Spot") {
		out.PermSpot = true
		out.CanTrade = true
	}

	if strings.Contains(in.Permission, "Futures") {
		out.PermFutures = true
		out.CanTrade = true
	}

	if strings.Contains(in.Permission, "Withdrawal") {
		out.CanTransfer = true
	}

	return out
}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertBalance(in []spot_Balance) (out []entity.AssetsBalance) {
	if len(in) == 0 {
		return out
	}

	for _, i := range in {
		if i.Type == "main" || (i.Balance == "" || i.Balance == "0") {
			continue
		}
		out = append(out, entity.AssetsBalance{
			Asset:   i.Currency,
			Balance: i.Balance,
			Locked:  i.Holds,
		})
	}
	return out
}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {

		priceP := utils.GetPrecisionFromStr(item.PriceIncrement)
		sizeP := utils.GetPrecisionFromStr(item.BaseIncrement)
		state := "OFF"
		if item.EnableTrading {
			state = "LIVE"
		}
		out = append(out, entity.Spot_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseCurrency,
			Quote:          item.QuoteCurrency,
			MinQty:         item.BaseMinSize,
			MinNotional:    item.QuoteMinSize,
			PricePrecision: priceP,
			SizePrecision:  sizeP,
			State:          strings.ToUpper(state),
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {

	oID := in.OrderId
	if oID == "" {
		oID = in.CancelledOrderIds
	}
	out = append(out, entity.PlaceOrder{
		OrderID:       oID,
		ClientOrderID: in.СlientOid,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertOrderList(in []spot_orderList) (out []entity.Spot_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.Spot_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.ID,
			ClientOrderID: item.ClientOid,
			Size:          item.Size,
			ExecutedSize:  item.DealSize,
			Side:          strings.ToUpper(item.Side),
			Price:         item.Price,
			Type:          strings.ToUpper(item.Type),
			Status:        strings.ToUpper(item.TradeType),
			CreateTime:    item.CreatedAt,
			UpdateTime:    time.Now().UTC().UnixMilli(),
		})
	}
	return out
}

func (c *spot_converts) convertOrdersHistory(in []spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.Spot_OrdersHistory{
			Symbol:  item.Symbol,
			OrderID: item.OrderId,
			// ClientOrderID: item.ClOrdId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Size,
			Price:         item.Price,
			ExecutedSize:  item.Size,
			ExecutedPrice: item.Price,
			Fee:           item.Fee,
			FeeAsset:      item.FeeCurrency,
			Type:          strings.ToUpper(item.Type),
			// Status:        strings.ToUpper(item.TradeType),
			Status:     "FILLED",
			CreateTime: item.CreatedAt,
			UpdateTime: item.CreatedAt,
		})
	}
	return out
}

func (c *spot_converts) convertListenKey(in spot_listenKey) (out entity.Spot_ListenKey) {
	out.ListenKey = in.Token
	return out
}

// ===============FUTURES=================

type futures_converts struct{}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, i := range in {
		out = append(out, entity.FuturesBalance{
			Asset:            i.Currency,
			Balance:          utils.FloatToStringAll(i.MarginBalance),
			Equity:           utils.FloatToStringAll(i.AccountEquity),
			Available:        utils.FloatToStringAll(i.AvailableBalance),
			UnrealizedProfit: utils.FloatToStringAll(i.UnrealisedPNL),
		})
	}
	return out
}

func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		state := "LIVE"
		if strings.ToUpper(item.Status) != "OPEN" {
			state = strings.ToUpper(item.Status)
		}
		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseCurrency,
			Quote:          item.QuoteCurrency,
			MinQty:         utils.FloatToStringAll(item.LotSize),
			PricePrecision: utils.GetPrecisionFromStr(utils.FloatToStringAll(item.IndexPriceTickSize)),
			SizePrecision:  utils.GetPrecisionFromStr(utils.FloatToStringAll(item.LotSize)),
			MaxLeverage:    utils.Int64ToString(item.MaxLeverage),
			State:          state,
			IsSizeContract: true,
			Multiplier:     utils.FloatToStringAll(item.Multiplier),
			ContractSize:   utils.FloatToStringAll(item.LotSize * item.Multiplier),
		})
	}
	return out
}

func (c *futures_converts) convertLeverage(in futures_leverage) (out entity.Futures_Leverage) {
	out.Symbol = in.Symbol
	out.Leverage = in.Leverage
	return out
}

func (c *futures_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {
	oID := in.OrderId
	if oID == "" {
		oID = in.CancelledOrderIds
	}
	out = append(out, entity.PlaceOrder{
		OrderID:       oID,
		ClientOrderID: in.СlientOid,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertOrderList(answ []futures_orderList) (res []entity.Futures_OrdersList) {
	for _, item := range answ {
		positionSide := "LONG"
		if item.PositionSide == "BOTH" {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.PositionSide)
		}

		res = append(res, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.ID,
			ClientOrderID: item.ClientOid,
			PositionSide:  positionSide,
			Side:          strings.ToUpper(item.Side),
			PositionSize:  utils.FloatToStringAll(item.Size),
			ExecutedSize:  utils.FloatToStringAll(item.FilledSize),
			Price:         item.Price,
			Type:          strings.ToUpper(item.Type),
			MarginMode:    strings.ToUpper(item.MarginMode),
			Leverage:      item.Leverage,
			Status:        strings.ToUpper(item.Status),
			CreateTime:    item.CreatedAt,
			UpdateTime:    item.UpdatedAt,
			TpOrder:       item.TpOrder,
			SlOrder:       item.SlOrder,
		})
	}
	return res
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {

		if !item.IsOpen {
			continue
		}

		positionSide := "LONG"
		hedgeMode := false

		if item.PositionSide != "BOTH" {
			hedgeMode = true
			positionSide = strings.ToUpper(item.PositionSide)
		} else {
			if item.CurrentQty < 0 {
				positionSide = "SHORT"
				item.CurrentQty = 0 - item.CurrentQty
			}
		}

		res = append(res, entity.Futures_Positions{
			Symbol:           item.Symbol,
			PositionSide:     positionSide,
			PositionSize:     utils.FloatToStringAll(item.CurrentQty),
			Leverage:         utils.Int64ToString(item.Leverage),
			PositionID:       item.ID,
			EntryPrice:       utils.FloatToStringAll(item.AvgEntryPrice),
			MarkPrice:        utils.FloatToStringAll(item.MarkPrice),
			UnRealizedProfit: utils.FloatToStringAll(item.UnrealisedPnl),
			RealizedProfit:   utils.FloatToStringAll(item.RealisedPnl),
			Notional:         utils.FloatToStringAll(item.CurrentCost),
			HedgeMode:        hedgeMode,
			MarginMode:       strings.ToUpper(item.MarginMode),
			CreateTime:       item.OpeningTimestamp,
			UpdateTime:       item.CurrentTimestamp,
		})
	}
	return res
}

// func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {

// 	if len(in) == 0 {
// 		return out
// 	}

// 	for _, item := range in {
// 		positionSide := "LONG"
// 		hedgeMode := false

// 		if item.Status == "done" {
// 			item.Status = "FILLED"
// 		}
// 		if item.PositionSide != "BOTH" {
// 			hedgeMode = true
// 			positionSide = strings.ToUpper(item.PositionSide)
// 		} else {
// 			// if item.CurrentQty < 0 {
// 			// 	positionSide = "SHORT"
// 			// 	item.CurrentQty = 0 - item.CurrentQty
// 			// }
// 		}

// 		t := "LIMIT"
// 		if item.Liquidity == "taker" {
// 			t = "MARKET"
// 		}

// 		out = append(out, entity.Futures_OrdersHistory{
// 			Symbol:        item.Symbol,
// 			OrderID:       item.OrderId,
// 			ClientOrderID: item.ClientOid,
// 			Side:          strings.ToUpper(item.Side),
// 			PositionSide:  positionSide,
// 			PositionSize:  utils.FloatToStringAll(item.Size),
// 			ExecutedSize:  utils.FloatToStringAll(item.Size),
// 			Price:         item.Price,
// 			ExecutedPrice: item.Price,
// 			// 		RealisedProfit: item.Pnl,
// 			Fee:        item.Fee,
// 			FeeAsset:   item.FeeCurrency,
// 			Leverage:   item.Leverage,
// 			HedgeMode:  hedgeMode,
// 			MarginMode: strings.ToUpper(item.MarginMode),
// 			// Type:       strings.ToUpper(item.Type),
// 			Type: strings.ToUpper(t),
// 			// Status:     strings.ToUpper(item.Status),
// 			Status:     "FILLED",
// 			CreateTime: item.CreatedAt,
// 			UpdateTime: time.Now().UnixMilli(),
// 			// UpdateTime: item.UpdatedAt,
// 		})
// 	}
// 	return out
// }

func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		// history возвращаем только по реально исполненным ордерам
		filledSize := item.FilledSize
		if filledSize == 0 {
			filledSize = item.DealSize
		}
		if filledSize <= 0 {
			continue
		}

		positionSide := "LONG"
		hedgeMode := false

		if strings.ToUpper(item.PositionSide) != "BOTH" && item.PositionSide != "" {
			hedgeMode = true
			positionSide = strings.ToUpper(item.PositionSide)
		} else {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		}

		orderType := strings.ToUpper(item.Type)
		if orderType == "" {
			orderType = "LIMIT"
			if strings.ToLower(item.Liquidity) == "taker" {
				orderType = "MARKET"
			}
		}

		execPrice := item.AvgDealPrice
		if execPrice == "" || execPrice == "0" {
			execPrice = item.Price
		}

		price := item.Price
		if (price == "" || price == "0") && item.StopPrice != "" && item.StopPrice != "0" {
			price = item.StopPrice
		}

		isTP, isSL := kucoinStopFlags(item.Side, item.Stop)

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       item.ID,
			ClientOrderID: item.ClientOid,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  positionSide,
			PositionSize:  utils.FloatToStringAll(item.Size),
			ExecutedSize:  utils.FloatToStringAll(filledSize),
			Price:         price,
			ExecutedPrice: execPrice,
			Fee:           item.Fee,
			FeeAsset:      item.FeeCurrency,
			Leverage:      item.Leverage,
			HedgeMode:     hedgeMode,
			MarginMode:    strings.ToUpper(item.MarginMode),
			Type:          orderType,
			Status:        "FILLED",
			CreateTime:    item.CreatedAt,
			UpdateTime:    item.UpdatedAt,
			TpOrder:       isTP,
			SlOrder:       isSL,
		})
	}

	return out
}

func kucoinStopFlags(side string, stop string) (isTP bool, isSL bool) {
	s := strings.ToLower(strings.TrimSpace(side))
	st := strings.ToLower(strings.TrimSpace(stop))

	// Логика такая же, как у вас в active stop orders:
	// sell + up   => TP
	// sell + down => SL
	// buy  + up   => SL
	// buy  + down => TP
	switch s {
	case "sell":
		if st == "up" {
			return true, false
		}
		if st == "down" {
			return false, true
		}
	case "buy":
		if st == "up" {
			return false, true
		}
		if st == "down" {
			return true, false
		}
	}

	return false, false
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.Futures_PositionsHistory{
			Symbol:       item.Symbol,
			PositionID:   item.CloseId,
			PositionSide: strings.ToUpper(item.Side),
			// PositionAmt:         item.OpenMaxPos,
			// ExecutedPositionAmt: item.CloseTotalPos,
			AvgPrice:         item.OpenPrice,
			ExecutedAvgPrice: item.ClosePrice,
			RealisedProfit:   item.Pnl,
			Fee:              item.TradeFee,
			Funding:          item.FundingFee,
			Leverage:         item.Leverage,
			MarginMode:       strings.ToUpper(item.MarginMode),
			CreateTime:       item.OpenTime,
			UpdateTime:       item.CloseTime,
		})
	}
	return out
}
