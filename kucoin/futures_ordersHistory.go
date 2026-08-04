package kucoin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) StartTime(startTime int64) *futures_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_ordersHistory) EndTime(endTime int64) *futures_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_ordersHistory) Limit(limit int64) *futures_ordersHistory {
	s.limit = &limit
	return s
}

func (s *futures_ordersHistory) Page(page int64) *futures_ordersHistory {
	s.page = &page
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	// ------------------------------------------------
	// 1) Общая история done orders
	// ------------------------------------------------
	normalItems, err := s.fetchDoneOrders(ctx, opts...)
	if err != nil {
		return res, err
	}

	out := s.convert.convertOrdersHistory(normalItems)

	// ------------------------------------------------
	// 2) Done stop orders для разметки TP/SL
	// ------------------------------------------------
	stopItems, err := s.fetchDoneStopOrders(ctx, opts...)
	if err != nil {
		return res, err
	}

	// индексируем историю по clientOid и orderId
	indexByClientOid := make(map[string]int, len(out))
	indexByOrderID := make(map[string]int, len(out))

	for i := range out {
		if out[i].ClientOrderID != "" {
			indexByClientOid[out[i].ClientOrderID] = i
		}
		if out[i].OrderID != "" {
			indexByOrderID[out[i].OrderID] = i
		}
	}

	// размечаем уже исполненные ордера как TP/SL, если нашли соответствующий stop order
	for _, so := range stopItems {
		isTP, isSL := kucoinStopFlags(so.Side, so.Stop)
		if !isTP && !isSL {
			continue
		}

		// Сначала матчим по clientOid
		if so.ClientOid != "" {
			if idx, ok := indexByClientOid[so.ClientOid]; ok {
				if isTP {
					out[idx].TpOrder = true
				}
				if isSL {
					out[idx].SlOrder = true
				}
				continue
			}
		}

		// fallback по id
		if so.ID != "" {
			if idx, ok := indexByOrderID[so.ID]; ok {
				if isTP {
					out[idx].TpOrder = true
				}
				if isSL {
					out[idx].SlOrder = true
				}
				continue
			}
		}
	}

	return out, nil
}

func (s *futures_ordersHistory) fetchDoneOrders(ctx context.Context, opts ...utils.RequestOption) ([]futures_ordersHistory_Response, error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/orders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"status": "done",
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["pageSize"] = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m["currentPage"] = *s.page
	}
	if s.startTime != nil {
		m["startAt"] = *s.startTime
	}
	if s.endTime != nil {
		m["endAt"] = *s.endTime
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var answ struct {
		Result struct {
			Items []futures_ordersHistory_Response `json:"items"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	return answ.Result.Items, nil
}

func (s *futures_ordersHistory) fetchDoneStopOrders(ctx context.Context, opts ...utils.RequestOption) ([]futures_stopOrderHistory, error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/stopOrders",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"status": "done",
	}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m["pageSize"] = *s.limit
	}
	if s.page != nil && *s.page > 0 {
		m["currentPage"] = *s.page
	}
	if s.startTime != nil {
		m["startAt"] = *s.startTime
	}
	if s.endTime != nil {
		m["endAt"] = *s.endTime
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var answ struct {
		Result struct {
			Items []futures_stopOrderHistory `json:"items"`
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, err
	}

	return answ.Result.Items, nil
}

type futures_ordersHistory_Response struct {
	Symbol        string  `json:"symbol"`
	ID            string  `json:"id"`
	ClientOid     string  `json:"clientOid"`
	OpType        string  `json:"opType"`
	Type          string  `json:"type"`
	Side          string  `json:"side"`
	PositionSide  string  `json:"positionSide"`
	Price         string  `json:"price"`
	Stop          string  `json:"stop"`
	StopPrice     string  `json:"stopPrice"`
	AvgDealPrice  string  `json:"avgDealPrice"`
	Liquidity     string  `json:"liquidity"`
	Size          float64 `json:"size"`
	Value         string  `json:"value"`
	Leverage      string  `json:"leverage"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	MarginMode    string  `json:"marginMode"`
	FilledSize    float64 `json:"filledSize"`
	FilledValue   string  `json:"filledValue"`
	DealSize      float64 `json:"dealSize"`
	DealValue     string  `json:"dealValue"`
	Status        string  `json:"status"`
	Fee           string  `json:"fee"`
	FeeCurrency   string  `json:"feeCurrency"`
	ReduceOnly    bool    `json:"reduceOnly"`
	CloseOrder    bool    `json:"closeOrder"`
	StopTriggered bool    `json:"stopTriggered"`
	IsActive      bool    `json:"isActive"`
}

type futures_stopOrderHistory struct {
	Symbol        string  `json:"symbol"`
	ID            string  `json:"id"`
	ClientOid     string  `json:"clientOid"`
	OpType        string  `json:"opType"`
	Type          string  `json:"type"`
	Side          string  `json:"side"`
	Stop          string  `json:"stop"`
	PositionSide  string  `json:"positionSide"`
	Price         string  `json:"price"`
	StopPrice     string  `json:"stopPrice"`
	Size          float64 `json:"size"`
	FilledSize    float64 `json:"filledSize"`
	DealSize      float64 `json:"dealSize"`
	TradeType     string  `json:"tradeType"`
	MarginMode    string  `json:"marginMode"`
	Leverage      string  `json:"leverage"`
	Status        string  `json:"status"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	StopTriggered bool    `json:"stopTriggered"`
	CloseOrder    bool    `json:"closeOrder"`
	IsActive      bool    `json:"isActive"`
}
