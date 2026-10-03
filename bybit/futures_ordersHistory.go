package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
	cursor    *string
	category  *string

	orderID *string
}

func (s *futures_ordersHistory) Symbol(symbol string) *futures_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_ordersHistory) Category(category string) *futures_ordersHistory {
	s.category = &category
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

func (s *futures_ordersHistory) Cursor(cursor string) *futures_ordersHistory {
	s.cursor = &cursor
	return s
}

func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/order/history",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"category": "linear", "orderStatus": "Filled"}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.category != nil {
		m["category"] = *s.category
	}
	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}

	if s.cursor != nil && *s.cursor != "" {
		m["cursor"] = *s.cursor
	}

	if s.startTime != nil {
		m["startTime"] = *s.startTime
	}
	if s.endTime != nil {
		m["endTime"] = *s.endTime
	}

	if s.orderID != nil {
		m["orderId"] = *s.orderID
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_ordersHistory_Response `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	res = s.convert.convertOrdersHistory(answ.Result)
	s.attachLeverage(ctx, res, opts...)
	return res, nil
}

type futures_ordersHistory_Response struct {
	List []struct {
		Symbol        string            `json:"symbol"`
		OrderId       string            `json:"orderId"`
		OrderLinkId   string            `json:"orderLinkId"`
		StopOrderType string            `json:"stopOrderType"`
		CreateType    string            `json:"createType"`
		Side          string            `json:"side"`
		PositionIdx   int64             `json:"positionIdx"`
		Qty           string            `json:"qty"`
		CumExecQty    string            `json:"cumExecQty"`
		Price         string            `json:"price"`
		TriggerPrice  string            `json:"triggerPrice"`
		AvgPrice      string            `json:"avgPrice"`
		CumExecFee    string            `json:"cumExecFee"`
		CumFeeDetail  map[string]string `json:"cumFeeDetail"`
		OrderType     string            `json:"orderType"`
		OrderStatus   string            `json:"orderStatus"`
		CreatedTime   string            `json:"createdTime"`
		UpdatedTime   string            `json:"updatedTime"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}

const (
	closedPnlWindow      = 7 * 24 * time.Hour
	closedPnlMaxRequests = 5
)

// Order history carries no leverage; closing orders take the position leverage from their closed-pnl record.
func (s *futures_ordersHistory) attachLeverage(ctx context.Context, orders []entity.Futures_OrdersHistory, opts ...utils.RequestOption) {
	var from, to int64
	for _, order := range orders {
		if from == 0 || order.CreateTime < from {
			from = order.CreateTime
		}
		if order.UpdateTime > to {
			to = order.UpdateTime
		}
	}
	if from == 0 || to < from {
		return
	}

	leverage := make(map[string]string)
	requests := 0
	for end := to; end >= from && requests < closedPnlMaxRequests; end -= closedPnlWindow.Milliseconds() {
		start := max(end-closedPnlWindow.Milliseconds()+1, from)
		cursor := ""
		for requests < closedPnlMaxRequests {
			requests++
			page, next, err := s.fetchClosedPnl(ctx, start, end, cursor, opts...)
			if err != nil {
				return
			}
			for _, item := range page {
				if item.Leverage != "" {
					leverage[item.OrderId] = item.Leverage
				}
			}
			if next == "" || next == cursor || len(page) == 0 {
				break
			}
			cursor = next
		}
	}

	for i := range orders {
		if orders[i].Leverage == "" {
			orders[i].Leverage = leverage[orders[i].OrderID]
		}
	}
}

func (s *futures_ordersHistory) fetchClosedPnl(ctx context.Context, start, end int64, cursor string, opts ...utils.RequestOption) ([]futures_PositionsHistory_Response, string, error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/position/closed-pnl",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"category": "linear", "limit": 100, "startTime": start, "endTime": end}
	if s.category != nil {
		m["category"] = *s.category
	}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if cursor != "" {
		m["cursor"] = cursor
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return nil, "", err
	}

	var answ struct {
		Result struct {
			List           []futures_PositionsHistory_Response `json:"list"`
			NextPageCursor string                              `json:"nextPageCursor"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &answ); err != nil {
		return nil, "", err
	}
	return answ.Result.List, answ.Result.NextPageCursor, nil
}
