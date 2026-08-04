package bingx

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
		Endpoint: "/openApi/swap/v2/trade/leverage",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.leverage != nil {
		m["leverage"] = *s.leverage
	}

	if s.positionSide != nil {
		m["side"] = strings.ToUpper(string(*s.positionSide))
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_leverage `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	r2 := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/openApi/swap/v2/trade/leverage",
		SecType:  utils.SecTypeSigned,
	}

	m2 := utils.Params{}
	if s.symbol != nil {
		m2["symbol"] = *s.symbol
	}
	r2.SetParams(m2)

	data2, _, err2 := s.callAPI(ctx, r2, opts...)
	if err2 != nil {
		return res, err2
	}
	var answ2 struct {
		Result futures_leverage `json:"data"`
	}

	err2 = json.Unmarshal(data2, &answ2)
	if err2 != nil {
		return res, err2
	}

	return s.convert.convertLeverage(answ2.Result), nil

}

// type futures_leverage struct {
// 	Symbol           string `json:"symbol"`
// 	LongLeverage     int    `json:"longLeverage"`
// 	MaxLongLeverage  int    `json:"maxLongLeverage"`
// 	MaxShortLeverage int    `json:"maxShortLeverage"`
// }
