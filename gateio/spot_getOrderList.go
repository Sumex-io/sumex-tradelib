package gateio

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
		Endpoint: "/api/v4/spot/open_orders",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := []spot_orderList{}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertOrderList(answ), nil
}

type spot_orderList struct {
	Orders []struct {
		Currency_pair string `json:"currency_pair"`
		ID            string `json:"id"`
		Text          string `json:"text"`
		Create_time   int64  `json:"create_time_ms"`
		Update_time   int64  `json:"update_time_ms"`
		Status        string `json:"status"`
		Type          string `json:"type"`
		Side          string `json:"side"`
		Amount        string `json:"amount"`
		Filled_amount string `json:"filled_amount"`
		Price         string `json:"price"`
	} `json:"orders"`
}
