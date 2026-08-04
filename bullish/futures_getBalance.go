package bullish

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
	uid     *string
}

func (s *futures_getBalance) UID(uid string) *futures_getBalance {
	s.uid = &uid
	return s
}

func (s *futures_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.FuturesBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/trading-api/v1/accounts/asset",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.uid != nil {
		m["tradingAccountId"] = *s.uid
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := []futures_Balance{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res = s.convert.convertBalance(answ)
	return res, nil
}

type futures_Balance struct {
	AssetSymbol       string `json:"assetSymbol"`
	AvailableQuantity string `json:"availableQuantity"`
	LockedQuantity    string `json:"lockedQuantity"`
}
