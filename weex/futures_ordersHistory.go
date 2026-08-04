package weex

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
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
	page      *int64

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

func (s *futures_ordersHistory) Page(page int64) *futures_ordersHistory {
	s.page = &page
	return s
}

func (s *futures_ordersHistory) OrderID(orderID string) *futures_ordersHistory {
	s.orderID = &orderID
	return s
}

func (s *futures_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersHistory, err error) {
	{
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/capi/v3/order/history",
			SecType:  utils.SecTypeSigned,
		}

		m := utils.Params{}

		if s.symbol != nil && *s.symbol != "" {
			m["symbol"] = *s.symbol
		}
		if s.limit != nil && *s.limit > 0 {
			m["limit"] = *s.limit
		}
		if s.page != nil && *s.page >= 0 {
			m["page"] = *s.page
		}
		if s.startTime != nil {
			m["startTime"] = *s.startTime
		}
		if s.endTime != nil {
			m["endTime"] = *s.endTime
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ []futures_ordersHistory_Response
		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertOrdersHistory(answ)...)
	}

	if s.symbol != nil && *s.symbol != "" {
		r := &utils.Request{
			Method:   http.MethodGet,
			Endpoint: "/capi/v2/order/historyPlan",
			SecType:  utils.SecTypeSigned,
		}

		v2symbol := strings.ToLower(*s.symbol)
		if !strings.HasPrefix(v2symbol, "cmt_") {
			v2symbol = "cmt_" + v2symbol
		}

		m := utils.Params{
			"symbol": v2symbol,
		}

		if s.limit != nil && *s.limit > 0 {
			m["pageSize"] = *s.limit
		}
		if s.startTime != nil {
			m["startTime"] = *s.startTime
		}
		if s.endTime != nil {
			m["endTime"] = *s.endTime
		}

		r.SetParams(m)

		data, _, e := s.callAPI(ctx, r, opts...)
		if e != nil {
			return res, e
		}

		var answ futures_algoOrdersHistoryPage
		if e := json.Unmarshal(data, &answ); e != nil {
			return res, e
		}

		res = append(res, s.convert.convertAlgoOrdersHistory(answ.List)...)
	}

	if s.orderID != nil && *s.orderID != "" {
		filtered := make([]entity.Futures_OrdersHistory, 0, 1)
		for _, item := range res {
			if item.OrderID == *s.orderID {
				filtered = append(filtered, item)
			}
		}
		res = filtered
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].CreateTime > res[j].CreateTime
	})

	return res, nil
}

type futures_ordersHistory_Response struct {
	Symbol        string `json:"symbol"`
	OrderId       int64  `json:"orderId"`
	ClientOrderId string `json:"clientOrderId"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Type          string `json:"type"`
	OrigQty       string `json:"origQty"`
	Price         string `json:"price"`
	ExecutedQty   string `json:"executedQty"`
	AvgPrice      string `json:"avgPrice"`
	CumQuote      string `json:"cumQuote"`
	Status        string `json:"status"`
	Time          int64  `json:"time"`
	UpdateTime    int64  `json:"updateTime"`
	TimeInForce   string `json:"timeInForce"`
}

type futures_algoOrdersHistoryPage struct {
	List     []futures_algoOrdersHistoryItem `json:"list"`
	NextPage bool                            `json:"nextPage"`
}

type futures_algoOrdersHistoryItem struct {
	Symbol                string `json:"symbol"`
	Size                  string `json:"size"`
	ClientOID             string `json:"client_oid"`
	CreateTime            string `json:"createTime"`
	FilledQty             string `json:"filled_qty"`
	Fee                   string `json:"fee"`
	OrderID               string `json:"order_id"`
	Price                 string `json:"price"`
	PriceAvg              string `json:"price_avg"`
	Status                string `json:"status"`
	Type                  string `json:"type"`
	OrderType             string `json:"order_type"`
	TotalProfits          string `json:"totalProfits"`
	TriggerPrice          string `json:"triggerPrice"`
	TriggerPriceType      string `json:"triggerPriceType"`
	TriggerTime           string `json:"triggerTime"`
	PresetTakeProfitPrice string `json:"presetTakeProfitPrice"`
	PresetStopLossPrice   string `json:"presetStopLossPrice"`
}
