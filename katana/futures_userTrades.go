package katana

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_userTrades struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)

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

// Do maps GET /v1/fills. No client-side filter drops rows here, so unlike positionsHistory a single
// request cannot under-fill a page and no paging loop is needed. Sorted newest-first for the same
// pager reason as the other history actions.
func (s *futures_userTrades) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_UserTrades, err error) {
	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}

	nonce, _, err := newNonce()
	if err != nil {
		return res, err
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/fills",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
	})
	applyHistoryPaging(r, s.symbol, s.startTime, s.endTime, s.limit)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []katanaFill
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	out := s.convert.convertUserTrades(answ)

	sortDescendingByTime(out, func(t entity.Futures_UserTrades) int64 { return t.Time })
	return out, nil
}
