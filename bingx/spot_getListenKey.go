package bingx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type spot_getListenKey struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert spot_converts
}

func (s *spot_getListenKey) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Spot_ListenKey, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/openApi/user/auth/userDataStream",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := spot_listenKey{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertListenKey(answ), nil
}

type spot_listenKey struct {
	ListenKey string `json:"listenKey"`
}
