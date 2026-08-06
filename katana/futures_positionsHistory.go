package katana

import (
	"context"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_positionsHistory struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)

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

// Do derives position history from the fills that closed or reduced a position. Katana omits
// `realizedPnL` on a fill "for open actions", so a non-empty RealizedPnL is an exact closing signal
// — where the Hyperliquid connector has to approximate it with a heuristic.
//
// The closing filter necessarily runs client-side, AFTER Katana's own `limit` has capped the raw
// response, so a single request would return roughly half a page on a normal open/close mix and a
// pager would read that short page as end-of-data. fetchClosingFills pages the whole window
// instead, and returns the NEWEST `target` matches already sorted newest-first — a pager holds
// `start` fixed and narrows `end`, so it needs the most recent matches in the window, not the first
// ones encountered scanning forward.
func (s *futures_positionsHistory) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_PositionsHistory, err error) {
	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	mkts, err := s.markets(ctx, opts...)
	if err != nil {
		return res, err
	}

	marketFilter := ""
	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		marketFilter = strings.TrimSpace(*s.symbol)
	}
	target := resolvedLimit(s.limit, 50)

	closing, err := fetchClosingFills(ctx, s.callAPI, wallet, marketFilter, s.startTime, s.endTime, target, opts...)
	if err != nil {
		return res, err
	}

	overrides := resolveIMFOverridesBestEffort(ctx, s.callAPI, wallet, marketFilter, opts...)

	return s.convert.convertPositionsHistory(closing, mkts, overrides), nil
}

// fetchClosingFills collects every closing fill in the window, sorts newest-first, and only THEN
// trims to target. The order matters: fills arrive window-first, not recency-first, so trimming
// before sorting would keep the OLDEST target matches and merely re-sort an already-wrong subset.
func fetchClosingFills(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, market string,
	start, end *int64,
	target int,
	opts ...utils.RequestOption,
) ([]katanaFill, error) {
	all, err := fetchFillsPaged(ctx, callAPI, wallet, market, start, end, opts...)
	if err != nil {
		return nil, err
	}

	closing := make([]katanaFill, 0, len(all))
	for _, f := range all {
		if strings.TrimSpace(f.RealizedPnL) != "" {
			closing = append(closing, f)
		}
	}

	sortDescendingByTime(closing, func(f katanaFill) int64 { return f.Time })
	if len(closing) > target {
		closing = closing[:target]
	}
	return closing, nil
}
