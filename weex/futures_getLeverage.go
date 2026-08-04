package weex

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
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol is required")
	}

	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/capi/v3/account/symbolConfig",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{
		"symbol": *s.symbol,
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
		return res, errors.New("zero answer")
	}

	if s.marginMode != nil {
		answ[0].MarginType = strings.ToUpper(string(*s.marginMode))
	} else if answ[0].MarginType == "" {
		answ[0].MarginType = "CROSSED"
	}

	return s.convert.convertLeverage(answ), nil
}

type futures_leverage struct {
	Symbol                string `json:"symbol"`
	MarginType            string `json:"marginType"`
	SeparatedType         string `json:"separatedType"`
	CrossLeverage         string `json:"crossLeverage"`
	IsolatedLongLeverage  string `json:"isolatedLongLeverage"`
	IsolatedShortLeverage string `json:"isolatedShortLeverage"`
}
