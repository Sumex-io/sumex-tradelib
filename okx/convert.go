package okx

import (
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================
type account_converts struct{}

func (c *account_converts) convertAccountInfo(in accountInfo) (out entity.AccountInformation) {

	out.UID = in.UID
	out.Label = in.Label
	out.IP = in.Ip
	out.PermSpot = true

	if strings.Contains(in.Perm, "read") {
		out.CanRead = true
	}

	if strings.Contains(in.Perm, "trade") {
		out.CanTrade = true
	}

	if in.PosMode == "long_short_mode" {
		// out.HedgeMode = true
	}

	if in.AcctLv != "1" {
		out.PermFutures = true
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

		priceP := utils.GetPrecisionFromStr(item.TickSz)
		sizeP := utils.GetPrecisionFromStr(item.LotSz)

		out = append(out, entity.Spot_InstrumentsInfo{
			Symbol: item.InstId,
			Base:   item.BaseCcy,
			Quote:  item.QuoteCcy,
			MinQty: item.MinSz,
			// MinNotional: item,
			PricePrecision: priceP,
			SizePrecision:  sizeP,
			State:          strings.ToUpper(item.State),
		})
	}
	return out
}

func (c *spot_converts) convertBalance(in []spot_Balance) (out []entity.AssetsBalance) {
	if len(in) == 0 {
		return out
	}

	for _, i := range in {
		for _, item := range i.Details {
			out = append(out, entity.AssetsBalance{
				Asset:   item.Ccy,
				Balance: item.Eq,
				Locked:  item.FrozenBal,
			})
		}
	}
	return out
}

func (c *spot_converts) convertOrdersHistory(in []spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        item.InstId,
			OrderID:       item.OrdId,
			ClientOrderID: item.ClOrdId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Sz,
			Price:         item.Px,
			ExecutedSize:  item.AccFillSz,
			ExecutedPrice: item.AvgPx,
			Fee:           item.Fee,
			FeeAsset:      item.FeeCcy,
			Type:          strings.ToUpper(item.OrdType),
			Status:        strings.ToUpper(item.State),
			CreateTime:    utils.StringToInt64(item.CTime),
			UpdateTime:    utils.StringToInt64(item.UTime),
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrder(in []placeOrder_Response) (out []entity.PlaceOrder) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		out = append(out, entity.PlaceOrder{
			OrderID:       item.OrdId,
			ClientOrderID: item.ClOrdId,
			Ts:            time.Now().UTC().UnixMilli(),
		})
	}
	return out
}

func (c *spot_converts) convertOrderList(answ []spot_orderList) (res []entity.Spot_OrdersList) {
	for _, item := range answ {
		res = append(res, entity.Spot_OrdersList{
			Symbol:        item.InstId,
			OrderID:       item.OrdId,
			ClientOrderID: item.ClOrdId,
			Size:          item.Sz,
			ExecutedSize:  item.FillSz,
			Side:          strings.ToUpper(item.Side),
			Price:         item.Px,
			Type:          strings.ToUpper(item.OrdType),
			Status:        strings.ToUpper(item.State),
			CreateTime:    utils.StringToInt64(item.CTime),
			UpdateTime:    utils.StringToInt64(item.UTime),
		})
	}
	return res
}

// ===============FUTURES=================

type futures_converts struct{}

func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol:         item.InstId,
			Base:           item.CtValCcy,
			Quote:          item.SettleCcy,
			MinQty:         item.MinSz,
			PricePrecision: utils.GetPrecisionFromStr(item.TickSz),
			SizePrecision:  utils.GetPrecisionFromStr(item.MinSz),
			MaxLeverage:    item.Lever,
			State:          strings.ToUpper(item.State),
			IsSizeContract: true,
			Multiplier:     item.CtMult,
			ContractSize:   item.CtVal,
		})
	}
	return out
}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, i := range in {
		for _, item := range i.Details {
			out = append(out, entity.FuturesBalance{
				Asset:            item.Ccy,
				Balance:          item.CashBal,
				Equity:           item.Eq,
				Available:        item.AvailBal,
				UnrealizedProfit: item.Upl,
			})
		}
	}
	return out
}

