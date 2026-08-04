package mexc

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
	out.CanRead = true
	out.CanTrade = in.СanTrade
	out.CanTransfer = in.CanWithdraw

	for _, item := range in.Permissions {
		if item == "SPOT" {
			out.PermSpot = true
		} else if item == "FUTURES" {
			out.PermFutures = true
		}
	}
	return out
}

func (c *account_converts) convertFundingAccountBalance(in fundingBalance) (out []entity.FundingAccountBalance) {
	if len(in.Assets) == 0 {
		return out
	}

	for _, item := range in.Assets {
		out = append(out, entity.FundingAccountBalance{
			Asset:            item.Asset,
			Balance:          utils.FloatToStringAll(item.Free),
			AvailableBalance: utils.FloatToStringAll(item.Free),
			FrozenBalance:    utils.FloatToStringAll(item.Locked),
		})
	}

	return out
}

func (c *account_converts) convertTradingAccountBalance(in []tradingBalance) (out []entity.TradingAccountBalance) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		r := entity.TradingAccountBalance{
			TotalEquity:      item.TotalEq,
			AvailableEquity:  item.AvailEq,
			NotionalUsd:      item.NotionalUsd,
			UnrealizedProfit: item.Upl,
			UpdateTime:       utils.StringToInt64(item.UTime),
		}
		// for _, item := range item.Details {
		// 	r.Assets = append(r.Assets, entity.TradingAccountBalanceDetails{
		// 		Asset:            item.Ccy,
		// 		Balance:          item.CashBal,
		// 		EquityBalance:    item.Eq,
		// 		AvailableBalance: item.AvailBal,
		// 		AvailableEquity:  item.AvailEq,
		// 		UnrealizedProfit: item.Upl,
		// 	})
		// }

		out = append(out, r)
	}
	return out
}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in.Symbols) == 0 {
		return out
	}
	for _, item := range in.Symbols {
		state := "OTHER"
		if item.Status == "1" {
			state = "LIVE"
		} else if item.Status == "3" {
			state = "OFF"
		} else if item.Status == "2" {
			state = "SUSPENDED"
		}
		rec := entity.Spot_InstrumentsInfo{
			Symbol:         item.Symbol,
			Base:           item.BaseAsset,
			Quote:          item.QuoteAsset,
			MinQty:         item.BaseSizePrecision,
			MinNotional:    item.QuoteAmountPrecision,
			PricePrecision: fmt.Sprintf("%d", item.QuotePrecision),
			SizePrecision:  fmt.Sprintf("%d", item.BaseAssetPrecision),
			State:          state,
		}
		out = append(out, rec)
	}
	return
}

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

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {
	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.СlientOrderId,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *spot_converts) convertOrderList(answ []spot_orderList) (res []entity.Spot_OrdersList) {
	for _, item := range answ {
		res = append(res, entity.Spot_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			Side:          strings.ToUpper(item.Side),
			Price:         item.Price,
			Size:          item.OrigQty,
			ExecutedSize:  item.ExecutedQty,
			Type:          strings.ToUpper(item.Type),
			Status:        item.Status,
			CreateTime:    item.Time,
			UpdateTime:    item.UpdateTime,
		})
	}
	return res
}

func (c *spot_converts) convertOrdersHistory(in []spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			Side:          strings.ToUpper(item.Side),
			Size:          item.OrigQty,
			Price:         item.Price,
			ExecutedSize:  item.ExecutedQty,
			ExecutedPrice: item.Price,
			// Fee:           utils.FloatToStringAll(item.Fee),
			Type:       strings.ToUpper(item.Type),
			Status:     strings.ToUpper(item.Status),
			CreateTime: item.Time,
			UpdateTime: item.UpdateTime,
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

func (c *futures_converts) convertInstrumentsInfo(in []futures_instrumentsInfo) (out []entity.Futures_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	// for _, item := range in {
	// 	// state := "OTHER"
	// 	// if item.State == 1 {
	// 	// 	state = "LIVE"
	// 	// } else if item.State == 0 {
	// 	// 	state = "OFF"
	// 	// } else if item.State == 5 {
	// 	// 	state = "PRE-OPEN"
	// 	// } else if item.State == 25 {
	// 	// 	state = "SUSPENDED"
	// 	// }
	// 	rec := entity.Futures_InstrumentsInfo{
	// 		// Symbol:         item.Name,
	// 		// 	Base:           base,
	// 		// 	Quote:          quote,
	// 		// 	MinQty:         fmt.Sprintf("%d", item.Order_size_min),
	// 		// 	PricePrecision: utils.GetPrecisionFromStr(item.Mark_price_round),
	// 		// 	SizePrecision:  "0",
	// 		// 	MaxLeverage:    item.Leverage_max,
	// 		// 	State:          item.Status,
	// 		// 	IsSizeContract: true,
	// 		// 	Multiplier:     item.Quanto_multiplier,
	// 	}
	// 	out = append(out, rec)
	// }
	return
}
