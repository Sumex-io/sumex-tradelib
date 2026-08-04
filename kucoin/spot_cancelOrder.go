package kucoin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============TradeCancelOrders=================

type spot_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol  *string
	orderID *string
}

func (s *spot_cancelOrder) Symbol(symbol string) *spot_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_cancelOrder) OrderID(orderID string) *spot_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *spot_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodDelete,
		Endpoint: "/api/v1/orders/{orderId}",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// if s.symbol != nil {
	// 	m["instId"] = *s.symbol
	// }
	oID := ""
	if s.orderID != nil {
		oID = *s.orderID
		r.Endpoint = strings.Replace(r.Endpoint, "{orderId}", *s.orderID, 1)
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			CancelledOrderIds []string `json:"cancelledOrderIds"`
		} `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = append(res, entity.PlaceOrder{
		OrderID: oID,
		Ts:      time.Now().UTC().UnixMilli(),
	})
	return res, nil
}
