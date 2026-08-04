package whitebit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_ordersHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
}

func (s *spot_ordersHistory) Symbol(symbol string) *spot_ordersHistory {
	s.symbol = &symbol
	return s
}

func (s *spot_ordersHistory) StartTime(startTime int64) *spot_ordersHistory {
	s.startTime = &startTime
	return s
}

func (s *spot_ordersHistory) EndTime(endTime int64) *spot_ordersHistory {
	s.endTime = &endTime
	return s
}

func (s *spot_ordersHistory) Limit(limit int64) *spot_ordersHistory {
	s.limit = &limit
	return s
}

func (s *spot_ordersHistory) Page(page int64) *spot_ordersHistory {
	s.page = &page
	return s
}

func (s *spot_ordersHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_OrdersHistory, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/trade-account/executed-history",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// market опционален
	if s.symbol != nil {
		m["market"] = *s.symbol
	}

	if s.limit != nil && *s.limit > 0 {
		m["limit"] = *s.limit
	}

	if s.page != nil && *s.page > 0 {
		// WhiteBIT использует offset
		m["offset"] = *s.page
	}

	// startDate / endDate — в секундах Unix по докам.
	// В onetrades время обычно в мс, поэтому делим на 1000.
	if s.startTime != nil {
		m["startDate"] = *s.startTime / 1000
	}
	if s.endTime != nil {
		m["endDate"] = *s.endTime / 1000
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// Ответ: {"BTC_USDT":[ {...}, {...} ], "ETH_USDT":[ ... ] }
	raw := make(map[string][]spot_tradeHistoryWB)
	if err = json.Unmarshal(data, &raw); err != nil {
		return res, err
	}

	var flat []spot_tradeHistoryWB
	for market, trades := range raw {
		for _, t := range trades {
			t.Market = market
			flat = append(flat, t)
		}
	}

	return s.convert.convertOrdersHistory(flat), nil
}

// структура под /api/v4/trade-account/executed-history
type spot_tradeHistoryWB struct {
	ID            int64   `json:"id"`
	ClientOrderID string  `json:"clientOrderId"`
	Time          float64 `json:"time"` // seconds with fraction
	Side          string  `json:"side"`
	Role          int     `json:"role"`
	Amount        string  `json:"amount"`
	Price         string  `json:"price"`
	Deal          string  `json:"deal"`
	Fee           string  `json:"fee"`
	OrderID       int64   `json:"orderId"`
	FeeAsset      string  `json:"feeAsset"` // не используем, но пусть будет
	Market        string  `json:"-"`        // заполняем вручную из map key
}
