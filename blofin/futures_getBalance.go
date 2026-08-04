package blofin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts
}

func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/account/balance",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// data: {
	//   "code":"0",
	//   "msg":"success",
	//   "data":{
	//     "ts":"...",
	//     "totalEquity":"...",
	//     "isolatedEquity":"...",
	//     "details":[ {...}, {...} ]
	//   }
	// }

	var answ struct {
		Data futures_Balance `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	// конвертер ожидает []futures_Balance, как на OKX, поэтому оборачиваем в слайс
	return s.convert.convertBalance([]futures_Balance{answ.Data}), nil
}

// структура под Blofin /api/v1/account/balance
type futures_Balance struct {
	Ts             string `json:"ts"`
	TotalEquity    string `json:"totalEquity"`
	IsolatedEquity string `json:"isolatedEquity"`

	Details []struct {
		Currency              string `json:"currency"`
		Equity                string `json:"equity"`
		Balance               string `json:"balance"`
		Ts                    string `json:"ts"`
		IsolatedEquity        string `json:"isolatedEquity"`
		Available             string `json:"available"`
		AvailableEquity       string `json:"availableEquity"`
		Frozen                string `json:"frozen"`
		OrderFrozen           string `json:"orderFrozen"`
		EquityUsd             string `json:"equityUsd"`
		IsolatedUnrealizedPnl string `json:"isolatedUnrealizedPnl"`
		Bonus                 string `json:"bonus"`
	} `json:"details"`
}
