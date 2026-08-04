package blofin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

// GET Multiple Leverage (рекомендуемый у Blofin)
// GET /api/v1/account/batch-leverage-info
// Пример: /api/v1/account/batch-leverage-info?instId=BTC-USDT&marginMode=cross

type futures_getLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol     *string
	marginMode *entity.MarginModeType
}

func (s *futures_getLeverage) Symbol(symbol string) *futures_getLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_getLeverage) MarginMode(marginMode entity.MarginModeType) *futures_getLeverage {
	s.marginMode = &marginMode
	return s
}

func (s *futures_getLeverage) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_Leverage, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/account/batch-leverage-info",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	// По доке instId обязателен
	if s.symbol != nil {
		m["instId"] = *s.symbol
	} else {
		return res, errors.New("instId (symbol) is required for Blofin leverage-info")
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["marginMode"] = "cross"
		case entity.MarginModeTypeIsolated:
			m["marginMode"] = "isolated"
		}
	}

	r.SetParams(m)

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

	return s.convert.convertLeverage(answ.Result), nil
}

// Структура под ответ Blofin batch-leverage-info
//
//	{
//	  "leverage": "50",
//	  "marginMode": "cross",
//	  "instId": "BTC-USDT",
//	  "positionSide": "net"
//	}
type futures_leverage struct {
	InstId       string `json:"instId"`
	Leverage     string `json:"leverage"`
	MarginMode   string `json:"marginMode"`
	PositionSide string `json:"positionSide"` // net / long / short
}
