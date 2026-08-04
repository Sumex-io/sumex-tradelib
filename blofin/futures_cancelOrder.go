package blofin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	orderID       *string
	clientOrderID *string
}

func (s *futures_cancelOrder) Symbol(symbol string) *futures_cancelOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_cancelOrder) OrderID(orderID string) *futures_cancelOrder {
	s.orderID = &orderID
	return s
}

func (s *futures_cancelOrder) ClientOrderID(clientOrderID string) *futures_cancelOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *futures_cancelOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {

	// === Валидация: нужен хотя бы один идентификатор ордера ===
	if s.orderID == nil && s.clientOrderID == nil {
		return res, errors.New("either orderId or clientOrderId is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v1/trade/cancel-order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// instId — опционально
	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.orderID != nil {
		m["orderId"] = *s.orderID
	}
	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []placeOrder_Response `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	// проверяем внутренний code
	for _, item := range answ.Result {
		if item.Code != "0" {
			if item.Msg != "" {
				return res, errors.New(item.Msg)
			}
			return res, errors.New("cancel order failed with code " + item.Code)
		}
	}

	return s.convert.convertPlaceOrder(answ.Result), nil
}
