package bybit

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

	return s.convert.convertOrdersHistory(answ.Result), nil
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
