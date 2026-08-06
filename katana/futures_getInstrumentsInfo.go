package katana

import (
	"context"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_getInstrumentsInfo struct {
	convert futures_converts
	markets func(ctx context.Context, opts ...utils.RequestOption) (map[string]Market, error)
}

// Do returns the tradable perpetual catalog. It is the one action with no request of its own: the
// catalog is the client's cached markets() call, and convertInstrumentsInfo sorts by symbol because
// that call is backed by a Go map, whose iteration order is randomized — an unsorted response would
// differ between identical calls.
func (s *futures_getInstrumentsInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.Futures_InstrumentsInfo, err error) {
	mkts, err := s.markets(ctx, opts...)
	if err != nil {
		return res, err
	}

	return s.convert.convertInstrumentsInfo(mkts), nil
}
