package weex

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Sumex-io/sumex-tradelib/entity"
	"github.com/Sumex-io/sumex-tradelib/utils"
)

type futures_setPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)

	symbol     *string
	mode       *entity.PositionModeType
	marginMode *entity.MarginModeType
}

func (s *futures_setPositionMode) Symbol(symbol string) *futures_setPositionMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setPositionMode) Mode(mode entity.PositionModeType) *futures_setPositionMode {
	s.mode = &mode
	return s
}

func (s *futures_setPositionMode) MarginMode(marginMode entity.MarginModeType) *futures_setPositionMode {
	s.marginMode = &marginMode
	return s
}
func (s *futures_setPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol is required")
	}
	if s.mode == nil {
		return res, errors.New("mode is required")
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

	if *s.marginMode == entity.MarginModeTypeCross {
		m["marginType"] = "CROSSED"
	} else if *s.marginMode == entity.MarginModeTypeIsolated {
		m["marginType"] = "ISOLATED"
	} else {
		return res, errors.New("unsupported margin mode")
	}

	if *s.mode == entity.PositionModeTypeOneWay {
		m["separatedType"] = "COMBINED"
	} else if *s.mode == entity.PositionModeTypeHedge {
		m["separatedType"] = "SEPARATED"
	} else {
		return res, errors.New("unsupported position mode")
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Msg         string `json:"msg"`
		RequestTime int64  `json:"requestTime"`
		Code        string `json:"code"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.Code != "200" && answ.Code != "00000" && answ.Code != "" {
		if answ.Msg != "" {
			return res, errors.New(answ.Msg)
		}
		return res, errors.New("set position mode failed")
	}

	return entity.Futures_PositionsMode{
		HedgeMode: *s.mode == entity.PositionModeTypeHedge,
	}, nil
}
