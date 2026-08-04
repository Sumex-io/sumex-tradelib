package blofin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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
	// page/cursor Blofin обычно делает через before/after, можно при необходимости добавить позднее
	orderID *string
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

func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	// ------------------------------------------------
	// 1) Обычная история исполненных ордеров
	// ------------------------------------------------
	r1 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/trade/orders-history",
		SecType:  utils.SecTypeSigned,
	}

	m1 := utils.Params{
		"state": "filled",
	}

	if s.symbol != nil {
		m1["instId"] = *s.symbol
	}
	if s.startTime != nil {
		m1["startTime"] = *s.startTime
	}
	if s.endTime != nil {
		m1["endTime"] = *s.endTime
	}
	if s.limit != nil && *s.limit > 0 {
		m1["limit"] = *s.limit
	}
	if s.orderID != nil {
		m1["orderId"] = *s.orderID
	}

	r1.SetParams(m1)

	data1, _, err := s.callAPI(ctx, r1, opts...)
	if err != nil {
		return res, err
	}

	var answ1 struct {
		Result []futures_ordersHistory_Response `json:"data"`
	}

	if err = json.Unmarshal(data1, &answ1); err != nil {
		return res, err
	}

	out := convertOrdersHistoryBlofin(answ1.Result)

	// ------------------------------------------------
	// 2) История TP/SL ордеров
	//    Берём только effective, т.к. по вашему правилу
	//    в history возвращаем только реально исполнившиеся.
	// ------------------------------------------------
	r2 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/trade/orders-tpsl-history",
		SecType:  utils.SecTypeSigned,
	}

	m2 := utils.Params{
		"state": "effective",
	}

	if s.symbol != nil {
		m2["instId"] = *s.symbol
	}
	if s.limit != nil && *s.limit > 0 {
		m2["limit"] = *s.limit
	}
	if s.orderID != nil {
		// Для TP/SL history у Blofin используется tpslId
		m2["tpslId"] = *s.orderID
	}

	r2.SetParams(m2)

	data2, _, err := s.callAPI(ctx, r2, opts...)
	if err != nil {
		return res, err
	}

	var answ2 struct {
		Result []futures_tpslOrdersHistory_Response `json:"data"`
	}

	if err = json.Unmarshal(data2, &answ2); err != nil {
		return res, err
	}

	tpslOut := convertTPSLOrdersHistoryBlofin(answ2.Result)

	// ------------------------------------------------
	// 3) Merge:
	//    - сначала пытаемся матчить по algoId == tpslId
	//    - потом по clientOrderId
	//    - если не нашли, добавляем отдельную запись
	// ------------------------------------------------
	indexByAlgoID := make(map[string]int, len(out))
	indexByClientOrderID := make(map[string]int, len(out))

	for i := range out {
		if out[i].PositionID != "" {
			indexByAlgoID[out[i].PositionID] = i
		}
		if out[i].ClientOrderID != "" {
			indexByClientOrderID[out[i].ClientOrderID] = i
		}
	}

	for _, p := range tpslOut {
		if idx, ok := indexByAlgoID[p.OrderID]; ok {
			if p.TpOrder {
				out[idx].TpOrder = true
			}
			if p.SlOrder {
				out[idx].SlOrder = true
			}
			continue
		}

		if p.ClientOrderID != "" {
			if idx, ok := indexByClientOrderID[p.ClientOrderID]; ok {
				if p.TpOrder {
					out[idx].TpOrder = true
				}
				if p.SlOrder {
					out[idx].SlOrder = true
				}
				continue
			}
		}

		// fallback: если обычная история не дала связку, вернём отдельной строкой
		// Это лучше, чем потерять исполненный TP/SL совсем.
		out = append(out, p)
	}

	return out, nil
}

type futures_ordersHistory_Response struct {
	InstId            string `json:"instId"`
	OrderId           string `json:"orderId"`
	ClientOrderId     string `json:"clientOrderId"`
	AlgoClientOrderId string `json:"algoClientOrderId"`
	AlgoId            string `json:"algoId"`
	Side              string `json:"side"`
	PositionSide      string `json:"positionSide"`
	Size              string `json:"size"`
	FilledSize        string `json:"filledSize"`
	Price             string `json:"price"`
	AveragePrice      string `json:"averagePrice"`
	Fee               string `json:"fee"`
	Pnl               string `json:"pnl"`
	Leverage          string `json:"leverage"`
	OrderType         string `json:"orderType"`
	State             string `json:"state"`
	MarginMode        string `json:"marginMode"`
	OrderCategory     string `json:"orderCategory"`
	TpTriggerPrice    string `json:"tpTriggerPrice"`
	TpOrderPrice      string `json:"tpOrderPrice"`
	SlTriggerPrice    string `json:"slTriggerPrice"`
	SlOrderPrice      string `json:"slOrderPrice"`
	CreateTime        string `json:"createTime"`
	UpdateTime        string `json:"updateTime"`
}

