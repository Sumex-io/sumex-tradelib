package bitget

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================
type account_converts struct{}

func (c *account_converts) convertAccountInfo(in accountInfo) (out entity.AccountInformation) {

	out.UID = in.UserId
	out.IP = in.Ips

	for _, item := range in.Authorities {
		switch item {
		case "coor":
			out.PermFutures = true
			out.CanRead = true
		case "stor":
			out.PermSpot = true
			out.CanRead = true
		case "coow":
			out.PermFutures = true
			out.CanTrade = true
			out.CanRead = true
		case "stow":
			out.PermSpot = true
			out.CanTrade = true
			out.CanRead = true
		}
	}
	return out
}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {

		if item.Status == "online" {
			item.Status = "LIVE"
		}

		rec := entity.Spot_InstrumentsInfo{
			Symbol: item.Symbol,
			Base:   item.BaseCoin,
			Quote:  item.QuoteCoin,
			// MinQty:         utils.FloatToStringAll(item.MinQty),
			MinNotional:    item.MinTradeUSDT,
			PricePrecision: item.PricePrecision,
			SizePrecision:  item.QuantityPrecision,
			State:          item.Status,
		}
		out = append(out, rec)
	}
	return
}

func (c *spot_converts) convertBalance(in []spot_Balance) (out []entity.AssetsBalance) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		out = append(out, entity.AssetsBalance{
			Asset: item.Coin,
			// Balance: item.Available,
			Balance: utils.FloatToStringAll(utils.StringToFloat(item.Available) + utils.StringToFloat(item.Locked) + utils.StringToFloat(item.Frozen)),
			Locked:  utils.FloatToStringAll(utils.StringToFloat(item.Frozen) + utils.StringToFloat(item.Locked)),
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {

	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.ClientOid,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertOrdersHistory(in []spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	type feeDetailJson struct {
		NewFees struct {
			T float64 `json:"t"`
		} `json:"newFees"`
	}

	for _, item := range in {
		executedQty := item.BaseVolume
		// if strings.ToUpper(item.OrderType) == "MARKET" {
		// 	executedQty = item.QuoteVolume
		// }
		fee := "0"
		answ := feeDetailJson{}
		if item.FeeDetail != "" {
			err := json.Unmarshal([]byte(item.FeeDetail), &answ)
			if err == nil {
				fee = utils.FloatToStringAll(answ.NewFees.T)
			} else {
				log.Println("=Err convertOrdersHistory=", err)
			}
		}

		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOid,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Size,
			Price:         item.Price,
			ExecutedSize:  executedQty,
			ExecutedPrice: item.PriceAvg,
			Fee:           fee,
			Type:          strings.ToUpper(item.OrderType),
			Status:        strings.ToUpper(item.Status),
			// Status:     "FILLED",
			CreateTime: utils.StringToInt64(item.CTime),
			UpdateTime: utils.StringToInt64(item.UTime),
		})
	}
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
			ClientOrderID: item.ClientOid,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Size,
			Price:         item.PriceAvg,
			ExecutedSize:  item.BaseVolume,
			Type:          strings.ToUpper(item.OrderType),
			Status:        strings.ToUpper(item.Status),
			CreateTime:    utils.StringToInt64(item.CTime),
			UpdateTime:    utils.StringToInt64(item.UTime),
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
		if item.SymbolStatus == "normal" {
			item.SymbolStatus = "LIVE"
		}

		rec := entity.Futures_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseCoin,
			Quote:          item.QuoteCoin,
			MinQty:         item.MinTradeNum,
			MinNotional:    item.MinTradeUSDT,
			PricePrecision: item.PricePlace,
			SizePrecision:  item.VolumePlace,
			MaxLeverage:    item.MaxLever,
			Multiplier:     item.SizeMultiplier,
			State:          item.SymbolStatus,
		}
		out = append(out, rec)
	}
	return
}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.FuturesBalance{
			Asset:            item.MarginCoin,
			Balance:          item.AccountEquity,
			Equity:           item.AccountEquity,
			Available:        item.UnionAvailable,
			UnrealizedProfit: item.UnrealizedPL,
		})

		// if len(i.AssetList) == 0 {
		// 	continue
		// }
		// for _, item := range i.AssetList {
		// out = append(out, entity.FuturesBalance{
		// 	Asset:   item.Coin,
		// 	Balance: item.Balance,
		// 	// Equity:           item.AccountEquity,
		// 	Available:        item.Available,
		// 	UnrealizedProfit: i.UnrealizedPL,
		// })
		// }
	}
	return out
}

