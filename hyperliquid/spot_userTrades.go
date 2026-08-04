package hyperliquid

import (
	"context"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_userTrades struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	user            string
	symbol          *string
	orderID         *string
	startTime       *int64
	endTime         *int64
	limit           *int64
	page            *int64
	aggregateByTime *bool
}

func (s *spot_userTrades) Symbol(symbol string) *spot_userTrades {
	s.symbol = &symbol
	return s
}

func (s *spot_userTrades) OrderID(orderID string) *spot_userTrades {
	s.orderID = &orderID
	return s
}

func (s *spot_userTrades) StartTime(startTime int64) *spot_userTrades {
	s.startTime = &startTime
	return s
}

func (s *spot_userTrades) EndTime(endTime int64) *spot_userTrades {
	s.endTime = &endTime
	return s
}

func (s *spot_userTrades) Limit(limit int64) *spot_userTrades {
	s.limit = &limit
	return s
}

func (s *spot_userTrades) Page(page int64) *spot_userTrades {
	s.page = &page
	return s
}

func (s *spot_userTrades) AggregateByTime(v bool) *spot_userTrades {
	s.aggregateByTime = &v
	return s
}

func (s *spot_userTrades) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Spot_UserTrades, err error) {
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
		errPrefix:       "hyperliquid spot userTrades",
	}, opts...)
	if err != nil {
		return nil, err
	}

	out := s.convert.convertSpotUserTrades(fills)
	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		want := strings.TrimSpace(*s.symbol)
		filtered := make([]entity.Spot_UserTrades, 0, len(out))
		for _, item := range out {
			if item.Symbol == want {
				filtered = append(filtered, item)
			}
		}
		out = filtered
	}

	return paginateSpotUserTrades(out, s.limit, s.page), nil
}

func paginateSpotUserTrades(in []entity.Spot_UserTrades, limit *int64, page *int64) []entity.Spot_UserTrades {
	if limit == nil || *limit <= 0 {
		return in
	}

	p := int64(1)
	if page != nil && *page > 0 {
		p = *page
	}

	start := (p - 1) * *limit
	if start >= int64(len(in)) {
		return []entity.Spot_UserTrades{}
	}
	end := start + *limit
	if end > int64(len(in)) {
		end = int64(len(in))
	}
	return in[start:end]
}
