package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_amendOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle   *string
	symbol   *string
	orderID  *string
	newSize  *string
	newPrice *string
}

func (s *futures_amendOrder) Symbol(symbol string) *futures_amendOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_amendOrder) OrderID(orderID string) *futures_amendOrder {
	s.orderID = &orderID
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
	r := &utils.Request{
		Method:   http.MethodPut,
		Endpoint: "/api/v4/futures/{settle}/orders/{order_id}",
		SecType:  utils.SecTypeSigned,
	}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)
	if s.orderID != nil {
		r.Endpoint = strings.Replace(r.Endpoint, "{order_id}", *s.orderID, 1)
	}

	m := utils.Params{}

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

	answ := futures_placeOrder_Response{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder(answ), nil
}
