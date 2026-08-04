package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle *string
	mode   *entity.PositionModeType
}

func (s *futures_setPositionMode) Settle(settle string) *futures_setPositionMode {
	s.settle = &settle
	return s
}

func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v4/futures/{settle}/dual_mode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	if s.mode != nil {
		switch *s.mode {
		case entity.PositionModeTypeHedge:
			m["dual_mode"] = true
		case entity.PositionModeTypeOneWay:
			m["dual_mode"] = false
		}
	}
	r.SetParams(m)

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
