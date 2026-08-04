package binance

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	isCOINM bool
	convert futures_converts

	symbol *string
}

func (s *futures_getMarginMode) Symbol(symbol string) *futures_getMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/fapi/v1/symbolConfig",
		SecType:  utils.SecTypeSigned,
	}

	if s.isCOINM {
		r.Method = http.MethodPost
		r.Endpoint = "/dapi/v1/marginType"

		m := utils.Params{"marginType": "CROSSED"}
		if s.symbol != nil {
			m["symbol"] = *s.symbol
		}

		r.SetFormParams(m)

		data, _, err := s.callAPI(ctx, r, opts...)
		if err != nil {
			if strings.Contains(err.Error(), "No need to change margin type.") {
				return entity.Futures_MarginMode{MarginMode: string(entity.MarginModeTypeCross)}, nil
			}
			return res, err
		}
		var answ struct {
			Msg  string `json:"msg"`
			Code int64  `json:"code"`
		}

		err = json.Unmarshal(data, &answ)
		if err != nil {
			return res, err
		}

		if answ.Code != -4046 {
			return entity.Futures_MarginMode{MarginMode: string(entity.MarginModeTypeCross)}, nil
		}

		if answ.Code != 200 {
			return res, errors.New(answ.Msg)
		}

		return entity.Futures_MarginMode{MarginMode: string(entity.MarginModeTypeCross)}, nil
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
	answ := []futures_marginMode{}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ) == 0 {
		return res, nil
	}
	res.MarginMode = string(entity.MarginModeTypeCross)

	if answ[0].MarginType != "CROSSED" {
		res.MarginMode = string(entity.MarginModeTypeIsolated)
	}

	return res, nil
}

type futures_marginMode struct {
	Symbol     string `json:"symbol"`
	Leverage   int64  `json:"leverage"`
	MarginType string `json:"marginType"`
}
