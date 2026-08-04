package okx

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
		Method:   http.MethodPost,
		Endpoint: "/api/v5/trade/amend-order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.orderID != nil {
		m["ordId"] = *s.orderID
	}

	if s.newSize != nil {
		m["newSz"] = *s.newSize
	}

	if s.newPrice != nil {
		m["newPx"] = *s.newPrice
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []placeOrder_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	if answ.Result[0].SCode != "0" {
		return res, errors.New(answ.Result[0].SMsg)
	}
	return s.convert.convertPlaceOrder(answ.Result), nil
}
