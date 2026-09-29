package xemus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

const defaultHistoryLimit = 100

type futures_ordersHistory struct {
	callAPI callAPIFunc

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
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

// Do maps GET /v1/orders?status=COMPLETED — Orderly's filled, cancelled and rejected orders — newest
// first. A TP/SL that fired shows here as the regular order it became; one cancelled before
// triggering never became an order and does not.
func (s *futures_ordersHistory) Do(ctx context.Context) (res []entity.Futures_OrdersHistory, err error) {
	q := historyQuery(s.symbol, s.startTime, s.endTime, s.limit)
	q.Set("status", "COMPLETED")
	q.Set("sort_by", "CREATED_TIME_DESC")

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/orders", query: q, signed: true})
	if err != nil {
		return res, err
	}
	var answ ordersResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	res = make([]entity.Futures_OrdersHistory, 0, len(answ.Rows))
	for _, o := range answ.Rows {
		res = append(res, toOrderHistory(o))
	}
	return res, nil
}

// historyQuery builds the symbol / start_t / end_t / size query the paginated history reads share.
// The limit is clamped to 1..maxPageSize rather than refused: the caller's limit is a UI page
// size, and a larger one should still return the most one page gives.
func historyQuery(symbol *string, startTime, endTime, limit *int64) url.Values {
	q := url.Values{}
	if symbol != nil && strings.TrimSpace(*symbol) != "" {
		q.Set("symbol", strings.TrimSpace(*symbol))
	}
	if startTime != nil && *startTime > 0 {
		q.Set("start_t", strconv.FormatInt(*startTime, 10))
	}
	if endTime != nil && *endTime > 0 {
		q.Set("end_t", strconv.FormatInt(*endTime, 10))
	}
	q.Set("size", strconv.FormatInt(clampLimit(limit), 10))
	return q
}

func clampLimit(limit *int64) int64 {
	if limit == nil || *limit <= 0 {
		return defaultHistoryLimit
	}
	if *limit > maxPageSize {
		return maxPageSize
	}
	return *limit
}
