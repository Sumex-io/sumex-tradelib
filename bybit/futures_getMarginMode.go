package bybit

import (
	"context"
	"encoding/json"
	"net/http"

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
		Method:   http.MethodGet,
		Endpoint: "/v5/account/info",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}
	// if s.symbol != nil {
	// 	m["symbol"] = *s.symbol
	// }

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result futures_marginMode `json:"result"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	marginMode := string(entity.MarginModeTypeCross)

	if answ.Result.MarginMode == "ISOLATED_MARGIN" {
		marginMode = string(entity.MarginModeTypeIsolated)
	}

	return entity.Futures_MarginMode{MarginMode: marginMode}, nil
}

type futures_marginMode struct {
	MarginMode string `json:"marginMode"`
}
