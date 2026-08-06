package katana

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getOrderList struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)

	symbol *string
}

const katanaOrdersPageCap = katanaMaxPageLimit

func (s *futures_getOrderList) Symbol(symbol string) *futures_getOrderList {
	s.symbol = &symbol
	return s
}

// Do maps GET /v1/orders (closed=false, set explicitly rather than relied on). Both `limit` and
// `market` are load-bearing: without them the endpoint's default of 50 applies ACROSS EVERY MARKET,
// so a wallet with brackets on several symbols silently loses the tail — and a stop-loss past it
// renders as absent, making a protected position look unprotected. Katana offers no usable cursor
// here (`fromId` has no ordering guarantee to page against), so a full page is reported in the log
// rather than paged through.
func (s *futures_getOrderList) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_OrdersList, err error) {
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
		Endpoint: "/v1/orders",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
		"closed": "false",
	})

	marketFilter := ""
	if s.symbol != nil && strings.TrimSpace(*s.symbol) != "" {
		marketFilter = strings.TrimSpace(*s.symbol)
		r.SetParam("market", marketFilter)
	}
	r.SetParam("limit", strconv.Itoa(katanaOrdersPageCap))

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []katanaOrder
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	if len(answ) >= katanaOrdersPageCap {
		log.Printf("katana: GET /v1/orders returned a full %d-order page for wallet=%s market=%q; any further open orders are NOT in this response", katanaOrdersPageCap, wallet, marketFilter)
	}

	return s.convert.convertOrderList(answ), nil
}
