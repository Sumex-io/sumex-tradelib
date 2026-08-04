package huobi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getBalance struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts

	uid *string
}

func (s *spot_getBalance) UID(uid string) *spot_getBalance {
	s.uid = &uid
	return s
}

func (s *spot_getBalance) Do(ctx context.Context, opts ...utils.RequestOption) (res []entity.AssetsBalance, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/v1/account/accounts/{account-id}/balance",
		SecType:  utils.SecTypeSigned,
	}

	if s.uid != nil {
		r.Endpoint = strings.Replace(r.Endpoint, "{account-id}", *s.uid, 1)
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result struct {
			List []spot_Balance `json:"list"`
		} `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertBalance(answ.Result.List), nil
}

type spot_Balance struct {
	Currency  string `json:"currency"`
	Type      string `json:"type"`
	Balance   string `json:"balance"`
	Debt      string `json:"debt,omitempty"`
	Available string `json:"available,omitempty"`
	SeqNum    string `json:"seq-num,omitempty"`
}
