package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol       *string
	marginMode   *entity.MarginModeType
	positionMode *entity.PositionModeType
}

func (s *futures_setMarginMode) Symbol(symbol string) *futures_setMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setMarginMode) MarginMode(marginMode entity.MarginModeType) *futures_setMarginMode {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setMarginMode) PositionMode(positionMode entity.PositionModeType) *futures_setMarginMode {
	s.positionMode = &positionMode
	return s
}

func (s *futures_setMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol is required")
	}
	if s.marginMode == nil {
		return res, errors.New("margin mode is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/capi/v3/account/marginType",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"symbol": *s.symbol,
	}

	switch *s.marginMode {
	case entity.MarginModeTypeCross:
		m["marginType"] = "CROSSED"
	case entity.MarginModeTypeIsolated:
		m["marginType"] = "ISOLATED"
	default:
		return res, errors.New("unsupported margin mode")
	}

	if s.positionMode != nil {
		switch *s.positionMode {
		case entity.PositionModeTypeOneWay:
			m["separatedType"] = "COMBINED"
		case entity.PositionModeTypeHedge:
			m["separatedType"] = "SEPARATED"
		}
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Code        string `json:"code"`
		Msg         string `json:"msg"`
		RequestTime int64  `json:"requestTime"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.Code != "200" && answ.Code != "00000" && answ.Code != "" {
		if answ.Msg != "" {
			return res, errors.New(answ.Msg)
		}
		return res, errors.New("set margin mode failed")
	}

	return entity.Futures_MarginMode{MarginMode: string(*s.marginMode)}, nil
}
