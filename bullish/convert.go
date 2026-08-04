package bullish

import (
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================
type account_converts struct{}

// ===============FUTURES=================
type futures_converts struct{}

func (c *futures_converts) convertBalance(in []futures_Balance) (out []entity.FuturesBalance) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		out = append(out, entity.FuturesBalance{
			Asset:     item.AssetSymbol,
			Balance:   utils.FloatToStringAll(utils.StringToFloat(item.AvailableQuantity) + utils.StringToFloat(item.LockedQuantity)),
			Available: item.AvailableQuantity,
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
		if !item.CreateOrderEnabled {
			state = "OFF"
		}

		out = append(out, entity.Futures_InstrumentsInfo{
			Symbol: item.Symbol,
			Base:   item.BaseSymbol,
			Quote:  item.QuoteSymbol,
			MinQty: item.MinQuantityLimit,
			// MinNotional:    utils.FloatToStringAll(item.TradeMinUSDT),
			// PricePrecision: item.PricePrecision,
			PricePrecision: utils.GetPrecisionFromStr(utils.FloatToStringAll(utils.StringToFloat(item.TickSize))),
			SizePrecision:  item.BasePrecision,
			Multiplier:     item.ContractMultiplier,
			// ContractSize:   item.TickSize,
			State: state,
		})
	}
	return
}

func (c *futures_converts) convertMarketCandle(in []futures_marketCandle_responce) (out []entity.Futures_MarketCandle) {
	if len(in) == 0 {
		return out
	}
	for index, item := range in {

		complete := true
		if index == 0 {
			complete = false
		}
		out = append(out, entity.Futures_MarketCandle{
			OpenPrice:    item.Open,
			HighestPrice: item.High,
			LowestPrice:  item.Low,
			ClosePrice:   item.Close,
			Volume:       item.Volume,
			Time:         utils.StringToInt64(item.CreatedAtTimestamp),
			Complete:     complete,
		})
	}
	return
}

func (c *futures_converts) convertPlaceOrder(in futures_placeOrder_Response) (out []entity.PlaceOrder) {

	out = append(out, entity.PlaceOrder{
		OrderID:       in.OrderId,
		ClientOrderID: in.OrderLinkId,
		Ts:            time.Now().UTC().UnixMilli(),
	})
	return out
}

func (c *futures_converts) convertPositions(answ []futures_Position) (res []entity.Futures_Positions) {
	for _, item := range answ {

		if utils.StringToFloat(item.Quantity) == 0 {
			continue
		}
		marginMode := string(entity.MarginModeTypeCross)
		hedgeMode := false
		positionSide := ""

		if strings.ToUpper(item.Side) == "BUY" {
			positionSide = "LONG"
		} else {
			positionSide = "SHORT"
		}
		res = append(res, entity.Futures_Positions{
			Symbol:       item.Symbol,
			PositionSide: positionSide,
			// PositionID:   item.PositionId,
			PositionSize: item.Quantity,
			// EntryPrice:   item.EntryNotional,
			EntryPrice: utils.FloatToStringAll(utils.StringToFloat(item.EntryNotional) / utils.StringToFloat(item.Quantity)),
			// MarkPrice:    item.MarkPrice,
			// InitialMargin:    item.Initial_margin,
			UnRealizedProfit: item.ReportedMtmPnl,
			RealizedProfit:   item.RealizedPnl,
			Notional:         item.Notional,
			// MarginRatio:      item.Maintenance_rate,
			// Leverage:   item.Leverage,
			MarginMode: marginMode,
			HedgeMode:  hedgeMode,
			CreateTime: utils.StringToInt64(item.CreatedTime),
			UpdateTime: utils.StringToInt64(item.UpdatedTime),
		})
	}
	return res
}

func (c *futures_converts) convertOrderList(in []futures_orderList) (out []entity.Futures_OrdersList) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {

		positionSide := "LONG"
		if strings.ToUpper(item.Side) == "BUY" {
			positionSide = "LONG"
		} else {
			positionSide = "SHORT"
		}
		otype := "LIMIT"
		if item.Type != "LMT" {
			otype = strings.ToUpper(item.Type)
		}

		ostatus := "LIVE"
		if item.Status != "OPEN" {
			ostatus = strings.ToUpper(item.Status)
		}
		out = append(out, entity.Futures_OrdersList{
			Symbol:        item.Symbol,
			OrderID:       item.OrderId,
			ClientOrderID: item.ClientOrderId,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  positionSide,
			PositionSize:  item.Quantity,
			ExecutedSize:  item.QuantityFilled,
			Price:         item.Price,
			Type:          otype,
			Status:        ostatus,
			CreateTime:    utils.StringToInt64(item.CreatedAtTimestamp),
			// UpdateTime:    utils.StringToInt64(item.UpdatedTime),
		})
	}
	return out
}
