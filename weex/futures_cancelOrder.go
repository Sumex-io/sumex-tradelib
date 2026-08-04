package weex

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
	isTpSl        *bool
}

func (s *futures_cancelOrder) IsTpSl(v bool) *futures_cancelOrder {
	s.isTpSl = &v
	return s
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
	isAlgo := s.isTpSl != nil && *s.isTpSl

	r := &utils.Request{
		Method:  http.MethodDelete,
		SecType: utils.SecTypeSigned,
	}

	m := utils.Params{}

	if isAlgo {
		r.Endpoint = "/capi/v3/algoOrder"

		if s.orderID == nil || *s.orderID == "" {
			return res, errors.New("orderId required")
		}

		m["orderId"] = *s.orderID
	} else {
		r.Endpoint = "/capi/v3/order"

		if s.orderID != nil && *s.orderID != "" {
			m["orderId"] = *s.orderID
		}

		if s.clientOrderID != nil && *s.clientOrderID != "" {
			m["origClientOrderId"] = *s.clientOrderID
		}

		if len(m) == 0 {
			return res, errors.New("orderId or clientOrderId required")
		}
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ futures_cancelOrder_Response
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if !answ.Success {
		if answ.ErrorMessage != "" {
			return res, errors.New(answ.ErrorMessage)
		}
		if answ.ErrorCode != "" {
			return res, errors.New(answ.ErrorCode)
		}
		return res, errors.New("cancel order failed")
	}

	return s.convert.convertCancelOrder(answ), nil
}

type futures_cancelOrder_Response struct {
	OrderId           string `json:"orderId"`
	OrigClientOrderId string `json:"origClientOrderId"`
	Success           bool   `json:"success"`
	ErrorCode         string `json:"errorCode"`
	ErrorMessage      string `json:"errorMessage"`
}
