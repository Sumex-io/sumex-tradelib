package xemus

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
)

type futures_getBalance struct {
	callAPI callAPIFunc
}

// Do reads GET /v1/account/holdings for each collateral token's settled balance and GET
// /v1/positions for the PnL not settled into USDC yet and the account's free collateral.
func (s *futures_getBalance) Do(ctx context.Context) (res []entity.FuturesBalance, err error) {
	data, err := s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/account/holdings", signed: true})
	if err != nil {
		return res, err
	}
	var holdings holdingsResponse
	if err := json.Unmarshal(data, &holdings); err != nil {
		return res, err
	}
	data, err = s.callAPI(ctx, &request{method: http.MethodGet, path: "/v1/positions", signed: true})
	if err != nil {
		return res, err
	}
	var positions positionsResponse
	if err := json.Unmarshal(data, &positions); err != nil {
		return res, err
	}
	return toBalance(holdings, positions), nil
}
