package okx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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
		Endpoint: "/api/v5/trade/orders-pending",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"instType": "SPOT",
	}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.limit != nil {
		m["limit"] = *s.limit
	}

	if s.orderType != nil {
		m["ordType"] = strings.ToLower(string(*s.orderType))
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []spot_orderList `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result), nil
}

type spot_orderList struct {
	InstId         string `json:"instId"`
	OrdId          string `json:"ordId"`
	ClOrdId        string `json:"clOrdId"`
	Px             string `json:"px"`
	Sz             string `json:"sz"`
	FillSz         string `json:"fillSz"`
	AttachAlgoOrds []struct {
		AttachAlgoId string `json:"attachAlgoId"`
		SlOrdPx      string `json:"slOrdPx"`
		SlTriggerPx  string `json:"slTriggerPx"`
		TpOrdPx      string `json:"tpOrdPx"`
		TpTriggerPx  string `json:"tpTriggerPx"`
	} `json:"AttachAlgoOrds"`
	PosSide   string `json:"posSide"`
	OrdType   string `json:"ordType"`
	TdMode    string `json:"tdMode"`
	InstType  string `json:"instType"`
	Lever     string `json:"lever"`
	Side      string `json:"side"`
	State     string `json:"state"`
	IsTpLimit string `json:"isTpLimit"`
	UTime     string `json:"uTime"`
	CTime     string `json:"cTime"`
}
