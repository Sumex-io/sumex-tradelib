package bybit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol   *string
	category *string
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) Category(category string) *futures_getOrderList {
	s.category = &category
	return s
}

func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	m := utils.Params{
		"category": "linear",
		"limit":    50,
	}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.category != nil {
		m["category"] = *s.category
	}
	if s.symbol != nil || m["category"] == "inverse" {
		return s.fetch(ctx, m, opts...)
	}

	// linear needs symbol or settleCoin, and USDT and USDC perps settle separately
	for _, settleCoin := range []string{"USDT", "USDC"} {
		m["settleCoin"] = settleCoin
		list, err := s.fetch(ctx, m, opts...)
		if err != nil {
			return res, err
		}
		res = append(res, list...)
	}
	return res, nil
}

func (s *futures_getOrderList) fetch(ctx context.Context, m utils.Params, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/order/realtime",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			List []futures_orderList `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result.List), nil
}

type futures_orderList struct {
	Symbol        string `json:"symbol"`
	OrderId       string `json:"orderId"`
	OrderLinkId   string `json:"orderLinkId"`
	StopOrderType string `json:"stopOrderType"`
	Side          string `json:"side"`
	PositionIdx   int64  `json:"positionIdx"`
	Qty           string `json:"qty"`
	CumExecQty    string `json:"cumExecQty"`
	Price         string `json:"price"`
	TriggerPrice  string `json:"triggerPrice"`
	OrderType     string `json:"orderType"`
	OrderStatus   string `json:"orderStatus"`
	CreatedTime   string `json:"createdTime"`
	UpdatedTime   string `json:"updatedTime"`
}
