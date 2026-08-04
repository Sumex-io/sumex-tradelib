package bybit

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

	out.UID = fmt.Sprintf("%d", in.UserID)
	out.Label = in.Note
	out.IP = strings.Join(in.Ips, ",")
	out.CanRead = true

	if in.ReadOnly == 0 {
		out.CanTrade = true
	}

	for _, item := range in.Permissions.Spot {
		if item == "SpotTrade" {
			out.PermSpot = true
			out.CanTrade = true
			break
		}
	}

	for _, item := range in.Permissions.Derivatives {
		if item == "DerivativesTrade" {
			out.PermFutures = true
			out.CanTrade = true
			break
		}
	}

	for _, item := range in.Permissions.Wallet {
		if item == "AccountTransfer" {
			out.CanTransfer = true
			break
		}
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
		for _, item := range i.Coin {
			out = append(out, entity.AssetsBalance{
				Asset:           item.Coin,
				Balance:         item.WalletBalance,
				Equity:          item.Equity,
				Locked:          item.Locked,
				TotalPositionMM: item.TotalPositionMM,
				TotalPositionIM: item.TotalPositionIM,
				UnrealisedPnl:   item.UnrealisedPnl,
				// Locked: utils.FloatToStringAll(utils.StringToFloat(item.TotalPositionIM) + utils.StringToFloat(item.Locked) + utils.StringToFloat(item.UnrealisedPnl)),
				// Locked: utils.FloatToStringAll(utils.StringToFloat(item.TotalPositionIM) - utils.StringToFloat(item.TotalPositionMM) + utils.StringToFloat(item.UnrealisedPnl)),
			})
		}
	}
	return out
}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		if item.Status == "Trading" {
			item.Status = "LIVE"
		}

		sizeP := utils.GetPrecisionFromStr(item.LotSizeFilter.BasePrecision)
		priceP := utils.GetPrecisionFromStr(item.PriceFilter.TickSize)

		rec := entity.Spot_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseCoin,
			Quote:          item.QuoteCoin,
			MinQty:         item.LotSizeFilter.MinOrderQty,
			MinNotional:    item.LotSizeFilter.MinOrderAmt,
			PricePrecision: priceP,
			SizePrecision:  sizeP,
			State:          item.Status,
		}
		out = append(out, rec)
	}
	return out
}

func (c *spot_converts) convertOrdersHistory(in spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in.List) == 0 {
		return out
	}
	for _, item := range in.List {

		coin, _, _ := firstFeeCoin(item.CumFeeDetail)

		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.OrderLinkId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Qty,
			Price:         item.Price,
			ExecutedSize:  item.CumExecQty,
			ExecutedPrice: item.AvgPrice,
			Fee:           item.CumExecFee,
			FeeAsset:      coin,
			Type:          strings.ToUpper(item.OrderType),
			Status:        strings.ToUpper(item.OrderStatus),
			CreateTime:    utils.StringToInt64(item.CreatedTime),
			UpdateTime:    utils.StringToInt64(item.UpdatedTime),
			// Cursor:        in.NextPageCursor,
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {

	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.OrderLinkId,
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
			OrderID:       item.OrderId,
			ClientOrderID: item.OrderLinkId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Qty,
			Price:         item.Price,
			ExecutedSize:  item.CumExecQty,
			Type:          strings.ToUpper(item.OrderType),
			Status:        strings.ToUpper(item.OrderStatus),
			CreateTime:    utils.StringToInt64(item.CreatedTime),
			UpdateTime:    utils.StringToInt64(item.UpdatedTime),
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
		if item.Status == "Trading" {
			item.Status = "LIVE"
		}

		rec := entity.Futures_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseCoin,
			Quote:          item.QuoteCoin,
			MinQty:         item.LotSizeFilter.MinOrderQty,
			MinNotional:    item.LotSizeFilter.MinNotionalValue,
			PricePrecision: item.PriceScale,
			SizePrecision:  utils.GetPrecisionFromStr(item.LotSizeFilter.MinOrderQty),
			MaxLeverage:    item.LeverageFilter.MaxLeverage,
			State:          item.Status,
		}
		out = append(out, rec)
	}
	return
}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}
	for _, i := range in {
		for _, item := range i.Coin {
			if utils.StringToFloat(item.WalletBalance) == 0 {
				continue
			}
			out = append(out, entity.FuturesBalance{
				Asset:   item.Coin,
				Balance: item.WalletBalance,
				Equity:  item.Equity,
				// Available:        utils.FloatToStringAll(utils.StringToFloat(item.Equity) - utils.StringToFloat(item.TotalPositionIM) - utils.StringToFloat(item.Locked)),
				Available:        utils.FloatToStringAll(utils.StringToFloat(item.WalletBalance) - utils.StringToFloat(item.TotalPositionIM) - utils.StringToFloat(item.Locked)),
				UnrealizedProfit: item.UnrealisedPnl,
			})
		}
	}
	return out
}

