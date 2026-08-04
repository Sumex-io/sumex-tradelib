package binance

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
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
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
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/fapi/v1/order",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/order"
	}

	m := utils.Params{}

	// --- TP / SL separate order for existing futures position ---
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder

	// минимальная валидация: нельзя одновременно TP и SL
	if isTP && isSL {
		return res, errors.New("binance futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	if isTP || isSL {

		if s.symbol != nil {
			m["symbol"] = *s.symbol
		}

		if s.side != nil {
			m["side"] = strings.ToUpper(string(*s.side))
		}

		if s.positionSide != nil {
			m["positionSide"] = strings.ToUpper(string(*s.positionSide))
		} else {
			m["positionSide"] = "BOTH"
		}

		// Binance требует stopPrice для STOP_MARKET / TAKE_PROFIT_MARKET
		if s.price != nil {
			m["stopPrice"] = *s.price
		}

		if isTP {
			m["type"] = "TAKE_PROFIT_MARKET"
		} else {
			m["type"] = "STOP_MARKET"
		}

		// если size указан — частичное закрытие
		if s.size != nil {
			m["quantity"] = *s.size
			m["reduceOnly"] = "true"
		} else {
			// если size нет — закрыть всю позицию
			m["closePosition"] = "true"
		}

		if s.clientOrderID != nil {
			m["newClientOrderId"] = *s.clientOrderID
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		answ := futures_placeOrder_Response{}
		if err := json.Unmarshal(data, &answ); err != nil {
			return res, err
		}

		return s.convert.convertPlaceOrder(answ), nil
	}

	// --- end TP / SL branch ---
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.side != nil {
		m["side"] = strings.ToUpper(string(*s.side))
	}

	if s.positionSide != nil {
		m["positionSide"] = strings.ToUpper(string(*s.positionSide))
	} else {
		m["positionSide"] = "BOTH"
	}

	if s.reduce != nil && *s.reduce == true {
		m["reduceOnly"] = "true"
	}

	if s.orderType != nil {
		m["type"] = strings.ToUpper(string(*s.orderType))
		if *s.orderType == entity.OrderTypeLimit || *s.orderType == entity.OrderTypeStop || *s.orderType == entity.OrderTypeTakeProfit {
			m["timeInForce"] = "GTC"
		}
	}

	if s.size != nil {
		m["quantity"] = *s.size
	}

	if s.price != nil {
		if *s.orderType == entity.OrderTypeStop || *s.orderType == entity.OrderTypeTakeProfit {
			m["stopPrice"] = *s.price
		} else {
			m["price"] = *s.price
		}
	}

	if s.clientOrderID != nil {
		m["newClientOrderId"] = *s.clientOrderID
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := futures_placeOrder_Response{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder(answ), nil
}

type futures_placeOrder_Response struct {
	Symbol        string `json:"symbol"`
	ClientOrderId string `json:"clientOrderId"`
	OrderId       int64  `json:"orderId"`
}
