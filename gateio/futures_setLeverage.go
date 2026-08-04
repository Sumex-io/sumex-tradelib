package gateio

import (
	"context"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	settle       *string
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
		Endpoint: "/api/v4/futures/{settle}/positions/{contract}/leverage",
		// Endpoint: "/api/v4/margin/leverage/user_market_setting",
		SecType: utils.SecTypeSigned,
	}

	settleDefault := "usdt"

	if s.settle == nil {
		s.settle = &settleDefault
	}

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", *s.settle, 1)

	m := utils.Params{}
	if s.symbol != nil {
		r.Endpoint = strings.Replace(r.Endpoint, "{contract}", *s.symbol, 1)
		// m["currency_pair"] = *s.symbol
	}

	if s.leverage != nil && s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["leverage"] = 0
			m["cross_leverage_limit"] = *s.leverage
		case entity.MarginModeTypeIsolated:
			m["leverage"] = *s.leverage
		}

	}

	// r.SetFormParams(m)
	r.SetParams(m)

	// data, _, err := s.callAPI(ctx, r, opts...)
	_, _, err = s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	// if string(data) != "" {
	// 	return res, errors.New("Not Zero Answers")
	// }
	// var answ futures_leverage
	// err = json.Unmarshal(data, &answ)
	// if err != nil {
	// 	return res, err
	// }
	res.Symbol = *s.symbol
	res.Leverage = *s.leverage
	res.LongLeverage = *s.leverage
	res.ShortLeverage = *s.leverage
	return res, nil
	// return s.convert.convertLeverage(answ), nil
}