func (c *futures_converts) convertLeverage(in []futures_leverage) (out entity.Futures_Leverage) {
	if len(in) == 0 {
		return out
	} else if len(in) == 1 {
		out.Symbol = in[0].InstId
		out.Leverage = in[0].Lever
		out.LongLeverage = in[0].Lever
		out.ShortLeverage = in[0].Lever
	} else if len(in) == 2 {
		out.Symbol = in[0].InstId
		if in[0].Lever == in[1].Lever {
			out.Leverage = in[0].Lever
		} else if utils.StringToInt64(in[0].Lever) < utils.StringToInt64(in[1].Lever) {
			out.Leverage = in[0].Lever
		} else {
			out.Leverage = in[1].Lever
		}
		for _, item := range in {
			switch strings.ToUpper(item.PosSide) {
			case "LONG":
				out.LongLeverage = item.Lever
			case "SHORT":
				out.ShortLeverage = item.Lever
			}
		}
	}

	return out
}

func (c *futures_converts) convertPlaceOrder(in []placeOrder_Response) (out []entity.PlaceOrder) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		out = append(out, entity.PlaceOrder{
			OrderID:       item.OrdId,
			ClientOrderID: item.ClOrdId,
			Ts:            time.Now().UTC().UnixMilli(),
		})
	}
	return out
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {

		if item.Type == "1" || item.Type == "5" {
			continue
		}
		mMode := string(entity.MarginModeTypeCross)
		if item.MgnMode != "cross" {
			mMode = string(entity.MarginModeTypeIsolated)
		}

		if strings.ToUpper(item.PosSide) == "NET" {
			item.PosSide = item.Direction
		}
		out = append(out, entity.Futures_PositionsHistory{
			Symbol:              item.InstId,
			PositionID:          item.PosId,
			PositionSide:        strings.ToUpper(item.PosSide),
			PositionAmt:         item.OpenMaxPos,
			ExecutedPositionAmt: item.CloseTotalPos,
			AvgPrice:            item.OpenAvgPx,
			ExecutedAvgPrice:    item.CloseAvgPx,
			RealisedProfit:      item.RealizedPnl,
			Fee:                 item.Fee,
			Funding:             item.FundingFee,
			Leverage:            item.Lever,
			MarginMode:          mMode,
			CreateTime:          utils.StringToInt64(item.CTime),
			UpdateTime:          utils.StringToInt64(item.UTime),
		})
	}
	return out
}

func (c *futures_converts) convertOrderList(answ []futures_orderList) (res []entity.Futures_OrdersList) {
	for _, item := range answ {
		positionSide := "LONG"
		if item.PosSide == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.PosSide)
		}

		res = append(res, entity.Futures_OrdersList{
			Symbol:        item.InstId,
			OrderID:       item.OrdId,
			ClientOrderID: item.ClOrdId,
			PositionSide:  positionSide,
			Side:          strings.ToUpper(item.Side),
			PositionSize:  item.Sz,
			ExecutedSize:  item.FillSz,
			Price:         item.Px,
			// TpPrice:       tp,
			// SlPrice:       sl,
			Type:       strings.ToUpper(item.OrdType),
			MarginMode: strings.ToUpper(item.TdMode),
			// InstType:      item.InstType,
			Leverage: item.Lever,
			Status:   strings.ToUpper(item.State),
			// IsTpLimit:     b,
			CreateTime: utils.StringToInt64(item.CTime),
			UpdateTime: utils.StringToInt64(item.UTime),
		})
	}
	return res
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {
		positionSide := "LONG"
		if item.PosSide == "net" {
			if utils.StringToFloat(item.Pos) < 0 {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.PosSide)
		}

		hedgeMode := false

		if item.PosSide != "net" {
			hedgeMode = true

		}
		res = append(res, entity.Futures_Positions{
			Symbol:           item.InstID,
			PositionSide:     positionSide,
			PositionSize:     item.Pos,
			Leverage:         item.Lever,
			PositionID:       item.PosID,
			EntryPrice:       item.AvgPx,
			MarkPrice:        item.MarkPx,
			UnRealizedProfit: item.Upl,
			RealizedProfit:   item.RealizedPnl,
			Notional:         item.NotionalUsd,
			HedgeMode:        hedgeMode,
			MarginMode:       strings.ToUpper(item.MgnMode),
			CreateTime:       utils.StringToInt64(item.CTime),
			UpdateTime:       utils.StringToInt64(item.UTime),
		})
	}
	return res
}

// func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {

