package bitget

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	leverage      *string
	longLeverage  *string
	shortLeverage *string
	positionSide  *entity.PositionSideType
	marginMode    *entity.MarginModeType
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
	return s
}

func (s *futures_setLeverage) LongLeverage(longLeverage string) *futures_setLeverage {
	s.longLeverage = &longLeverage
	return s
}

func (s *futures_setLeverage) ShortLeverage(shortLeverage string) *futures_setLeverage {
	s.shortLeverage = &shortLeverage
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
		Endpoint: "/api/v2/mix/account/set-leverage",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"productType": "USDT-FUTURES", "marginCoin": "USDT"}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	// if s.leverage != nil {
	// 	m["leverage"] = *s.leverage
	// }

	if s.longLeverage != nil && s.shortLeverage != nil {
		m["longLeverage"] = *s.longLeverage
		m["shortLeverage"] = *s.shortLeverage
	} else if s.leverage != nil {
		m["leverage"] = *s.leverage
	}

	if s.positionSide != nil {
		m["holdSide"] = strings.ToLower(string(*s.positionSide))
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_leverage_extra `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertLeverage_extra(answ.Result), nil
}

type futures_leverage_extra struct {
	Symbol                string `json:"symbol"`
	MarginMode            string `json:"marginMode"`
	CrossedMarginLeverage string `json:"crossMarginLeverage"`
	IsolatedLongLever     string `json:"longLeverage"`
	IsolatedShortLever    string `json:"shortLeverage"`
}
