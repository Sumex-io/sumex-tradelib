package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle *string
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v4/futures/{settle}/accounts",
		SecType:  utils.SecTypeSigned,
	}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := futures_positionMode{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return entity.Futures_PositionsMode{HedgeMode: answ.In_dual_mode}, nil
}

type futures_positionMode struct {
	In_dual_mode bool `json:"in_dual_mode"`
}
