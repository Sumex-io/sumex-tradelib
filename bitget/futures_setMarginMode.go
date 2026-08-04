package bitget

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_setMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol       *string
	leverage     *string
	positionSide *entity.PositionSideType
	marginMode   *entity.MarginModeType
}

func (s *futures_setMarginMode) Symbol(symbol string) *futures_setMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_setMarginMode) MarginMode(marginMode entity.MarginModeType) *futures_setMarginMode {
	s.marginMode = &marginMode
	return s
}

func (s *futures_setMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/api/v2/mix/account/set-margin-mode",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{"productType": "USDT-FUTURES", "marginCoin": "USDT"}

	if s.symbol != nil {
		m["symbol"] = *s.symbol
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["marginMode"] = "crossed"
		case entity.MarginModeTypeIsolated:
			m["marginMode"] = "isolated"
		}
	}
	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result futures_marginMode `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	marginMode := string(entity.MarginModeTypeCross)

	if answ.Result.MarginMode == "isolated" {
		marginMode = string(entity.MarginModeTypeIsolated)
	}

	return entity.Futures_MarginMode{MarginMode: marginMode}, nil
}
