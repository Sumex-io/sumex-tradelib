package bybit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

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
		Method:   http.MethodPost,
		Endpoint: "/v5/order/cancel",
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
