package blofin

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
	convert futures_converts

	symbol        *string
	side          *entity.SideType
	size          *string
	price         *string
	orderType     *entity.OrderType
	clientOrderID *string
	positionSide  *entity.PositionSideType
	marginMode    *entity.MarginModeType
	reduceOnly    *bool
	tpTriggerPx   *string
	tpOrderPx     *string
	slTriggerPx   *string
	slOrderPx     *string

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

func (s *futures_placeOrder) ReduceOnly(reduceOnly bool) *futures_placeOrder {
	s.reduceOnly = &reduceOnly
	return s
}

func (s *futures_placeOrder) TpTriggerPrice(px string) *futures_placeOrder {
	s.tpTriggerPx = &px
	return s
}

func (s *futures_placeOrder) TpOrderPrice(px string) *futures_placeOrder {
	s.tpOrderPx = &px
	return s
}

func (s *futures_placeOrder) SlTriggerPrice(px string) *futures_placeOrder {
	s.slTriggerPx = &px
	return s
}

func (s *futures_placeOrder) SlOrderPrice(px string) *futures_placeOrder {
	s.slOrderPx = &px
	return s
}

func (s *futures_placeOrder) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.PlaceOrder, err error) {
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder

	// минимальная валидация: нельзя одновременно TP и SL
	if isTP && isSL {
		return res, errors.New("blofin futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	// --- TP/SL отдельным запросом (на существующую позицию) ---
	if isTP || isSL {
		r := &utils.Request{
			Method:   http.MethodPost,
			Endpoint: "/api/v1/trade/order-tpsl",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{}

		// instId
		if s.symbol != nil {
			m["instId"] = *s.symbol
		}

		// marginMode: cross / isolated
		if s.marginMode != nil {
			switch *s.marginMode {
			case entity.MarginModeTypeCross:
				m["marginMode"] = "cross"
			case entity.MarginModeTypeIsolated:
				m["marginMode"] = "isolated"
			default:
				// без дополнительной валидации: пусть биржа вернет ошибку
				m["marginMode"] = strings.ToLower(string(*s.marginMode))
			}
		}

		// positionSide: net / long / short (как у вас в обычном ордере)
		if s.positionSide != nil {
			ps := strings.ToLower(string(*s.positionSide))
			if ps == "both" || ps == "one_way" {
				ps = "net"
			}
			m["positionSide"] = ps
		}

		// side: buy / sell
		if s.side != nil {
			m["side"] = strings.ToLower(string(*s.side))
		}

		// size: если не задан — ставим "-1" (entire positions по доке)
		if s.size != nil && *s.size != "" {
			m["size"] = *s.size
		} else {
			m["size"] = "-1"
		}

		// reduceOnly: для TP/SL обычно true; если явно задано — используем
		if s.reduceOnly != nil {
			if *s.reduceOnly {
				m["reduceOnly"] = "true"
			} else {
				m["reduceOnly"] = "false"
			}
		} else if s.reduce != nil && *s.reduce {
			m["reduceOnly"] = "true"
		} else {
			m["reduceOnly"] = "true"
		}

		// clientOrderId
		if s.clientOrderID != nil {
			m["clientOrderId"] = *s.clientOrderID
		}

		// trigger + order price
		// В нашем унифицированном варианте: s.price = trigger.
		// orderPrice по умолчанию "-1" (market) — в доке для tp/sl order price это допустимо. :contentReference[oaicite:1]{index=1}
		if s.price != nil {
			if isTP {
				m["tpTriggerPrice"] = *s.price
				if s.tpOrderPx != nil && *s.tpOrderPx != "" {
					m["tpOrderPrice"] = *s.tpOrderPx
				} else {
					m["tpOrderPrice"] = "-1"
				}
			} else {
				m["slTriggerPrice"] = *s.price
				if s.slOrderPx != nil && *s.slOrderPx != "" {
					m["slOrderPrice"] = *s.slOrderPx
				} else {
					m["slOrderPrice"] = "-1"
				}
			}
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		var answ struct {
			Code string `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				TpslId        string `json:"tpslId"`
				ClientOrderId any    `json:"clientOrderId"`
				Code          string `json:"code"`
				Msg           any    `json:"msg"`
			} `json:"data"`
		}

		if err = json.Unmarshal(data, &answ); err != nil {
			return res, err
		}

		// оставляем вашу текущую философию: если API вернул code!=0 в теле — превращаем в error
		if answ.Code != "0" {
			if answ.Msg != "" {
				return res, errors.New(answ.Msg)
			}
			return res, errors.New("place tpsl failed with code " + answ.Code)
		}
		if answ.Data.Code != "" && answ.Data.Code != "0" {
			// msg может быть null/any
			if s, ok := answ.Data.Msg.(string); ok && s != "" {
				return res, errors.New(s)
			}
			return res, errors.New("place tpsl failed with code " + answ.Data.Code)
		}

		// адаптируем под существующий convert.convertPlaceOrder([]placeOrder_Response)
		out := []placeOrder_Response{
			{
				OrderId:       answ.Data.TpslId,
				ClientOrderId: "",
				Code:          "0",
				Msg:           "",
			},
		}
		// clientOrderId может быть null/string
		if s, ok := answ.Data.ClientOrderId.(string); ok {
			out[0].ClientOrderId = s
		}

		return s.convert.convertPlaceOrder(out), nil
	}

	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol (instId) is required")
	}
	if s.marginMode == nil {
		return res, errors.New("marginMode is required")
	}
	if s.positionSide == nil {
		// на Blofin positionSide обязателен даже в one-way, по доке
		return res, errors.New("positionSide is required")
	}
	if s.side == nil {
		return res, errors.New("side is required")
	}
	if s.orderType == nil {
		return res, errors.New("orderType is required")
	}
	if s.size == nil || *s.size == "" {
		return res, errors.New("size is required")
	}

	ordTypeStr := strings.ToLower(string(*s.orderType))
	if ordTypeStr != "market" && s.price == nil {
		return res, errors.New("price is required for non-market orders")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v1/trade/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// instId
	m["instId"] = *s.symbol

	// marginMode: cross / isolated
	switch *s.marginMode {
	case entity.MarginModeTypeCross:
		m["marginMode"] = "cross"
	case entity.MarginModeTypeIsolated:
		m["marginMode"] = "isolated"
	default:
		return res, errors.New("unsupported marginMode: " + string(*s.marginMode))
	}

	// positionSide: net / long / short
	if s.positionSide != nil {
		ps := strings.ToLower(string(*s.positionSide))
		// как и в setLeverage: ONE_WAY/BOTH -> net
		if ps == "both" || ps == "one_way" {
			ps = "net"
		}
		m["positionSide"] = ps
	}

	// side: buy / sell
	if s.side != nil {
		m["side"] = strings.ToLower(string(*s.side))
	}

	// orderType
	m["orderType"] = ordTypeStr

	// price (кроме market)
	if s.price != nil && ordTypeStr != "market" {
		m["price"] = *s.price
	}

	// size (контракты)
	m["size"] = *s.size

	// reduceOnly
	if s.reduceOnly != nil {
		if *s.reduceOnly {
			m["reduceOnly"] = "true"
		} else {
			m["reduceOnly"] = "false"
		}
	}

	// clientOrderId
	if s.clientOrderID != nil {
		m["clientOrderId"] = *s.clientOrderID
	}

	// TP/SL – по доке: tpTriggerPrice/tpOrderPrice/slTriggerPrice/slOrderPrice
	if s.tpTriggerPx != nil && s.tpOrderPx != nil {
		m["tpTriggerPrice"] = *s.tpTriggerPx
		m["tpOrderPrice"] = *s.tpOrderPx
	}
	if s.slTriggerPx != nil && s.slOrderPx != nil {
		m["slTriggerPrice"] = *s.slTriggerPx
		m["slOrderPrice"] = *s.slOrderPx
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []placeOrder_Response `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	// проверяем внутренний code по каждому ордеру
	for _, item := range answ.Result {
		if item.Code != "0" {
			// если есть текст — вернём его
			if item.Msg != "" {
				return res, errors.New(item.Msg)
			}
			return res, errors.New("place order failed with code " + item.Code)
		}
	}

	return s.convert.convertPlaceOrder(answ.Result), nil
}

// структура под ответ Blofin Place Order
type placeOrder_Response struct {
	OrderId       string `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Msg           string `json:"msg"`
	Code          string `json:"code"`
}
