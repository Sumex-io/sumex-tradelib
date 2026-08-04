package bingx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) OrderType(orderType entity.OrderType) *futures_getOrderList {
	s.orderType = &orderType
	return s
}

func (s *futures_getOrderList) Limit(limit int) *futures_getOrderList {
	s.limit = &limit
	return s
}

func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/swap/v2/trade/openOrders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_orderList `json:"data"`
	}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertOrderList(answ.Result)
	return res, nil
}

type futures_orderList struct {
	Orders []struct {
		Symbol        string `json:"symbol"`
		OrderId       int64  `json:"orderId"`
		ClientOrderId string `json:"clientOrderId"`
		PositionID    int64  `json:"positionID"`
		Side          string `json:"side"`
		PositionSide  string `json:"positionSide"`
		Type          string `json:"type"`
		OrigQty       string `json:"origQty"`
		ExecutedQty   string `json:"executedQty"`
		Price         string `json:"price"`
		AvgPrice      string `json:"avgPrice"`
		Leverage      string `json:"leverage"`
		Status        string `json:"status"`
		StopPrice     string `json:"stopPrice"`
		Time          int64  `json:"time"`
		UpdateTime    int64  `json:"updateTime"`
	} `json:"orders"`
}
