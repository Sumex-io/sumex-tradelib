package sumex

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_getBalance struct {
	callAPI callAPIFunc
}

// Do reads the balance off GET /v1/positions rather than /v1/account/holdings: the positions
// response carries Orderly's account-level total_collateral_value and free_collateral next to the
// rows the unrealized PnL is computed from, so one call yields a self-consistent snapshot.
func (s *futures_getBalance) Do(ctx context.Context) (res []entity.FuturesBalance, err error) {
	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/positions", signed: true})
	if err != nil {
		return res, err
	}
	var answ positionsResponse
	if err := json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	return toBalance(answ), nil
}
