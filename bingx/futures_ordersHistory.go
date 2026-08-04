package bingx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
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

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/swap/v2/trade/allOrders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}

	if s.startTime != nil {
		m["startTime"] = *s.startTime
	}
	if s.endTime != nil {
		m["endTime"] = *s.endTime
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	// log.Println("=029fc1=", string(data))

	if err != nil {
		return res, err
	}

	var answ struct {
		Result struct {
			Orders []futures_ordersHistory_Response `json:"orders"`
		} `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrdersHistory(answ.Result.Orders), nil
}

type futures_ordersHistory_Response struct {
	Symbol          string `json:"symbol"`
	OrderId         int64  `json:"orderId"`
	Side            string `json:"side"`
	PositionSide    string `json:"positionSide"`
	Type            string `json:"type"`
	OrigQty         string `json:"origQty"`
	Price           string `json:"price"`
	ExecutedQty     string `json:"executedQty"`
	AvgPrice        string `json:"avgPrice"`
	CumQuote        string `json:"cumQuote"`
	Profit          string `json:"profit"`
	Commission      string `json:"commission"`
	Status          string `json:"status"`
	ClientOrderId   string `json:"clientOrderId"`
	Leverage        string `json:"leverage"`
	PositionID      int64  `json:"positionID"`
	OnlyOnePosition bool   `json:"onlyOnePosition"`
	Time            int64  `json:"time"`
	UpdateTime      int64  `json:"updateTime"`
}
