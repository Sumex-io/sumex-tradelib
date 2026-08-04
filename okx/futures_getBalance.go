package okx

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
		Endpoint: "/api/v5/account/balance",
		// Endpoint: "/api/v5/asset/balances",
		SecType: utils.SecTypeSigned,
	}

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
	Details []struct {
		Ccy      string `json:"ccy"`
		CashBal  string `json:"cashBal"`
		AvailBal string `json:"availBal"`
		AvailEq  string `json:"availEq,omitempty"`
		Upl      string `json:"upl,omitempty"`
		Eq       string `json:"eq,omitempty"`
	} `json:"details"`
}
