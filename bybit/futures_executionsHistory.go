package bybit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_executionsHistory struct {
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

func (s *futures_executionsHistory) Symbol(symbol string) *futures_executionsHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_executionsHistory) Category(category string) *futures_executionsHistory {
	s.category = &category
	return s
}

func (s *futures_executionsHistory) StartTime(startTime int64) *futures_executionsHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_executionsHistory) EndTime(endTime int64) *futures_executionsHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_executionsHistory) Limit(limit int64) *futures_executionsHistory {
	s.limit = &limit
	return s
}

func (s *futures_executionsHistory) Page(page int64) *futures_executionsHistory {
	s.page = &page
	return s
}

func (s *futures_executionsHistory) Cursor(cursor string) *futures_executionsHistory {
	s.cursor = &cursor
	return s
}

func (s *futures_executionsHistory) OrderID(orderID string) *futures_executionsHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_executionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_ExecutionsHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/execution/list",
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
		Result futures_executionsHistory_Response `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertExecutionsHistory(answ.Result), nil
}

type futures_executionsHistory_Response struct {
	List []struct {
		Symbol      string `json:"symbol"`
		OrderId     string `json:"orderId"`
		OrderLinkId string `json:"orderLinkId"`
		Side        string `json:"side"`
		OrderQty    string `json:"orderQty"`
		ExecQty     string `json:"execQty"`
		ClosedSize  string `json:"closedSize"`
		OrderPrice  string `json:"orderPrice"`
		ExecPrice   string `json:"execPrice"`
		MarkPrice   string `json:"markPrice"`

		ExecFee   string `json:"execFee"`
		ExecValue string `json:"execValue"`

		OrderType string `json:"orderType"`
		ExecType  string `json:"execType"`
		ExecTime  string `json:"execTime"`
	} `json:"list"`
	NextPageCursor string `json:"nextPageCursor"`
}
