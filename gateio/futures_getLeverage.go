package gateio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getLeverage struct {
	callAPI    func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert    futures_converts
	marginMode *entity.MarginModeType
	symbol     *string
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
		Method: http.MethodGet,
		// Endpoint: "/api/v4/margin/user/account",
		Endpoint: "/api/v4/futures/{settle}/positions/{contract}",
		SecType:  utils.SecTypeSigned,
	}

	settleDefault := "usdt"

	r.Endpoint = strings.Replace(r.Endpoint, "{settle}", settleDefault, 1)

	m := utils.Params{}
	if s.symbol != nil {
		// m["currency_pair"] = *s.symbol
		res.Symbol = *s.symbol
		r.Endpoint = strings.Replace(r.Endpoint, "{contract}", *s.symbol, 1)
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ []futures_leverage

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ) == 0 {
		return res, errors.New("Zero Answers")
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			res.Leverage = answ[0].Cross_leverage_limit
			res.LongLeverage = answ[0].Cross_leverage_limit
			res.ShortLeverage = answ[0].Cross_leverage_limit
		case entity.MarginModeTypeIsolated:
			res.Leverage = answ[0].Leverage
			res.LongLeverage = answ[0].Leverage
			res.ShortLeverage = answ[0].Leverage
		}
	} else {
		return res, errors.New("Invalid marginMode")
	}

	return res, nil
	// return s.convert.convertLeverage(answ[0]), nil
}

type futures_leverage struct {
	Currency_pair        string `json:"currency_pair"`
	Сontract             string `json:"contract"`
	Leverage             string `json:"leverage"`
	Cross_leverage_limit string `json:"cross_leverage_limit"`
}

// type futures_leverage struct {
// 	Currency_pair        string `json:"currency_pair"`
// 	Сontract             string `json:"contract"`
// 	Leverage             string `json:"leverage"`
// 	Cross_leverage_limit string `json:"cross_leverage_limit"`
// }