func (c *futures_converts) convertLeverage(in futures_leverage) (out entity.Futures_Leverage) {

	out.Symbol = in.Symbol
	out.Leverage = in.Leverage
	return out
}

func (c *futures_converts) convertLeverage_Extra(in []futures_leverage) (out entity.Futures_Leverage) {
	if len(in) == 0 {
		return out
	}
	out.Symbol = in[0].Symbol

	for _, item := range in {
		switch item.PositionIdx {
		case 0:
			out.Leverage = item.Leverage
		case 1:
			out.LongLeverage = item.Leverage
		case 2:
			out.ShortLeverage = item.Leverage
		}
	}
	return out
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		positionSide := ""
		switch item.PositionIdx {
		case 0:
			if strings.ToUpper(item.Side) == "BUY" {
				positionSide = "SHORT"
			} else {
				positionSide = "LONG"
			}
		case 1:
			positionSide = "LONG"
		case 2:
			positionSide = "SHORT"
		}
		out = append(out, entity.Futures_PositionsHistory{
			Symbol:              item.Symbol,
			PositionID:          item.OrderId,
			PositionSide:        positionSide,
			PositionAmt:         item.Qty,
			ExecutedPositionAmt: item.ClosedSize,
			AvgPrice:            item.AvgEntryPrice,
			ExecutedAvgPrice:    item.AvgExitPrice,
			RealisedProfit:      item.ClosedPnl,
			Fee:                 utils.FloatToStringAll((0 - utils.StringToFloat(item.OpenFee)) + (0 - utils.StringToFloat(item.CloseFee))),
			// Funding:             item.TotalFunding,
			// MarginMode:          mMode,
			Leverage:   item.Leverage,
			CreateTime: utils.StringToInt64(item.CreatedTime),
			UpdateTime: utils.StringToInt64(item.UpdatedTime),
		})
	}
	return out
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrder_Response) (out []entity.PlaceOrder) {

	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.OrderLinkId,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertOrderList(in []futures_orderList) (out []entity.Futures_OrdersList) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {

		positionSide := ""
		switch item.PositionIdx {
		case 0:
			if strings.ToUpper(item.Side) == "BUY" {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}
		case 1:
			positionSide = "LONG"
		case 2:
			positionSide = "SHORT"
		}
		price := item.Price
		if price == "" || price == "0" {
			price = item.TriggerPrice
		}

		istp := false
		issl := false
		if item.StopOrderType != "" && (strings.ToLower(item.StopOrderType) == "stoploss" || strings.ToLower(item.StopOrderType) == "partialstoploss") {
			issl = true
		}
		if item.StopOrderType != "" && (strings.ToLower(item.StopOrderType) == "takeprofit" || strings.ToLower(item.StopOrderType) == "partialtakeprofit") {
			istp = true
		}
		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.OrderLinkId,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  positionSide,
			PositionSize:  item.Qty,
			ExecutedSize:  item.CumExecQty,
			Price:         price,
			Type:          strings.ToUpper(item.OrderType),
			Status:        strings.ToUpper(item.OrderStatus),
			CreateTime:    utils.StringToInt64(item.CreatedTime),
			UpdateTime:    utils.StringToInt64(item.UpdatedTime),
			TpOrder:       istp,
			SlOrder:       issl,
		})
	}
	return out
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {

		if utils.StringToFloat(item.Size) == 0 {
			continue
		}

		marginMode := ""
		hedgeMode := false
		positionSide := ""

		if item.PositionIdx != 0 {
			hedgeMode = true
		}

		switch item.PositionIdx {
		case 0:
			if strings.ToUpper(item.Side) == "BUY" {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}
		case 1:
			positionSide = "LONG"
		case 2:
			positionSide = "SHORT"
		}

		res = append(res, entity.Futures_Positions{
			Symbol:       item.Symbol,
			PositionSide: positionSide,
			// PositionID:   item.PositionId,
			PositionSize: item.Size,
			EntryPrice:   item.AvgPrice,
			MarkPrice:    item.MarkPrice,
			// InitialMargin:    item.Initial_margin,
			UnRealizedProfit: item.UnrealisedPnl,
			RealizedProfit:   item.CurRealisedPnl,
			Notional:         item.PositionValue,
			// MarginRatio:      item.Maintenance_rate,
			Leverage:   item.Leverage,
			MarginMode: marginMode,
			HedgeMode:  hedgeMode,
			CreateTime: utils.StringToInt64(item.CreatedTime),
			UpdateTime: utils.StringToInt64(item.UpdatedTime),
		})
	}
	return res
}

