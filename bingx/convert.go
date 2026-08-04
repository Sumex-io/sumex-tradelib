package bingx

import (
	"fmt"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================
type account_converts struct{}

func (c *account_converts) convertAccountInfo(in accountInfo) (out entity.AccountInformation) {
	out.Label = in.Note
	out.IP = strings.Join(in.IpAddresses, ",")
	for _, item := range in.Permissions {
		switch item {
		case 1:
			out.CanTrade = true
			out.PermSpot = true
		case 2:
			out.CanRead = true
		case 3:
			out.PermFutures = true
		case 4:
			out.CanTransfer = true
		}
	}
	return out
}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertBalance(in spot_Balance) (out []entity.AssetsBalance) {
	for _, item := range in.Balances {
		out = append(out, entity.AssetsBalance{
			Asset:   item.Asset,
			Balance: utils.FloatToStringAll(utils.StringToFloat(item.Free) + utils.StringToFloat(item.Locked)),
			Locked:  item.Locked,
		})
	}
	return out
}

func (c *spot_converts) convertInstrumentsInfo(in spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in.Symbols) == 0 {
		return out
	}
	for _, item := range in.Symbols {
		state := "OTHER"
		base := ""
		quote := ""

		sp := strings.Split(item.Symbol, "-")
		if len(sp) == 2 {
			base = sp[0]
			quote = sp[1]
		}

		priceP := utils.GetPrecisionFromStr(utils.FloatToStringAll(item.TickSize))
		sizeP := utils.GetPrecisionFromStr(utils.FloatToStringAll(item.StepSize))
		if item.Status == 1 {
			state = "LIVE"
		} else if item.Status == 0 {
			state = "OFF"
		} else if item.Status == 5 {
			state = "PRE-OPEN"
		} else if item.Status == 25 {
			state = "SUSPENDED"
		}
		rec := entity.Spot_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           base,
			Quote:          quote,
			MinQty:         utils.FloatToStringAll(item.MinQty),
			MinNotional:    utils.FloatToStringAll(item.MinNotional),
			PricePrecision: priceP,
			SizePrecision:  sizeP,
			State:          state,
		}
		out = append(out, rec)
	}
	return
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       fmt.Sprintf("%d", in.OrderId),
		ClientOrderID: in.ClientOrderID,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertOrderList(in spot_orderList) (out []entity.Spot_OrdersList) {
	if len(in.Orders) == 0 {
		return out
	}
	for _, item := range in.Orders {
		out = append(out, entity.Spot_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       fmt.Sprintf("%d", item.OrderId),
			ClientOrderID: item.ClientOrderId,
			Side:          item.Side,
			Size:          item.OrigQty,
			Price:         item.Price,
			ExecutedSize:  item.ExecutedQty,
			Type:          strings.ToUpper(item.Type),
			Status:        item.Status,
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
		})
	}
	return out
}

func (c *spot_converts) convertOrdersHistory(in spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in.Orders) == 0 {
		return out
	}
	for _, item := range in.Orders {
		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       fmt.Sprintf("%d", item.OrderId),
			ClientOrderID: item.ClientOrderId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.OrigQty,
			Price:         item.Price,
			ExecutedSize:  item.ExecutedQty,
			ExecutedPrice: utils.FloatToStringAll(item.AvgPrice),
			Fee:           utils.FloatToStringAll(item.Fee),
			Type:          strings.ToUpper(item.Type),
			Status:        item.Status,
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
		})
	}
	return out
}

func (c *spot_converts) convertListenKey(in spot_listenKey) (out entity.Spot_ListenKey) {
	out.ListenKey = in.ListenKey
	return out
}

// ===============FUTURES=================

type futures_converts struct{}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.FuturesBalance{
			Asset:     item.Asset,
			Balance:   item.Balance,
			Equity:    item.Equity,
			Available: item.AvailableMargin,
			// AvailableMargin:  item.AvailableMargin,
			UnrealizedProfit: item.UnrealizedProfit,
		})
	}
	return out
}

func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		state := "OTHER"
		switch item.Status {
		case 1:
			state = "LIVE"
		case 0:
			state = "OFF"
		case 5:
			state = "PRE-OPEN"
		case 25:
			state = "SUSPENDED"
		}
		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.Asset,
			Quote:          item.Currency,
			MinQty:         utils.FloatToStringAll(item.TradeMinQuantity),
			MinNotional:    utils.FloatToStringAll(item.TradeMinUSDT),
			PricePrecision: fmt.Sprintf("%d", item.PricePrecision),
			SizePrecision:  fmt.Sprintf("%d", item.QuantityPrecision),
			State:          state,
		})
	}
	return
}

