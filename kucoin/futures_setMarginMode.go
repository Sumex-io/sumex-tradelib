package kucoin

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol     *string
	marginMode *entity.MarginModeType
}

func (s *futures_setMarginMode) Symbol(symbol string) *futures_setMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setMarginMode) MarginMode(marginMode entity.MarginModeType) *futures_setMarginMode {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v2/position/changeMarginMode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["marginMode"] = "CROSS"
		case entity.MarginModeTypeIsolated:
			m["marginMode"] = "ISOLATED"
		}
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_marginMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return entity.Futures_MarginMode{MarginMode: strings.ToUpper(answ.Result.MarginMode)}, nil
}
