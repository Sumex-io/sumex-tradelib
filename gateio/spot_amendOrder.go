package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_amendOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol   *string
	orderID  *string
	newSize  *string
	newPrice *string
}

func (s *spot_amendOrder) Symbol(symbol string) *spot_amendOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_amendOrder) OrderID(orderID string) *spot_amendOrder {
	s.orderID = &orderID
	return s
}

func (s *spot_amendOrder) NewSize(newSize string) *spot_amendOrder {
	s.newSize = &newSize
	return s
}

func (s *spot_amendOrder) NewPrice(newPrice string) *spot_amendOrder {
	s.newPrice = &newPrice
	return s
}

func (s *spot_amendOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodPatch,
		Endpoint: "/api/v4/spot/orders/{order_id}",
		SecType:  utils.SecTypeSigned,
	}
	if s.orderID != nil {
		r.Endpoint = strings.Replace(r.Endpoint, "{order_id}", *s.orderID, 1)
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["currency_pair"] = *s.symbol
	}

	if s.newSize != nil {
		m["amount"] = *s.newSize
	}

	if s.newPrice != nil {
		m["price"] = *s.newPrice
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := placeOrder_Response{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder(answ), nil
}
