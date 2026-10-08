package katana

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

// The two account-level actions every exchange in this library is expected to answer. They live
// here rather than under a spot_ prefix because Katana is PERPETUALS ONLY: it lists no spot markets
// at all, so both clients expose exactly these two and no spot trading action exists to be called.

// ===================GetAccountInfo==================

type getAccountInfo struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       account_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	sign          *katanaSigner
	now           func() time.Time
}

// errSessionKeyExpired is the user-facing reason a connection stops trading: Katana delegated
// ("session") keys expire 30 days after authorization and can also be revoked on Katana.
const errSessionKeyExpired = "katana: the Katana session key of this connection has expired or was revoked. Create a new session key on Katana and reconnect this account in Sumex"

// katanaDelegatedKey is one entry of GET /v1/delegatedKeys; `expires` is epoch milliseconds.
type katanaDelegatedKey struct {
	DelegatedKey string `json:"delegatedKey"`
	Expires      int64  `json:"expires"`
}

// Do proves the supplied credentials actually work. resolveWallet exercises the API key/secret;
// delegatedAddress exercises the delegated private key every trade action signs with, so a
// malformed key is refused at connect time instead of surfacing as a raw signing error on the
// user's first order. The delegated key must also still be authorized on Katana: it expires 30
// days after authorization, and an expired key fails every order, cancel and leverage change while
// reads keep working, so the connection would otherwise look healthy. sumex-api re-runs this
// check on its key refresh, which marks such a connection invalid. resolveWallet runs first
// because CanRead is derived from it.
func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	delegated, err := s.sign.delegatedAddress()
	if err != nil {
		return res, fmt.Errorf("katana: the delegated key supplied with this connection is not a usable private key: %w", err)
	}
	if err := s.assertDelegatedKeyAuthorized(ctx, wallet, delegated, opts...); err != nil {
		return res, err
	}

	return s.convert.convertAccountInfo(wallet), nil
}

// assertDelegatedKeyAuthorized looks the delegated key up in GET /v1/delegatedKeys (HMAC, read
// scope) and refuses it when it is missing (revoked, or authorized for another wallet) or expired.
func (s *getAccountInfo) assertDelegatedKeyAuthorized(ctx context.Context, wallet, delegated string, opts ...utils.RequestOption) error {
	nonce, _, err := newNonce()
	if err != nil {
		return err
	}
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/delegatedKeys",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
	})

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return err
	}
	var keys []katanaDelegatedKey
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}

	nowMs := s.now().UnixMilli()
	for _, key := range keys {
		if strings.EqualFold(key.DelegatedKey, delegated) && key.Expires > nowMs {
			return nil
		}
	}
	return fmt.Errorf(errSessionKeyExpired)
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
// entity.SignAuthStream carries only a Signature for it to land in.
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
