package sumex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_positionsHistory struct {
	callAPI callAPIFunc

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
}

func (s *futures_positionsHistory) Symbol(symbol string) *futures_positionsHistory {
	s.symbol = &symbol
	return s
}

func (s *futures_positionsHistory) StartTime(startTime int64) *futures_positionsHistory {
	s.startTime = &startTime
	return s
}

func (s *futures_positionsHistory) EndTime(endTime int64) *futures_positionsHistory {
	s.endTime = &endTime
	return s
}

func (s *futures_positionsHistory) Limit(limit int64) *futures_positionsHistory {
	s.limit = &limit
	return s
}

// Do maps GET /v1/history/positions. That route takes only symbol and limit, so the time window is
// applied here, on the close time. Rows that have closed nothing yet (a position still fully open)
// are left out: this is a history of closed positions.
func (s *futures_positionsHistory) Do(ctx context.Context) (res []entity.Futures_PositionsHistory, err error) {
	q := url.Values{}
	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		q.Set("symbol", strings.TrimSpace(*s.symbol))
	}
	q.Set("limit", strconv.FormatInt(clampLimit(s.limit), 10))

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/history/positions", query: q, signed: true})
	if err != nil {
		return res, err
	}
	var answ positionHistoryResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	res = []entity.Futures_PositionsHistory{}
	for _, row := range answ.Rows {
		if isZero(row.ClosedPositionQty.String()) {
			continue
		}
		h := toPositionHistory(row)
		if s.startTime != nil && *s.startTime > 0 && h.UpdateTime < *s.startTime {
			continue
		}
		if s.endTime != nil && *s.endTime > 0 && h.UpdateTime > *s.endTime {
			continue
		}
		res = append(res, h)
	}
	return res, nil
}