func (c *futures_converts) convertOrdersHistory(in futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in.List) == 0 {
		return out
	}

	for _, item := range in.List {
		status := strings.ToUpper(item.OrderStatus)
		if status != "FILLED" {
			continue
		}

		positionSide := ""
		switch item.PositionIdx {
		case 0:
			if strings.ToUpper(item.Side) == "BUY" {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}
		case 1:
			positionSide = "LONG"
		case 2:
			positionSide = "SHORT"
		}

		price := item.Price
		if price == "" || price == "0" {
			price = item.TriggerPrice
		}

		stopOrderType := strings.ToLower(item.StopOrderType)
		createType := strings.ToLower(item.CreateType)

		istp := false
		issl := false

		switch stopOrderType {
		case "takeprofit", "partialtakeprofit":
			istp = true
		case "stoploss", "partialstoploss":
			issl = true
		}

		if !istp && !issl {
			switch createType {
			case "createbytakeprofit", "createbypartialtakeprofit":
				istp = true
			case "createbystoploss", "createbypartialstoploss":
				issl = true
			}
		}

		coin, _, _ := firstFeeCoin(item.CumFeeDetail)

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.OrderLinkId,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  positionSide,
			PositionSize:  item.Qty,
			Price:         price,
			ExecutedSize:  item.CumExecQty,
			ExecutedPrice: item.AvgPrice,
			Fee:           fmt.Sprintf("-%s", item.CumExecFee),
			FeeAsset:      coin,
			Type:          strings.ToUpper(item.OrderType),
			Status:        status,
			CreateTime:    utils.StringToInt64(item.CreatedTime),
			UpdateTime:    utils.StringToInt64(item.UpdatedTime),
			TpOrder:       istp,
			SlOrder:       issl,
			// Cursor:      in.NextPageCursor,
		})
	}

	return out
}

func firstFeeCoin(m map[string]string) (coin, fee string, ok bool) {
	for k, v := range m {
		return k, v, true
	}
	return "", "", false
}

func (c *futures_converts) convertExecutionsHistory(in futures_executionsHistory_Response) (out []entity.Futures_ExecutionsHistory) {

	if len(in.List) == 0 {
		return out
	}
	for _, item := range in.List {

		if utils.StringToFloat(item.ExecQty) < utils.StringToFloat(item.OrderQty) {
			continue
		}
		out = append(out, entity.Futures_ExecutionsHistory{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.OrderLinkId,
			Side:          strings.ToUpper(item.Side),
			PositionSize:  item.OrderQty,
			Price:         item.OrderPrice,
			ExecutedSize:  item.ExecQty,
			ExecutedPrice: item.ExecPrice,
			Fee:           fmt.Sprintf("-%s", item.ExecFee),
			Type:          strings.ToUpper(item.OrderType),
			Status:        "FILLED",
			// CreateTime:    utils.StringToInt64(item.CreatedTime),
			UpdateTime: utils.StringToInt64(item.ExecTime),
		})
	}
	return out
}
