package weex

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
		Endpoint: "/api/v3/account/",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ spot_balanceResponse
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ.Balances), nil
}

type spot_balanceResponse struct {
	Balances []spot_Balance `json:"balances"`
}

type spot_Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}