// 	if len(in) == 0 {
// 		return out
// 	}

// 	for _, item := range in {

// 		hedgeMode := false
// 		if item.PosSide == "net" {
// 			if utils.StringToFloat(item.Sz) < 0 {
// 				item.PosSide = "SHORT"
// 			} else {
// 				item.PosSide = "LONG"
// 			}
// 		} else {
// 			hedgeMode = true
// 		}

// 		out = append(out, entity.Futures_OrdersHistory{
// 			Symbol:         item.InstId,
// 			OrderID:        item.OrdId,
// 			ClientOrderID:  item.ClOrdId,
// 			Side:           strings.ToUpper(item.Side),
// 			PositionSide:   strings.ToUpper(item.PosSide),
// 			PositionSize:   item.Sz,
// 			ExecutedSize:   item.AccFillSz,
// 			Price:          item.Px,
// 			ExecutedPrice:  item.AvgPx,
// 			RealisedProfit: item.Pnl,
// 			Fee:            item.Fee,
// 			FeeAsset:       item.FeeCcy,
// 			Leverage:       item.Lever,
// 			HedgeMode:      hedgeMode,
// 			MarginMode:     strings.ToUpper(item.TdMode),
// 			Type:           strings.ToUpper(item.OrdType),
// 			Status:         strings.ToUpper(item.State),
// 			CreateTime:     utils.StringToInt64(item.CTime),
// 			UpdateTime:     utils.StringToInt64(item.UTime),
// 		})
// 	}
// 	return out

// }
func (c *futures_converts) convertOrdersHistory(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	return convertOrdersHistoryOKX(in)
}

// =======OLD

func convertPlaceOrder(in []placeOrder_Response) (out []entity.PlaceOrder) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		out = append(out, entity.PlaceOrder{
			OrderID:       item.OrdId,
			ClientOrderID: item.ClOrdId,
			Ts:            utils.StringToInt64(item.Ts),
		})
	}
	return out
}

func (c *futures_converts) convertAlgoOrderList(in []futures_algoOrder) (out []entity.Futures_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, it := range in {
		// В algo-ордерах нет обычного ordId; там algoId.
		// Приводим в единый формат: OrderID = algoId, ClientOrderID = algoClOrdId
		price := ""
		// Для conditional TP/SL мы хотим показывать trigger price.
		if it.TpTriggerPx != "" && it.TpTriggerPx != "0" && it.TpTriggerPx != "0.0" {
			price = it.TpTriggerPx
		} else if it.SlTriggerPx != "" && it.SlTriggerPx != "0" && it.SlTriggerPx != "0.0" {
			price = it.SlTriggerPx
		} else if it.TriggerPx != "" {
			price = it.TriggerPx
		}

		// Type: оставляем CONDITIONAL, чтобы отличалось от обычного LIMIT/MARKET
		// (если хочешь, можем писать TP/SL по наличию tpTriggerPx/slTriggerPx)
		istp := false
		issl := false

		typ := "CONDITIONAL"
		if it.TpTriggerPx != "" && it.TpTriggerPx != "0" && it.TpTriggerPx != "0.0" {
			typ = "TP"
			istp = true
		} else if it.SlTriggerPx != "" && it.SlTriggerPx != "0" && it.SlTriggerPx != "0.0" {
			typ = "SL"
			issl = true
		}

		out = append(out, entity.Futures_OrdersList{
			Symbol:        it.InstId,
			OrderID:       it.AlgoId,
			ClientOrderID: it.AlgoClOrdId,
			PositionSide:  strings.ToUpper(it.PosSide),
			Side:          strings.ToUpper(it.Side),
			PositionSize:  it.Sz,
			ExecutedSize:  "0", // pending algo обычно без fillSz (либо он не нужен)
			Price:         price,
			Type:          typ,
			Status:        strings.ToUpper(it.State),
			CreateTime:    strToInt64Safe(it.CTime),
			UpdateTime:    strToInt64Safe(it.UTime),
			TpOrder:       istp,
			SlOrder:       issl,
		})
	}

	return out
}

// локальный помощник (как минимум чтобы не падать)
// если у тебя уже есть такой в utils — можно заменить на utils.StringToInt64
func strToInt64Safe(s string) int64 {
	// OKX обычно отдаёт ms timestamp строкой
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int64(ch-'0')
	}
	return n
}
