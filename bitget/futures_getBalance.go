package bitget

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
		Endpoint: "/api/v2/mix/account/accounts",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"productType": "USDT-FUTURES"}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []futures_Balance `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ.Result), nil
}

type futures_Balance struct {
	MarginCoin string `json:"marginCoin"`
	// Balance    string `json:"balance"`
	Available           string `json:"available"`
	СrossedMaxAvailable string `json:"crossedMaxAvailable"`
	UnionAvailable      string `json:"unionAvailable"`
	AccountEquity       string `json:"accountEquity"`
	UnrealizedPL        string `json:"unrealizedPL"`
	AssetList           []struct {
		Coin      string `json:"coin"`
		Balance   string `json:"balance"`
		Available string `json:"available"`
	} `json:"assetList"`
}
