package blofin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============AmendOrder=================

type futures_amendOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	orderID       *string
	clientOrderID *string
	newSize       *string
	newPrice      *string
}

func (s *futures_amendOrder) Symbol(symbol string) *futures_amendOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_amendOrder) OrderID(orderID string) *futures_amendOrder {
	s.orderID = &orderID
	return s
}

func (s *futures_amendOrder) ClientOrderID(clientOrderID string) *futures_amendOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *futures_amendOrder) NewSize(newSize string) *futures_amendOrder {
	s.newSize = &newSize
	return s
}

func (s *futures_amendOrder) NewPrice(newPrice string) *futures_amendOrder {
	s.newPrice = &newPrice
	return s
}

func (s *futures_amendOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	// Минимальная валидация
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol (instId) is required")
	}
	if s.orderID == nil && s.clientOrderID == nil {
		return res, errors.New("either orderId or clientOrderId is required")
	}
	if s.newSize == nil && s.newPrice == nil {
		return res, errors.New("either new size or new price must be provided")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v1/trade/amend-order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"instId": *s.symbol,
	}

	if s.orderID != nil {
		m["orderId"] = *s.orderID
	}
	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	}
	if s.newSize != nil {
		m["size"] = *s.newSize
	}
	if s.newPrice != nil {
		m["price"] = *s.newPrice
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
			return res, errors.New("amend order failed with code " + item.Code)
		}
	}

	return s.convert.convertPlaceOrder(answ.Result), nil
}
