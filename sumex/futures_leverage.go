package sumex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

// ===================GetLeverage==================

type futures_getLeverage struct {
	callAPI callAPIFunc

	symbol *string
}

func (s *futures_getLeverage) Symbol(symbol string) *futures_getLeverage {
	s.symbol = &symbol
	return s
}

// Do reads the symbol's leverage setting from GET /v1/account/leverages. A symbol the account has
// never set a leverage on has no row there; Orderly then applies the account-wide leverage, which
// GET /v1/account/info reports as max_leverage, so that is the answer.
func (s *futures_getLeverage) Do(ctx context.Context) (res entity.Futures_Leverage, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, errNoSymbol
	}
	symbol := strings.TrimSpace(*s.symbol)

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/account/leverages", signed: true})
	if err != nil {
		return res, err
	}
	var rows []leverageRow
	if err := decodeRows(data, &rows); err != nil {
		return res, err
	}
	for _, row := range rows {
		if row.Symbol == symbol {
			return leverageResult(symbol, row.Leverage.String(), normalizeMarginMode(row.MarginMode)), nil
		}
	}

	data, err = s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/account/info", signed: true})
	if err != nil {
		return res, err
	}
	var info accountInfoResponse
	if err := json.Unmarshal(data, &info); err != nil {
		return res, err
	}
	if info.MaxLeverage == "" {
		return res, errors.New("sumex: no leverage setting found for " + symbol)
	}
	return leverageResult(symbol, info.MaxLeverage.String(), ""), nil
}

// leverageResult fills Long/ShortLeverage with the same value: Orderly has one leverage per symbol
// and margin mode, not one per side.
func leverageResult(symbol, leverage, marginMode string) entity.Futures_Leverage {
	return entity.Futures_Leverage{
		Symbol:        symbol,
		Leverage:      leverage,
		LongLeverage:  leverage,
		ShortLeverage: leverage,
		MarginMode:    marginMode,
	}
}

// ===================SetLeverage==================

type futures_setLeverage struct {
	callAPI callAPIFunc

	symbol     *string
	leverage   *string
	marginMode *string
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
	return s
}

// MarginMode picks which of the symbol's two leverage settings (cross or isolated) is changed.
// Unset, perp-api changes the one for the symbol's current default margin mode.
func (s *futures_setLeverage) MarginMode(marginMode string) *futures_setLeverage {
	s.marginMode = &marginMode
	return s
}

// Do calls POST /v1/account/leverage. The response is passed through by perp-api unvalidated, so
// the result is built from the request that Orderly accepted rather than read back from it.
func (s *futures_setLeverage) Do(ctx context.Context) (res entity.Futures_Leverage, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, errNoSymbol
	}
	if s.leverage == nil {
		return res, errors.New("sumex: setLeverage requires a leverage")
	}
	symbol := strings.TrimSpace(*s.symbol)
	leverage, err := wireLeverage(*s.leverage)
	if err != nil {
		return res, err
	}
	body := map[string]interface{}{"symbol": symbol, "leverage": leverage}
	marginMode := ""
	if s.marginMode != nil && strings.TrimSpace(*s.marginMode) != "" {
		marginMode, err = wireMarginMode(*s.marginMode)
		if err != nil {
			return res, err
		}
		body["margin_mode"] = marginMode
	}

	if _, err := s.callAPI(ctx, &request{method: http.MethodPost, path: "/v1/account/leverage", body: body, signed: true}); err != nil {
		return res, err
	}
	return leverageResult(symbol, formatInt(leverage), marginMode), nil
}

// ===================GetMarginMode==================

type futures_getMarginMode struct {
	callAPI callAPIFunc

	symbol *string
}

func (s *futures_getMarginMode) Symbol(symbol string) *futures_getMarginMode {
	s.symbol = &symbol
	return s
}

// Do reads the symbol's default margin mode (GET /v1/account/margin-modes). A symbol with no row
// has never been switched and is on Orderly's default, CROSS.
func (s *futures_getMarginMode) Do(ctx context.Context) (res entity.Futures_MarginMode, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, errNoSymbol
	}
	symbol := strings.TrimSpace(*s.symbol)

	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/account/margin-modes", signed: true})
	if err != nil {
		return res, err
	}
	var rows []marginModeRow
	if err := decodeRows(data, &rows); err != nil {
		return res, err
	}
	res.MarginMode = "CROSS"
	for _, row := range rows {
		if row.Symbol == symbol {
			res.MarginMode = normalizeMarginMode(row.DefaultMarginMode)
			break
		}
	}
	return res, nil
}

// ===================SetMarginMode==================

type futures_setMarginMode struct {
	callAPI callAPIFunc

	symbol     *string
	marginMode *string
}

func (s *futures_setMarginMode) Symbol(symbol string) *futures_setMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setMarginMode) MarginMode(marginMode string) *futures_setMarginMode {
	s.marginMode = &marginMode
	return s
}

// Do sets the symbol's default margin mode (POST /v1/account/margin-mode): the mode an order placed
// without an explicit margin mode trades in. It does not move an existing position.
func (s *futures_setMarginMode) Do(ctx context.Context) (res entity.Futures_MarginMode, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, errNoSymbol
	}
	if s.marginMode == nil {
		return res, errors.New("sumex: setMarginMode requires a margin mode")
	}
	marginMode, err := wireMarginMode(*s.marginMode)
	if err != nil {
		return res, err
	}
	body := map[string]interface{}{
		"symbol":              strings.TrimSpace(*s.symbol),
		"default_margin_mode": marginMode,
	}
	if _, err := s.callAPI(ctx, &request{method: http.MethodPost, path: "/v1/account/margin-mode", body: body, signed: true}); err != nil {
		return res, err
	}
	res.MarginMode = marginMode
	return res, nil
}

// ===================GetPositionMode==================

type futures_getPositionMode struct{}

// Do answers from an invariant: Orderly nets one position per symbol (one-way mode only).
func (s *futures_getPositionMode) Do(ctx context.Context) (res entity.Futures_PositionsMode, err error) {
	res.HedgeMode = false
	return res, nil
}

// wireMarginMode validates a margin mode strictly. Unlike normalizeMarginMode, which reads Orderly's
// responses and defaults to CROSS, a write must never turn a typo into a real change.
func wireMarginMode(m string) (string, error) {
	switch v := strings.ToUpper(strings.TrimSpace(m)); v {
	case "CROSS", "ISOLATED":
		return v, nil
	default:
		return "", errors.New("sumex: margin mode must be CROSS or ISOLATED")
	}
}