func (c *futures_converts) convertLeverage(in futures_leverage) (out entity.Futures_Leverage) {

	out.Symbol = in.Symbol
	if in.LongLeverage != 0 && in.LongLeverage == in.ShortLeverage {
		out.Leverage = fmt.Sprintf("%d", in.LongLeverage)
	}

	if in.Leverage != 0 {
		out.Leverage = fmt.Sprintf("%d", in.Leverage)
	}
	out.LongLeverage = fmt.Sprintf("%d", in.LongLeverage)
	out.ShortLeverage = fmt.Sprintf("%d", in.ShortLeverage)
	return out
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       utils.Int64ToString(in.Order.OrderID),
		ClientOrderID: in.Order.ClientOrderId,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertPlaceOrder_extra(in futures_placeOrder_Response_Extra) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       in.Order.OrderID.(string),
		ClientOrderID: in.Order.ClientOrderId.(string),
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertOrderList(in futures_orderList) (out []entity.Futures_OrdersList) {
	if len(in.Orders) == 0 {
		return out
	}

	for _, item := range in.Orders {
		// positionSide := "LONG"
		// // if item.PosSide == "net" {
		// // 	if strings.ToUpper(item.Side) == "SELL" {
		// // 		positionSide = "SHORT"
		// // 	}
		// // } else {
		// // 	positionSide = strings.ToUpper(item.PosSide)
		// // }
		istp := false
		isls := false

		if strings.ToUpper(item.Type) == "TAKE_PROFIT_MARKET" {
			istp = true
			item.Price = item.StopPrice
		}

		if strings.ToUpper(item.Type) == "STOP_MARKET" {
			isls = true
			item.Price = item.StopPrice
		}
		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       fmt.Sprintf("%d", item.OrderId),
			ClientOrderID: item.ClientOrderId,
			PositionID:    fmt.Sprintf("%d", item.PositionID),
			Side:          item.Side,
			PositionSide:  item.PositionSide,
			Type:          strings.ToUpper(item.Type),
			PositionSize:  item.OrigQty,
			ExecutedSize:  item.ExecutedQty,
			Price:         item.Price,
			Leverage:      strings.Replace(item.Leverage, "X", "", 1),
			Status:        strings.ToUpper(item.Status),
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
			TpOrder:       istp,
			SlOrder:       isls,
		})
	}
	return out
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {

		marginMode := string(entity.MarginModeTypeIsolated)
		hedgeMode := false

		if !item.OnlyOnePosition {
			hedgeMode = true
		}

		if !item.Isolated {
			marginMode = string(entity.MarginModeTypeCross)
		}
		res = append(res, entity.Futures_Positions{
			Symbol:       item.Symbol,
			PositionSide: item.PositionSide,
			PositionID:   item.PositionId,
			PositionSize: item.PositionAmt,
			EntryPrice:   item.AvgPrice,
			MarkPrice:    item.MarkPrice,
			// InitialMargin:    item.Initial_margin,
			UnRealizedProfit: item.UnrealizedProfit,
			RealizedProfit:   item.RealisedProfit,
			Notional:         item.PositionValue,
			// MarginRatio:      item.Maintenance_rate,
			Leverage:   utils.Int64ToString(item.Leverage),
			MarginMode: marginMode,
			HedgeMode:  hedgeMode,
			CreateTime: item.CreateTime,
			UpdateTime: item.UpdateTime,
		})
	}
	return res
}

func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		marginMode := ""
		hedgeMode := false

		if !item.OnlyOnePosition {
			hedgeMode = true
		}

		orderType := strings.ToUpper(item.Type)

		tpOrder := false
		slOrder := false

		switch orderType {
		case "TAKE_PROFIT", "TAKE_PROFIT_MARKET":
			tpOrder = true
		case "STOP", "STOP_MARKET":
			slOrder = true
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:         item.Symbol,
			OrderID:        utils.Int64ToString(item.OrderId),
			ClientOrderID:  item.ClientOrderId,
			PositionID:     utils.Int64ToString(item.PositionID),
			Side:           strings.ToUpper(item.Side),
			PositionSide:   strings.ToUpper(item.PositionSide),
			PositionSize:   item.OrigQty,
			ExecutedSize:   item.ExecutedQty,
			Price:          item.Price,
			ExecutedPrice:  item.AvgPrice,
			RealisedProfit: item.Profit,
			Fee:            item.Commission,
			Type:           orderType,
			Leverage:       strings.Replace(item.Leverage, "X", "", 1),
			Status:         strings.ToUpper(item.Status),
			HedgeMode:      hedgeMode,
			MarginMode:     marginMode,
			CreateTime:     item.Time,
			UpdateTime:     item.UpdateTime,
			TpOrder:        tpOrder,
			SlOrder:        slOrder,
		})
	}

	return out
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		mMode := "cross"
		if item.Isolated {
			mMode = "isolated"
		}
		out = append(out, entity.Futures_PositionsHistory{
			Symbol:              item.Symbol,
			PositionID:          item.PositionId,
			PositionSide:        strings.ToUpper(item.PositionSide),
			PositionAmt:         item.PositionAmt,
			ExecutedPositionAmt: item.ClosePositionAmt,
			AvgPrice:            item.AvgPrice,
			ExecutedAvgPrice:    item.AvgClosePrice,
			RealisedProfit:      item.RealisedProfit,
			Fee:                 item.PositionCommission,
			Funding:             item.TotalFunding,
			MarginMode:          mMode,
			Leverage:            utils.Int64ToString(item.Leverage),
			CreateTime:          item.OpenTime,
			UpdateTime:          item.UpdateTime,
		})
	}
	return out
}

func (c *futures_converts) convertListenKey(in futures_listenKey) (out entity.Futures_ListenKey) {
	out.ListenKey = in.ListenKey
	return out
}
