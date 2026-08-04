package weex

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
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
		Endpoint: "/capi/v3/userTrades",
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
		m["startTime"] = *s.startTime
	}
	if s.endTime != nil {
		m["endTime"] = *s.endTime
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []futures_userTrade
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertUserTrades(answ), nil
}

type futures_userTrade struct {
	ID              int64  `json:"id"`
	OrderID         int64  `json:"orderId"`
	Symbol          string `json:"symbol"`
	Buyer           bool   `json:"buyer"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	Maker           bool   `json:"maker"`
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	QuoteQty        string `json:"quoteQty"`
	RealizedPnl     string `json:"realizedPnl"`
	Side            string `json:"side"`
	PositionSide    string `json:"positionSide"`
	Time            int64  `json:"time"`
}
