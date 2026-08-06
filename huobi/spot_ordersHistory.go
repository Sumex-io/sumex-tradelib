package huobi

import (
	"context"
	"encoding/json"
	"net/http"

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
		Endpoint: "/v1/order/history",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"size": 1000}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["size"] = *s.limit
	}
	// if s.page != nil && *s.page > 0 {
	// 	m["page"] = *s.page
	// }
	if s.startTime != nil {
		m["start-time"] = *s.startTime
	}
	if s.endTime != nil {
		m["end-time"] = *s.endTime
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []spot_ordersHistory_Response `json:"data"`
	}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrdersHistory(answ.Result), nil
}

type spot_ordersHistory_Response struct {
	Symbol          string `json:"symbol"`
	ID              int64  `json:"id"`
	Client_order_id string `json:"client-order-id"`
	Type            string `json:"type"`
	Amount          string `json:"amount"`
	Filled_amount   string `json:"field-amount"`
	Price           string `json:"price"`
	// Fill_price    string `json:"fill_price"`
	Fill_cash string `json:"field-cash-amount"`
	Fee       string `json:"field-fees"`
	State     string `json:"state"`

	Created_at int64 `json:"created-at"`
	Updated_at int64 `json:"updated-at"`
}
