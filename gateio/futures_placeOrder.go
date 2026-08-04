package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	positionSide *entity.PositionSideType
	hedgeMode    *bool

	settle  *string
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

func (s *futures_placeOrder) HedgeMode(hedgeMode bool) *futures_placeOrder {
	s.hedgeMode = &hedgeMode
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
		Endpoint: "/api/v4/futures/{settle}/orders",
		SecType:  utils.SecTypeSigned,
	}

	// settle default
	settleDefault := "usdt"
	if s.settle == nil {
		s.settle = &settleDefault
	}
	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder
	if isTP && isSL {
		return res, errors.New("gateio futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	// ---------------- TP/SL via price_orders ----------------
	if isTP || isSL {
		// We need contract + trigger price at minimum
		if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
			return res, errors.New("gateio futures_placeOrder(TP/SL): Symbol is required")
		}
		if s.price == nil || strings.TrimSpace(*s.price) == "" {
			return res, errors.New("gateio futures_placeOrder(TP/SL): Price (trigger) is required")
		}

		dual := s.hedgeMode != nil && *s.hedgeMode

		// In dual(hedge) mode for correct TP/SL semantics we should know which position we are closing.
		// For partial close we use reduce_only + signed size; sign maps to posSide in dual mode.
		if dual && s.positionSide == nil {
			return res, errors.New("gateio futures_placeOrder(TP/SL): PositionSide is required in hedgeMode")
		}

		// endpoint
		r.Endpoint = "/api/v4/futures/{settle}/price_orders"
		r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

		// -------- initial (order to be placed when trigger fires) --------
		init := gateioPriceOrderInitial{
			Contract:   *s.symbol,
			Price:      "0",   // market
			Tif:        "ioc", // market-like
			ReduceOnly: true,
		}

		// tag / client id
		if s.clientOrderID != nil && strings.TrimSpace(*s.clientOrderID) != "" {
			init.Text = *s.clientOrderID
		} else {
			init.Text = "api"
		}

		// if caller explicitly asked reduce=false, allow it (but default true for safety)
		if s.reduce != nil && *s.reduce == false {
			init.ReduceOnly = false
		}

		// size logic:
		// - if size not provided => close full position
		// - if size provided => partial close
		hasSize := s.size != nil && strings.TrimSpace(*s.size) != ""
		if !hasSize {
			// full close
			init.Size = 0
			if dual {
				// dual mode: close direction comes from auto_size, and size must be 0
				switch *s.positionSide {
				case entity.PositionSideTypeLong:
					init.AutoSize = "close_long"
				case entity.PositionSideTypeShort:
					init.AutoSize = "close_short"
				default:
					return res, errors.New("gateio futures_placeOrder(TP/SL): unsupported PositionSide for hedgeMode")
				}
				init.ReduceOnly = true
				// IMPORTANT: do NOT set Close=true in dual mode
			} else {
				// single mode: can use close=true with size=0
				v := true
				init.Close = &v
				init.ReduceOnly = true
			}
		} else {
			// partial close
			sz := utils.StringToInt64(*s.size)
			if sz < 0 {
				sz = -sz
			}
			if sz == 0 {
				return res, errors.New("gateio futures_placeOrder(TP/SL): Size must be > 0 for partial close")
			}

			if dual {
				// dual mode: sign of size selects which position is reduced:
				// negative => long, positive => short
				switch *s.positionSide {
				case entity.PositionSideTypeLong:
					init.Size = -sz
				case entity.PositionSideTypeShort:
					init.Size = sz
				default:
					return res, errors.New("gateio futures_placeOrder(TP/SL): unsupported PositionSide for hedgeMode")
				}
				init.ReduceOnly = true
				// IMPORTANT: do NOT set Close=true and do NOT set AutoSize for partial
				init.Close = nil
				init.AutoSize = ""
			} else {
				// single mode: use Side to choose sign (SELL => negative) if provided
				if s.side != nil && *s.side == entity.SideTypeSell {
					init.Size = -sz
				} else {
					init.Size = sz
				}
				init.ReduceOnly = true
				init.Close = nil
				init.AutoSize = ""
			}
		}

		// -------- trigger --------
		tr := gateioPriceOrderTrigger{
			StrategyType: 0,
			PriceType:    0,     // last price
			Expiration:   86400, // 24h
			Price:        *s.price,
		}

		// rule:
		// 1 => >= trigger
		// 2 => <= trigger
		//
		// Use PositionSide for robust TP/SL direction:
		// LONG: TP up (>=), SL down (<=)
		// SHORT: TP down (<=), SL up (>=)
		if s.positionSide != nil {
			switch *s.positionSide {
			case entity.PositionSideTypeLong:
				if isTP {
					tr.Rule = 1
				} else {
					tr.Rule = 2
				}
			case entity.PositionSideTypeShort:
				if isTP {
					tr.Rule = 2
				} else {
					tr.Rule = 1
				}
			}
		} else {
			// fallback (single mode): infer from side if present; otherwise default like LONG close
			if s.side != nil && *s.side == entity.SideTypeBuy {
				// treating as closing SHORT
				if isTP {
					tr.Rule = 2
				} else {
					tr.Rule = 1
				}
			} else {
				// treating as closing LONG
				if isTP {
					tr.Rule = 1
				} else {
					tr.Rule = 2
				}
			}
		}

		j_i, err := json.Marshal(init)
		if err != nil {
			return res, err
		}

		j_t, err := json.Marshal(tr)
		if err != nil {
			return res, err
		}

		body := utils.Params{
			"initial": j_i,
			"trigger": j_t,
		}

		// IMPORTANT: send nested json as struct (not map->string)
		r.SetFormParamsStruct(body)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		// response: { "id": 123 }
		var answ struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(data, &answ); err != nil {
			return res, err
		}

		out := futures_placeOrder_Response{
			Contract:    *s.symbol,
			ID:          answ.ID,
			Text:        init.Text,
			Create_time: 0,
		}
		return s.convert.convertPlaceOrder(out), nil
	}

	// ---------------- normal orders ----------------
	m := utils.Params{}

	if s.price != nil {
		m["price"] = *s.price
	}
	if s.symbol != nil {
		m["contract"] = *s.symbol
	}
	if s.size != nil {
		m["size"] = *s.size
	}
	if s.side != nil && s.size != nil {
		if *s.side == entity.SideTypeSell {
			m["size"] = fmt.Sprintf("-%s", *s.size)
		}
	}
	if s.orderType != nil {
		if *s.orderType == entity.OrderTypeMarket {
			m["price"] = "0"
			m["tif"] = "ioc"
		}
	}
	if s.clientOrderID != nil {
		m["text"] = *s.clientOrderID
	}
	if s.hedgeMode != nil && s.positionSide != nil && s.side != nil {
		if *s.hedgeMode && ((strings.ToUpper(string(*s.positionSide)) == "LONG" && strings.ToUpper(string(*s.side)) == "SELL") ||
			(strings.ToUpper(string(*s.positionSide)) == "SHORT" && strings.ToUpper(string(*s.side)) == "BUY")) {
			m["reduce_only"] = true
		}
	}
	if s.reduce != nil && *s.reduce == true {
		m["reduce_only"] = true
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

type futures_placeOrder_Response struct {
	Contract    string  `json:"contract"`
	ID          int64   `json:"id"`
	Text        string  `json:"text"`
	Create_time float64 `json:"create_time"`
}

// Gate Futures price_orders payload

type gateioPriceOrderInitial struct {
	Contract   string `json:"contract"`            // e.g. DOGE_USDT
	Size       int64  `json:"size"`                // signed; in dual mode: <0 => long, >0 => short; 0 => close all (with close/auto_size)
	Price      string `json:"price"`               // "0" for market
	Tif        string `json:"tif"`                 // "ioc"
	Close      *bool  `json:"close,omitempty"`     // single mode full close only
	ReduceOnly bool   `json:"reduce_only"`         // true recommended
	Text       string `json:"text,omitempty"`      // client tag
	AutoSize   string `json:"auto_size,omitempty"` // dual mode full close: close_long / close_short (size must be 0)
}

type gateioPriceOrderTrigger struct {
	StrategyType int    `json:"strategy_type"` // 0
	PriceType    int    `json:"price_type"`    // 0 last price
	Rule         int    `json:"rule"`          // 1 >=, 2 <=
	Expiration   int    `json:"expiration"`    // seconds
	Price        string `json:"price"`         // trigger price
}

type gateioPriceOrderBody struct {
	Initial gateioPriceOrderInitial `json:"initial"`
	Trigger gateioPriceOrderTrigger `json:"trigger"`
}
