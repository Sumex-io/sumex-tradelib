package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type spot_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol        *string
	orderID       *string
	clientOrderID *string
}

func (s *spot_cancelOrder) Symbol(symbol string) *spot_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_cancelOrder) OrderID(orderID string) *spot_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *spot_cancelOrder) ClientOrderID(clientOrderID string) *spot_cancelOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *spot_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodDelete,
		Endpoint: "/api/v3/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.orderID != nil {
		m["orderId"] = *s.orderID
	}

	if s.clientOrderID != nil {
		m["origClientOrderId"] = *s.clientOrderID
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		OrderId json.Number `json:"orderId"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.OrderId == "" {
		return res, errors.New("cancel order failed")
	}

	out := cancelOrder_Response{OrderID: answ.OrderId.String()}
	if s.clientOrderID != nil {
		out.ClientOrderID = *s.clientOrderID
	}

	return s.convert.convertCancelOrder(out), nil
}

type cancelOrder_Response struct {
	OrderID       string
	ClientOrderID string
}
