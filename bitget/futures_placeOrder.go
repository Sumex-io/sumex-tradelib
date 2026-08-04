package bitget

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
		Endpoint: "/api/v2/mix/order/place-order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"productType": "USDT-FUTURES", "marginCoin": "USDT"}

	// --- TP / SL separate order for existing futures position (Bitget) ---
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder

	// минимальная валидация: нельзя одновременно TP и SL
	if isTP && isSL {
		return res, errors.New("bitget futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	if isTP || isSL {
		// переключаемся на trigger/plan endpoint
		r.Endpoint = "/api/v2/mix/order/place-tpsl-order"

		// symbol
		if s.symbol != nil {
			m["symbol"] = *s.symbol
		}

		// triggerPrice
		if s.price != nil {
			m["triggerPrice"] = *s.price
		}

		// executePrice: 0 => market execution (по докам)
		m["executePrice"] = "0"

		// planType:
		// - если size указан -> profit_plan/loss_plan (количественный TP/SL)
		// - если size не указан -> pos_profit/pos_loss (позиционный TP/SL на всю позицию)
		if s.size != nil && *s.size != "" {
			m["size"] = *s.size
			if isTP {
				m["planType"] = "profit_plan"
			} else {
				m["planType"] = "loss_plan"
			}
		} else {
			if isTP {
				m["planType"] = "pos_profit"
			} else {
				m["planType"] = "pos_loss"
			}
		}

		// holdSide:
		// hedge-mode: long/short
		// one-way: buy/sell
		// (без лишней валидации — если не передали, биржа вернёт ошибку)
		if s.hedgeMode != nil && *s.hedgeMode {
			if s.positionSide != nil {
				switch *s.positionSide {
				case entity.PositionSideTypeLong:
					m["holdSide"] = "long"
				case entity.PositionSideTypeShort:
					m["holdSide"] = "short"
				}
			}
		} else {
			// one-way: buy/sell (это направление позиции)
			if s.side != nil {
				m["holdSide"] = strings.ToLower(string(*s.side))
			}
		}

		// clientOid (опционально)
		if s.clientOrderID != nil {
			m["clientOid"] = *s.clientOrderID
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		// Ответ этого эндпоинта: data { orderId, clientOid } (symbol нет)
		var answ struct {
			Result struct {
				OrderId   string `json:"orderId"`
				ClientOid string `json:"clientOid"`
			} `json:"data"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		// адаптируем под существующий конвертер
		tmp := futures_placeOrder_Response{
			Symbol:    "",
			ClientOid: answ.Result.ClientOid,
			OrderId:   answ.Result.OrderId,
		}
		if s.symbol != nil {
			tmp.Symbol = *s.symbol
		}

		return s.convert.convertPlaceOrder(tmp), nil
	}
	// --- end TP / SL branch ---

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.size != nil {
		m["size"] = *s.size
	}

	if s.price != nil {
		m["price"] = *s.price
	}

	if s.side != nil {
		m["side"] = strings.ToLower(string(*s.side))
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["marginMode"] = "crossed"
		case entity.MarginModeTypeIsolated:
			m["marginMode"] = "isolated"
		}
	}

	if s.orderType != nil {
		m["orderType"] = strings.ToLower(string(*s.orderType))
		// if *s.orderType == entity.OrderTypeLimit {

		// }
	}

	if s.clientOrderID != nil {
		m["clientOid"] = *s.clientOrderID
	}

	if s.hedgeMode != nil && *s.hedgeMode {
		if s.positionSide != nil {
			switch *s.positionSide {
			case entity.PositionSideTypeLong:
				if s.side != nil && *s.side == entity.SideTypeBuy {
					m["tradeSide"] = "open"
				} else if s.side != nil && *s.side == entity.SideTypeSell {
					m["tradeSide"] = "close"
					m["side"] = strings.ToLower(string(entity.SideTypeBuy))
				}
			case entity.PositionSideTypeShort:
				if s.side != nil && *s.side == entity.SideTypeSell {
					m["tradeSide"] = "open"
				} else if s.side != nil && *s.side == entity.SideTypeBuy {
					m["tradeSide"] = "close"
					m["side"] = strings.ToLower(string(entity.SideTypeSell))
				}
			}
		}
	}

	if s.reduce != nil && *s.reduce == true {
		m["reduceOnly"] = "YES"
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_placeOrder_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertPlaceOrder(answ.Result), nil
}

type futures_placeOrder_Response struct {
	Symbol    string `json:"symbol"`
	ClientOid string `json:"clientOid"`
	OrderId   string `json:"orderId"`
}
