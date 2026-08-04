package blofin

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
	convert futures_converts

	mode *entity.PositionModeType
}

func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	if s.mode == nil {
		return res, errors.New("position mode is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v1/account/set-position-mode",
		SecType:  utils.SecTypeSigned,
	}

	var modeStr string
	switch *s.mode {
	case entity.PositionModeTypeHedge:
		modeStr = "long_short_mode"
	case entity.PositionModeTypeOneWay:
		modeStr = "net_mode"
	default:
		return res, errors.New("unsupported position mode: " + string(*s.mode))
	}

	form := utils.Params{
		"positionMode": modeStr,
	}
	r.SetFormParams(form)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// по аналогии с GET: "data": { "positionMode": "..." }
	var answ struct {
		Data *futures_positionMode `json:"data"`
	}

	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}

	if answ.Data == nil {
		return res, errors.New("Zero Answer")
	}

	return s.convert.convertPositionMode(*answ.Data), nil
}
