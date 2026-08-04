package gateio

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getMarginMode) Symbol(symbol string) *futures_getMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
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
		r.Endpoint = strings.Replace(r.Endpoint, "{contract}", *s.symbol, 1)
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ []futures_margin

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	res.MarginMode = string(entity.MarginModeTypeCross)
	sym := ""
	if s.symbol != nil {
		sym = *s.symbol
	}
	for _, item := range answ {
		if item.Сontract == sym {
			if strings.ToUpper(item.Pos_margin_mode) == "ISOLATED" {
				res.MarginMode = string(entity.MarginModeTypeIsolated)
			}
			break
		}
	}

	return res, nil
}

type futures_margin struct {
	Currency_pair        string `json:"currency_pair"`
	Сontract             string `json:"contract"`
	Leverage             string `json:"leverage"`
	Cross_leverage_limit string `json:"cross_leverage_limit"`
	Pos_margin_mode      string `json:"pos_margin_mode"`
}
