package katana

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getPositions struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)

	symbol *string
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

// Do maps GET /v1/positions. Zero-quantity rows are dropped by convertPositions: Katana returns
// closed positions with quantity "0", which would otherwise render as a phantom size-0 LONG. The
// IMF override lookup is best-effort — it only decorates a leverage number and must never empty the
// whole positions panel.
func (s *futures_getPositions) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_Positions, err error) {
	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	mkts, err := s.markets(ctx, opts...)
	if err != nil {
		return res, err
	}

	nonce, _, err := newNonce()
	if err != nil {
		return res, err
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/positions",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
	})

	marketFilter := ""
	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		marketFilter = strings.TrimSpace(*s.symbol)
		r.SetParam("market", marketFilter)
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []katanaPosition
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	overrides := resolveIMFOverridesBestEffort(ctx, s.callAPI, wallet, marketFilter, opts...)

	return s.convert.convertPositions(answ, mkts, overrides), nil
}