func (c *futures_converts) convertLeverage(in futures_leverage) (out entity.Futures_Leverage) {

	out.Symbol = in.Symbol

	if in.MarginMode == "crossed" {
		out.Leverage = utils.Int64ToString(in.CrossedMarginLeverage)
		out.LongLeverage = utils.Int64ToString(in.CrossedMarginLeverage)
		out.ShortLeverage = utils.Int64ToString(in.CrossedMarginLeverage)
	} else {
		out.LongLeverage = utils.Int64ToString(in.IsolatedLongLever)
		out.ShortLeverage = utils.Int64ToString(in.IsolatedShortLever)
	}
	return out
}

func (c *futures_converts) convertLeverage_extra(in futures_leverage_extra) (out entity.Futures_Leverage) {

	out.Symbol = in.Symbol

	if in.MarginMode == "crossed" {
		out.Leverage = in.CrossedMarginLeverage
		out.LongLeverage = in.CrossedMarginLeverage
		out.ShortLeverage = in.CrossedMarginLeverage
	} else {
		out.LongLeverage = in.IsolatedLongLever
		out.ShortLeverage = in.IsolatedShortLever
	}
	return out
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.ClientOid,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		mMode := string(entity.MarginModeTypeCross)
		if item.MarginMode == "isolated" {
			mMode = string(entity.MarginModeTypeIsolated)
		}

		fee := utils.StringToFloat(item.OpenFee) + utils.StringToFloat(item.CloseFee)

		out = append(out, entity.Futures_PositionsHistory{
			Symbol:              item.Symbol,
			PositionID:          item.PositionId,
			PositionSide:        strings.ToUpper(item.HoldSide),
			PositionAmt:         item.OpenTotalPos,
			ExecutedPositionAmt: item.CloseTotalPos,
			AvgPrice:            item.OpenAvgPrice,
			ExecutedAvgPrice:    item.CloseAvgPrice,
			RealisedProfit:      item.Pnl,
			Fee:                 utils.FloatToStringAll(fee),
			Funding:             item.TotalFunding,
			MarginMode:          mMode,
			CreateTime:          utils.StringToInt64(item.CTime),
			UpdateTime:          utils.StringToInt64(item.UTime),
		})
	}
	return out
}

func (c *futures_converts) convertOrderList(in futures_orderList) (out []entity.Futures_OrdersList) {
	if len(in.Orders) == 0 {
		return out
	}

	for _, item := range in.Orders {

		side := strings.ToUpper(item.Side)
		positionSide := "LONG"

		if strings.ToUpper(item.PositionSide) == "LONG" && strings.ToUpper(item.TradeSide) == "CLOSE" {
			side = "SELL"
		} else if strings.ToUpper(item.PositionSide) == "SHORT" && strings.ToUpper(item.TradeSide) == "CLOSE" {
			side = "BUY"
		}
		if item.PositionSide == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.PositionSide)
		}

		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			// PositionID:    fmt.Sprintf("%d", item.PositionID),
			Side:         side,
			PositionSide: positionSide,
			Type:         strings.ToUpper(item.Type),
			PositionSize: item.Size,
			ExecutedSize: item.BaseVolume,
			Price:        item.Price,
			Leverage:     item.Leverage,
			Status:       strings.ToUpper(item.Status),
			CreateTime:   utils.StringToInt64(item.Time),
			UpdateTime:   utils.StringToInt64(item.UpdateTime),
		})
	}
	return out
}