type futures_tpslOrdersHistory_Response struct {
	TpslId         string `json:"tpslId"`
	ClientOrderId  string `json:"clientOrderId"`
	InstId         string `json:"instId"`
	MarginMode     string `json:"marginMode"`
	PositionSide   string `json:"positionSide"`
	Side           string `json:"side"`
	OrderType      string `json:"orderType"`
	Size           string `json:"size"`
	ActualSize     string `json:"actualSize"`
	Leverage       string `json:"leverage"`
	State          string `json:"state"`
	OrderCategory  string `json:"orderCategory"`
	TpTriggerPrice string `json:"tpTriggerPrice"`
	TpOrderPrice   string `json:"tpOrderPrice"`
	SlTriggerPrice string `json:"slTriggerPrice"`
	SlOrderPrice   string `json:"slOrderPrice"`
	CreateTime     string `json:"createTime"`
}

func convertOrdersHistoryBlofin(in []futures_ordersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		status := strings.ToUpper(item.State)
		if status != "FILLED" {
			continue
		}

		hedgeMode := false
		posSide := item.PositionSide

		if strings.ToLower(posSide) == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				posSide = "SHORT"
			} else {
				posSide = "LONG"
			}
		} else {
			hedgeMode = true
		}

		orderCategory := strings.ToLower(item.OrderCategory)
		tpOrder := orderCategory == "tp"
		slOrder := orderCategory == "sl"

		clientOrderID := item.ClientOrderId
		if clientOrderID == "" {
			clientOrderID = item.AlgoClientOrderId
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        item.InstId,
			OrderID:       item.OrderId,
			ClientOrderID: clientOrderID,
			// Временный мост для merge с TP/SL history:
			// сюда кладём algoId, если он есть.
			PositionID:     item.AlgoId,
			Side:           strings.ToUpper(item.Side),
			PositionSide:   strings.ToUpper(posSide),
			PositionSize:   item.Size,
			ExecutedSize:   item.FilledSize,
			Price:          item.Price,
			ExecutedPrice:  item.AveragePrice,
			RealisedProfit: item.Pnl,
			Fee:            item.Fee,
			Leverage:       item.Leverage,
			HedgeMode:      hedgeMode,
			MarginMode:     strings.ToUpper(item.MarginMode),
			Type:           strings.ToUpper(item.OrderType),
			Status:         status,
			CreateTime:     utils.StringToInt64(item.CreateTime),
			UpdateTime:     utils.StringToInt64(item.UpdateTime),
			TpOrder:        tpOrder,
			SlOrder:        slOrder,
		})
	}

	return out
}

func convertTPSLOrdersHistoryBlofin(in []futures_tpslOrdersHistory_Response) (out []entity.Futures_OrdersHistory) {
	if len(in) == 0 {
		return out
	}

	for _, item := range in {
		status := strings.ToUpper(item.State)
		if status != "EFFECTIVE" {
			continue
		}

		orderCategory := strings.ToLower(item.OrderCategory)
		tpOrder := orderCategory == "tp"
		slOrder := orderCategory == "sl"

		// На всякий случай fallback по заполненным trigger-полям
		if !tpOrder && item.TpTriggerPrice != "" {
			tpOrder = true
		}
		if !slOrder && item.SlTriggerPrice != "" {
			slOrder = true
		}

		if !tpOrder && !slOrder {
			continue
		}

		hedgeMode := false
		posSide := item.PositionSide

		if strings.ToLower(posSide) == "net" {
			if strings.ToUpper(item.Side) == "SELL" {
				posSide = "SHORT"
			} else {
				posSide = "LONG"
			}
		} else {
			hedgeMode = true
		}

		price := ""
		if tpOrder {
			price = item.TpTriggerPrice
			if price == "" {
				price = item.TpOrderPrice
			}
		}
		if slOrder {
			price = item.SlTriggerPrice
			if price == "" {
				price = item.SlOrderPrice
			}
		}

		out = append(out, entity.Futures_OrdersHistory{
			Symbol:        item.InstId,
			OrderID:       item.TpslId,
			ClientOrderID: item.ClientOrderId,
			Side:          strings.ToUpper(item.Side),
			PositionSide:  strings.ToUpper(posSide),
			PositionSize:  item.Size,
			ExecutedSize:  item.ActualSize,
			Price:         price,
			ExecutedPrice: "",
			Leverage:      item.Leverage,
			HedgeMode:     hedgeMode,
			MarginMode:    strings.ToUpper(item.MarginMode),
			Type:          strings.ToUpper(item.OrderType),
			Status:        status,
			CreateTime:    utils.StringToInt64(item.CreateTime),
			UpdateTime:    utils.StringToInt64(item.CreateTime),
			TpOrder:       tpOrder,
			SlOrder:       slOrder,
		})
	}

	return out
}
