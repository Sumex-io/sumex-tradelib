package weex

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

	symbol       *string
	leverage     *string
	positionSide *entity.PositionSideType
	marginMode   *entity.MarginModeType
}

func (s *futures_setLeverage) Symbol(symbol string) *futures_setLeverage {
	s.symbol = &symbol
	return s
}

func (s *futures_setLeverage) Leverage(leverage string) *futures_setLeverage {
	s.leverage = &leverage
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
	if s.symbol == nil || *s.symbol == "" {
		return res, errors.New("symbol is required")
	}
	if s.leverage == nil || *s.leverage == "" {
		return res, errors.New("leverage is required")
	}

	r := &utils.Request{
		Method:   http.MethodPost,
		Endpoint: "/capi/v3/account/leverage",
		SecType:  utils.SecTypeSigned,
	}
	m := utils.Params{
		"symbol": *s.symbol,
	}

	if s.marginMode != nil {
		switch *s.marginMode {
		case entity.MarginModeTypeCross:
			m["marginType"] = "CROSSED"
			m["crossLeverage"] = *s.leverage
		case entity.MarginModeTypeIsolated:
			m["marginType"] = "ISOLATED"
			m["isolatedLongLeverage"] = *s.leverage
			m["isolatedShortLeverage"] = *s.leverage
		default:
			return res, errors.New("unsupported margin mode")
		}
	}

	r.SetFormParams(m)

	data, _, err := s.callAPI(ctx, r, opts...)
	if err != nil {
		return res, err
	}
	var answ futures_leverage
	err = json.Unmarshal(data, &answ)
	if err != nil {
		return res, err
	}

	return s.convert.convertLeverage([]futures_leverage{answ}), nil
}
