package bitget

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type getAccountInfo struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *getAccountInfo) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.AccountInformation, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v2/spot/account/info",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result accountInfo `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	return s.convert.convertAccountInfo(answ.Result), nil
}

type accountInfo struct {
	UserId      string   `json:"userId"`
	Ips         string   `json:"ips"`
	Authorities []string `json:"authorities"`
}

// ===================SignAuthStream==================

type signAuthStream struct {
	sec       string
	timeStamp *int64
}

func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = &timeStamp
	return s
}

func (s *signAuthStream) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.SignAuthStream, err error) {
	sf, err := utils.SignFunc(utils.KeyTypeHmacBase64)
	if err != nil {
		return res, err
	}

	t := int64(0)

	if s.timeStamp != nil {
		t = *s.timeStamp
	}

	raw := fmt.Sprintf("%dGET/user/verify", t)
	sign, err := sf(s.sec, raw)
	if err != nil {
		return res, err
	}
	res.Signature = *sign
	return res, nil
}
