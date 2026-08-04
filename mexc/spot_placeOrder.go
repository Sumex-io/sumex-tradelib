package mexc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============placeOrder=================
type spot_placeOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol        *string
	side          *entity.SideType
	orderType     *entity.OrderType
	size          *string
	price         *string
	clientOrderID *string

	tpPrice *string
	slPrice *string
}

func (s *spot_placeOrder) Symbol(symbol string) *spot_placeOrder {
	s.symbol = &symbol
	return s
}

func (s *spot_placeOrder) Side(side entity.SideType) *spot_placeOrder {
	s.side = &side
	return s
}

func (s *spot_placeOrder) OrderType(orderType entity.OrderType) *spot_placeOrder {
	s.orderType = &orderType
	return s
}

func (s *spot_placeOrder) Size(size string) *spot_placeOrder {
	s.size = &size
	return s
}

func (s *spot_placeOrder) Price(price string) *spot_placeOrder {
	s.price = &price
	return s
}

func (s *spot_placeOrder) ClientOrderID(clientOrderID string) *spot_placeOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *spot_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v3/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.side != nil {
		m["side"] = string(*s.side)
	}

	if s.orderType != nil {
		m["type"] = string(*s.orderType)
	}

	if s.size != nil {
		m["quantity"] = *s.size
	}

	if s.price != nil {
		m["price"] = *s.price
	}

	if s.clientOrderID != nil {
		m["newClientOrderId"] = *s.clientOrderID
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

type placeOrder_Response struct {
	Symbol        string `json:"symbol"`
	OrderId       string `json:"orderId"`
	СlientOrderId string `json:"clientOrderId"`
	Price         string `json:"price"`
	OrigQty       string `json:"origQty"`
	Type          string `json:"type"`
	Side          string `json:"side"`
	TransactTime  int64  `json:"transactTime"`
}
