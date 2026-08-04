package bybit

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

	symbol   *string
	category *string
	mode     *entity.PositionModeType
}

func (s *futures_setPositionMode) Symbol(symbol string) *futures_setPositionMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setPositionMode) Category(category string) *futures_setPositionMode {
	s.category = &category
	return s
}

func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/v5/position/switch-mode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"category": "linear",
	}

	b := false
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	if s.category != nil {
		m["category"] = *s.category
	}
	if s.mode != nil {
		switch *s.mode {
		case entity.PositionModeTypeHedge:
			m["mode"] = 3
			b = true
			res.HedgeMode = true
		case entity.PositionModeTypeOneWay:
			m["mode"] = 0
		}
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		RetMsg string `json:"retMsg"`
		// Result futures_positionMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.RetMsg != "OK" {
		return res, errors.New("Wrong Answer")
	}

	return entity.Futures_PositionsMode{HedgeMode: b}, nil
}
