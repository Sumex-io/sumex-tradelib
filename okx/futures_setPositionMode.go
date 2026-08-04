package okx

import (
	"context"
	"encoding/json"
	"errors"
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
		Endpoint: "/api/v5/account/set-position-mode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.mode != nil {
		if *s.mode == entity.PositionModeTypeHedge {
			m["posMode"] = "long_short_mode"
		} else if *s.mode == entity.PositionModeTypeOneWay {
			m["posMode"] = "net_mode"
		}
	}
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_positionMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	b := false

	if answ.Result[0].PosMode == "long_short_mode" {
		b = true
	}
	return entity.Futures_PositionsMode{HedgeMode: b}, nil
}
