package xemus

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_userTrades struct {
	callAPI callAPIFunc

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
}

func (s *futures_userTrades) Symbol(symbol string) *futures_userTrades {
	s.symbol = &symbol
	return s
}

func (s *futures_userTrades) StartTime(startTime int64) *futures_userTrades {
	s.startTime = &startTime
	return s
}

func (s *futures_userTrades) EndTime(endTime int64) *futures_userTrades {
	s.endTime = &endTime
	return s
}

func (s *futures_userTrades) Limit(limit int64) *futures_userTrades {
	s.limit = &limit
	return s
}

// Do maps the account's fills, GET /v1/trades.
func (s *futures_userTrades) Do(ctx context.Context) (res []entity.Futures_UserTrades, err error) {
	q := historyQuery(s.symbol, s.startTime, s.endTime, s.limit)

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/trades", query: q, signed: true})
	if err != nil {
		return res, err
	}
	var answ tradesResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	res = make([]entity.Futures_UserTrades, 0, len(answ.Rows))
	for _, t := range answ.Rows {
		res = append(res, toUserTrade(t))
	}
	return res, nil
}
