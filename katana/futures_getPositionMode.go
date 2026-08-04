package katana

import (
	"context"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

// futures_getPositionMode answers from a Katana invariant — no hedge mode, always one-way — so it
// is not wired to the transport at all and cannot fail.
type futures_getPositionMode struct {
	convert futures_converts
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	return s.convert.convertPositionMode(), nil
}
