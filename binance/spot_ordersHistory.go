package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
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
	// if s.page != nil && *s.page > 0 {
	// 	m["page"] = *s.page
	// }
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
	answ := []spot_ordersHistory_Response{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrdersHistory(answ), nil
}

type spot_ordersHistory_Response struct {
	Symbol              string `json:"symbol"`
	OrderId             int64  `json:"orderId"`
	ClientOrderId       string `json:"clientOrderId"`
	Side                string `json:"side"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	СummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Price               string `json:"price"`
	// AvgPx     string `json:"avgPx"`
	// Fee       string `json:"fee"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	Time       int64  `json:"time"`
	UpdateTime int64  `json:"updateTime"`
}
