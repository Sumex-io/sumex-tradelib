package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
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

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/fapi/v1/allOrders",
		// Endpoint: "/fapi/v1/userTrades",
		SecType: utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/allOrders"
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

	if s.orderID != nil {
		m["orderId"] = *s.orderID
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := []futures_ordersHistory_Response{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrdersHistory(answ), nil
	//===========
}

type futures_ordersHistory_Response struct {
	Symbol        string `json:"symbol"`
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	Price         string `json:"price"`
	AvgPrice      string `json:"avgPrice"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Time          int64  `json:"time"`
	UpdateTime    int64  `json:"updateTime"`
}
