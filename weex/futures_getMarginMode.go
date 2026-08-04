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
	var answ []futures_marginMode
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ) == 0 {
		return res, errors.New("zero answer")
	}

	marginMode := string(entity.MarginModeTypeCross)
	if strings.ToUpper(answ[0].MarginType) == "ISOLATED" {
		marginMode = string(entity.MarginModeTypeIsolated)
	}

	return entity.Futures_MarginMode{MarginMode: marginMode}, nil
}

type futures_marginMode struct {
	Symbol                string `json:"symbol"`
	MarginType            string `json:"marginType"`
	SeparatedType         string `json:"separatedType"`
	CrossLeverage         string `json:"crossLeverage"`
	IsolatedLongLeverage  string `json:"isolatedLongLeverage"`
	IsolatedShortLeverage string `json:"isolatedShortLeverage"`
}
