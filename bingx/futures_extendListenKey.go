package bingx

import (
	"context"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_extendListenKey struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	listenKey *string
}

func (s *futures_extendListenKey) ListenKey(listenKey string) *futures_extendListenKey {
	s.listenKey = &listenKey
	return s
}

func (s *futures_extendListenKey) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_ListenKey, err error) {
	r := &utils.Request{
		Method:   http.MethodPut,
		Endpoint: "/openApi/user/auth/userDataStream",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.listenKey != nil {
		m["listenKey"] = *s.listenKey
		res.ListenKey = *s.listenKey
	}

	r.SetParams(m)

	_, _, err = s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	return res, err
}
