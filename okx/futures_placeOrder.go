package okx

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
	// tpPrice       *string
	// slPrice       *string

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
		Endpoint: "/api/v5/trade/order",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// --- TP / SL separate order for existing futures position (OKX) ---
	isTP := s.tpOrder != nil && *s.tpOrder
	isSL := s.slOrder != nil && *s.slOrder

	// минимальная валидация: нельзя одновременно TP и SL
	if isTP && isSL {
		return res, errors.New("okx futures_placeOrder: TpOrder and SlOrder cannot both be true")
	}

	if isTP || isSL {
		// переключаемся на algo endpoint
		r.Endpoint = "/api/v5/trade/order-algo"

		// tdMode
		if s.marginMode != nil {
			switch *s.marginMode {
			case entity.MarginModeTypeCross:
				m["tdMode"] = "cross"
			case entity.MarginModeTypeIsolated:
				m["tdMode"] = "isolated"
			}
		}

		// instId
		if s.symbol != nil {
			m["instId"] = *s.symbol
		}

		// side
		if s.side != nil {
			m["side"] = strings.ToLower(string(*s.side))
		}

		// ordType: conditional = one-way stop order (в т.ч. TP/SL) :contentReference[oaicite:2]{index=2}
		m["ordType"] = "conditional"

		// sz (если не передадут — биржа сама вернёт ошибку; мы не валидируем)
		if s.size != nil {
			m["sz"] = *s.size
		}

		// posSide (если передан)
		if s.positionSide != nil {
			m["posSide"] = strings.ToLower(string(*s.positionSide))
		}

		// reduceOnly (на TP/SL обычно true; если у вас уже передают — используем)
		if s.reduce != nil && *s.reduce == true {
			m["reduceOnly"] = true
		} else {
			// по умолчанию тоже true, чтобы не "увеличить" позицию случайно
			m["reduceOnly"] = true
		}

		// algoClOrdId (у algo-ордера отдельное поле, не clOrdId) :contentReference[oaicite:3]{index=3}
		if s.clientOrderID != nil {
			m["algoClOrdId"] = *s.clientOrderID
		}

		// TP/SL: trigger = s.price, исполнение market => OrdPx = "-1" :contentReference[oaicite:4]{index=4}
		if s.price != nil {
			if isTP {
				m["tpTriggerPx"] = *s.price
				m["tpOrdPx"] = "-1"
			} else {
				m["slTriggerPx"] = *s.price
				m["slOrdPx"] = "-1"
			}
		}

		// tag (brokerID)
		if s.brokerID != "" {
			m["tag"] = s.brokerID
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}

		// Ответ order-algo отличается от /trade/order, поэтому читаем в отдельную структуру
		var answ struct {
			Result []struct {
				AlgoId      string `json:"algoId"`
				AlgoClOrdId string `json:"algoClOrdId"`
				SCode       string `json:"sCode"`
				SMsg        string `json:"sMsg"`
				Ts          string `json:"ts"`
			} `json:"data"`
		}

		if err = json.Unmarshal(data, &answ); err != nil {
			return res, err
		}
		if len(answ.Result) == 0 {
			return res, errors.New("Zero Answer")
		}
		if answ.Result[0].SCode != "0" {
			return res, errors.New(answ.Result[0].SMsg)
		}

		// Маппим к вашему placeOrder_Response (он уже используется конвертером в okx пакете)
		tmp := make([]placeOrder_Response, 0, len(answ.Result))
		for _, it := range answ.Result {
			tmp = append(tmp, placeOrder_Response{
				ClOrdId: it.AlgoClOrdId,
				OrdId:   it.AlgoId,
				Tag:     s.brokerID,
				Ts:      it.Ts,
				SCode:   it.SCode,
				SMsg:    it.SMsg,
			})
		}

		return s.convert.convertPlaceOrder(tmp), nil
	}
	// --- end TP / SL branch ---

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["tdMode"] = "cross"
		case entity.MarginModeTypeIsolated:
			m["tdMode"] = "isolated"
		}
	}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.side != nil {
		m["side"] = strings.ToLower(string(*s.side))
	}

	if s.size != nil {
		m["sz"] = *s.size
	}

	if s.price != nil {
		m["px"] = *s.price
	}

	if s.orderType != nil {
		m["ordType"] = strings.ToLower(string(*s.orderType))
	}

	if s.clientOrderID != nil {
		m["clOrdId"] = *s.clientOrderID
	}

	if s.positionSide != nil {
		m["posSide"] = strings.ToLower(string(*s.positionSide))
	}

	if s.reduce != nil && *s.reduce == true {
		m["reduceOnly"] = true
	}

	// if s.tpPrice != nil || s.slPrice != nil {
	// 	attachAlgoOrds := []orderList_attachAlgoOrds{{}}
	// 	if s.tpPrice != nil {
	// 		attachAlgoOrds[0].TpTriggerPx = *s.tpPrice
	// 		attachAlgoOrds[0].TpOrdPx = "-1"
	// 	}

	// 	if s.slPrice != nil {
	// 		attachAlgoOrds[0].SlTriggerPx = *s.slPrice
	// 		attachAlgoOrds[0].SlOrdPx = "-1"
	// 	}
	// 	j, err := json.Marshal(attachAlgoOrds)
	// 	if err != nil {
	// 		return res, err
	// 	}

	// 	m["attachAlgoOrds"] = string(j)
	// }

	if s.brokerID != "" {
		m["tag"] = s.brokerID
	}
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []placeOrder_Response `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	if answ.Result[0].SCode != "0" {
		return res, errors.New(answ.Result[0].SMsg)
	}
	return s.convert.convertPlaceOrder(answ.Result), nil
}

// type futures_placeOrder_Response struct {
// 	ClOrdId string `json:"clOrdId"`
// 	OrdId   string `json:"ordId"`
// 	Tag     string `json:"tag"`
// 	Ts      string `json:"ts"`
// 	SCode   string `json:"sCode"`
// 	SMsg    string `json:"sMsg"`
// }
