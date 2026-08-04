package mexc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============spot_getOrderList=================
type spot_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol *string
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v3/openOrders",
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

	answ := []spot_orderList{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ), nil
}

type spot_orderList struct {
	Symbol              string `json:"symbol"`
	OrderId             string `json:"orderId"`
	ClientOrderId       string `json:"clientOrderId"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
	Status              string `json:"status"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	StopPrice           string `json:"stopPrice"`
	Time                int64  `json:"time"`
	UpdateTime          int64  `json:"updateTime"`
	IsWorking           bool   `json:"isWorking"`
	OrigQuoteOrderQty   string `json:"origQuoteOrderQty"`
}
