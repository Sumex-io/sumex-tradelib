package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_placeOrder struct {
	callAPI  func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	brokerID string

	convert futures_converts

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string
	positionSide  *entity.PositionSideType
	marginMode    *entity.MarginModeType

	reduce  *bool
	tpOrder *bool
	slOrder *bool
}

func (s *futures_placeOrder) Reduce(reduce bool) *futures_placeOrder {
	s.reduce = &reduce
	return s
}

func (s *futures_placeOrder) TpOrder(v bool) *futures_placeOrder {
	s.tpOrder = &v
	return s
}

func (s *futures_placeOrder) SlOrder(v bool) *futures_placeOrder {
	s.slOrder = &v
	return s
}

func (s *futures_placeOrder) MarginMode(marginMode entity.MarginModeType) *futures_placeOrder {
	s.marginMode = &marginMode
	return s
}

func (s *futures_placeOrder) Symbol(symbol string) *futures_placeOrder {
	s.symbol = &symbol
	return s
}

func (s *futures_placeOrder) Side(side entity.SideType) *futures_placeOrder {
	s.side = &side
	return s
}

func (s *futures_placeOrder) Size(size string) *futures_placeOrder {
	s.size = &size
	return s
}

func (s *futures_placeOrder) Price(price string) *futures_placeOrder {
	s.price = &price
	return s
}

func (s *futures_placeOrder) OrderType(orderType entity.OrderType) *futures_placeOrder {
	s.orderType = &orderType
	return s
}

func (s *futures_placeOrder) ClientOrderID(clientOrderID string) *futures_placeOrder {
	s.clientOrderID = &clientOrderID
	return s
}

func (s *futures_placeOrder) PositionSide(positionSide entity.PositionSideType) *futures_placeOrder {
	s.positionSide = &positionSide
	return s
}

func (s *futures_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.tpOrder != nil && *s.tpOrder && s.slOrder != nil && *s.slOrder {
		return res, errors.New("TpOrder and SlOrder cannot both be true")
	}

	if s.tpOrder != nil && *s.tpOrder || s.slOrder != nil && *s.slOrder {
		return s.doAlgoOrder(ctx, opts...)
	}

	return s.doNormalOrder(ctx, opts...)
}

func (s *futures_placeOrder) doNormalOrder(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/capi/v3/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.side != nil {
		m["side"] = strings.ToUpper(string(*s.side))
	}

	if s.positionSide != nil {
		m["positionSide"] = strings.ToUpper(string(*s.positionSide))
	}

	if s.orderType != nil {
		m["type"] = strings.ToUpper(string(*s.orderType))
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

	if s.reduce != nil && *s.reduce {
		m["reduceOnly"] = true
	}

	if s.orderType != nil && strings.ToUpper(string(*s.orderType)) == "LIMIT" {
		m["timeInForce"] = "GTC"
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ futures_placeOrder_Response
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
		return res, errors.New("place order failed")
	}

	return s.convert.convertPlaceOrder(answ), nil
}

func (s *futures_placeOrder) doAlgoOrder(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol is required")
	}
	if s.side == nil {
		return res, errors.New("side is required")
	}
	if s.positionSide == nil {
		return res, errors.New("position side is required")
	}
	if s.size == nil || *s.size == "" {
		return res, errors.New("size is required")
	}
	if s.price == nil || *s.price == "" {
		return res, errors.New("trigger price is required")
	}
	if s.clientOrderID == nil || *s.clientOrderID == "" {
		return res, errors.New("client order id is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/capi/v3/algoOrder",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"symbol":       *s.symbol,
		"side":         strings.ToUpper(string(*s.side)),
		"positionSide": strings.ToUpper(string(*s.positionSide)),
		"quantity":     *s.size,
		"triggerPrice": *s.price,
		"clientAlgoId": *s.clientOrderID,
	}

	if s.tpOrder != nil && *s.tpOrder {
		m["type"] = "TAKE_PROFIT_MARKET"
	}

	if s.slOrder != nil && *s.slOrder {
		m["type"] = "STOP_MARKET"
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ futures_placeOrder_Response
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
		return res, errors.New("place algo order failed")
	}

	return s.convert.convertPlaceOrder(answ), nil
}

type futures_placeOrder_Response struct {
	OrderId       string `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Success       bool   `json:"success"`
	ErrorCode     string `json:"errorCode"`
	ErrorMessage  string `json:"errorMessage"`
}
