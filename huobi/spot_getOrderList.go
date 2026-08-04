package huobi

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol *string
}

func (s *spot_getOrderList) Symbol(symbol string) *spot_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *spot_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersList, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/order/openOrders",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []spot_orderList `json:"data"`
	}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ.Result), nil
}

type spot_orderList struct {
	Symbol          string `json:"symbol"`
	Price           string `json:"price"`
	Amount          string `json:"amount"`
	Created_at      int64  `json:"created-at"`
	Client_order_id string `json:"client-order-id"`
	Filled_amount   string `json:"filled-amount"`
	ID              int64  `json:"id"`
	Type            string `json:"type"`
	State           string `json:"state"`
}
