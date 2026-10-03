package weex

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

const (
	weexUserTradesPageLimit = 100
	weexUserTradesMaxPages  = 20
	weexUserTradesWindowMs  = int64(7 * 24 * time.Hour / time.Millisecond)
	weexIncomeMaxRangeMs    = int64(100 * 24 * time.Hour / time.Millisecond)
	weexLiquidationMatchMs  = int64(time.Minute / time.Millisecond)
)

type futures_positionsHistory struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	startTime *int64
	endTime   *int64
	limit     *int64
	page      *int64
	orderID   *string
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

func (s *futures_positionsHistory) Page(page int64) *futures_positionsHistory {
	s.page = &page
	return s
}

func (s *futures_positionsHistory) OrderID(orderID string) *futures_positionsHistory {
	s.orderID = &orderID
	return s
}

// WeEx serves no positions history endpoint, so positions are replayed from the account fills.
func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	end := time.Now().UnixMilli()
	if s.endTime != nil && *s.endTime > 0 {
		end = *s.endTime
	}

	start := end - weexUserTradesWindowMs + 1
	if s.startTime != nil && *s.startTime > 0 {
		start = *s.startTime
	}

	fills, err := s.fetchFills(ctx, start, end, opts...)
	if err != nil {
		return res, err
	}

	out := s.convert.convertPositionsHistory(fills)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdateTime != out[j].UpdateTime {
			return out[i].UpdateTime > out[j].UpdateTime
		}
		return out[i].PositionID > out[j].PositionID
	})

	if s.limit != nil && *s.limit > 0 {
		page := int64(1)
		if s.page != nil && *s.page > 0 {
			page = *s.page
		}

		from := (page - 1) * *s.limit
		if from >= int64(len(out)) {
			return []entity.Futures_PositionsHistory{}, nil
		}
		to := from + *s.limit
		if to > int64(len(out)) {
			to = int64(len(out))
		}
		out = out[from:to]
	}

	s.markLiquidations(ctx, out, start, end, opts...)

	return out, nil
}

// Fills carry no liquidation marker, so closes are matched to the account's liquidation bills; best effort.
func (s *futures_positionsHistory) markLiquidations(ctx context.Context, out []entity.Futures_PositionsHistory, start, end int64, opts ...utils.RequestOption) {
	if len(out) == 0 {
		return
	}
	if end-start > weexIncomeMaxRangeMs {
		start = end - weexIncomeMaxRangeMs
	}

	billTimes := make(map[string][]int64)
	for _, incomeType := range []string{"start_liquidate", "finish_liquidate"} {
		m := utils.Params{"incomeType": incomeType, "startTime": start, "endTime": end, "limit": 100}
		if s.symbol != nil && *s.symbol != "" {
			m["symbol"] = *s.symbol
		}

		r := &utils.Request{
			Method:   http.MethodPost,
			Endpoint: "/capi/v3/account/income",
			SecType:  utils.SecTypeSigned,
		}
		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			continue
		}

		var answ futures_income_Response
		if err := json.Unmarshal(data, &answ); err != nil {
			continue
		}
		for _, item := range answ.Items {
			billTimes[item.Symbol] = append(billTimes[item.Symbol], item.Time)
		}
	}

	for i := range out {
		for _, billTime := range billTimes[out[i].Symbol] {
			if billTime-out[i].UpdateTime <= weexLiquidationMatchMs && out[i].UpdateTime-billTime <= weexLiquidationMatchMs {
				out[i].IsLiquidation = true
				break
			}
		}
	}
}

type futures_income_Response struct {
	Items []struct {
		Symbol string `json:"symbol"`
		Time   int64  `json:"time"`
	} `json:"items"`
}

// fetchFills walks the range in the 7-day windows /capi/v3/userTrades accepts.
func (s *futures_positionsHistory) fetchFills(ctx context.Context, start, end int64, opts ...utils.RequestOption) ([]entity.Futures_UserTrades, error) {
	var fills []entity.Futures_UserTrades
	seen := make(map[string]struct{})

	for windowEnd := end; windowEnd >= start; windowEnd -= weexUserTradesWindowMs {
		lo, hi := windowEnd-weexUserTradesWindowMs+1, windowEnd
		if lo < start {
			lo = start
		}

		for page := 0; page < weexUserTradesMaxPages && lo <= hi; page++ {
			req := (&futures_userTrades{callAPI: s.callAPI, convert: s.convert}).StartTime(lo).EndTime(hi).Limit(weexUserTradesPageLimit)
			if s.symbol != nil && *s.symbol != "" {
				req.Symbol(*s.symbol)
			}

			trades, err := req.Do(ctx, opts...)
			if err != nil {
				return nil, err
			}

			for _, trade := range trades {
				if _, ok := seen[trade.TradeID]; ok {
					continue
				}
				seen[trade.TradeID] = struct{}{}
				fills = append(fills, trade)
			}

			if len(trades) < weexUserTradesPageLimit {
				break
			}

			// The docs leave the sort order open, so narrow the range from whichever end was served.
			first, last := trades[0].Time, trades[len(trades)-1].Time
			switch {
			case first > last:
				hi = last
			case first < last:
				lo = last
			case first == hi:
				hi = first - 1
			default:
				lo = first + 1
			}
		}
	}

	return fills, nil
}
