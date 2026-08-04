package gateio

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
		Endpoint: "/api/v4/account/detail",
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
	User_id      int64    `json:"user_id"`
	Ip_whitelist []string `json:"ip_whitelist"`
}

// ===================SignAuthStream==================

type signAuthStream struct {
	sec       string
	timeStamp int64
	channel   string
	event     string
}

func (s *signAuthStream) TimeStamp(timeStamp int64) *signAuthStream {
	s.timeStamp = timeStamp
	return s
}

func (s *signAuthStream) Channel(channel string) *signAuthStream {
	s.channel = channel
	return s
}

func (s *signAuthStream) Event(event string) *signAuthStream {
	s.event = event
	return s
}

func (s *signAuthStream) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.SignAuthStream, err error) {
	sf, err := utils.SignFunc(utils.KeyTypeHmacHex512)
	if err != nil {
		return res, err
	}

	raw := fmt.Sprintf("channel=%s&event=%s&time=%d", s.channel, s.event, s.timeStamp)
	sign, err := sf(s.sec, raw)
	if err != nil {
		return res, err
	}
	res.Signature = *sign
	return res, nil
}
