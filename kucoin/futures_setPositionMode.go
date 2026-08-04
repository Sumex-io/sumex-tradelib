package kucoin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	mode *entity.PositionModeType
}

func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v2/position/switchPositionMode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.mode != nil {
		switch *s.mode {
		case entity.PositionModeTypeHedge:
			m["positionMode"] = 1
		case entity.PositionModeTypeOneWay:
			m["positionMode"] = 0
		}
	}
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result futures_positionMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	b := false

	if answ.Result.PositionMode == 1 {
		b = true
	}
	return entity.Futures_PositionsMode{HedgeMode: b}, nil
}
