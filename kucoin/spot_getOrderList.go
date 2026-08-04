package kucoin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol    *string
	orderType *entity.OrderType
	limit     *int
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) OrderType(orderType entity.OrderType) *spot_getOrderList {
	s.orderType = &orderType
	return s
}

func (s *spot_getOrderList) Limit(limit int) *spot_getOrderList {
	s.limit = &limit
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"status": "active",
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.limit != nil {
		m["pageSize"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result struct {
			Items []spot_orderList `json:"items"`
		} `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result.Items), nil
}

type spot_orderList struct {
	Symbol    string `json:"symbol"`
	ID        string `json:"id"`
	ClientOid string `json:"clientOid"`
	OpType    string `json:"opType"`
	Type      string `json:"type"`
	Side      string `json:"side"`
	Price     string `json:"price"`
	Size      string `json:"size"`
	DealSize  string `json:"dealSize"`
	TradeType string `json:"tradeType"`
	CreatedAt int64  `json:"createdAt"`
}
