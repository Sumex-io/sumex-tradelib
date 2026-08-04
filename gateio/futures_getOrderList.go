package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getOrderList struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle    *string
	symbol    *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_getOrderList) Settle(settle string) *futures_getOrderList {
	s.settle = &settle
	return s
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) OrderType(orderType entity.OrderType) *futures_getOrderList {
	s.orderType = &orderType
	return s
}

func (s *futures_getOrderList) Limit(limit int) *futures_getOrderList {
	s.limit = &limit
	return s
}

func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
	settleDefault := "usdt"
	if s.settle == nil {
		s.settle = &settleDefault
	}

	// ---------------- 1) NORMAL OPEN ORDERS ----------------
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/api/v4/futures/{settle}/orders",
			SecType:  utils.SecTypeSigned,
		}
		r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

		m := utils.Params{
			"status": "open",
		}
		if s.symbol != nil && *s.symbol != "" {
			m["contract"] = *s.symbol
		}
		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		orders := []futures_orderList{}
		if e := json.Unmarshal(data, &orders); e != nil {
			return res, e
		}

		// используем существующий конвертер для обычных ордеров
		res = append(res, s.convert.convertOrderList(orders)...)
	}

	// ---------------- 2) TRIGGER (TP/SL) OPEN ORDERS ----------------
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/api/v4/futures/{settle}/price_orders",
			SecType:  utils.SecTypeSigned,
		}
		r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

		m := utils.Params{
			"status": "open",
		}
		if s.symbol != nil && *s.symbol != "" {
			m["contract"] = *s.symbol
		}
		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		priceOrders := []gateio_priceOrderList{}
		if e := json.Unmarshal(data, &priceOrders); e != nil {
			return res, e
		}

		res = append(res, convertGateioPriceOrdersToStd(priceOrders)...)
	}

	return res, nil
}

// ---------------- normal orders (as was) ----------------

type futures_orderList struct {
	ID          int64   `json:"id"`
	Contract    string  `json:"contract"`
	Status      string  `json:"status"`
	Size        int64   `json:"size"`
	Left        int64   `json:"left"`
	Is_close    bool    `json:"is_close"`
	Text        string  `json:"text"`
	Price       string  `json:"price"`
	Fill_price  string  `json:"fill_price"`
	Create_time float64 `json:"create_time"`
	Update_time float64 `json:"update_time"`
}

// ---------------- price_orders (trigger/auto orders) ----------------

// Gate API v4 futures price_orders обычно возвращает массив объектов,
// где внутри есть initial/trigger.
// Нам достаточно минимального набора полей для unified order list.
type gateio_priceOrderList struct {
	ID          int64   `json:"id"`
	Status      string  `json:"status"`
	Create_time float64 `json:"create_time"`
	Update_time float64 `json:"update_time"`

	Initial gateio_priceOrderInitialList `json:"initial"`
	Trigger gateio_priceOrderTriggerList `json:"trigger"`
}

type gateio_priceOrderInitialList struct {
	Contract   string `json:"contract"`
	Size       int64  `json:"size"` // иногда обязано быть 0 при auto_size=close_long/close_short
	Price      string `json:"price"`
	Tif        string `json:"tif"`
	ReduceOnly bool   `json:"reduce_only"`
	Text       string `json:"text,omitempty"`

	// Ключевое для hedge/dual mode закрытия:
	// "close_long" / "close_short" / иногда пусто
	AutoSize string `json:"auto_size,omitempty"`
}

type gateio_priceOrderTriggerList struct {
	StrategyType int    `json:"strategy_type"`
	PriceType    int    `json:"price_type"`
	Rule         int    `json:"rule"` // 1 => >= , 2 => <=
	Expiration   int    `json:"expiration"`
	Price        string `json:"price"`
}

func convertGateioPriceOrdersToStd(in []gateio_priceOrderList) (out []entity.Futures_OrdersList) {
	if len(in) == 0 {
		return out
	}

	for _, po := range in {
		symbol := po.Initial.Contract
		orderID := int64ToString(po.ID)

		// derive side/positionSide for TP/SL-close orders
		side := ""
		positionSide := ""

		auto := strings.ToLower(strings.TrimSpace(po.Initial.AutoSize))
		switch auto {
		case "close_long":
			// закрываем LONG -> SELL
			side = "SELL"
			positionSide = "LONG"
		case "close_short":
			// закрываем SHORT -> BUY
			side = "BUY"
			positionSide = "SHORT"
		default:
			// fallback по знаку size (эвристика)
			if po.Initial.Size < 0 {
				side = "SELL"
				positionSide = "LONG"
			} else if po.Initial.Size > 0 {
				side = "BUY"
				positionSide = "SHORT"
			} else {
				// size==0 и auto_size пустой — не угадаем корректно, но пусть будет пусто
				side = ""
				positionSide = ""
			}
		}

		// TP vs SL:
		// если закрываем LONG: TP обычно rule=1 (>=), SL rule=2 (<=)
		// если закрываем SHORT: TP обычно rule=2 (<=), SL rule=1 (>=)
		isTP := false
		isSL := false
		if positionSide == "LONG" {
			if po.Trigger.Rule == 1 {
				isTP = true
			} else if po.Trigger.Rule == 2 {
				isSL = true
			}
		} else if positionSide == "SHORT" {
			if po.Trigger.Rule == 2 {
				isTP = true
			} else if po.Trigger.Rule == 1 {
				isSL = true
			}
		}

		// create/update times: gate отдаёт seconds (float), в проекте обычно ms
		ctMs := int64(po.Create_time * 1000.0)
		utMs := int64(po.Update_time * 1000.0)

		// Для trigger ордеров логичнее показывать триггерную цену:
		price := po.Trigger.Price

		// Status приводим к верхнему регистру как у остальных
		status := strings.ToUpper(po.Status)
		if status == "" {
			status = "OPEN"
		}

		if po.Initial.Size < 0 {
			po.Initial.Size = 0 - po.Initial.Size
		}
		out = append(out, entity.Futures_OrdersList{
			Symbol:        symbol,
			OrderID:       orderID,
			ClientOrderID: po.Initial.Text,

			Side:         side,
			PositionSide: positionSide,

			// тут у Gate trigger-ордеров нет “executed size” как у лимитки;
			// оставляем 0/пусто — безопаснее.
			PositionSize: utils.Int64ToString(po.Initial.Size),
			ExecutedSize: "0",

			Price:  price,
			Type:   "TRIGGER", // чтобы визуально отличалось от LIMIT/MARKET
			Status: status,

			CreateTime: ctMs,
			UpdateTime: utMs,

			// для унификации:
			TpOrder: isTP,
			SlOrder: isSL,
		})
	}

	return out
}

func int64ToString(v int64) string {
	// без fmt чтобы не тянуть лишнее
	return utils.Int64ToString(v)
}
