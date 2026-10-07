package weex

import (
	"context"
	"errors"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_setPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol     *string
	mode       *entity.PositionModeType
	marginMode *entity.MarginModeType
}

func (s *futures_setPositionMode) Symbol(symbol string) *futures_setPositionMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) MarginMode(marginMode entity.MarginModeType) *futures_setPositionMode {
	s.marginMode = &marginMode
	return s
}
func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	// WEEX v3 only reads dualSidePosition; marginType's separatedType is merged/split positions, not hedge.
	return res, errors.New("Exchange does not support this method.")
}
