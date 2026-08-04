package bingx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============futures_placeOrder=================
type futures_placeOrder struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string
	positionSide  *entity.PositionSideType
	marginMode    *entity.MarginModeType
	hedgeMode     *bool

	reduce  *bool
	tpOrder *bool
	slOrder *bool
}

func (s *futures_placeOrder) Reduce(reduce bool) *futures_placeOrder {
	s.reduce = &reduce
	return s
}

func (s *futures_placeOrder) MarginMode(marginMode entity.MarginModeType) *futures_placeOrder {
	s.marginMode = &marginMode
	return s
}

func (s *futures_placeOrder) HedgeMode(hedgeMode bool) *futures_placeOrder {
	s.hedgeMode = &hedgeMode
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
		Endpoint: "/openApi/swap/v2/trade/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"positionSide": strings.ToUpper(string(entity.PositionSideTypeBoth))}

	// --- TP / SL separate order for existing futures position (BingX) ---
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder

	// минимальная валидация: нельзя одновременно TP и SL
	if isTP && isSL {
		return res, errors.New("bingx futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	if isTP || isSL {

		// symbol
		if s.symbol != nil {
			m["symbol"] = *s.symbol
		}

		// side (для закрытия: LONG->SELL, SHORT->BUY)
		if s.side != nil {
			m["side"] = strings.ToUpper(string(*s.side))
		}

		// positionSide: BOTH/LONG/SHORT (в hedge режиме важен)
		if s.hedgeMode != nil && *s.hedgeMode {
			if s.positionSide != nil {
				m["positionSide"] = strings.ToUpper(string(*s.positionSide))
			}
		}

		// trigger price
		if s.price != nil {
			// у BingX для STOP_MARKET/TAKE_PROFIT_MARKET используется stopPrice
			m["stopPrice"] = *s.price
		}

		// type
		if isTP {
			m["type"] = "TAKE_PROFIT_MARKET"
		} else {
			m["type"] = "STOP_MARKET"
		}

		// если size указан — частичное закрытие позиции
		if s.size != nil {
			m["quantity"] = *s.size
			if s.hedgeMode == nil || *s.hedgeMode == false {
				m["reduceOnly"] = "true"
			}
		} else {
			// если size нет — закрыть всю позицию (BingX поддерживает closePosition)
			m["closePosition"] = "true"
		}

		// reduceOnly если явно задан (пусть переопределяет/дублирует — биржа разрулит)
		if s.reduce != nil && *s.reduce == true {
			m["reduceOnly"] = "true"
		}

		// clientOrderID
		if s.clientOrderID != nil {
			m["clientOrderId"] = *s.clientOrderID
		}

		r.SetParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		var answ struct {
			Result futures_placeOrder_Response_Extra `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		return s.convert.convertPlaceOrder_extra(answ.Result), nil
	}

	// --- end TP / SL branch ---
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.orderType != nil {
		m["type"] = strings.ToUpper(string(*s.orderType))
	}

	if s.side != nil {
		m["side"] = strings.ToUpper(string(*s.side))
	}

	if s.price != nil {
		m["price"] = *s.price
		if *s.orderType == entity.OrderTypeStop || *s.orderType == entity.OrderTypeTakeProfit {
			m["stopPrice"] = *s.price
		}
	}

	if s.reduce != nil && *s.reduce == true {
		m["reduceOnly"] = "true"
	}

	if s.size != nil {
		m["quantity"] = *s.size
	}

	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	}

	if s.hedgeMode != nil && *s.hedgeMode {
		if s.positionSide != nil {
			m["positionSide"] = strings.ToUpper(string(*s.positionSide))
		}
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result futures_placeOrder_Response_Extra `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder_extra(answ.Result), nil
}

type futures_placeOrder_Response_Extra struct {
	Order struct {
		Symbol        string `json:"symbol"`
		OrderID       any    `json:"orderId"`
		ClientOrderId any    `json:"clientOrderId"`
	} `json:"order"`
}

type futures_placeOrder_Response struct {
	Order struct {
		Symbol        string `json:"symbol"`
		OrderID       int64  `json:"orderId"`
		ClientOrderId string `json:"clientOrderId"`
	} `json:"order"`
}
