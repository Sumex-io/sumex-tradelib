package okx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
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
		Endpoint: "/api/v5/trade/orders-history",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"instType": "SPOT",
		"state":    "filled",
	}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m["page"] = *s.page
	}
	if s.startTime != nil {
		m["begin"] = *s.startTime
	}
	if s.endTime != nil {
		m["end"] = *s.endTime
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
	InstId    string `json:"instId"`
	OrdId     string `json:"ordId"`
	ClOrdId   string `json:"clOrdId"`
	Side      string `json:"side"`
	Sz        string `json:"sz"`
	FillSz    string `json:"fillSz"`
	AccFillSz string `json:"accFillSz"`
	Px        string `json:"px"`
	AvgPx     string `json:"avgPx"`
	Fee       string `json:"fee"`
	FeeCcy    string `json:"feeCcy"`
	OrdType   string `json:"ordType"`
	State     string `json:"state"`
	CTime     string `json:"cTime"`
	UTime     string `json:"uTime"`
}
