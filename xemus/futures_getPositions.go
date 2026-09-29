package xemus

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_getPositions struct {
	callAPI callAPIFunc

	symbol *string
}

func (s *futures_getPositions) Symbol(symbol string) *futures_getPositions {
	s.symbol = &symbol
	return s
}

// Do maps GET /v1/positions. The symbol filter is applied here rather than through
// GET /v1/positions/:symbol, which answers a symbol with no position with a zero-filled row
// (symbol "") instead of nothing. Zero-quantity rows — positions Orderly keeps listing after they
// close — are dropped so they cannot render as a phantom size-0 LONG.
func (s *futures_getPositions) Do(ctx context.Context) (res []entity.Futures_Positions, err error) {
	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/positions", signed: true})
	if err != nil {
		return res, err
	}
	var answ positionsResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	symbol := ""
	if s.symbol != nil {
		symbol = strings.TrimSpace(*s.symbol)
	}
	res = []entity.Futures_Positions{}
	for _, row := range answ.Rows {
		if isZero(row.PositionQty.String()) {
			continue
		}
		if symbol != "" && row.Symbol != symbol {
			continue
		}
		res = append(res, toPosition(row))
	}
	return res, nil
}
