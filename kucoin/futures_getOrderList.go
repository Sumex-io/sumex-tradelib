package kucoin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ==============getOrderList=================
type futures_getOrderList struct {
	convert futures_converts

	callAPI   func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	symbol    *string
	instType  *string
	orderType *entity.OrderType
	limit     *int
}

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

func (s *futures_getOrderList) InstType(instType string) *futures_getOrderList {
	s.instType = &instType
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
	// 1) Обычные активные ордера
	normalItems, err := s.fetchActiveOrders(ctx, opts...)
	if err != nil {
		return res, err
	}

	// 2) Активные stop/tp/sl (KuCoin Futures держит их отдельно)
	stopItems, err := s.fetchActiveStopOrders(ctx, opts...)
	if err != nil {
		return res, err
	}

	// merge + dedupe by OrderID (id)
	merged := make([]futures_orderList, 0, len(normalItems)+len(stopItems))
	seen := map[string]struct{}{}

	for _, it := range normalItems {
		if it.ID == "" {
			merged = append(merged, it)
			continue
		}
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		merged = append(merged, it)
	}

	for _, it := range stopItems {
		if it.ID == "" {
			merged = append(merged, it)
			continue
		}
		if _, ok := seen[it.ID]; ok {
			continue
		}
		seen[it.ID] = struct{}{}
		merged = append(merged, it)
	}

	return s.convert.convertOrderList(merged), nil
}

// ---------------- internal fetchers ----------------

func (s *futures_getOrderList) fetchActiveOrders(ctx context.Context, opts ...utils.RequestOption) ([]futures_orderList, error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"status": "active",
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.limit != nil {
		m["pageSize"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var answ struct {
		Result struct {
			Items []futures_orderList `json:"items"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	return answ.Result.Items, nil
}

func (s *futures_getOrderList) fetchActiveStopOrders(ctx context.Context, opts ...utils.RequestOption) ([]futures_orderList, error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/stopOrders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"status": "active",
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.limit != nil {
		m["pageSize"] = *s.limit
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}
	// У stopOrders структура отличается, поэтому парсим в отдельную и маппим в futures_orderList
	var answ struct {
		Result struct {
			Items []futures_stopOrder `json:"items"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	out := make([]futures_orderList, 0, len(answ.Result.Items))
	for _, so := range answ.Result.Items {
		// Маппинг в общий формат.
		// Примечание: некоторые поля у stop-ордеров могут отсутствовать — оставляем нули/пустые.
		istp := false
		issl := false
		if so.Side == "sell" {
			if so.Stop == "up" {
				istp = true
			} else {
				issl = true
			}
		} else {
			if so.Stop == "up" {
				issl = true
			} else {
				istp = true
			}
		}
		out = append(out, futures_orderList{
			Symbol:       so.Symbol,
			ID:           so.ID,
			ClientOid:    so.ClientOid,
			OpType:       so.OpType,
			Type:         so.Type, // часто будет "limit" или "market"
			Side:         so.Side,
			PositionSide: so.PositionSide,
			Price:        so.StopPrice,
			Size:         so.Size,
			FilledSize:   so.FilledSize,
			TradeType:    so.TradeType,
			MarginMode:   so.MarginMode,
			Leverage:     so.Leverage,
			Status:       so.Status,
			CreatedAt:    so.CreatedAt,
			UpdatedAt:    so.UpdatedAt,
			TpOrder:      istp,
			SlOrder:      issl,
		})
	}

	return out, nil
}

// ---------------- response structs ----------------

// Обычные active orders (как у тебя уже было)
type futures_orderList struct {
	Symbol       string  `json:"symbol"`
	ID           string  `json:"id"`
	ClientOid    string  `json:"clientOid"`
	OpType       string  `json:"opType"`
	Type         string  `json:"type"`
	Side         string  `json:"side"`
	PositionSide string  `json:"positionSide"`
	Price        string  `json:"price"`
	StopPrice    string  `json:"stopPrice"`
	Size         float64 `json:"size"`
	FilledSize   float64 `json:"filledSize"`
	TradeType    string  `json:"tradeType"`
	MarginMode   string  `json:"marginMode"`
	Leverage     string  `json:"leverage"`
	Status       string  `json:"status"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
	TpOrder      bool    `json:"tpOrder" bson:"tpOrder"`
	SlOrder      bool    `json:"slOrder" bson:"slOrder"`
}

// Stop orders (TP/SL) — поля могут немного отличаться, но базовые обычно такие.
// Если у тебя в логах увидишь другие названия — скажешь, я подправлю struct 1-в-1.
type futures_stopOrder struct {
	Symbol       string `json:"symbol"`
	ID           string `json:"id"`
	ClientOid    string `json:"clientOid"`
	OpType       string `json:"opType"`
	Type         string `json:"type"`
	Side         string `json:"side"`
	Stop         string `json:"stop"`
	PositionSide string `json:"positionSide"`

	Price      string  `json:"price"`
	StopPrice  string  `json:"stopPrice"`
	Size       float64 `json:"size"`
	FilledSize float64 `json:"filledSize"`

	TradeType  string `json:"tradeType"`
	MarginMode string `json:"marginMode"`
	Leverage   string `json:"leverage"`
	Status     string `json:"status"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`

	// В stopOrders часто есть extra поля вроде stopPrice / stop / triggerPrice и т.п.
	// Мы их сейчас не используем для unified OrdersList.
	// StopPrice string `json:"stopPrice"`
	// Stop      string `json:"stop"`
}
