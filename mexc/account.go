package mexc

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===================GetAccountInfo==================
type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v3/account",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := accountInfo{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertAccountInfo(answ), nil
}

type accountInfo struct {
	СanTrade    bool     `json:"canTrade"`
	CanWithdraw bool     `json:"canWithdraw"`
	Permissions []string `json:"permissions"`
}

//===================getFundingAccountBalance==================

type getFundingAccountBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getFundingAccountBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FundingAccountBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/fund/v1/account/balance",
		SecType:  utils.SecTypeSigned,
	}
	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result fundingBalance `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertFundingAccountBalance(answ.Result), nil
}

type fundingBalance struct {
	Assets []struct {
		Asset  string  `json:"asset"`
		Free   float64 `json:"free"`
		Locked float64 `json:"locked"`
	} `json:"assets"`
}

//===================getTradingAccountBalance==================

type getTradingAccountBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getTradingAccountBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.TradingAccountBalance, err error) {
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
		Result []tradingBalance `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertTradingAccountBalance(answ.Result), nil
}

type tradingBalance struct {
	TotalEq            string `json:"totalEq"`
	IsoEq              string `json:"isoEq"`
	AdjEq              string `json:"adjEq,omitempty"`
	AvailEq            string `json:"availEq,omitempty"`
	OrdFroz            string `json:"ordFroz,omitempty"`
	Imr                string `json:"imr,omitempty"`
	Mmr                string `json:"mmr,omitempty"`
	MgnRatio           string `json:"mgnRatio,omitempty"`
	NotionalUsd        string `json:"notionalUsd,omitempty"`
	NotionalUsdForSwap string `json:"notionalUsdForSwap,omitempty"`
	Upl                string `json:"upl,omitempty"`
	UTime              string `json:"uTime"`
}
