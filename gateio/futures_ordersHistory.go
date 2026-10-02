package gateio

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle    *string
	symbol    *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_ordersHistory) Settle(settle string) *futures_ordersHistory {
	s.settle = &settle
	return s
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) OrderType(orderType entity.OrderType) *futures_ordersHistory {
	s.orderType = &orderType
	return s
}

func (s *futures_ordersHistory) Limit(limit int) *futures_ordersHistory {
	s.limit = &limit
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	settleDefault := "usdt"
	if s.settle == nil {
		s.settle = &settleDefault
	}

	// ---------------- 1) NORMAL FINISHED ORDERS ----------------
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/api/v4/futures/{settle}/orders",
			SecType:  utils.SecTypeSigned,
		}
		r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

		m := utils.Params{
			"status": "finished",
		}
		if s.symbol != nil && *s.symbol != "" {
			m["contract"] = *s.symbol
		}
		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		orders := []futures_ordersHistory_Response{}
		if e := json.Unmarshal(data, &orders); e != nil {
			return res, e
		}

		res = append(res, convertGateioFinishedOrdersToHistory(orders)...)
		s.attachFees(ctx, res, opts...)
	}

	// ---------------- 2) FINISHED TRIGGER / TP-SL ORDERS ----------------
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/api/v4/futures/{settle}/price_orders",
			SecType:  utils.SecTypeSigned,
		}
		r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

		m := utils.Params{
			"status": "finished",
		}
		if s.symbol != nil && *s.symbol != "" {
			m["contract"] = *s.symbol
		}
		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}
		priceOrders := []gateio_priceOrderHistory_Response{}
		if e := json.Unmarshal(data, &priceOrders); e != nil {
			return res, e
		}

		res = append(res, convertGateioFinishedPriceOrdersToHistory(priceOrders)...)
	}

	return res, nil
}

type futures_ordersHistory_Response struct {
	ID          int64   `json:"id"`
	Contract    string  `json:"contract"`
	Status      string  `json:"status"`
	FinishAs    string  `json:"finish_as"`
	Size        int64   `json:"size"`
	Left        int64   `json:"left"`
	Is_close    bool    `json:"is_close"`
	Text        string  `json:"text"`
	Mkfr        string  `json:"mkfr"`
	Tkfr        string  `json:"tkfr"`
	Price       string  `json:"price"`
	Fill_price  string  `json:"fill_price"`
	Create_time float64 `json:"create_time"`
	Update_time float64 `json:"update_time"`
	Finish_time float64 `json:"finish_time"`
}

type gateio_priceOrderHistory_Response struct {
	ID          int64   `json:"id"`
	Status      string  `json:"status"`
	FinishAs    string  `json:"finish_as"`
	Reason      string  `json:"reason"`
	Create_time float64 `json:"create_time"`
	Finish_time float64 `json:"finish_time"`
	TradeID     int64   `json:"trade_id"`

	Initial gateio_priceOrderInitialHistory `json:"initial"`
	Trigger gateio_priceOrderTriggerHistory `json:"trigger"`

	OrderType string `json:"order_type"`
}

type gateio_priceOrderInitialHistory struct {
	Contract   string `json:"contract"`
	Size       int64  `json:"size"`
	Price      string `json:"price"`
	Tif        string `json:"tif"`
	Text       string `json:"text"`
	ReduceOnly bool   `json:"reduce_only"`
	Close      bool   `json:"close"`
	AutoSize   string `json:"auto_size"`
}

type gateio_priceOrderTriggerHistory struct {
	StrategyType int    `json:"strategy_type"`
	PriceType    int    `json:"price_type"`
	Rule         int    `json:"rule"`
	Expiration   int    `json:"expiration"`
	Price        string `json:"price"`
}

func convertGateioFinishedOrdersToHistory(answ []futures_ordersHistory_Response) (res []entity.Futures_OrdersHistory) {
	for _, item := range answ {
		if strings.ToLower(item.Status) != "finished" {
			continue
		}
		if strings.ToLower(item.FinishAs) != "filled" {
			continue
		}

		positionSide := "LONG"
		side := "BUY"
		if item.Size < 0 {
			positionSide = "SHORT"
			side = "SELL"
		}

		sizeAbs := item.Size
		if sizeAbs < 0 {
			sizeAbs = -sizeAbs
		}

		res = append(res, entity.Futures_OrdersHistory{
			Symbol:        item.Contract,
			OrderID:       utils.Int64ToString(item.ID),
			ClientOrderID: item.Text,
			Side:          side,
			PositionSide:  positionSide,
			PositionSize:  utils.Int64ToString(sizeAbs),
			ExecutedSize:  utils.Int64ToString(sizeAbs),
			Price:         item.Price,
			ExecutedPrice: item.Fill_price,
			Status:        strings.ToUpper(item.Status),
			CreateTime:    int64(item.Create_time * 1000.0),
			UpdateTime:    int64(item.Update_time * 1000.0),
		})
	}
	return res
}

