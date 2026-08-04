package okx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol       *string
	leverage     *string
	positionSide *entity.PositionSideType
	marginMode   *entity.MarginModeType
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
	return s
}

func (s *futures_setLeverage) MarginMode(marginMode entity.MarginModeType) *futures_setLeverage {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setLeverage) PositionSide(positionSide entity.PositionSideType) *futures_setLeverage {
	s.positionSide = &positionSide
	return s
}

func (s *futures_setLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v5/account/set-leverage",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.leverage != nil {
		m["lever"] = *s.leverage
	}

	if s.positionSide != nil {
		m["posSide"] = strings.ToLower(string(*s.positionSide))
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["mgnMode"] = "cross"
		case entity.MarginModeTypeIsolated:
			m["mgnMode"] = "isolated"
		}
	}
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_leverage `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	//=== Проверка плечей

	r2 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/account/leverage-info",
		SecType:  utils.SecTypeSigned,
	}

	m2 := utils.Params{}

	if s.symbol != nil {
		m2["instId"] = *s.symbol
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m2["mgnMode"] = "cross"
		case entity.MarginModeTypeIsolated:
			m2["mgnMode"] = "isolated"
		}
	}

	r2.SetParams(m2)

	data2, _, err := s.callAPI(ctx, r2, opts...)
	if err != nil {
		return res, err
	}
	var answ2 struct {
		Result []futures_leverage `json:"data"`
	}

	err = json.Unmarshal(data2, &answ2)
	if err != nil {
		return res, err
	}

	if len(answ2.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	return s.convert.convertLeverage(answ2.Result), nil
}
