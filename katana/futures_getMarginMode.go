package katana

import (
	"context"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

// futures_getMarginMode answers from a Katana invariant — cross-margin only — so it is not wired to
// the transport at all and cannot fail.
type futures_getMarginMode struct {
	convert futures_converts

	symbol *string
}

// Symbol is accepted so the call site reads the same as every other connector's, but it cannot
// reach the answer: entity.Futures_MarginMode carries only MarginMode, no symbol. The mode is a
// Katana-wide invariant anyway, so no per-symbol answer is being dropped.
func (s *futures_getMarginMode) Symbol(symbol string) *futures_getMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	return s.convert.convertMarginMode(), nil
}
