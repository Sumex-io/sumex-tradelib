package okx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Betarost/onetrades/entity"
	"github.com/Betarost/onetrades/utils"
)

type futures_getMarginMode struct {
	callAPI func(ctx context.Context, r *utils.Request, opts ...utils.RequestOption) (data []byte, header *http.Header, err error)
	convert futures_converts

	symbol    *string
	tradeMode *entity.MarginModeType
}

func (s *futures_getMarginMode) Symbol(symbol string) *futures_getMarginMode {
	s.symbol = &symbol
	return s
}

func (s *futures_getMarginMode) TradeMode(tradeMode entity.MarginModeType) *futures_getMarginMode {
	s.tradeMode = &tradeMode
	return s
}

func (s *futures_getMarginMode) Do(ctx context.Context, opts ...utils.RequestOption) (res entity.Futures_MarginMode, err error) {
	r := &utils.Request{
		Method:   http.MethodGet,
		Endpoint: "/api/v5/account/leverage-info",
		SecType:  utils.SecTypeSigned,
	}

	m := utils.Params{}

	if s.symbol != nil {
		m["instId"] = *s.symbol
	}

	if s.tradeMode != nil {
		switch *s.tradeMode {
		case entity.MarginModeTypeCross:
			m["mgnMode"] = "cross"
		case entity.MarginModeTypeIsolated:
			m["mgnMode"] = "isolated"
		}
	}

	r.SetParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}

	var answ struct {
		Result []futures_leverage `json:"data"`
	}

	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	if len(answ.Result) == 0 {
		return res, errors.New("Zero Answer")
	}

	marginMode := string(entity.MarginModeTypeCross)

	// if answ.Result.MarginMode == "isolated" {
	// 	marginMode = "isolated"
	// }

	return entity.Futures_MarginMode{MarginMode: marginMode}, nil
	// return s.convert.convertLeverage(answ.Result[0]), nil
}

type futures_marginMode struct {
	InstId string `json:"instId"`
	Lever  string `json:"lever"`
}
