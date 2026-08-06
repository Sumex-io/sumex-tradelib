package katana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_setLeverage struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)
	sign          *katanaSigner

	symbol   *string
	leverage *string
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
	return s
}

// katanaIMFOverrideParams is the POST /v1/initialMarginFractionOverride `parameters` object. The
// field is `initialMarginFractionOverride`, matching the EIP-712 field imfOverrideTypedData signs.
type katanaIMFOverrideParams struct {
	Nonce                         string `json:"nonce"`
	Wallet                        string `json:"wallet"`
	DelegatedKey                  string `json:"delegatedKey,omitempty"`
	Market                        string `json:"market"`
	InitialMarginFractionOverride string `json:"initialMarginFractionOverride"`
}

// Do writes an initial-margin-fraction override, because Katana has no leverage concept of its own
// and no set-leverage endpoint: leverage is 1/initialMarginFraction, and the only writable input is
// an override that can TIGHTEN a market's margin requirement but never loosen it below the market's
// own floor. The response reports whatever the exchange echoes back — the server-accepted value —
// falling back to the locally clamped one only if the response omits it.
//
// There is no MarginMode input: Katana is cross-margin only, and the source connector had no such
// field either.
func (s *futures_setLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, errors.New("katana: setLeverage requires a symbol")
	}
	if s.leverage == nil || strings.TrimSpace(*s.leverage) == "" {
		return res, errors.New("katana: setLeverage requires a leverage")
	}
	symbol := strings.TrimSpace(*s.symbol)

	mkts, err := s.markets(ctx, opts...)
	if err != nil {
		return res, err
	}
	market, ok := mkts[symbol]
	if !ok {
		return res, fmt.Errorf("katana: market not found: %s", symbol)
	}

	requestedIMF, err := leverageToIMF(*s.leverage)
	if err != nil {
		return res, err
	}
	effectiveIMF, err := clampIMFToMarketFloor(requestedIMF, market.InitialMarginFraction)
	if err != nil {
		return res, err
	}

	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	delegated, err := s.sign.delegatedAddress()
	if err != nil {
		return res, err
	}
	nonceStr, nonceBig, err := newNonce()
	if err != nil {
		return res, err
	}

	sig, err := s.sign.signIMFOverride(nonceBig, wallet, symbol, effectiveIMF)
	if err != nil {
		return res, err
	}

	body := katanaSignedRequest[katanaIMFOverrideParams]{
		Parameters: katanaIMFOverrideParams{
			Nonce:                         nonceStr,
			Wallet:                        wallet,
			DelegatedKey:                  delegated,
			Market:                        symbol,
			InitialMarginFractionOverride: effectiveIMF,
		},
		Signature: sig,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return res, err
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/v1/initialMarginFractionOverride",
		SecType:  utils.SecTypeSigned,
	}
	r.BodyString = string(b)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := katanaIMFOverride{}
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	confirmedIMF := effectiveIMF
	if strings.TrimSpace(answ.InitialMarginFractionOverride) != "" {
		confirmedIMF = answ.InitialMarginFractionOverride
	}
	confirmedLeverage, err := imfToLeverage(confirmedIMF)
	if err != nil {
		return res, err
	}

	return s.convert.convertLeverage(symbol, confirmedLeverage), nil
}

// clampIMFToMarketFloor enforces both bounds on a leverage override. An override may only TIGHTEN
// ("initial margin fraction may only be increased from the market default"), and since
// leverage = 1/IMF, a higher IMF is a lower leverage — so the market's own default IMF is the
// FLOOR. Asking for more leverage than the market allows clamps up to that floor rather than
// erroring, matching how a sizing UI expects out-of-range values to behave.
func clampIMFToMarketFloor(requestedIMF, marketDefaultIMF string) (string, error) {
	requested, err := parsePositiveDecimal(requestedIMF, "initialMarginFractionOverride")
	if err != nil {
		return "", err
	}
	floor, err := parsePositiveDecimal(marketDefaultIMF, "initialMarginFraction")
	if err != nil {
		return "", err
	}
	// An IMF above 1 means LESS than 1x leverage, which Katana rejects outright ("Override value,
	// between 1 and the market's initialMarginFraction").
	ceiling := big.NewFloat(1)
	switch {
	case requested.Cmp(ceiling) > 0:
		return formatDecimal8(ceiling), nil
	case requested.Cmp(floor) < 0:
		return formatDecimal8(floor), nil
	default:
		return formatDecimal8(requested), nil
	}
}
