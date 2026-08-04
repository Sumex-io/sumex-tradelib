package weex

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getOrderList struct {
	convert futures_converts

	callAPI   func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	symbol    *string
	instType  *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) InstType(instType string) *futures_getOrderList {
	s.instType = &instType
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
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/capi/v3/openOrders",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{}

		if s.symbol != nil && *s.symbol != "" {
			m["symbol"] = *s.symbol
		}

		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ []futures_orderList
		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertOrderList(answ)...)
	}

	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/capi/v3/openAlgoOrders",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{}

		if s.symbol != nil && *s.symbol != "" {
			m["symbol"] = *s.symbol
		}

		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ []futures_algoOrder
		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertAlgoOrderList(answ)...)
	}

	if s.orderType != nil {
		want := strings.ToUpper(string(*s.orderType))
		filtered := make([]entity.Futures_OrdersList, 0, len(res))
		for _, item := range res {
			if strings.ToUpper(item.Type) == want {
				filtered = append(filtered, item)
			}
		}
		res = filtered
	}

	return res, nil
}

type futures_orderList struct {
	AvgPrice                string `json:"avgPrice"`
	ClientOrderId           string `json:"clientOrderId"`
	CumQuote                string `json:"cumQuote"`
	ExecutedQty             string `json:"executedQty"`
	OrderId                 int64  `json:"orderId"`
	OrigQty                 string `json:"origQty"`
	Price                   string `json:"price"`
	ReduceOnly              bool   `json:"reduceOnly"`
	Side                    string `json:"side"`
	PositionSide            string `json:"positionSide"`
	Status                  string `json:"status"`
	StopPrice               string `json:"stopPrice"`
	Symbol                  string `json:"symbol"`
	Time                    int64  `json:"time"`
	TimeInForce             string `json:"timeInForce"`
	Type                    string `json:"type"`
	UpdateTime              int64  `json:"updateTime"`
	WorkingType             string `json:"workingType"`
	PriceProtect            bool   `json:"priceProtect"`
	PriceMatch              string `json:"priceMatch"`
	SelfTradePreventionMode string `json:"selfTradePreventionMode"`
	GoodTillDate            int64  `json:"goodTillDate"`
}

type futures_algoOrder struct {
	AlgoId         int64  `json:"algoId"`
	ClientAlgoId   string `json:"clientAlgoId"`
	AlgoType       string `json:"algoType"`
	OrderType      string `json:"orderType"`
	Symbol         string `json:"symbol"`
	Side           string `json:"side"`
	PositionSide   string `json:"positionSide"`
	TimeInForce    string `json:"timeInForce"`
	Quantity       string `json:"quantity"`
	AlgoStatus     string `json:"algoStatus"`
	ActualOrderId  *int64 `json:"actualOrderId"`
	ActualPrice    string `json:"actualPrice"`
	TriggerPrice   string `json:"triggerPrice"`
	Price          string `json:"price"`
	TpTriggerPrice string `json:"tpTriggerPrice"`
	TpPrice        string `json:"tpPrice"`
	SlTriggerPrice string `json:"slTriggerPrice"`
	SlPrice        string `json:"slPrice"`
	TpOrderType    string `json:"tpOrderType"`
	WorkingType    string `json:"workingType"`
	ClosePosition  bool   `json:"closePosition"`
	ReduceOnly     bool   `json:"reduceOnly"`
	CreateTime     int64  `json:"createTime"`
	UpdateTime     int64  `json:"updateTime"`
	TriggerTime    int64  `json:"triggerTime"`
}