func (c *futures_converts) convertPlanOrderList(in futures_planOrderList) (out []entity.Futures_OrdersList) {
	if len(in.Orders) == 0 {
		return out
	}

	for _, item := range in.Orders {
		side := strings.ToUpper(item.Side)
		positionSide := "LONG"

		// Логика close/open похожа на вашу для обычных ордеров
		if strings.ToUpper(item.PosSide) == "LONG" && strings.ToUpper(item.TradeSide) == "CLOSE" {
			side = "SELL"
		} else if strings.ToUpper(item.PosSide) == "SHORT" && strings.ToUpper(item.TradeSide) == "CLOSE" {
			side = "BUY"
		}

		if item.PosSide == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		} else if item.PosSide != "" {
			positionSide = strings.ToUpper(item.PosSide)
		}

		// Для TP/SL разумнее показать triggerPrice как Price (в едином интерфейсе “цена”)
		price := item.TriggerPrice
		if price == "" {
			// fallback
			price = item.ExecutePrice
		}

		ordType := strings.ToUpper(item.OrderType)
		if ordType == "" {
			// чтобы хотя бы что-то было
			ordType = "TRIGGER"
		}

		status := strings.ToUpper(item.Status)
		if status == "" {
			status = "LIVE"
		}

		istp := false
		issl := false

		if strings.ToLower(item.PlanType) == "loss_plan" {
			issl = true
		}

		if strings.ToLower(item.PlanType) == "profit_plan" {
			istp = true
		}

		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOid,
			Side:          side,
			PositionSide:  positionSide,
			Type:          ordType,
			PositionSize:  item.Size,
			Price:         price,
			Status:        status,
			CreateTime:    utils.StringToInt64(item.CTime),
			UpdateTime:    utils.StringToInt64(item.UTime),
			TpOrder:       istp,
			SlOrder:       issl,
		})
	}

	return out
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {

		marginMode := string(entity.MarginModeTypeIsolated)
		hedgeMode := false

		if item.PosMode != "one_way_mode" {
			hedgeMode = true
		}

		if item.MarginMode == "crossed" {
			marginMode = string(entity.MarginModeTypeCross)
		}
		res = append(res, entity.Futures_Positions{
			Symbol:       item.Symbol,
			PositionSide: strings.ToUpper(item.HoldSide),
			// PositionID:   item.PositionId,
			PositionSize: item.Total,
			EntryPrice:   item.OpenPriceAvg,
			MarkPrice:    item.MarkPrice,
			// InitialMargin:    item.Initial_margin,
			UnRealizedProfit: item.UnrealizedPL,
			RealizedProfit:   item.AchievedProfits,
			// Notional:         item.PositionValue,
			// MarginRatio:      item.Maintenance_rate,
			Leverage:   item.Leverage,
			MarginMode: marginMode,
			HedgeMode:  hedgeMode,
			CreateTime: utils.StringToInt64(item.CreateTime),
			UpdateTime: utils.StringToInt64(item.UpdateTime),
		})
	}
	return res
}

func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		marginMode := string(entity.MarginModeTypeIsolated)
		hedgeMode := false

		if item.PosMode != "one_way_mode" {
			hedgeMode = true
		}

		if item.MarginMode == "crossed" {
			marginMode = string(entity.MarginModeTypeCross)
		}

		// side := strings.ToUpper(item.Side)
		positionSide := "LONG"

		// if strings.ToUpper(item.PosSide) == "LONG" && strings.ToUpper(item.TradeSide) == "CLOSE" {
		// 	side = "SELL"
		// } else if strings.ToUpper(item.PosSide) == "SHORT" && strings.ToUpper(item.TradeSide) == "CLOSE" {
		// 	side = "BUY"
		// }
		if item.PosSide == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.PosSide)
		}
		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOid,
			// PositionID:     utils.Int64ToString(item.PositionID),
			Side:           strings.ToUpper(item.Side),
			PositionSide:   strings.ToUpper(positionSide),
			PositionSize:   item.Size,
			ExecutedSize:   item.BaseVolume,
			Price:          item.Price,
			ExecutedPrice:  item.PriceAvg,
			RealisedProfit: item.TotalProfits,
			Fee:            item.Fee,
			Type:           strings.ToUpper(item.OrderType),
			Leverage:       item.Leverage,
			Status:         strings.ToUpper(item.Status),
			HedgeMode:      hedgeMode,
			MarginMode:     marginMode,
			CreateTime:     utils.StringToInt64(item.CTime),
			UpdateTime:     utils.StringToInt64(item.UTime),
		})
	}
	return out
}
