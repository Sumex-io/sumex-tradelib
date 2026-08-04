package katana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getBalance struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
}

// Do reads the resolved wallet's collateral (GET /v1/wallets, scoped to that one wallet).
// includePositions is false: the balance row needs none of them, and asking for them would put a
// whole positions page on every balance poll.
func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
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
		Endpoint: "/v1/wallets",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":            nonce,
		"wallet":           wallet,
		"includePositions": "false",
	})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ []katanaWallet
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	if len(answ) == 0 {
		return res, fmt.Errorf("katana: wallet %s not found", wallet)
	}

	return s.convert.convertBalance(answ[0]), nil
}
