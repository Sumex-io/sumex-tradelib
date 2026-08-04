package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getBalance struct {
	callAPI   func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM   bool
	isPMargin bool
	convert   futures_converts
}

func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/fapi/v2/account",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/account"
	}

	if s.isPMargin {
		r.Endpoint = "/sapi/v1/portfolio/balance"

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}
		answ := []futures_Balance_PMargin{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}
		return s.convert.convertBalance_PMargin(answ), nil
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := futures_Balance{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertBalance(answ), nil
}

type futures_Balance struct {
	Assets []struct {
		Asset            string `json:"asset"`
		WalletBalance    string `json:"walletBalance"`
		AvailableBalance string `json:"availableBalance"`
		UnrealizedProfit string `json:"unrealizedProfit"`
	} `json:"assets"`
}

type futures_Balance_PMargin struct {
	Asset              string `json:"asset"`
	TotalWalletBalance string `json:"totalWalletBalance"`
	// AvailableBalance string `json:"availableBalance"`
	UmUnrealizedPNL string `json:"umUnrealizedPNL"`
}
