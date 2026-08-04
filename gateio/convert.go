package gateio

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

	out.UID = fmt.Sprintf("%d", in.User_id)
	out.IP = strings.Join(in.Ip_whitelist, ",")
	out.PermSpot = true
	out.PermFutures = true
	out.CanRead = true
	out.CanTrade = true
	return out
}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		if item.Trade_status == "tradable" {
			item.Trade_status = "LIVE"
		}
		rec := entity.Spot_InstrumentsInfo{
			Symbol:         item.ID,
			Base:           item.Base,
			Quote:          item.Quote,
			State:          item.Trade_status,
			MinQty:         item.Min_base_amount,
			MinNotional:    item.Min_quote_amount,
			PricePrecision: fmt.Sprintf("%d", item.Precision),
			SizePrecision:  fmt.Sprintf("%d", item.Amount_precision),
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
			Asset: item.Currency,
			// Balance: item.Available,
			Balance: utils.FloatToStringAll(utils.StringToFloat(item.Available) + utils.StringToFloat(item.Locked)),
			Locked:  item.Locked,
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
			Symbol:        item.Currency_pair,
			OrderID:       item.ID,
			ClientOrderID: item.Text,
			Side:          strings.ToUpper(item.Side),
			Size:          item.Amount,
			Price:         item.Price,
			ExecutedSize:  item.Filled_amount,
			ExecutedPrice: item.Avg_deal_price,
			Fee:           item.Fee,
			FeeAsset:      item.Fee_currency,
			Type:          strings.ToUpper(item.Type),
			Status:        strings.ToUpper(item.Finish_as),
			CreateTime:    item.Create_time,
			UpdateTime:    item.Update_time,
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {

	out = append(out, entity.PlaceOrder{
		OrderID:       in.ID,
		ClientOrderID: in.Text,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertOrderList(in []spot_orderList) (out []entity.Spot_OrdersList) {
	if len(in) == 0 {
		return out
	}
	for _, i := range in {
		for _, item := range i.Orders {

			out = append(out, entity.Spot_OrdersList{
				Symbol:        item.Currency_pair,
				OrderID:       item.ID,
				ClientOrderID: item.Text,
				Side:          strings.ToUpper(item.Side),
				Size:          item.Amount,
				Price:         item.Price,
				ExecutedSize:  item.Filled_amount,
				Type:          strings.ToUpper(item.Type),
				Status:        strings.ToUpper(item.Status),
				CreateTime:    item.Create_time,
				UpdateTime:    item.Update_time,
			})
		}
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
		if item.Status == "trading" {
			item.Status = "LIVE"
		}
		if item.In_delisting {
			item.Status = "OFF"
		}

		base := ""
		quote := ""

		sp := strings.Split(item.Name, "_")
		if len(sp) == 2 {
			base = sp[0]
			quote = sp[1]
		}

		rec := entity.Futures_InstrumentsInfo{
			Symbol:         item.Name,
			Base:           base,
			Quote:          quote,
			MinQty:         fmt.Sprintf("%d", item.Order_size_min),
			PricePrecision: utils.GetPrecisionFromStr(item.Mark_price_round),
			SizePrecision:  "0",
			MaxLeverage:    item.Leverage_max,
			State:          item.Status,
			IsSizeContract: true,
			Multiplier:     "1",
			// Multiplier:     item.Quanto_multiplier,
			ContractSize: item.Quanto_multiplier,
		}
		out = append(out, rec)
	}
	return
}

func (c *futures_converts) convertBalance(in futures_Balance) (out []entity.FuturesBalance) {

	out = append(out, entity.FuturesBalance{
		Asset:   in.Currency,
		Balance: in.Total,
		// Equity:  in.Equity,
		Available:        in.Available,
		UnrealizedProfit: in.Unrealised_pnl,
	})
	return out
}

func (c *futures_converts) convertLeverage(in futures_leverage) (out entity.Futures_Leverage) {

	out.Symbol = in.Currency_pair
	// if in.Сontract != "" {
	// 	out.Symbol = in.Сontract
	// }
	out.Leverage = in.Leverage
	// if utils.StringToInt(in.Leverage) == 0 {
	// 	out.Leverage = in.Cross_leverage_limit
	// }
	return out

	// out.Symbol = in.Currency_pair
	// if in.Сontract != "" {
	// 	out.Symbol = in.Сontract
	// }
	// out.Leverage = in.Leverage
	// if utils.StringToInt(in.Leverage) == 0 {
	// 	out.Leverage = in.Cross_leverage_limit
	// }
	// return out
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrder_Response) (out []entity.PlaceOrder) {

	out = append(out, entity.PlaceOrder{
		OrderID:       fmt.Sprintf("%d", in.ID),
		ClientOrderID: in.Text,
		Ts:            int64(in.Create_time),
	})
	return out
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {
		if item.Size == 0 {
			continue
		}
		positionSide := "LONG"
		if item.Mode == "single" {
			if item.Size < 0 {
				positionSide = "SHORT"
			}
		} else {
			positionSide = strings.ToUpper(item.Mode)
			spt := strings.Split(positionSide, "_")
			if len(spt) > 1 {
				positionSide = spt[1]
			}
		}
		leverage := item.Cross_leverage_limit
		hedgeMode := false
		marginMode := string(entity.MarginModeTypeCross)

		if item.Leverage != "0" {
			marginMode = string(entity.MarginModeTypeIsolated)
			leverage = item.Leverage
		}

		if item.Mode != "single" {
			hedgeMode = true
		}

		if item.Size < 0 {
			item.Size = 0 - item.Size
		}

		res = append(res, entity.Futures_Positions{
			Symbol:       item.Contract,
			PositionSide: positionSide,
			PositionSize: fmt.Sprintf("%d", item.Size),
			Leverage:     leverage,
			// PositionID:       item.PosID,
			EntryPrice:       item.Entry_price,
			MarkPrice:        item.Mark_price,
			UnRealizedProfit: item.Unrealised_pnl,
			RealizedProfit:   item.Realised_pnl,
			Notional:         item.Value,
			HedgeMode:        hedgeMode,
			MarginMode:       marginMode,
			CreateTime:       item.Open_time,
			UpdateTime:       item.Update_time,
		})
	}
	return res
}

func (c *futures_converts) convertOrderList(answ []futures_orderList) (res []entity.Futures_OrdersList) {
	for _, item := range answ {
		positionSide := "LONG"
		side := "BUY"
		if item.Size < 0 {
			positionSide = "SHORT"
			side = "SELL"

		}
		// if item.PosSide == "net" {
		// 	if strings.ToUpper(item.Side) == "SELL" {
		// 		positionSide = "SHORT"
		// 	}
		// } else {
		// 	positionSide = strings.ToUpper(item.PosSide)
		// }

		if item.Size < 0 {
			item.Size = 0 - item.Size
		}

		res = append(res, entity.Futures_OrdersList{
			Symbol:        item.Contract,
			OrderID:       fmt.Sprintf("%d", item.ID),
			ClientOrderID: item.Text,
			// PositionID: ,
			Side:         side,
			PositionSide: positionSide,
			PositionSize: fmt.Sprintf("%d", item.Size),
			// ExecutedSize: utils.Int64ToString(item.Left),
			Price: item.Price,
			// Leverage:      item.Lever,
			Type:       "LIMIT",
			Status:     strings.ToUpper(item.Status),
			CreateTime: int64(item.Create_time),
			UpdateTime: int64(item.Update_time),
		})
	}
	return res
}

func (c *futures_converts) convertOrdersHistory(answ []futures_ordersHistory_Response) (res []entity.Futures_OrdersHistory) {
	for _, item := range answ {
		positionSide := "LONG"
		side := "BUY"
		if item.Size < 0 {
			positionSide = "SHORT"
			side = "SELL"

		}
		// if item.PosSide == "net" {
		// 	if strings.ToUpper(item.Side) == "SELL" {
		// 		positionSide = "SHORT"
		// 	}
		// } else {
		// 	positionSide = strings.ToUpper(item.PosSide)
		// }

		if item.Size < 0 {
			item.Size = 0 - item.Size
		}

		res = append(res, entity.Futures_OrdersHistory{
			Symbol:        item.Contract,
			OrderID:       fmt.Sprintf("%d", item.ID),
			ClientOrderID: item.Text,
			// PositionID: ,
			Side:          side,
			PositionSide:  positionSide,
			PositionSize:  fmt.Sprintf("%d", item.Size),
			ExecutedSize:  utils.Int64ToString(item.Size),
			Price:         item.Price,
			ExecutedPrice: item.Fill_price,
			Fee:           utils.FloatToStringAll(utils.StringToFloat(item.Mkfr) + utils.StringToFloat(item.Tkfr)),
			// Leverage:      item.Lever,
			// Type:       "LIMIT",
			Status:     strings.ToUpper(item.Status),
			CreateTime: int64(item.Create_time),
			UpdateTime: int64(item.Update_time),
		})
	}
	return res
}

func (c *futures_converts) convertPositionsHistory(in []futures_PositionsHistory_Response) (out []entity.Futures_PositionsHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		avgPrice := item.Long_price
		executedAvgPrice := item.Short_price
		if strings.ToUpper(item.Side) == "SHORT" {
			avgPrice = item.Short_price
			executedAvgPrice = item.Long_price
		}
		// mMode := "cross"
		// if item.Isolated {
		// 	mMode = "isolated"
		// }

		out = append(out, entity.Futures_PositionsHistory{
			Symbol: item.Contract,
			// PositionID:          item.PositionId,
			PositionSide:        strings.ToUpper(item.Side),
			PositionAmt:         item.Max_size,
			ExecutedPositionAmt: item.Accum_size,
			AvgPrice:            avgPrice,
			ExecutedAvgPrice:    executedAvgPrice,
			RealisedProfit:      item.Pnl,
			Fee:                 item.Pnl_fee,
			Funding:             item.Pnl_fund,
			// MarginMode:          mMode,
			Leverage:   item.Leverage,
			CreateTime: item.First_open_time * 1000,
			UpdateTime: item.Time * 1000,
		})
	}
	return out
}