func convertGateioFinishedPriceOrdersToHistory(in []gateio_priceOrderHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, po := range in {
		if strings.ToLower(po.Status) != "finished" {
			continue
		}
		if strings.ToLower(po.FinishAs) != "succeeded" {
			continue
		}

		side := ""
		positionSide := ""

		orderType := strings.ToLower(strings.TrimSpace(po.OrderType))
		auto := strings.ToLower(strings.TrimSpace(po.Initial.AutoSize))

		switch {
		case strings.Contains(orderType, "close-long"), auto == "close_long":
			side = "SELL"
			positionSide = "LONG"
		case strings.Contains(orderType, "close-short"), auto == "close_short":
			side = "BUY"
			positionSide = "SHORT"
		default:
			// fallback по знаку size
			if po.Initial.Size < 0 {
				side = "SELL"
				positionSide = "LONG"
			} else if po.Initial.Size > 0 {
				side = "BUY"
				positionSide = "SHORT"
			}
		}

		isTP := false
		isSL := false
		if positionSide == "LONG" {
			if po.Trigger.Rule == 1 {
				isTP = true
			} else if po.Trigger.Rule == 2 {
				isSL = true
			}
		} else if positionSide == "SHORT" {
			if po.Trigger.Rule == 2 {
				isTP = true
			} else if po.Trigger.Rule == 1 {
				isSL = true
			}
		}

		sizeAbs := po.Initial.Size
		if sizeAbs < 0 {
			sizeAbs = -sizeAbs
		}

		price := po.Trigger.Price
		if price == "" {
			price = po.Initial.Price
		}

		tsMs := int64(po.Finish_time * 1000.0)
		if tsMs == 0 {
			tsMs = int64(po.Create_time * 1000.0)
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        po.Initial.Contract,
			OrderID:       utils.Int64ToString(po.ID),
			ClientOrderID: po.Initial.Text,
			Side:          side,
			PositionSide:  positionSide,
			PositionSize:  utils.Int64ToString(sizeAbs),
			ExecutedSize:  utils.Int64ToString(sizeAbs),
			Price:         price,
			ExecutedPrice: "",
			Type:          "TRIGGER",
			Status:        strings.ToUpper(po.Status),
			CreateTime:    int64(po.Create_time * 1000.0),
			UpdateTime:    tsMs,
			TpOrder:       isTP,
			SlOrder:       isSL,
		})
	}

	return out
}

type gateio_futuresTrade struct {
	OrderID string `json:"order_id"`
	Fee     string `json:"fee"`
}

// Gate orders carry only fee rates (mkfr/tkfr), so the fee is summed from one page of fills; without them it stays empty.
func (s *futures_ordersHistory) attachFees(ctx context.Context, orders []entity.Futures_OrdersHistory, opts ...utils.RequestOption) {
	var from, to int64
	for _, order := range orders {
		if from == 0 || order.CreateTime < from {
			from = order.CreateTime
		}
		if order.UpdateTime > to {
			to = order.UpdateTime
		}
	}
	if from == 0 || to < from {
		return
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: strings.Replace("/api/v4/futures/{settle}/my_trades_timerange", "{settle}", *s.settle, 1),
		SecType:  utils.SecTypeSigned,
	}
	m := utils.Params{"from": from / 1000, "to": to/1000 + 1, "limit": 1000}
	if s.symbol != nil && *s.symbol != "" {
		m["contract"] = *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return
	}

	var trades []gateio_futuresTrade
	if err := json.Unmarshal(data, &trades); err != nil {
		return
	}

	fees := make(map[string]*big.Rat)
	for _, trade := range trades {
		fee, ok := new(big.Rat).SetString(strings.TrimSpace(trade.Fee))
		if !ok {
			continue
		}
		if fees[trade.OrderID] == nil {
			fees[trade.OrderID] = new(big.Rat)
		}
		fees[trade.OrderID].Add(fees[trade.OrderID], fee)
	}

	for i := range orders {
		if fee, ok := fees[orders[i].OrderID]; ok {
			paid := strings.TrimRight(new(big.Rat).Neg(fee).FloatString(12), "0")
			orders[i].Fee = strings.TrimSuffix(paid, ".")
		}
	}
}
