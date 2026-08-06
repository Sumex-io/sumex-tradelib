package hyperliquid

import (
	"context"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_userTrades struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	user            string
	symbol          *string
	orderID         *string
	startTime       *int64
	endTime         *int64
	limit           *int64
	page            *int64
	aggregateByTime *bool
}

func (s *futures_userTrades) Symbol(symbol string) *futures_userTrades {
	s.symbol = &symbol
	return s
}

func (s *futures_userTrades) OrderID(orderID string) *futures_userTrades {
	s.orderID = &orderID
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

func (s *futures_userTrades) Page(page int64) *futures_userTrades {
	s.page = &page
	return s
}

func (s *futures_userTrades) AggregateByTime(v bool) *futures_userTrades {
	s.aggregateByTime = &v
	return s
}

func (s *futures_userTrades) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_UserTrades, err error) {
	fills, err := requestHyperliquidUserFills(ctx, hyperliquidUserTradesParams{
		callAPI:         s.callAPI,
		user:            s.user,
		symbol:          s.symbol,
		orderID:         s.orderID,
		startTime:       s.startTime,
		endTime:         s.endTime,
		limit:           s.limit,
		page:            s.page,
		aggregateByTime: s.aggregateByTime,
		errPrefix:       "hyperliquid futures userTrades",
	}, opts...)
	if err != nil {
		return nil, err
	}

	out := s.convert.convertFuturesUserTrades(fills)
	if s.symbol != nil && *s.symbol != "" {
		filtered := make([]entity.Futures_UserTrades, 0, len(out))
		for _, item := range out {
			if futuresOrdersHistoryMatchSymbol(item.Symbol, *s.symbol) {
				filtered = append(filtered, item)
			}
		}
		out = filtered
	}

	return paginateFuturesUserTrades(out, s.limit, s.page), nil
}

func paginateFuturesUserTrades(in []entity.Futures_UserTrades, limit *int64, page *int64) []entity.Futures_UserTrades {
	if limit == nil || *limit <= 0 {
		return in
	}

	p := int64(1)
	if page != nil && *page > 0 {
		p = *page
	}

	start := (p - 1) * *limit
	if start >= int64(len(in)) {
		return []entity.Futures_UserTrades{}
	}
	end := start + *limit
	if end > int64(len(in)) {
		end = int64(len(in))
	}
	return in[start:end]
}
