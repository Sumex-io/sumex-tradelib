package bybit

import (
	"context"
	"encoding/json"
	"net/http"

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
		Method:   http.MethodPost,
		Endpoint: "/v5/order/amend",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"category": "spot",
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.orderID != nil {
		m["orderId"] = *s.orderID
	}

	if s.newSize != nil {
		m["qty"] = *s.newSize
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
		Result placeOrder_Response `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder(answ.Result), nil
}
