package katana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

// The two account-level actions every exchange in this library is expected to answer. They live
// here rather than under a spot_ prefix because Katana is PERPETUALS ONLY: it lists no spot markets
// at all, so both clients expose exactly these two and no spot trading action exists to be called.
// The original connector had to reject spot trading at a dispatcher; here the rejection is
// structural — SpotClient offers no order, position or balance action to reach in the first place.

// ===================GetAccountInfo==================

type getAccountInfo struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       account_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	sign          *katanaSigner
}

// Do proves the supplied credentials actually work. resolveWallet exercises the API key/secret;
// delegatedAddress exercises the delegated private key every trade action signs with, so a
// malformed key is refused at connect time instead of surfacing as a raw signing error on the
// user's first order. resolveWallet runs first because CanRead is derived from it.
func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	if _, err := s.sign.delegatedAddress(); err != nil {
		return res, fmt.Errorf("katana: the delegated key supplied with this connection is not a usable private key: %w", err)
	}

	return s.convert.convertAccountInfo(wallet), nil
}

// ===================SignAuthStream==================

type signAuthStream struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       account_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)

	timeStamp *int64
}

// TimeStamp is accepted for parity with every other connector's signAuthStream, but Katana has
// nothing to do with it: its token is minted server-side with no timestamp input, and
// entity.SignAuthStream carries only a Signature. The original connector echoed the caller's
// timestamp back in its own response struct; that field has no counterpart here.
func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = &timeStamp
	return s
}

// Do mints a private-channel WebSocket auth token (GET /v1/wsToken). Whether the token is scoped
// per wallet or per API key is undocumented (API_NOTES.md Gaps), so the request carries the
// resolved wallet — the only scoping the docs actually specify.
func (s *signAuthStream) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.SignAuthStream, err error) {
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
		Endpoint: "/v1/wsToken",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
	})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := katanaWsTokenResponse{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	return s.convert.convertSignAuthStream(answ), nil
}
