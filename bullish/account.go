package bullish

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// ===================NewGenerateJWT==================

type generateJWT struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert account_converts
}

func (s *generateJWT) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_ListenKey, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/trading-api/v1/users/hmac/login",
		SecType:  utils.SecTypeSigned,
	}

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	answ := response_generateJWT{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}
	res.ListenKey = answ.Token
	return res, nil
}

type response_generateJWT struct {
	Token string `json:"token"`
}
