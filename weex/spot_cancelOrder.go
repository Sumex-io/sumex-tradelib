package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
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
		Method:   http.MethodPost,
		Endpoint: "/api/v2/trade/cancel-order",
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
		m["clientOid"] = *s.clientOrderID
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Code string               `json:"code"`
		Msg  string               `json:"msg"`
		Data cancelOrder_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.Code != "00000" && answ.Code != "" {
		if answ.Msg != "" {
			return res, errors.New(answ.Msg)
		}
		return res, errors.New("cancel order failed")
	}

	if !answ.Data.Result {
		if answ.Data.ErrMsg != "" {
			return res, errors.New(answ.Data.ErrMsg)
		}
		return res, errors.New("cancel order failed")
	}

	return s.convert.convertCancelOrder(answ.Data), nil
}

type cancelOrder_Response struct {
	OrderID       string `json:"order_id"`
	ClientOrderID string `json:"client_oid"`
	Symbol        string `json:"symbol"`
	Result        bool   `json:"result"`
	ErrCode       string `json:"err_code"`
	ErrMsg        string `json:"err_msg"`
}
