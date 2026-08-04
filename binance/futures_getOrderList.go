package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============getOrderList=================
type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
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
		Endpoint: "/fapi/v1/openOrders",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/openOrders"
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
	answ := []futures_orderList{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ), nil
}

type futures_orderList struct {
	Symbol        string `json:"symbol"`
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	Price         string `json:"price"`
	Type          string `json:"type"`
	Status        string `json:"status"`

	Time       int64 `json:"time"`
	UpdateTime int64 `json:"updateTime"`
}

type futures_orderList_attachAlgoOrds struct {
	AttachAlgoId string `json:"attachAlgoId"`
	SlOrdPx      string `json:"slOrdPx"`
	SlTriggerPx  string `json:"slTriggerPx"`
	TpOrdPx      string `json:"tpOrdPx"`
	TpTriggerPx  string `json:"tpTriggerPx"`
}
