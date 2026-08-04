package huobi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_cancelOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol  *string
	orderID *string
	uid     *string
}

func (s *spot_cancelOrder) UID(uid string) *spot_cancelOrder {
	s.uid = &uid
	return s
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
		Endpoint: "/v1/order/orders/{order-id}/submitcancel",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.orderID != nil {
		r.Endpoint = strings.Replace(r.Endpoint, "{order-id}", *s.orderID, 1)
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result        string `json:"data"`
		ClientOrderId string `json:"clientOrderId"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return []entity.PlaceOrder{{OrderID: answ.Result, ClientOrderID: answ.ClientOrderId, Ts: time.Now().UTC().UnixMilli()}}, nil
	// return s.convert.convertPlaceOrder(answ.Result), nil
}
