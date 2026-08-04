package blofin

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
	// базовая валидация параметров
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol is required")
	}
	if s.leverage == nil || *s.leverage == "" {
		return res, errors.New("leverage is required")
	}
	if s.marginMode == nil {
		return res, errors.New("margin mode is required")
	}

	// ============= 1. SET LEVERAGE =============
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v1/account/set-leverage",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"instId":   *s.symbol,
		"leverage": *s.leverage,
	}

	// marginMode: cross / isolated
	switch *s.marginMode {
	case entity.MarginModeTypeCross:
		m["marginMode"] = "cross"
	case entity.MarginModeTypeIsolated:
		m["marginMode"] = "isolated"
	default:
		return res, errors.New("unsupported margin mode: " + string(*s.marginMode))
	}

	// positionSide: net / long / short
	if s.positionSide != nil {
		// твой enum — типа LONG / SHORT / BOTH, мапим:
		ps := strings.ToLower(string(*s.positionSide))
		// по Blofin: one-way → "net"
		if ps == "both" {
			ps = "net"
		}
		m["positionSide"] = ps
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// ожидание: {"code":"0","msg":"success","data":{...}} — можно не разбирать глубоко
	var answ struct {
		Data json.RawMessage `json:"data"`
	}
	if err = json.Unmarshal(data, &answ); err != nil {
		return res, err
	}
	// На этом шаге считаем, что если code==0 (apiError уже проверил), то всё ок.

	// ============= 2. GET LEVERAGE (проверка) =============
	r2 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/account/batch-leverage-info",
		SecType:  utils.SecTypeSigned,
	}

	m2 := utils.Params{
		"instId": *s.symbol,
	}

	switch *s.marginMode {
	case entity.MarginModeTypeCross:
		m2["marginMode"] = "cross"
	case entity.MarginModeTypeIsolated:
		m2["marginMode"] = "isolated"
	}
	r2.SetParams(m2)

	data2, _, err := s.callAPI(ctx, r2, opts...)
	if err != nil {
		return res, err
	}

	var answ2 struct {
		Result []futures_leverage `json:"data"`
	}

	if err = json.Unmarshal(data2, &answ2); err != nil {
		return res, err
	}

	if len(answ2.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	return s.convert.convertLeverage(answ2.Result), nil
}
