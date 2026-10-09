package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
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

	base := ""
	if s.clientOrderID != nil {
		base = *s.clientOrderID
	}

	if clientOrderID := resolveClientOrderID(s.brokerID, base); clientOrderID != "" {
		m["newClientOrderId"] = clientOrderID
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
	if s.positionSide == nil {
		return res, errors.New("position side is required")
	}
	if s.price == nil || *s.price == "" {
		return res, errors.New("trigger price is required")
	}
	if s.clientOrderID == nil || *s.clientOrderID == "" {
		return res, errors.New("client order id is required")
	}

	// /capi/v3/algoOrder would leave a standalone conditional order that WeEx does not show on the position.
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/capi/v3/placeTpSlOrder",
		SecType:  utils.SecTypeSigned,
	}

	planType := "STOP_LOSS"
	if s.tpOrder != nil && *s.tpOrder {
		planType = "TAKE_PROFIT"
	}

	m := utils.Params{
		"symbol":       *s.symbol,
		"clientAlgoId": *s.clientOrderID,
		"planType":     planType,
		"triggerPrice": *s.price,
		"positionSide": strings.ToUpper(string(*s.positionSide)),
	}

	// Without a quantity WeEx sets the TP/SL on the whole position; any quantity, even the full size, makes it partial.
	if s.size != nil && *s.size != "" {
		m["quantity"] = *s.size
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []futures_placeTpSlOrder_Response
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ) == 0 {
		return res, errors.New("place tp/sl order failed")
	}

	if !answ[0].Success {
		if answ[0].ErrorMessage != "" {
			return res, errors.New(answ[0].ErrorMessage)
		}
		if answ[0].ErrorCode != "" {
			return res, errors.New(answ[0].ErrorCode)
		}
		return res, errors.New("place tp/sl order failed")
	}

	return s.convert.convertPlaceOrder(futures_placeOrder_Response{
		OrderId:       answ[0].OrderId.String(),
		ClientOrderId: *s.clientOrderID,
		Success:       true,
	}), nil
}

type futures_placeTpSlOrder_Response struct {
	OrderId      json.Number `json:"orderId"`
	Success      bool        `json:"success"`
	ErrorCode    string      `json:"errorCode"`
	ErrorMessage string      `json:"errorMessage"`
}

type futures_placeOrder_Response struct {
	OrderId       string `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Success       bool   `json:"success"`
	ErrorCode     string `json:"errorCode"`
	ErrorMessage  string `json:"errorMessage"`
}
