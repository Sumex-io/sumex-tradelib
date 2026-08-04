package bybit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol        *string
	category      *string
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

func (s *futures_setLeverage) Category(category string) *futures_setLeverage {
	s.category = &category
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
		Endpoint: "/v5/position/set-leverage",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"category": "linear"}
	if s.symbol != nil {
		m["symbol"] = *s.symbol
		res.Symbol = *s.symbol
	}
	if s.category != nil {
		m["category"] = *s.category
	}
	// if s.leverage != nil {
	// 	m["buyLeverage"] = *s.leverage
	// 	m["sellLeverage"] = *s.leverage
	// }

	if s.longLeverage != nil && s.shortLeverage != nil {
		m["buyLeverage"] = *s.longLeverage
		m["sellLeverage"] = *s.shortLeverage
		res.LongLeverage = *s.longLeverage
		res.ShortLeverage = *s.shortLeverage
	} else if s.leverage != nil {
		m["buyLeverage"] = *s.leverage
		m["sellLeverage"] = *s.leverage
		res.LongLeverage = *s.leverage
		res.ShortLeverage = *s.leverage
	}

	// if s.positionSide != nil {
	// 	m["side"] = strings.ToUpper(string(*s.positionSide))
	// }

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		RetMsg string `json:"retMsg"`
		// Result futures_leverage `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if answ.RetMsg != "OK" {
		return res, errors.New(answ.RetMsg)
	}
	return res, nil
}

// type futures_leverage struct {
// 	Symbol           string `json:"symbol"`
// 	LongLeverage     int    `json:"longLeverage"`
// 	MaxLongLeverage  int    `json:"maxLongLeverage"`
// 	MaxShortLeverage int    `json:"maxShortLeverage"`
// }
