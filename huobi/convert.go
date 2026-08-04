package huobi

import (
	"fmt"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===============ACCOUNT=================
type account_converts struct{}

func (c *account_converts) convertAccountInfo(in []accountInfo) (out entity.AccountInformation) {
	extra := []entity.AccountInformationExtraInfo{}
	for _, item := range in {
		t := "SPOT"
		if strings.ToUpper(item.Type) != "SPOT" {
			t = "FUTURES"
		}
		extra = append(extra, entity.AccountInformationExtraInfo{
			UID:  utils.Int64ToString(item.ID),
			Type: t,
		})
	}
	out.ExtraInfo = extra
	return out
}

// ===============SPOT=================

type spot_converts struct{}

func (c *spot_converts) convertInstrumentsInfo(in []spot_instrumentsInfo) (out []entity.Spot_InstrumentsInfo) {
	if len(in) == 0 {
		return out
	}
	for _, item := range in {
		// if strings.ToUpper(item.Symbol) == "BTCUSDT" {
		// 	log.Printf("=28606a= %+v", item)
		// }
		if item.Status == "online" {
			item.Status = "LIVE"
		}

		rec := entity.Spot_InstrumentsInfo{
			Symbol: strings.ToUpper(item.Symbol),
			Base:   strings.ToUpper(item.BaseCoin),
			Quote:  strings.ToUpper(item.QuoteCoin),
			// MinQty:         utils.FloatToStringAll(item.MinQty),
			// MinNotional:    item.MinTradeUSDT,
			PricePrecision: utils.FloatToStringAll(item.PricePrecision),
			SizePrecision:  utils.FloatToStringAll(item.QuantityPrecision),
			State:          strings.ToUpper(item.Status),
		}
		out = append(out, rec)
	}
	return
}

func (c *spot_converts) convertBalance(in []spot_Balance) (out []entity.AssetsBalance) {
	if len(in) == 0 {
		return out
	}

	mapsAssets := map[string]entity.AssetsBalance{}

	for _, item := range in {
		t, is := mapsAssets[item.Currency]
		if !is {
			t = entity.AssetsBalance{Asset: item.Currency}
		}
		switch item.Type {
		case "trade":
			t.Balance = item.Balance
		case "frozen":
			t.Locked = item.Balance
		}
		mapsAssets[item.Currency] = t

	}

	for _, i := range mapsAssets {
		if i.Balance != "0" || i.Locked != "0" {
			i.Balance = utils.FloatToStringAll(utils.StringToFloat(i.Balance) + utils.StringToFloat(i.Locked))
			out = append(out, i)
		}
	}
	return out
}

func (c *spot_converts) convertOrdersHistory(in []spot_ordersHistory_Response) (out []entity.Spot_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {

		side := ""
		typeOrd := ""

		sp := strings.Split(item.Type, "-")
		if len(sp) == 2 {
			side = sp[0]
			typeOrd = sp[1]
		}

		if item.State != "filled" || utils.StringToFloat(item.Filled_amount) == 0 {
			continue
		}
		out = append(out, entity.Spot_OrdersHistory{
			Symbol:        strings.ToUpper(item.Symbol),
			OrderID:       fmt.Sprintf("%d", item.ID),
			ClientOrderID: item.Client_order_id,
			Side:          strings.ToUpper(side),
			Size:          item.Amount,
			Price:         item.Price,
			ExecutedSize:  item.Filled_amount,
			ExecutedPrice: utils.FloatToStringAll(utils.StringToFloat(item.Fill_cash) / utils.StringToFloat(item.Filled_amount)),
			Fee:           item.Fee,
			Type:          strings.ToUpper(typeOrd),
			Status:        strings.ToUpper(item.State),
			CreateTime:    item.Created_at,
			UpdateTime:    item.Updated_at,
		})
	}
	return out
}

func (c *spot_converts) convertPlaceOrder(in placeOrder_Response) (out []entity.PlaceOrder) {

	// out = append(out, entity.PlaceOrder{
	// 	OrderID:       in.ID,
	// 	ClientOrderID: in.Text,
	// 	Ts:            time.Now().UTC().UnixMilli(),
	// })
	return out
}

func (c *spot_converts) convertOrderList(in []spot_orderList) (out []entity.Spot_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {

		side := ""
		typeOrd := ""

		sp := strings.Split(item.Type, "-")
		if len(sp) == 2 {
			side = sp[0]
			typeOrd = sp[1]
		}
		out = append(out, entity.Spot_OrdersList{
			Symbol:        strings.ToUpper(item.Symbol),
			OrderID:       fmt.Sprintf("%d", item.ID),
			ClientOrderID: item.Client_order_id,
			Side:          strings.ToUpper(side),
			Size:          item.Amount,
			Price:         item.Price,
			ExecutedSize:  item.Filled_amount,
			Type:          strings.ToUpper(typeOrd),
			Status:        item.State,
			CreateTime:    item.Created_at,
			// UpdateTime:    utils.StringToInt64(item.UTime),
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
		state := "OTHER"
		if item.Contract_status == 1 {
			state = "LIVE"
		}
		rec := entity.Futures_InstrumentsInfo{
			Symbol:         item.Contract_code,
			Base:           item.Symbol,
			Quote:          item.Trade_partition,
			MinQty:         "1",
			PricePrecision: utils.GetPrecisionFromStr(utils.FloatToStringAll(item.Price_tick)),
			SizePrecision:  utils.GetPrecisionFromStr(utils.FloatToStringAll(item.Contract_size)),
			State:          state,
			ContractSize:   utils.FloatToStringAll(item.Contract_size),
			Multiplier:     "1",
			IsSizeContract: true,
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
