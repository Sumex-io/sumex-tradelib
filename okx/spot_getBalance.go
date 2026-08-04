package okx

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
		Endpoint: "/api/v5/account/balance",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result []spot_Balance `json:"data"`
	}
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertBalance(answ.Result), nil
}

type spot_Balance struct {
	Details []struct {
		Ccy           string `json:"ccy"`
		Eq            string `json:"eq"`
		CashBal       string `json:"cashBal"`
		IsoEq         string `json:"isoEq,omitempty"`
		AvailEq       string `json:"availEq,omitempty"`
		DisEq         string `json:"disEq"`
		AvailBal      string `json:"availBal"`
		FrozenBal     string `json:"frozenBal"`
		OrdFrozen     string `json:"ordFrozen"`
		Liab          string `json:"liab,omitempty"`
		Upl           string `json:"upl,omitempty"`
		UplLib        string `json:"uplLib,omitempty"`
		CrossLiab     string `json:"crossLiab,omitempty"`
		IsoLiab       string `json:"isoLiab,omitempty"`
		MgnRatio      string `json:"mgnRatio,omitempty"`
		Interest      string `json:"interest,omitempty"`
		Twap          string `json:"twap,omitempty"`
		MaxLoan       string `json:"maxLoan,omitempty"`
		EqUsd         string `json:"eqUsd"`
		NotionalLever string `json:"notionalLever,omitempty"`
		StgyEq        string `json:"stgyEq"`
		IsoUpl        string `json:"isoUpl,omitempty"`
		UTime         string `json:"uTime"`
	} `json:"details"`
}
