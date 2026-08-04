package binance

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getLeverage struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
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
		Endpoint: "/fapi/v1/symbolConfig",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Endpoint = "/dapi/v1/positionRisk"

		m := utils.Params{}

		symbol := ""
		if s.symbol != nil {
			symbol = *s.symbol
		}

		r.SetParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			return res, err
		}
		answ := []futures_Position{}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		if len(answ) == 0 || symbol == "" {
			return res, nil
		}
		for _, item := range answ {
			if item.Symbol == symbol {
				return s.convert.convertLeverage_COINM(item), nil
			}
		}
		return res, nil

	}

	m := utils.Params{}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	answ := []futures_leverage{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ) == 0 {
		return res, nil
	}
	return s.convert.convertLeverage(answ[0]), nil
}

type futures_leverage struct {
	Symbol     string `json:"symbol"`
	Leverage   int64  `json:"leverage"`
	MarginType string `json:"marginType"`
}
