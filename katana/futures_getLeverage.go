package katana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getLeverage struct {
	callAPI       func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert       futures_converts
	resolveWallet func(ctx context.Context, opts ...utils.RequestOption) (string, error)
	markets       func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)

	symbol *string
}

func (s *futures_getLeverage) Symbol(symbol string) *futures_getLeverage {
	s.symbol = &symbol
	return s
}

// Do resolves the effective (override-aware) leverage for one market. A nil symbol is rejected:
// entity.Futures_Leverage is single-symbol shaped, so there is no all-markets answer to give.
//
// Deliberately NOT using resolveIMFOverridesBestEffort: everywhere else the override merely
// decorates a row whose payload comes from elsewhere, but here the override IS the answer. A wallet
// that set 5x would otherwise be told "20x" — the market default — with no error, feeding a
// silently wrong leverage into the position-sizing UI.
func (s *futures_getLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	if s.symbol == nil || strings.TrimSpace(*s.symbol) == "" {
		return res, errors.New("katana: getLeverage requires a symbol")
	}
	sym := strings.TrimSpace(*s.symbol)

	wallet, err := s.resolveWallet(ctx, opts...)
	if err != nil {
		return res, err
	}
	mkts, err := s.markets(ctx, opts...)
	if err != nil {
		return res, err
	}
	market, ok := mkts[sym]
	if !ok {
		return res, fmt.Errorf("katana: market not found: %s", sym)
	}

	overrides, err := resolveIMFOverrides(ctx, s.callAPI, wallet, sym, opts...)
	if err != nil {
		return res, err
	}
	effectiveIMF := market.InitialMarginFraction
	if ov, hasOverride := overrides[sym]; hasOverride && strings.TrimSpace(ov) != "" {
		effectiveIMF = ov
	}

	leverage, err := imfToLeverage(effectiveIMF)
	if err != nil {
		return res, err
	}

	return s.convert.convertLeverage(sym, leverage), nil
}

// ===============IMF OVERRIDES=================
//
// The initialMarginFractionOverride lookup is Katana's only leverage control, so it lives with the
// leverage action even though positions and the two history actions also decorate their rows with
// it. It is a free function rather than a client method for the same reason hyperliquid's
// resolvePerpAsset is: it needs nothing from the client but its transport, and caching it would
// serve a stale leverage right after a setLeverage call.

// katanaIMFOverride is the GET/POST /v1/initialMarginFractionOverride response object. GET's exact
// shape is undocumented (API_NOTES.md Gap 7) beyond reusing this one inside an array. The field is
// null when no override is set, and encoding/json leaves a plain string "" on a null, so no pointer
// handling is needed.
type katanaIMFOverride struct {
	Wallet                        string `json:"wallet"`
	Market                        string `json:"market"`
	InitialMarginFractionOverride string `json:"initialMarginFractionOverride"`
}

// resolveIMFOverrides returns symbol -> override for the markets that actually have one set. A
// market absent from the map has no override, and callers fall back to its own default IMF.
func resolveIMFOverrides(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, market string,
	opts ...utils.RequestOption,
) (map[string]string, error) {
	nonce, _, err := newNonce()
	if err != nil {
		return nil, err
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/initialMarginFractionOverride",
		SecType:  utils.SecTypeSigned,
	}
	r.SetParams(utils.Params{
		"nonce":  nonce,
		"wallet": wallet,
	})
	if market != "" {
		r.SetParam("market", market)
	}

	data, _, err := callAPI(ctx, r, opts...)
	if err != nil {
		return nil, err
	}

	var raw []katanaIMFOverride
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	out := make(map[string]string, len(raw))
	for _, o := range raw {
		if strings.TrimSpace(o.InitialMarginFractionOverride) == "" {
			continue
		}
		out[o.Market] = o.InitialMarginFractionOverride
	}
	return out, nil
}

// The failure log is rate-limited PER PROCESS, not per client: positions is a poll path and a
// caller may build a fresh client per request, so a once-per-client flag would fire on every poll
// and suppress nothing. Without this, a persistently failing lookup writes roughly a line a second
// per user into the caller's on-disk log.
const imfOverrideLogInterval = 5 * time.Minute

var (
	imfOverrideLogMu    sync.Mutex
	imfOverrideLoggedAt time.Time
)

// logIMFOverrideFailure emits at most one line per imfOverrideLogInterval process-wide. The first
// failure after a quiet window always logs, so an isolated failure is never swallowed.
func logIMFOverrideFailure(wallet, market string, cause error) {
	imfOverrideLogMu.Lock()
	defer imfOverrideLogMu.Unlock()
	if !imfOverrideLoggedAt.IsZero() && time.Since(imfOverrideLoggedAt) < imfOverrideLogInterval {
		return
	}
	imfOverrideLoggedAt = time.Now()
	log.Printf("katana: GET /v1/initialMarginFractionOverride failed, falling back to market default IMF for wallet=%s market=%q (further occurrences suppressed for %s): %v", wallet, market, imfOverrideLogInterval, cause)
}

// resolveIMFOverridesBestEffort is for the actions where the override only decorates a leverage
// number on a row whose payload comes from elsewhere. The GET's response shape is undocumented
// (API_NOTES.md Gap 7), and a hard failure would empty an entire positions or history response over
// a cosmetic field, so on error it logs and returns an empty map — callers then fall back to each
// market's default IMF, exactly as if no override were set. getLeverage does NOT use this.
func resolveIMFOverridesBestEffort(
	ctx context.Context,
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error),
	wallet, market string,
	opts ...utils.RequestOption,
) map[string]string {
	overrides, err := resolveIMFOverrides(ctx, callAPI, wallet, market, opts...)
	if err != nil {
		logIMFOverrideFailure(wallet, market, err)
		return map[string]string{}
	}
	return overrides
}
