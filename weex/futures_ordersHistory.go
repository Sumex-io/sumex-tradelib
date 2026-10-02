package weex

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"sort"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

// WeEx documents `limit` on GET /capi/v3/order/history as 1..1000 and applies a 500-record default
// when it is omitted; from 2026-09-14 the request weight scales with the page size.
const (
	futuresOrdersHistoryDefaultLimit int64 = 100
	futuresOrdersHistoryMaxLimit     int64 = 1000
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64

	orderID *string
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) StartTime(startTime int64) *futures_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_ordersHistory) EndTime(endTime int64) *futures_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_ordersHistory) Limit(limit int64) *futures_ordersHistory {
	s.limit = &limit
	return s
}

func (s *futures_ordersHistory) Page(page int64) *futures_ordersHistory {
	s.page = &page
	return s
}

func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	s.orderID = &orderID
	return s
}

// pageLimit keeps the caller's Limit inside WeEx's documented range and never returns 0, so the
// request always carries an explicit page size instead of billing weight for their 500-row default.
func (s *futures_ordersHistory) pageLimit() int64 {
	if s.limit == nil || *s.limit <= 0 {
		return futuresOrdersHistoryDefaultLimit
	}
	if *s.limit > futuresOrdersHistoryMaxLimit {
		return futuresOrdersHistoryMaxLimit
	}
	return *s.limit
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/capi/v3/order/history",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{}

		if s.symbol != nil && *s.symbol != "" {
			m["symbol"] = *s.symbol
		}
		m["limit"] = s.pageLimit()
		if s.page != nil && *s.page >= 0 {
			m["page"] = *s.page
		}
		if s.startTime != nil {
			m["startTime"] = *s.startTime
		}
		if s.endTime != nil {
			m["endTime"] = *s.endTime
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ []futures_ordersHistory_Response
		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertOrdersHistory(answ)...)
		s.attachFills(ctx, res, opts...)
	}

	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/capi/v3/allAlgoOrders",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{"limit": s.pageLimit()}
		if s.symbol != nil && *s.symbol != "" {
			m["symbol"] = *s.symbol
		}
		if s.startTime != nil {
			m["startTime"] = *s.startTime
		}
		if s.endTime != nil {
			m["endTime"] = *s.endTime
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ struct {
			Orders []futures_algoOrder `json:"orders"`
		}
		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		executed := make(map[string]entity.Futures_OrdersHistory, len(res))
		var oldest int64
		for _, order := range res {
			executed[order.OrderID] = order
			if oldest == 0 || order.CreateTime < oldest {
				oldest = order.CreateTime
			}
		}

		for _, order := range s.convert.convertAlgoOrdersHistory(answ.Orders, executed) {
			// Past a full orders page the caller pages by the oldest createTime, so older rows would skip orders.
			if int64(len(res)) >= s.pageLimit() && order.CreateTime < oldest {
				continue
			}
			res = append(res, order)
		}
	}

	if s.orderID != nil && *s.orderID != "" {
		filtered := make([]entity.Futures_OrdersHistory, 0, 1)
		for _, item := range res {
			if item.OrderID == *s.orderID {
				filtered = append(filtered, item)
			}
		}
		res = filtered
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].CreateTime > res[j].CreateTime
	})

	return res, nil
}

type futures_ordersHistory_Response struct {
	Symbol        string `json:"symbol"`
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Type          string `json:"type"`
	OrigQty       string `json:"origQty"`
	Price         string `json:"price"`
	ExecutedQty   string `json:"executedQty"`
	AvgPrice      string `json:"avgPrice"`
	CumQuote      string `json:"cumQuote"`
	Status        string `json:"status"`
	Time          int64  `json:"time"`
	UpdateTime    int64  `json:"updateTime"`
	TimeInForce   string `json:"timeInForce"`
	ReduceOnly    bool   `json:"reduceOnly"`
}

// WeEx orders carry no fee or PnL, so both are summed from the account fills; without fills the order stays as served.
func (s *futures_ordersHistory) attachFills(ctx context.Context, orders []entity.Futures_OrdersHistory, opts ...utils.RequestOption) {
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

	fills, err := (&futures_positionsHistory{callAPI: s.callAPI, convert: s.convert, symbol: s.symbol}).fetchFills(ctx, from, to, opts...)
	if err != nil {
		return
	}

	type orderFills struct {
		fee, profit *big.Rat
		feeAsset    string
	}
	byOrder := make(map[string]*orderFills)
	for _, fill := range fills {
		sum := byOrder[fill.OrderID]
		if sum == nil {
			sum = &orderFills{fee: new(big.Rat), profit: new(big.Rat)}
			byOrder[fill.OrderID] = sum
		}
		sum.fee.Add(sum.fee, weexDecimal(fill.Commission))
		sum.profit.Add(sum.profit, weexDecimal(fill.RealisedProfit))
		sum.feeAsset = fill.CommissionAsset
	}

	for i := range orders {
		sum, ok := byOrder[orders[i].OrderID]
		if !ok {
			continue
		}
		orders[i].Fee = weexDecimalString(new(big.Rat).Neg(sum.fee))
		orders[i].FeeAsset = sum.feeAsset
		orders[i].RealisedProfit = weexDecimalString(sum.profit)
	}
}
