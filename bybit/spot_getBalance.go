package bybit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
}

func (s *spot_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.AssetsBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v5/account/wallet-balance",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"accountType": "UNIFIED",
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	// log.Println("=SPOT=", string(data))
	var answ struct {
		Result struct {
			List []spot_Balance `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ.Result.List), nil
}

type spot_Balance struct {
	Coin []struct {
		Coin            string `json:"coin"`
		WalletBalance   string `json:"walletBalance"`
		Locked          string `json:"locked"`
		Equity          string `json:"equity"`
		TotalPositionMM string `json:"totalPositionMM"`
		TotalPositionIM string `json:"totalPositionIM"`
		UnrealisedPnl   string `json:"unrealisedPnl"`
	} `json:"coin"`
}
