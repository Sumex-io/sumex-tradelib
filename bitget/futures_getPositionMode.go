package bitget

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getPositionMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol *string
}

func (s *futures_getPositionMode) Symbol(symbol string) *futures_getPositionMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getPositionMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_PositionsMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v2/mix/account/account",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"productType": "USDT-FUTURES", "marginCoin": "USDT"}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}
	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ struct {
		Result futures_positionMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	b := false

	if answ.Result.PosMode == "hedge_mode" {
		b = true
	}

	return entity.Futures_PositionsMode{HedgeMode: b}, nil
}

type futures_positionMode struct {
	PosMode string `json:"posMode"`
}
