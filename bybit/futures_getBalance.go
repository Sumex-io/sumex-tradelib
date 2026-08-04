package bybit

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
		Endpoint: "/v5/account/wallet-balance",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"accountType": "UNIFIED",
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	// log.Println("=Futures=", string(data))
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			List []futures_Balance `json:"list"`
		} `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ.Result.List), nil
}

type futures_Balance struct {
	Coin []struct {
		Coin                string `json:"coin"`
		WalletBalance       string `json:"walletBalance"`
		AvailableToWithdraw string `json:"availableToWithdraw"`
		Equity              string `json:"equity"`
		TotalPositionMM     string `json:"totalPositionMM"`
		TotalPositionIM     string `json:"totalPositionIM"`
		Locked              string `json:"locked"`
		UnrealisedPnl       string `json:"unrealisedPnl"`
	} `json:"coin"`
}
