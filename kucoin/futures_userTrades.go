package kucoin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_userTrades struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	orderID   *string
	startTime *int64
	endTime   *int64
	limit     *int64
}

func (s *futures_userTrades) Symbol(symbol string) *futures_userTrades {
	s.symbol = &symbol
	return s
}

func (s *futures_userTrades) OrderID(orderID string) *futures_userTrades {
	s.orderID = &orderID
	return s
}

func (s *futures_userTrades) StartTime(startTime int64) *futures_userTrades {
	s.startTime = &startTime
	return s
}

func (s *futures_userTrades) EndTime(endTime int64) *futures_userTrades {
	s.endTime = &endTime
	return s
}

func (s *futures_userTrades) Limit(limit int64) *futures_userTrades {
	s.limit = &limit
	return s
}

func (s *futures_userTrades) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_UserTrades, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/fills",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil && *s.symbol != "" {
		m["symbol"] = *s.symbol
	}
	if s.orderID != nil && *s.orderID != "" {
		m["orderId"] = *s.orderID
	}
	if s.startTime != nil {
		m["startAt"] = *s.startTime
	}
	if s.endTime != nil {
		m["endAt"] = *s.endTime
	}
	if s.limit != nil && *s.limit > 0 {
		m["pageSize"] = *s.limit
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result struct {
			Items []futures_userTrade `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertUserTrades(dropInverse(answ.Result.Items, func(t futures_userTrade) string { return t.Symbol })), nil
}

type futures_userTrade struct {
	TradeID      string `json:"tradeId"`
	OrderID      string `json:"orderId"`
	Symbol       string `json:"symbol"`
	Side         string `json:"side"`
	PositionSide string `json:"positionSide"`
	Liquidity    string `json:"liquidity"`
	Price        string `json:"price"`
	Value        string `json:"value"`
	OpenFeePay   string `json:"openFeePay"`
	CloseFeePay  string `json:"closeFeePay"`
	Fee          string `json:"fee"`
	FeeCurrency  string `json:"feeCurrency"`
	TradeTime    int64  `json:"tradeTime"`
}
