package weex

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"strconv"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type spot_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
}

func (s *spot_ordersHistory) Symbol(symbol string) *spot_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *spot_ordersHistory) StartTime(startTime int64) *spot_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *spot_ordersHistory) EndTime(endTime int64) *spot_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *spot_ordersHistory) Limit(limit int64) *spot_ordersHistory {
	s.limit = &limit
	return s
}

func (s *spot_ordersHistory) Page(page int64) *spot_ordersHistory {
	s.page = &page
	return s
}

func (s *spot_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v3/allOrders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m["page"] = *s.page
	}
	if s.startTime != nil {
		m["startTime"] = *s.startTime
	}
	if s.endTime != nil {
		m["endTime"] = *s.endTime
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []spot_ordersHistory_Response
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	res = s.convert.convertOrdersHistory(answ)
	s.attachFees(ctx, res, opts...)
	return res, nil
}

type spot_ordersHistory_Response struct {
	Symbol              string `json:"symbol"`
	OrderId             int64  `json:"orderId"`
	ClientOrderId       string `json:"clientOrderId"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Time                int64  `json:"time"`
	UpdateTime          int64  `json:"updateTime"`
	IsWorking           bool   `json:"isWorking"`
}

type spot_userTrade struct {
	OrderID    int64  `json:"orderId"`
	Commission string `json:"commission"`
}

// WeEx spot orders carry no fee, so it is summed from one page of the symbol's trades; without them the fee stays empty.
func (s *spot_ordersHistory) attachFees(ctx context.Context, orders []entity.Spot_OrdersHistory, opts ...utils.RequestOption) {
	if s.symbol == nil || *s.symbol == "" || len(orders) == 0 {
		return
	}

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
		Endpoint: "/api/v3/myTrades",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{"symbol": *s.symbol, "startTime": from, "endTime": to, "limit": 200})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return
	}

	var trades []spot_userTrade
	if err := json.Unmarshal(data, &trades); err != nil {
		return
	}

	fees := make(map[string]*big.Rat)
	for _, trade := range trades {
		id := strconv.FormatInt(trade.OrderID, 10)
		if fees[id] == nil {
			fees[id] = new(big.Rat)
		}
		fees[id].Add(fees[id], weexDecimal(trade.Commission))
	}

	for i := range orders {
		if fee, ok := fees[orders[i].OrderID]; ok {
			orders[i].Fee = weexDecimalString(fee)
		}
	}
}
